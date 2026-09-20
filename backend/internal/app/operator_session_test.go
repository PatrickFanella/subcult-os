package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExpiredOperatorSessionRefreshAndPermissionBoundary(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Before refresh", 2), "id")
	login := authJSON(t, fx.app, http.MethodPost, "/api/auth/login", map[string]any{
		"email": fx.email("owner"), "password": "secret1234",
	}, nil, http.StatusOK)
	if _, err := fx.app.db.Exec(t.Context(), `update identity_sessions set access_expires_at=now()-interval '1 second' where access_token_hash=$1`, tokenHash(login.AccessToken)); err != nil {
		t.Fatal(err)
	}
	path := "/api/events/" + eventID
	getJSON(t, fx.app, login.AccessCookie, path, http.StatusUnauthorized)
	patchJSON(t, fx.app, login.AccessCookie, path, map[string]any{"title": "After refresh"}, http.StatusUnauthorized)
	getJSON(t, fx.app, fx.ownerCookie, path, http.StatusOK)
	refresh := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/refresh", bytes.NewBufferString(`{}`))
	refresh.Header.Set(refreshTokenHeader, login.RefreshToken)
	rotated := authRequest(t, fx.app, refresh, http.StatusOK)
	patchJSON(t, fx.app, rotated.AccessCookie, path, map[string]any{"title": "After refresh"}, http.StatusOK)
	other := newLifecycleFixture(t, fx.app)
	getJSON(t, fx.app, other.ownerCookie, path, http.StatusForbidden)
	patchJSON(t, fx.app, other.ownerCookie, path, map[string]any{"title": "Forbidden"}, http.StatusForbidden)
	var title string
	if err := fx.app.db.QueryRow(t.Context(), `select title from events where id=$1`, eventID).Scan(&title); err != nil || title != "After refresh" {
		t.Fatalf("denied request changed event: title=%q err=%v", title, err)
	}
}
