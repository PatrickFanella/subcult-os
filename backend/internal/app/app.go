package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Greeting(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + "!"
}

type App struct {
	config        Config
	db            *pgxpool.Pool
	mux           *http.ServeMux
	loginMu       sync.Mutex
	loginAttempts map[string]loginAttempt
}

func New(config Config, db *pgxpool.Pool) *App {
	a := &App{config: config, db: db, mux: http.NewServeMux(), loginAttempts: map[string]loginAttempt{}}
	a.routes()
	return a
}

func (a *App) Handler() http.Handler { return a.originGuard(a.mux) }

func (a *App) routes() {
	a.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.mux.HandleFunc("GET /api/ready", a.handleReady)
	a.mux.HandleFunc("POST /api/auth/signup", a.handleSignup)
	a.mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	a.mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	a.mux.HandleFunc("GET /api/me", a.handleMe)
	a.mux.HandleFunc("GET /api/dev/email-outbox", a.handleDevEmailOutbox)
	a.mux.HandleFunc("POST /api/workspaces", a.handleCreateWorkspace)
	a.mux.HandleFunc("GET /api/workspaces/current", a.handleCurrentWorkspace)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}", a.handleGetWorkspace)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/invitations", a.handleCreateInvitation)
	a.mux.HandleFunc("POST /api/invitations/{token}/accept", a.handleAcceptInvitation)
	a.mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/members/{memberID}", a.handleRemoveMember)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/events", a.handleListEvents)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/events", a.handleCreateEvent)
	a.mux.HandleFunc("GET /api/events/{eventID}", a.handleGetEvent)
	a.mux.HandleFunc("PATCH /api/events/{eventID}", a.handleUpdateEvent)
	a.mux.HandleFunc("POST /api/events/{eventID}/publish", a.handlePublishEvent)
	a.mux.HandleFunc("POST /api/events/{eventID}/end-of-night", a.handleEndOfNight)
	a.mux.HandleFunc("GET /api/events/{eventID}/report", a.handleGetReport)
	a.mux.HandleFunc("GET /api/public/events/{slug}", a.handlePublicEvent)
	a.mux.HandleFunc("POST /api/public/events/{slug}/reservations", a.handleReserveTicket)
	a.mux.HandleFunc("GET /api/tickets/{code}", a.handleGetTicket)
	a.mux.HandleFunc("GET /api/events/{eventID}/door/tickets", a.handleDoorTicketSearch)
	a.mux.HandleFunc("POST /api/events/{eventID}/door/check-ins", a.handleDoorCheckIn)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *App) originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestNeedsOriginCheck(r) && !a.allowedOrigin(r) {
			writeError(w, http.StatusForbidden, "origin not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestNeedsOriginCheck(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return false
	}
	if _, err := r.Cookie(authCookieName); err != nil {
		return false
	}
	return r.Header.Get("Origin") != ""
}

func (a *App) allowedOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if strings.EqualFold(parsed.Host, r.Host) {
		return true
	}
	publicWebURL := strings.TrimSpace(a.config.PublicWebURL)
	if publicWebURL == "" {
		return false
	}
	publicParsed, err := url.Parse(publicWebURL)
	if err != nil || publicParsed.Scheme == "" || publicParsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, publicParsed.Scheme) && strings.EqualFold(parsed.Host, publicParsed.Host)
}
