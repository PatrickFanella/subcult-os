package app

import (
	"bytes"
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
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("set TEST_DATABASE_URL to run lifecycle acceptance test")
	}

	ctx := t.Context()
	db, err := OpenDB(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	app := New(Config{AppEnv: "test", PublicWebURL: "http://public.test", SessionSecret: "test-secret"}, db)
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-")) + fmt.Sprintf("-%d", time.Now().UnixNano())
	ownerEmail := "owner+" + suffix + "@example.test"
	memberEmail := "member+" + suffix + "@example.test"
	guestEmail := "guest+" + suffix + "@example.test"

	ownerCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email": ownerEmail, "password": "secret1234", "displayName": "Owner"}).Cookie
	workspace := postJSON(t, app, ownerCookie, "/api/workspaces", map[string]any{"name": "Signal Collective"}).JSON
	workspaceID := workspace["id"].(string)

	invite := postJSON(t, app, ownerCookie, "/api/workspaces/"+workspaceID+"/invitations", map[string]any{"email": memberEmail}).JSON
	memberCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email": memberEmail, "password": "secret1234", "displayName": "Door"}).Cookie
	postJSON(t, app, memberCookie, "/api/invitations/"+invite["token"].(string)+"/accept", map[string]any{})

	event := postJSON(t, app, ownerCookie, "/api/workspaces/"+workspaceID+"/events", map[string]any{"title": "Night Market", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Free community event.", "locationDisplay": "Warehouse District", "ticketAllocation": 2}).JSON
	eventID := event["id"].(string)
	published := postJSON(t, app, ownerCookie, "/api/events/"+eventID+"/publish", map[string]any{}).JSON
	slug := published["publicSlug"].(string)

	ticket := postJSON(t, app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": guestEmail, "displayName": "Guest"}).JSON
	postJSON(t, app, memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": ticket["code"].(string)})
	report := postJSON(t, app, ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}).JSON

	if report["ticketsReserved"].(float64) != 1 || report["ticketsCheckedIn"].(float64) != 1 || report["noShows"].(float64) != 0 {
		t.Fatalf("unexpected report counts: %#v", report)
	}
}

type testResponse struct {
	Cookie *http.Cookie
	JSON   map[string]any
}

func postJSON(t *testing.T, app *App, cookie *http.Cookie, path string, payload map[string]any) testResponse {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("POST %s got %d: %s", path, rec.Code, rec.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	var cookieOut *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "subcult_session" {
			cookieOut = c
		}
	}
	return testResponse{Cookie: cookieOut, JSON: decoded}
}
