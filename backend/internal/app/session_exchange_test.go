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
			name: "access header wins",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(authTokenHeader, "access-header")
				req.Header.Set("Authorization", "Bearer bearer-token")
				req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
				return req
			},
			want: "access-header",
		},
		{
			name: "bearer second",
			build: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", "Bearer bearer-token")
				req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
				return req
			},
			want: "bearer-token",
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sessionTokenFromRequest(test.build()); got != test.want {
				t.Fatalf("sessionTokenFromRequest() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRefreshTokenFromRequestPrefersHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.Header.Set(refreshTokenHeader, "refresh-header")
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh-cookie"})
	if got := refreshTokenFromRequest(req); got != "refresh-header" {
		t.Fatalf("refreshTokenFromRequest() = %q", got)
	}
}

func TestSetSessionEmitsHttpOnlyCookiesAndMobileHeaders(t *testing.T) {
	app := NewTestApp(t)
	recorder := httptest.NewRecorder()
	now := time.Now().UTC()
	tokens := sessionTokens{
		AccessToken: "access", RefreshToken: "refresh",
		AccessExpiresAt: now.Add(accessTokenLifetime), RefreshExpiresAt: now.Add(refreshTokenLifetime),
	}

	request := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/login", nil)
	app.setSession(recorder, request, tokens)
	if recorder.Header().Get(authTokenHeader) != "access" || recorder.Header().Get(refreshTokenHeader) != "refresh" {
		t.Fatalf("missing mobile token headers: %#v", recorder.Header())
	}
	cookies := recorder.Header().Values("Set-Cookie")
	if len(cookies) != 2 {
		t.Fatalf("Set-Cookie count = %d, want 2", len(cookies))
	}
	joined := strings.Join(cookies, "\n")
	for _, expected := range []string{authCookieName + "=access", refreshCookieName + "=refresh", "HttpOnly"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("Set-Cookie missing %q: %s", expected, joined)
		}
	}
}

func TestSetSessionDoesNotExposeCredentialsToBrowserJavaScript(t *testing.T) {
	app := NewTestApp(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	now := time.Now().UTC()
	app.setSession(recorder, request, sessionTokens{
		AccessToken: "access", RefreshToken: "refresh",
		AccessExpiresAt: now.Add(accessTokenLifetime), RefreshExpiresAt: now.Add(refreshTokenLifetime),
	})
	if recorder.Header().Get(authTokenHeader) != "" || recorder.Header().Get(refreshTokenHeader) != "" {
		t.Fatalf("browser response exposed session credentials: %#v", recorder.Header())
	}
	if len(recorder.Header().Values("Set-Cookie")) != 2 {
		t.Fatal("browser response did not retain HttpOnly cookie transport")
	}
}
