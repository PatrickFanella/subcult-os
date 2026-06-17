package app

import (
	"context"
	"encoding/json"
	"log"
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
	payments      paymentProvider
	mux           *http.ServeMux
	loginMu       sync.Mutex
	loginAttempts map[string]loginAttempt
}

func New(config Config, db *pgxpool.Pool) *App {
	a := &App{config: config, db: db, payments: newStripePaymentProvider(config.StripeSecretKey), mux: http.NewServeMux(), loginAttempts: map[string]loginAttempt{}}
	a.routes()
	return a
}

func (a *App) Handler() http.Handler { return a.requestLogger(a.cors(a.originGuard(a.mux))) }

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
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/event-templates", a.handleListEventTemplates)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/event-templates", a.handleCreateEventTemplate)
	a.mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/event-templates/{templateID}", a.handleUpdateEventTemplate)
	a.mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/event-templates/{templateID}", a.handleDeleteEventTemplate)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/contacts", a.handleListContacts)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/contacts", a.handleCreateContact)
	a.mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/contacts/{contactID}", a.handleUpdateContact)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/commitments", a.handleListWorkspaceCommitments)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/commitments", a.handleCreateCommitment)
	a.mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/commitments/{commitmentID}", a.handleUpdateCommitment)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/reminders", a.handleListWorkspaceReminders)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/reminders/sweep", a.handleSweepWorkspaceReminders)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/archives", a.handleListWorkspaceArchives)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/invitations", a.handleCreateInvitation)
	a.mux.HandleFunc("POST /api/invitations/{token}/accept", a.handleAcceptInvitation)
	a.mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/members/{memberID}", a.handleRemoveMember)
	a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/events", a.handleListEvents)
	a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/events", a.handleCreateEvent)
	a.mux.HandleFunc("GET /api/events/{eventID}", a.handleGetEvent)
	a.mux.HandleFunc("POST /api/events/{eventID}/apply-template", a.handleApplyEventTemplate)
	a.mux.HandleFunc("PATCH /api/events/{eventID}", a.handleUpdateEvent)
	a.mux.HandleFunc("POST /api/events/{eventID}/image", a.handleUploadEventImage)
	a.mux.HandleFunc("POST /api/events/{eventID}/publish", a.handlePublishEvent)
	a.mux.HandleFunc("GET /api/events/{eventID}/commitments", a.handleListEventCommitments)
	a.mux.HandleFunc("GET /api/events/{eventID}/reminders", a.handleListEventReminders)
	a.mux.HandleFunc("GET /api/events/{eventID}/notifications", a.handleListEventNotifications)
	a.mux.HandleFunc("GET /api/events/{eventID}/roles", a.handleListEventRoles)
	a.mux.HandleFunc("POST /api/events/{eventID}/roles", a.handleCreateEventRole)
	a.mux.HandleFunc("GET /api/events/{eventID}/staffing", a.handleListEventStaffing)
	a.mux.HandleFunc("POST /api/events/{eventID}/staffing", a.handleCreateEventStaffing)
	a.mux.HandleFunc("PATCH /api/events/{eventID}/staffing/{staffingID}", a.handleUpdateEventStaffing)
	a.mux.HandleFunc("GET /api/events/{eventID}/role-applications", a.handleListEventRoleApplications)
	a.mux.HandleFunc("GET /api/events/{eventID}/participants", a.handleListEventParticipants)
	a.mux.HandleFunc("PATCH /api/events/{eventID}/role-applications/{applicationID}", a.handleReviewEventRoleApplication)
	a.mux.HandleFunc("POST /api/events/{eventID}/end-of-night", a.handleEndOfNight)
	a.mux.HandleFunc("GET /api/events/{eventID}/report", a.handleGetReport)
	a.mux.HandleFunc("GET /api/events/{eventID}/archive", a.handleGetArchive)
	a.mux.HandleFunc("POST /api/events/{eventID}/archive/notes", a.handleCreateArchiveNote)
	a.mux.HandleFunc("POST /api/events/{eventID}/archive/seed-draft", a.handleSeedDraftFromArchive)
	a.mux.HandleFunc("GET /api/events/{eventID}/settlement", a.handleGetSettlement)
	a.mux.HandleFunc("POST /api/events/{eventID}/settlement/finalize", a.handleFinalizeSettlement)
	a.mux.HandleFunc("POST /api/events/{eventID}/settlement/adjustments", a.handleCreateSettlementAdjustment)
	a.mux.HandleFunc("GET /api/public/events", a.handleListPublicEvents)
	a.mux.HandleFunc("GET /api/public/events/{slug}", a.handlePublicEvent)
	a.mux.HandleFunc("GET /api/public/events/{slug}/roles", a.handleListPublicEventRoles)
	a.mux.HandleFunc("POST /api/public/events/{slug}/role-applications", a.handleSubmitPublicRoleApplication)
	a.mux.HandleFunc("POST /api/public/events/{slug}/reservations", a.handleReserveTicket)
	a.mux.HandleFunc("POST /api/public/events/{slug}/paid-reservations", a.handleCreatePaidReservation)
	a.mux.HandleFunc("POST /api/stripe/webhook", a.handleStripeWebhook)
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

func (a *App) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" && a.allowedOrigin(r) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PATCH, DELETE, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}

func (a *App) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("method=%s path=%s status=%d duration=%s", r.Method, r.URL.Path, recorder.status, time.Since(started).Round(time.Millisecond))
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
	if a.isDevelopment() && isLocalDevOrigin(parsed) {
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

func (a *App) isDevelopment() bool {
	return strings.EqualFold(strings.TrimSpace(a.config.AppEnv), "development") || strings.TrimSpace(a.config.AppEnv) == ""
}

func isLocalDevOrigin(origin *url.URL) bool {
	if origin.Scheme != "http" && origin.Scheme != "https" {
		return false
	}
	hostname := strings.ToLower(origin.Hostname())
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}
