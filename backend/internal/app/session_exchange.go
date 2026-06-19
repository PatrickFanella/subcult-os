package app

import (
	"net/http"
	"strings"
	"time"
)

func (a *App) setSession(w http.ResponseWriter, token string, expiresAt time.Time) {
	w.Header().Set(authSessionHeader, sessionCookieValue(token))
	http.SetCookie(w, sessionCookie(token, expiresAt, a.cookieSecure()))
}

func sessionTokenFromRequest(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get(authTokenHeader)); token != "" {
		return token
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		scheme, token, ok := strings.Cut(auth, " ")
		if ok && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token)
		}
	}
	if header := strings.TrimSpace(r.Header.Get(authSessionHeader)); header != "" {
		for _, part := range strings.Split(header, ";") {
			name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
			if ok && name == authCookieName && value != "" {
				return value
			}
		}
	}
	if cookie, err := r.Cookie(authCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func sessionCookieValue(token string) string {
	return authCookieName + "=" + token
}
