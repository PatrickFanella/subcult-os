package app

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerNeverRecordsPathCredentialsOrQueries(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	mux := http.NewServeMux()
	for _, pattern := range []string{"GET /api/tickets/{code}", "POST /api/invitations/{token}/accept", "DELETE /api/v1/auth/atproto/links/{did}"} {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	}
	a := &App{}
	for _, tc := range []struct {
		method, path, route string
		status              int
	}{
		{"GET", "/api/tickets/private-ticket?email=private-email&token=private-query", "GET /api/tickets/{code}", 204},
		{"POST", "/api/invitations/private-invite/accept", "POST /api/invitations/{token}/accept", 204},
		{"DELETE", "/api/v1/auth/atproto/links/did:plc:private-did", "DELETE /api/v1/auth/atproto/links/{did}", 204},
		{"GET", "/unknown/private-unmatched?state=private-state", "unmatched", 404},
		{"PATCH", "/api/tickets/private-wrong-method", "unmatched", 405},
	} {
		output.Reset()
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		a.requestLogger(mux).ServeHTTP(w, r)
		line := output.String()
		if strings.Contains(line, "private-") || strings.Contains(line, "did:plc:") || strings.Contains(line, "path=") {
			t.Fatalf("request secrets leaked: %q", line)
		}
		if !strings.Contains(line, `route="`+tc.route+`"`) || w.Code != tc.status {
			t.Fatalf("route/status changed: %q status=%d", line, w.Code)
		}
	}
}

func TestRequestLoggerRedactsPreRoutingDenial(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	a := &App{}
	deny := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	w := httptest.NewRecorder()
	a.requestLogger(deny).ServeHTTP(w, httptest.NewRequest("POST", "/api/invitations/private-invite/accept", nil))
	if strings.Contains(output.String(), "private-invite") || !strings.Contains(output.String(), `route="unmatched"`) || w.Code != 403 {
		t.Fatalf("unsafe pre-routing log: %q", output.String())
	}
}
