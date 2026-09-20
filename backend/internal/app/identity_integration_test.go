package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type authTestResult struct {
	Status       int
	Body         string
	JSON         any
	AccessToken  string
	RefreshToken string
	AccessCookie *http.Cookie
}

func TestIdentityVerificationAndProtectedLookup(t *testing.T) {
	app := newIdentityTestApp(t)
	email := "Person+Verify@Example.TEST"

	signup := authJSON(t, app, http.MethodPost, "/api/auth/signup", map[string]any{
		"email": email, "password": "secret1234", "displayName": "Person",
	}, nil, http.StatusAccepted)
	if signup.AccessToken != "" || signup.RefreshToken != "" || signup.AccessCookie != nil {
		t.Fatal("signup issued a session before email verification")
	}
	authJSON(t, app, http.MethodPost, "/api/auth/login", map[string]any{
		"email": email, "password": "secret1234",
	}, nil, http.StatusForbidden)

	token := latestIdentityToken(t, app, "identity_verification")
	verified := authJSON(t, app, http.MethodPost, "/api/auth/verify-email", map[string]any{"token": token}, nil, http.StatusOK)
	if verified.AccessToken == "" || verified.RefreshToken == "" || verified.AccessCookie == nil {
		t.Fatal("verification did not issue both session credentials")
	}
	authJSON(t, app, http.MethodPost, "/api/auth/verify-email", map[string]any{"token": token}, nil, http.StatusUnauthorized)

	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set(authTokenHeader, verified.AccessToken)
	me := authRequest(t, app, request, http.StatusOK)
	if mustString(t, me.JSON, "email") != normalizeEmail(email) {
		t.Fatalf("verified identity email = %#v", me.JSON)
	}

	var ciphertext []byte
	var lookupHash string
	if err := app.db.QueryRow(t.Context(), `select email_ciphertext, email_lookup_hash from email_identities`).Scan(&ciphertext, &lookupHash); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext), normalizeEmail(email)) || strings.Contains(lookupHash, "person") {
		t.Fatal("protected identity storage contains recognizable email")
	}
}

func TestRotatedRefreshReplayRevokesDescendants(t *testing.T) {
	app := newIdentityTestApp(t)
	first := signupAndVerifyIdentity(t, app, "rotate@example.test")

	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/refresh", bytes.NewReader([]byte(`{}`)))
	refreshRequest.Header.Set("Content-Type", "application/json")
	refreshRequest.Header.Set(refreshTokenHeader, first.RefreshToken)
	second := authRequest(t, app, refreshRequest, http.StatusOK)
	if second.AccessToken == "" || second.RefreshToken == "" || second.RefreshToken == first.RefreshToken {
		t.Fatalf("refresh did not rotate credentials: %#v", second)
	}

	assertAccessStatus(t, app, first.AccessToken, http.StatusUnauthorized)
	assertAccessStatus(t, app, second.AccessToken, http.StatusOK)

	replayRequest := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/refresh", bytes.NewReader([]byte(`{}`)))
	replayRequest.Header.Set("Content-Type", "application/json")
	replayRequest.Header.Set(refreshTokenHeader, first.RefreshToken)
	authRequest(t, app, replayRequest, http.StatusUnauthorized)
	assertAccessStatus(t, app, second.AccessToken, http.StatusUnauthorized)

	var revoked, reuse int
	if err := app.db.QueryRow(t.Context(), `
		select count(*) filter (where revoked_at is not null), count(*) filter (where reuse_detected_at is not null)
		from identity_sessions
	`).Scan(&revoked, &reuse); err != nil {
		t.Fatal(err)
	}
	if revoked != 2 || reuse != 2 {
		t.Fatalf("family replay state revoked=%d reuse=%d, want 2/2", revoked, reuse)
	}
}

func TestSessionExpiryLogoutAndRevokeAll(t *testing.T) {
	app := newIdentityTestApp(t)
	first := signupAndVerifyIdentity(t, app, "sessions@example.test")
	second := authJSON(t, app, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "sessions@example.test", "password": "secret1234",
	}, nil, http.StatusOK)

	logout := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/logout", bytes.NewReader([]byte(`{}`)))
	logout.Header.Set(authTokenHeader, first.AccessToken)
	logout.Header.Set(refreshTokenHeader, first.RefreshToken)
	authRequest(t, app, logout, http.StatusOK)
	assertAccessStatus(t, app, first.AccessToken, http.StatusUnauthorized)
	assertAccessStatus(t, app, second.AccessToken, http.StatusOK)

	logoutAll := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/logout-all", bytes.NewReader([]byte(`{}`)))
	logoutAll.Header.Set(authTokenHeader, second.AccessToken)
	authRequest(t, app, logoutAll, http.StatusOK)
	assertAccessStatus(t, app, second.AccessToken, http.StatusUnauthorized)

	third := authJSON(t, app, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "sessions@example.test", "password": "secret1234",
	}, nil, http.StatusOK)
	if _, err := app.db.Exec(t.Context(), `update identity_sessions set access_expires_at = now() - interval '1 second' where access_token_hash = $1`, tokenHash(third.AccessToken)); err != nil {
		t.Fatal(err)
	}
	assertAccessStatus(t, app, third.AccessToken, http.StatusUnauthorized)
	if _, err := app.db.Exec(t.Context(), `update identity_sessions set refresh_expires_at = now() - interval '1 second' where refresh_token_hash = $1`, tokenHash(third.RefreshToken)); err != nil {
		t.Fatal(err)
	}
	expiredRefresh := httptest.NewRequest(http.MethodPost, "/api/mobile/auth/refresh", bytes.NewReader([]byte(`{}`)))
	expiredRefresh.Header.Set(refreshTokenHeader, third.RefreshToken)
	authRequest(t, app, expiredRefresh, http.StatusUnauthorized)
}

func TestPasswordRecoveryRevokesSessions(t *testing.T) {
	app := newIdentityTestApp(t)
	session := signupAndVerifyIdentity(t, app, "recover@example.test")
	authJSON(t, app, http.MethodPost, "/api/auth/recovery/request", map[string]any{"email": "missing@example.test"}, nil, http.StatusAccepted)
	authJSON(t, app, http.MethodPost, "/api/auth/recovery/request", map[string]any{"email": "recover@example.test"}, nil, http.StatusAccepted)

	token := latestIdentityToken(t, app, "identity_recovery")
	authJSON(t, app, http.MethodPost, "/api/auth/recovery/complete", map[string]any{
		"token": token, "newPassword": "new-secret-1234",
	}, nil, http.StatusOK)
	assertAccessStatus(t, app, session.AccessToken, http.StatusUnauthorized)
	authJSON(t, app, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "recover@example.test", "password": "secret1234",
	}, nil, http.StatusUnauthorized)
	authJSON(t, app, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "recover@example.test", "password": "new-secret-1234",
	}, nil, http.StatusOK)
	authJSON(t, app, http.MethodPost, "/api/auth/recovery/complete", map[string]any{
		"token": token, "newPassword": "another-secret",
	}, nil, http.StatusUnauthorized)
}

func TestIdentityDoesNotMergeMatchingEmailOrDID(t *testing.T) {
	app := newIdentityTestApp(t)
	first := signupAndVerifyIdentity(t, app, "unique@example.test")
	_ = first
	authJSON(t, app, http.MethodPost, "/api/auth/signup", map[string]any{
		"email": "  UNIQUE@EXAMPLE.TEST ", "password": "different1234",
	}, nil, http.StatusConflict)
	second := signupAndVerifyIdentity(t, app, "second@example.test")

	var firstPersonID, secondPersonID string
	if err := app.db.QueryRow(t.Context(), `select person_id from identity_sessions where access_token_hash = $1`, tokenHash(first.AccessToken)).Scan(&firstPersonID); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow(t.Context(), `select person_id from identity_sessions where access_token_hash = $1`, tokenHash(second.AccessToken)).Scan(&secondPersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(t.Context(), `insert into did_links (person_id, did, verified_at) values ($1, 'did:plc:identitytest', now())`, firstPersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(t.Context(), `insert into did_links (person_id, did, verified_at) values ($1, 'did:plc:identitytest', now())`, secondPersonID); !isUniqueViolation(err) {
		t.Fatalf("cross-account DID claim error = %v, want unique violation", err)
	}
}

func newIdentityTestApp(t *testing.T) *App {
	t.Helper()
	pool := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	return New(Config{AppEnv: "test", PublicWebURL: "http://public.test", SessionSecret: "identity-test-secret"}, pool)
}

func signupAndVerifyIdentity(t *testing.T, app *App, email string) authTestResult {
	t.Helper()
	authJSON(t, app, http.MethodPost, "/api/auth/signup", map[string]any{
		"email": email, "password": "secret1234", "displayName": "Identity Test",
	}, nil, http.StatusAccepted)
	token := latestIdentityToken(t, app, "identity_verification")
	return authJSON(t, app, http.MethodPost, "/api/auth/verify-email", map[string]any{"token": token}, nil, http.StatusOK)
}

func latestIdentityToken(t *testing.T, app *App, relatedType string) string {
	t.Helper()
	var body string
	if err := app.db.QueryRow(t.Context(), `
		select body from email_outbox where related_type = $1 order by created_at desc limit 1
	`, relatedType).Scan(&body); err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(body)
	if len(fields) == 0 {
		t.Fatal("identity email body has no URL")
	}
	parsed, err := url.Parse(fields[len(fields)-1])
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("identity email URL has no token: %q", body)
	}
	return token
}

func assertAccessStatus(t *testing.T, app *App, accessToken string, status int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set(authTokenHeader, accessToken)
	authRequest(t, app, request, status)
}

func authJSON(t *testing.T, app *App, method, path string, payload any, headers map[string]string, wantStatus int) authTestResult {
	t.Helper()
	if strings.HasPrefix(path, "/api/auth/") {
		path = "/api/mobile" + strings.TrimPrefix(path, "/api")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	return authRequest(t, app, request, wantStatus)
}

func authRequest(t *testing.T, app *App, request *http.Request, wantStatus int) authTestResult {
	t.Helper()
	recorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d: %s", request.Method, request.URL.Path, recorder.Code, wantStatus, recorder.Body.String())
	}
	result := authTestResult{
		Status: recorder.Code, Body: recorder.Body.String(),
		AccessToken: recorder.Header().Get(authTokenHeader), RefreshToken: recorder.Header().Get(refreshTokenHeader),
	}
	if strings.TrimSpace(recorder.Body.String()) != "" {
		if err := json.Unmarshal(recorder.Body.Bytes(), &result.JSON); err != nil {
			t.Fatal(err)
		}
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == authCookieName {
			result.AccessCookie = cookie
		}
	}
	return result
}

func TestIdentityChallengeExpiry(t *testing.T) {
	app := newIdentityTestApp(t)
	authJSON(t, app, http.MethodPost, "/api/auth/signup", map[string]any{
		"email": "expired@example.test", "password": "secret1234",
	}, nil, http.StatusAccepted)
	token := latestIdentityToken(t, app, "identity_verification")
	if _, err := app.db.Exec(t.Context(), `update identity_challenges set expires_at = $1`, time.Now().UTC().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	authJSON(t, app, http.MethodPost, "/api/auth/verify-email", map[string]any{"token": token}, nil, http.StatusUnauthorized)
}
