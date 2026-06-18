package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const (
	authCookieName     = "subcult_session"
	authSessionHeader  = "X-Subcult-Session"
	authTokenHeader    = "X-Subcult-Session-Token"
	sessionLifetime    = 30 * 24 * time.Hour
	passwordScheme     = "bcrypt"
	legacySHA256Scheme = "sha256"
	sessionTokenBytes  = 32
	passwordSaltBytes  = 16
	bcryptCost         = 12
	maxLoginFailures   = 5
	loginFailureWindow = 5 * time.Minute
	loginLockout       = 5 * time.Minute
)

type loginAttempt struct {
	Failures    int
	FirstFailed time.Time
	LockedUntil time.Time
}

type signupRequest struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName *string `json:"displayName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type personRow struct {
	ID           string
	Email        string
	DisplayName  sql.NullString
	PasswordHash string
}

type userRow struct {
	ID          string
	Email       string
	DisplayName sql.NullString
}

type workspaceSummaryRow struct {
	ID   string
	Name string
	Role string
}

type CurrentUserDTO struct {
	ID            string                `json:"id"`
	Email         string                `json:"email"`
	DisplayName   *string               `json:"displayName"`
	Workspaces    []WorkspaceSummaryDTO `json:"workspaces"`
	SessionCookie string                `json:"sessionCookie,omitempty"`
}

type WorkspaceSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func (a *App) handleSignup(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var req signupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	email := normalizeEmail(req.Email)
	if !validSignup(email, req.Password) {
		writeError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	displayName := normalizeDisplayName(req.DisplayName)
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var personID string
	err = tx.QueryRow(r.Context(), `
		insert into people (email, display_name, password_hash)
		values ($1, $2, $3)
		returning id
	`, email, displayName, passwordHash).Scan(&personID)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	token, tokenHash := newToken()
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	_, err = tx.Exec(r.Context(), `
		insert into sessions (person_id, token_hash, expires_at)
		values ($1, $2, $3)
	`, personID, tokenHash, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save session")
		return
	}

	a.setSession(w, token, expiresAt)
	user := CurrentUserDTO{ID: personID, Email: email, DisplayName: displayName, Workspaces: []WorkspaceSummaryDTO{}, SessionCookie: sessionCookieValue(token)}
	if displayName == nil {
		user.DisplayName = nil
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid email or password")
		return
	}
	loginKey := a.loginAttemptKey(r, email)
	if retryAfter := a.loginRetryAfter(loginKey); retryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}

	var person personRow
	err := a.db.QueryRow(r.Context(), `
		select id, email, display_name, password_hash
		from people
		where email = $1
	`, email).Scan(&person.ID, &person.Email, &person.DisplayName, &person.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			a.recordLoginFailure(loginKey)
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	validPassword, upgradedHash := verifyPassword(req.Password, person.PasswordHash)
	if !validPassword {
		a.recordLoginFailure(loginKey)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	a.clearLoginFailures(loginKey)
	if upgradedHash != "" {
		_, _ = a.db.Exec(r.Context(), `update people set password_hash = $2 where id = $1`, person.ID, upgradedHash)
	}

	token, tokenHash := newToken()
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	_, err = a.db.Exec(r.Context(), `
		insert into sessions (person_id, token_hash, expires_at)
		values ($1, $2, $3)
	`, person.ID, tokenHash, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	a.setSession(w, token, expiresAt)
	user, err := a.loadCurrentUser(r.Context(), person.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load current user")
		return
	}
	user.SessionCookie = sessionCookieValue(token)
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if a.db != nil {
		if cookie, err := r.Cookie(authCookieName); err == nil && cookie.Value != "" {
			_, _ = a.db.Exec(r.Context(), `delete from sessions where token_hash = $1`, tokenHash(cookie.Value))
		}
	}
	http.SetCookie(w, expiredSessionCookie(a.cookieSecure()))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) setSession(w http.ResponseWriter, token string, expiresAt time.Time) {
	w.Header().Set(authSessionHeader, sessionCookieValue(token))
	http.SetCookie(w, sessionCookie(token, expiresAt, a.cookieSecure()))
}

func sessionCookieValue(token string) string {
	return authCookieName + "=" + token
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := a.loadCurrentUser(r.Context(), personID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load current user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleMobileAuthDebug(w http.ResponseWriter, r *http.Request) {
	token := sessionTokenFromRequest(r)
	personID := ""
	recognized := false
	if token != "" && a.db != nil {
		err := a.db.QueryRow(r.Context(), `
			select person_id
			from sessions
			where token_hash = $1
			  and expires_at > now()
		`, tokenHash(token)).Scan(&personID)
		recognized = err == nil
	}
	_, cookieErr := r.Cookie(authCookieName)
	writeJSON(w, http.StatusOK, map[string]any{
		"hasCookie":        cookieErr == nil,
		"hasAuthorization": strings.TrimSpace(r.Header.Get("Authorization")) != "",
		"hasSessionHeader": strings.TrimSpace(r.Header.Get(authSessionHeader)) != "",
		"hasTokenHeader":   strings.TrimSpace(r.Header.Get(authTokenHeader)) != "",
		"hasToken":         token != "",
		"recognized":       recognized,
		"personId":         personID,
	})
}

func (a *App) requirePersonID(r *http.Request) (string, bool) {
	if a.db == nil {
		return "", false
	}
	token := sessionTokenFromRequest(r)
	if token == "" {
		return "", false
	}
	var personID string
	err := a.db.QueryRow(r.Context(), `
		select person_id
		from sessions
		where token_hash = $1
		  and expires_at > now()
	`, tokenHash(token)).Scan(&personID)
	if err != nil {
		return "", false
	}
	return personID, true
}

func sessionTokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(authCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		scheme, token, ok := strings.Cut(auth, " ")
		if ok && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token)
		}
	}
	if token := strings.TrimSpace(r.Header.Get(authTokenHeader)); token != "" {
		return token
	}
	header := strings.TrimSpace(r.Header.Get(authSessionHeader))
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && name == authCookieName && value != "" {
			return value
		}
	}
	return ""
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s$%s", passwordScheme, string(hash)), nil
}

func verifyPassword(password, hash string) (bool, string) {
	parts := strings.Split(hash, "$")
	if len(parts) >= 2 && parts[0] == passwordScheme {
		stored := strings.TrimPrefix(hash, passwordScheme+"$")
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil, ""
	}
	if len(parts) != 3 || parts[0] != legacySHA256Scheme {
		return false, ""
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false, ""
	}
	want, err := hex.DecodeString(parts[2])
	if err != nil {
		return false, ""
	}
	got, err := hex.DecodeString(passwordSum(salt, password))
	if err != nil {
		return false, ""
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, ""
	}
	upgraded, err := hashPassword(password)
	if err != nil {
		return true, ""
	}
	return true, upgraded
}

func newToken() (string, string) {
	raw := make([]byte, sessionTokenBytes)
	_, _ = rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, tokenHash(token)
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func passwordSum(salt []byte, password string) string {
	sum := sha256.Sum256(append(append([]byte{}, salt...), []byte(password)...))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeDisplayName(displayName *string) *string {
	if displayName == nil {
		return nil
	}
	v := strings.TrimSpace(*displayName)
	if v == "" {
		return nil
	}
	return &v
}

func validSignup(email, password string) bool {
	return email != "" && strings.Contains(email, "@") && len(password) >= 8
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func sessionCookie(token string, expiresAt time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
}

func expiredSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (a *App) loadCurrentUser(ctx context.Context, personID string) (CurrentUserDTO, error) {
	var person userRow
	err := a.db.QueryRow(ctx, `
		select id, email, display_name
		from people
		where id = $1
	`, personID).Scan(&person.ID, &person.Email, &person.DisplayName)
	if err != nil {
		return CurrentUserDTO{}, err
	}

	user := CurrentUserDTO{ID: person.ID, Email: person.Email, Workspaces: []WorkspaceSummaryDTO{}}
	if person.DisplayName.Valid {
		user.DisplayName = &person.DisplayName.String
	}

	rows, err := a.db.Query(ctx, `
		select w.id, w.name, wm.role
		from workspace_members wm
		join workspaces w on w.id = wm.workspace_id
		where wm.person_id = $1
		  and wm.removed_at is null
		order by w.created_at, w.name
	`, personID)
	if err != nil {
		return CurrentUserDTO{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var row workspaceSummaryRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Role); err != nil {
			return CurrentUserDTO{}, err
		}
		user.Workspaces = append(user.Workspaces, WorkspaceSummaryDTO{ID: row.ID, Name: row.Name, Role: row.Role})
	}
	if err := rows.Err(); err != nil {
		return CurrentUserDTO{}, err
	}

	return user, nil
}

func (a *App) cookieSecure() bool {
	env := strings.ToLower(strings.TrimSpace(a.config.AppEnv))
	return env != "development" && env != "test"
}

func (a *App) loginAttemptKey(r *http.Request, email string) string {
	remote := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if remote != "" {
		remote = strings.TrimSpace(strings.Split(remote, ",")[0])
	} else {
		remote = r.RemoteAddr
	}
	return normalizeEmail(email) + "|" + remote
}

func (a *App) loginRetryAfter(key string) time.Duration {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	attempt := a.loginAttempts[key]
	if attempt.LockedUntil.IsZero() {
		return 0
	}
	now := time.Now().UTC()
	if now.After(attempt.LockedUntil) {
		delete(a.loginAttempts, key)
		return 0
	}
	return time.Until(attempt.LockedUntil)
}

func (a *App) recordLoginFailure(key string) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	now := time.Now().UTC()
	attempt := a.loginAttempts[key]
	if attempt.FirstFailed.IsZero() || now.Sub(attempt.FirstFailed) > loginFailureWindow {
		attempt = loginAttempt{FirstFailed: now}
	}
	attempt.Failures++
	if attempt.Failures >= maxLoginFailures {
		attempt.LockedUntil = now.Add(loginLockout)
	}
	a.loginAttempts[key] = attempt
}

func (a *App) clearLoginFailures(key string) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	delete(a.loginAttempts, key)
}
