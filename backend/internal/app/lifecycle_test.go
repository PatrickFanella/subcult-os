package app

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFirstEventLifecycle(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 2)
	eventID := mustString(t, event, "id")
	published := publishEvent(t, fx, eventID)
	slug := mustString(t, published, "publicSlug")
	guestEmail := fx.email("guest")
	ticket := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": guestEmail, "displayName": "Guest"}, http.StatusOK)
	checkIn := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": mustString(t, ticket.JSON, "code")}, http.StatusOK)
	report := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	reportObj := mustObject(t, report.JSON)

	if int(reportObj["ticketsReserved"].(float64)) != 1 || int(reportObj["ticketsCheckedIn"].(float64)) != 1 || int(reportObj["noShows"].(float64)) != 0 {
		t.Fatalf("unexpected report counts: %#v", report.JSON)
	}
	if mustString(t, checkIn.JSON, "status") != "checked_in" {
		t.Fatalf("unexpected check-in status: %#v", checkIn.JSON)
	}
}

func TestFirstEventLifecycleFullCapacity(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")

	firstGuest := fx.email("guest-a")
	secondGuest := fx.email("guest-b")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": firstGuest, "displayName": "Guest One"}, http.StatusOK)
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": secondGuest, "displayName": "Guest Two"}, http.StatusConflict)

	publicEvent := getJSON(t, fx.app, nil, "/api/public/events/"+slug, http.StatusOK)
	if int(publicEvent.JSON.(map[string]any)["reservedCount"].(float64)) != 1 || int(publicEvent.JSON.(map[string]any)["remainingTickets"].(float64)) != 0 || publicEvent.JSON.(map[string]any)["isFull"].(bool) != true {
		t.Fatalf("event should remain full with one reservation: %#v", publicEvent.JSON)
	}
}

func TestFirstEventLifecycleDuplicateCheckIn(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	ticket := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusOK)
	code := mustString(t, ticket.JSON, "code")
	firstCheckIn := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": code}, http.StatusOK)
	secondCheckIn := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": code}, http.StatusOK)

	first := firstCheckIn.JSON.(map[string]any)
	second := secondCheckIn.JSON.(map[string]any)
	if first["status"] != "checked_in" || second["status"] != "checked_in" {
		t.Fatalf("expected checked-in ticket: first=%#v second=%#v", firstCheckIn.JSON, secondCheckIn.JSON)
	}
	if first["checkedInAt"] != second["checkedInAt"] {
		t.Fatalf("expected duplicate check-in to return same checkedInAt, got %v and %v", first["checkedInAt"], second["checkedInAt"])
	}
}

func TestFirstEventLifecycleDraftReservationsBlocked(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	if event["publicSlug"] != nil {
		t.Fatalf("draft event should not have a public slug: %#v", event["publicSlug"])
	}

	getJSON(t, fx.app, nil, "/api/public/events/"+eventID, http.StatusNotFound)
	postJSON(t, fx.app, nil, "/api/public/events/"+eventID+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusNotFound)
}

func TestFirstEventLifecyclePermissions(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")

	updated := patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"title": "Night Market Updated"}, http.StatusOK)
	if mustString(t, updated.JSON, "title") != "Night Market Updated" {
		t.Fatalf("member update did not persist: %#v", updated.JSON)
	}

	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/publish", map[string]any{}, http.StatusForbidden)
	published := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/publish", map[string]any{}, http.StatusOK)
	slug := mustString(t, published.JSON, "publicSlug")
	guestTicket := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusOK)
	search := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/tickets?query=guest", http.StatusOK)
	tickets := search.JSON.([]any)
	if len(tickets) != 1 {
		t.Fatalf("expected one ticket in door search, got %#v", search.JSON)
	}
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": mustString(t, guestTicket.JSON, "code")}, http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusForbidden)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
}

func TestDevEmailOutbox(t *testing.T) {
	fx := newLifecycleFixture(t)
	inviteEmail := fx.email("invitee")
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": inviteEmail}, http.StatusOK)

	outbox := getJSON(t, fx.app, fx.ownerCookie, "/api/dev/email-outbox", http.StatusOK).JSON.([]any)
	if len(outbox) == 0 {
		t.Fatal("expected at least one email in outbox")
	}

	matched := false
	for _, item := range outbox {
		msg := mustObject(t, item)
		if msg["recipientEmail"] == inviteEmail && strings.Contains(msg["body"].(string), "/invite/") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("expected invitation email for %s in outbox: %#v", inviteEmail, outbox)
	}
}

func TestDevEmailOutboxUnauthorized(t *testing.T) {
	fx := newLifecycleFixture(t)
	getJSON(t, fx.app, nil, "/api/dev/email-outbox", http.StatusUnauthorized)
}

func TestDevEmailOutboxProductionHidden(t *testing.T) {
	fx := newLifecycleFixture(t)
	prodApp := New(Config{AppEnv: "production", PublicWebURL: "http://public.test", SessionSecret: "test-secret"}, fx.app.db)
	getJSON(t, prodApp, fx.ownerCookie, "/api/dev/email-outbox", http.StatusNotFound)
}

func TestSignupStoresBcryptPasswordHash(t *testing.T) {
	fx := newLifecycleFixture(t)
	email := fx.email("bcrypt")
	postJSON(t, fx.app, nil, "/api/auth/signup", map[string]any{"email": email, "password": "secret1234", "displayName": "Hash"}, http.StatusOK)

	var storedHash string
	if err := fx.app.db.QueryRow(t.Context(), `select password_hash from people where email = $1`, email).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedHash, passwordScheme+"$") || strings.Contains(storedHash, legacySHA256Scheme+"$") {
		t.Fatalf("expected bcrypt password hash, got %q", storedHash)
	}
	valid, upgraded := verifyPassword("secret1234", storedHash)
	if !valid || upgraded != "" {
		t.Fatalf("expected bcrypt password to verify without upgrade, valid=%v upgraded=%q", valid, upgraded)
	}
}

func TestLegacyPasswordHashUpgradesOnLogin(t *testing.T) {
	fx := newLifecycleFixture(t)
	email := fx.email("legacy")
	legacyHash := legacyPasswordHashForTest(t, "secret1234")
	if _, err := fx.app.db.Exec(t.Context(), `insert into people (email, password_hash) values ($1, $2)`, email, legacyHash); err != nil {
		t.Fatal(err)
	}

	postJSON(t, fx.app, nil, "/api/auth/login", map[string]any{"email": email, "password": "secret1234"}, http.StatusOK)

	var storedHash string
	if err := fx.app.db.QueryRow(t.Context(), `select password_hash from people where email = $1`, email).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedHash, passwordScheme+"$") {
		t.Fatalf("expected legacy hash to upgrade to bcrypt, got %q", storedHash)
	}
}

func TestOriginGuardRejectsCrossSiteCookieMutations(t *testing.T) {
	fx := newLifecycleFixture(t)
	postJSONWithOrigin(t, fx.app, fx.ownerCookie, "/api/workspaces", map[string]any{"name": "Evil"}, "http://evil.example", http.StatusForbidden)
	postJSONWithOrigin(t, fx.app, fx.ownerCookie, "/api/workspaces", map[string]any{"name": "Allowed"}, "http://public.test", http.StatusOK)
}

func TestLoginRateLimit(t *testing.T) {
	fx := newLifecycleFixture(t)
	email := fx.email("missing")
	for range maxLoginFailures {
		postJSON(t, fx.app, nil, "/api/auth/login", map[string]any{"email": email, "password": "wrong-password"}, http.StatusUnauthorized)
	}
	resp := postJSON(t, fx.app, nil, "/api/auth/login", map[string]any{"email": email, "password": "wrong-password"}, http.StatusTooManyRequests)
	if !strings.Contains(resp.Body, "too many login attempts") {
		t.Fatalf("expected rate-limit response, got %s", resp.Body)
	}
}

type testResponse struct {
	Status int
	Cookie *http.Cookie
	JSON   any
	Body   string
}

type lifecycleFixture struct {
	app          *App
	ownerCookie  *http.Cookie
	memberCookie *http.Cookie
	workspaceID  string
	suffix       string
}

func (f lifecycleFixture) email(prefix string) string {
	return prefix + "+" + f.suffix + "@example.test"
}

func newLifecycleFixture(t *testing.T) lifecycleFixture {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("set TEST_DATABASE_URL to run lifecycle acceptance test")
	}

	ctx := t.Context()
	db, err := OpenDB(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	app := New(Config{AppEnv: "test", PublicWebURL: "http://public.test", SessionSecret: "test-secret"}, db)
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-")) + fmt.Sprintf("-%d", time.Now().UnixNano())
	ownerEmail := "owner+" + suffix + "@example.test"
	memberEmail := "member+" + suffix + "@example.test"

	ownerCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email": ownerEmail, "password": "secret1234", "displayName": "Owner"}, http.StatusOK).Cookie
	workspace := postJSON(t, app, ownerCookie, "/api/workspaces", map[string]any{"name": "Signal Collective"}, http.StatusOK)
	workspaceID := mustString(t, workspace.JSON, "id")
	invite := postJSON(t, app, ownerCookie, "/api/workspaces/"+workspaceID+"/invitations", map[string]any{"email": memberEmail}, http.StatusOK)
	memberCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email": memberEmail, "password": "secret1234", "displayName": "Door"}, http.StatusOK).Cookie
	postJSON(t, app, memberCookie, "/api/invitations/"+mustString(t, invite.JSON, "token")+"/accept", map[string]any{}, http.StatusOK)

	return lifecycleFixture{app: app, ownerCookie: ownerCookie, memberCookie: memberCookie, workspaceID: workspaceID, suffix: suffix}
}

func createEvent(t *testing.T, fx lifecycleFixture, title string, ticketAllocation int) map[string]any {
	t.Helper()
	resp := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": title, "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Free community event.", "locationDisplay": "Warehouse District", "ticketAllocation": ticketAllocation}, http.StatusOK)
	return mustObject(t, resp.JSON)
}

func publishEvent(t *testing.T, fx lifecycleFixture, eventID string) map[string]any {
	t.Helper()
	return mustObject(t, publishEventResult(t, fx.app, fx.ownerCookie, eventID).JSON)
}

func publishEventResult(t *testing.T, app *App, cookie *http.Cookie, eventID string) testResponse {
	t.Helper()
	return postJSON(t, app, cookie, "/api/events/"+eventID+"/publish", map[string]any{}, http.StatusOK)
}

func doJSON(t *testing.T, method string, app *App, cookie *http.Cookie, path string, payload any, wantStatus int) testResponse {
	t.Helper()
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s got %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var decoded any
	if strings.TrimSpace(rec.Body.String()) != "" {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	var cookieOut *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == authCookieName {
			cookieOut = c
		}
	}
	return testResponse{Status: rec.Code, Cookie: cookieOut, JSON: decoded, Body: rec.Body.String()}
}

func postJSON(t *testing.T, app *App, cookie *http.Cookie, path string, payload any, wantStatus int) testResponse {
	t.Helper()
	return doJSON(t, http.MethodPost, app, cookie, path, payload, wantStatus)
}

func postJSONWithOrigin(t *testing.T, app *App, cookie *http.Cookie, path string, payload any, origin string, wantStatus int) testResponse {
	t.Helper()
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Host = "public.test"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("POST %s origin %s got %d, want %d: %s", path, origin, rec.Code, wantStatus, rec.Body.String())
	}
	var decoded any
	if strings.TrimSpace(rec.Body.String()) != "" {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	return testResponse{Status: rec.Code, JSON: decoded, Body: rec.Body.String()}
}

func getJSON(t *testing.T, app *App, cookie *http.Cookie, path string, wantStatus int) testResponse {
	t.Helper()
	return doJSON(t, http.MethodGet, app, cookie, path, nil, wantStatus)
}

func patchJSON(t *testing.T, app *App, cookie *http.Cookie, path string, payload any, wantStatus int) testResponse {
	t.Helper()
	return doJSON(t, http.MethodPatch, app, cookie, path, payload, wantStatus)
}

func mustObject(t *testing.T, value any) map[string]any {
	t.Helper()
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object, got %#v", value)
	}
	return obj
}

func mustString(t *testing.T, value any, key string) string {
	t.Helper()
	obj := mustObject(t, value)
	v, ok := obj[key].(string)
	if !ok {
		t.Fatalf("expected %s string, got %#v", key, obj[key])
	}
	return v
}

func legacyPasswordHashForTest(t *testing.T, password string) string {
	t.Helper()
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s$%s$%s", legacySHA256Scheme, base64.RawURLEncoding.EncodeToString(salt), passwordSum(salt, password))
}
