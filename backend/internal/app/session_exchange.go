package app

import (
	"net/http"
	"strings"
	"time"
)

func (a *App) setSession(w http.ResponseWriter, r *http.Request, tokens sessionTokens) {
	if strings.HasPrefix(r.URL.Path, "/api/mobile/auth/") {
		w.Header().Set(authTokenHeader, tokens.AccessToken)
		w.Header().Set(refreshTokenHeader, tokens.RefreshToken)
	}
	http.SetCookie(w, authCookie(authCookieName, tokens.AccessToken, tokens.AccessExpiresAt, "/", a.cookieSecure()))
	http.SetCookie(w, authCookie(refreshCookieName, tokens.RefreshToken, tokens.RefreshExpiresAt, "/api/auth", a.cookieSecure()))
}

func (a *App) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, expiredAuthCookie(authCookieName, "/", a.cookieSecure()))
	http.SetCookie(w, expiredAuthCookie(refreshCookieName, "/api/auth", a.cookieSecure()))
}

func sessionTokenFromRequest(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get(authTokenHeader)); token != "" {
		return token
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		scheme, token, ok := strings.Cut(auth, " ")
		if ok && strings.EqualFold(scheme, "Bearer") {
			return strings.TrimSpace(token)
		}
	}
	if cookie, err := r.Cookie(authCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func refreshTokenFromRequest(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get(refreshTokenHeader)); token != "" {
		return token
	}
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func authCookie(name, token string, expiresAt time.Time, path string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: token, Path: path, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secure, Expires: expiresAt,
		MaxAge: int(time.Until(expiresAt).Seconds()),
	}
}

func expiredAuthCookie(name, path string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: "", Path: path, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secure, Expires: time.Unix(0, 0), MaxAge: -1,
	}
}
