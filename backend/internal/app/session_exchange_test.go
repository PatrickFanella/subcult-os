package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionTokenFromRequestPriorityOrder(t *testing.T) {
	tests := []struct {
		name  string
		build func() *http.Request
		want  string
	}{
		{
			name: "token header wins",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(authTokenHeader, "token-header")
				req.Header.Set("Authorization", "Bearer bearer-token")
				req.Header.Set(authSessionHeader, sessionCookieValue("session-header"))
				req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
				return req
			},
			want: "token-header",
		},
		{
			name: "bearer header second",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", "Bearer bearer-token")
				req.Header.Set(authSessionHeader, sessionCookieValue("session-header"))
				req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
				return req
			},
			want: "bearer-token",
		},
		{
			name: "session header third",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(authSessionHeader, sessionCookieValue("session-header")+"; theme=dark")
				req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
				return req
			},
			want: "session-header",
		},
		{
			name: "cookie last",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
				return req
			},
			want: "cookie-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionTokenFromRequest(tt.build()); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestSetSessionEmitsCookieAndSessionHeader(t *testing.T) {
	app := NewTestApp(t)
	rec := httptest.NewRecorder()
	expiresAt := time.Now().UTC().Add(sessionLifetime)

	app.setSession(rec, "abc123", expiresAt)

	if got := rec.Header().Get(authSessionHeader); got != sessionCookieValue("abc123") {
		t.Fatalf("expected session header %q, got %q", sessionCookieValue("abc123"), got)
	}
	setCookie := rec.Header().Values("Set-Cookie")
	if len(setCookie) == 0 {
		t.Fatal("expected session cookie to be set")
	}
	if !strings.Contains(setCookie[0], authCookieName+"=abc123") {
		t.Fatalf("expected set-cookie to contain session token, got %q", setCookie[0])
	}
	if !strings.Contains(setCookie[0], "HttpOnly") {
		t.Fatalf("expected set-cookie to be HttpOnly, got %q", setCookie[0])
	}
}
