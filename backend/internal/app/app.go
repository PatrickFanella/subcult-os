package app

import (
	"encoding/json"
	"net/http"
	"strings"

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
	config Config
	db     *pgxpool.Pool
	mux    *http.ServeMux
}

func New(config Config, db *pgxpool.Pool) *App {
	a := &App{config: config, db: db, mux: http.NewServeMux()}
	a.routes()
	return a
}

func (a *App) Handler() http.Handler { return a.mux }

func (a *App) routes() {
	a.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.mux.HandleFunc("POST /api/auth/signup", a.handleSignup)
	a.mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	a.mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	a.mux.HandleFunc("GET /api/me", a.handleMe)
	a.mux.HandleFunc("POST /api/workspaces", a.handleCreateWorkspace)
	a.mux.HandleFunc("GET /api/workspaces/current", a.handleCurrentWorkspace)
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
