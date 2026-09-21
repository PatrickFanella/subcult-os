package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	authCookieName         = "subcult_access"
	refreshCookieName      = "subcult_refresh"
	authTokenHeader        = "X-Subcult-Access-Token"
	refreshTokenHeader     = "X-Subcult-Refresh-Token"
	accessTokenLifetime    = 15 * time.Minute
	refreshTokenLifetime   = 30 * 24 * time.Hour
	verificationLifetime   = 30 * time.Minute
	recoveryLifetime       = 30 * time.Minute
	passwordScheme         = "bcrypt"
	sessionTokenBytes      = 32
	bcryptCost             = 12
	maxLoginFailures       = 5
	loginFailureWindow     = 5 * time.Minute
	loginLockout           = 5 * time.Minute
	identityVerifyPurpose  = "verify_email"
	identityRecoverPurpose = "recover_password"
)

var errInvalidSession = errors.New("invalid session")

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

type tokenRequest struct {
	Token string `json:"token"`
}

type recoveryCompleteRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type emailRequest struct {
	Email string `json:"email"`
}

type signupResultDTO struct {
	VerificationRequired bool   `json:"verificationRequired"`
	Email                string `json:"email"`
}

type personRow struct {
	ID              string
	DisplayName     sql.NullString
	PasswordHash    string
	EmailCiphertext []byte
	VerifiedAt      sql.NullTime
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
	ID          string                `json:"id"`
	Email       string                `json:"email"`
	DisplayName *string               `json:"displayName"`
	Workspaces  []WorkspaceSummaryDTO `json:"workspaces"`
}

type WorkspaceSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type sessionTokens struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type identityQueryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (a *App) handleSignup(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
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

	ciphertext, lookupHash, err := a.identity.protectEmail(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not protect identity")
		return
	}
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
	if err := tx.QueryRow(r.Context(), `
		insert into people (email, display_name, password_hash)
		values ($1, $2, $3)
		returning id
	`, email, normalizeDisplayName(req.DisplayName), passwordHash).Scan(&personID); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "account already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	var identityID string
	if err := tx.QueryRow(r.Context(), `
		insert into email_identities (person_id, email_ciphertext, email_lookup_hash)
		values ($1, $2, $3)
		returning id
	`, personID, ciphertext, lookupHash).Scan(&identityID); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "account already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create email identity")
		return
	}
	if err := a.createIdentityChallenge(r.Context(), tx, personID, identityID, email, identityVerifyPurpose, verificationLifetime); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create verification")
		return
	}
	if err := a.recordAuthEvent(r.Context(), tx, personID, "", "signup_requested"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record authentication event")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save account")
		return
	}
	writeJSON(w, http.StatusAccepted, signupResultDTO{VerificationRequired: true, Email: email})
}

func (a *App) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
		return
	}
	var req tokenRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Token) == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	personID, identityID, err := consumeIdentityChallenge(r.Context(), tx, req.Token, identityVerifyPurpose)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired verification")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update email_identities
		set verified_at = coalesce(verified_at, now()), updated_at = now()
		where id = $1
	`, identityID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify email")
		return
	}
	tokens, familyID, err := issueSession(r.Context(), tx, personID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := a.recordAuthEvent(r.Context(), tx, personID, familyID, "email_verified"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record authentication event")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save verification")
		return
	}
	a.setSession(w, r, tokens)
	user, err := a.loadCurrentUser(r.Context(), personID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load current user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleRequestVerification(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
		return
	}
	var req emailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	lookupHash := a.identity.emailLookupHash(req.Email)
	var personID, identityID string
	var ciphertext []byte
	var verifiedAt sql.NullTime
	err := a.db.QueryRow(r.Context(), `
		select person_id, id, email_ciphertext, verified_at
		from email_identities
		where email_lookup_hash = $1
	`, lookupHash).Scan(&personID, &identityID, &ciphertext, &verifiedAt)
	if err == nil && !verifiedAt.Valid {
		email, revealErr := a.identity.revealEmail(ciphertext)
		if revealErr == nil {
			tx, beginErr := a.db.Begin(r.Context())
			if beginErr == nil {
				defer func() { _ = tx.Rollback(r.Context()) }()
				if challengeErr := a.createIdentityChallenge(r.Context(), tx, personID, identityID, email, identityVerifyPurpose, verificationLifetime); challengeErr == nil {
					_ = tx.Commit(r.Context())
				}
			}
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
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
		select p.id, p.display_name, p.password_hash, ei.email_ciphertext, ei.verified_at
		from email_identities ei
		join people p on p.id = ei.person_id
		where ei.email_lookup_hash = $1
	`, a.identity.emailLookupHash(email)).Scan(&person.ID, &person.DisplayName, &person.PasswordHash, &person.EmailCiphertext, &person.VerifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		a.recordLoginFailure(loginKey)
		_ = a.recordAuthEvent(r.Context(), a.db, "", "", "login_failed")
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load account")
		return
	}
	validPassword, upgradedHash := verifyPassword(req.Password, person.PasswordHash)
	if !validPassword {
		a.recordLoginFailure(loginKey)
		_ = a.recordAuthEvent(r.Context(), a.db, person.ID, "", "login_failed")
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !person.VerifiedAt.Valid {
		writeError(w, http.StatusForbidden, "email verification required")
		return
	}
	a.clearLoginFailures(loginKey)
	if upgradedHash != "" {
		_, _ = a.db.Exec(r.Context(), `update people set password_hash = $2 where id = $1`, person.ID, upgradedHash)
	}
	tokens, familyID, err := issueSession(r.Context(), a.db, person.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	_ = a.recordAuthEvent(r.Context(), a.db, person.ID, familyID, "login_succeeded")
	a.setSession(w, r, tokens)
	user, err := a.loadCurrentUser(r.Context(), person.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load current user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleRefreshSession(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
		return
	}
	refreshToken := refreshTokenFromRequest(r)
	if refreshToken == "" {
		a.clearSession(w)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tokens, err := a.rotateSession(r.Context(), refreshToken)
	if err != nil {
		a.clearSession(w)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	a.setSession(w, r, tokens)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if a.db != nil {
		accessToken := sessionTokenFromRequest(r)
		refreshToken := refreshTokenFromRequest(r)
		_, _ = a.db.Exec(r.Context(), `
			update identity_sessions
			set revoked_at = coalesce(revoked_at, now())
			where family_id in (
				select family_id from identity_sessions
				where access_token_hash = $1 or refresh_token_hash = $2
			)
		`, tokenHash(accessToken), tokenHash(refreshToken))
	}
	a.clearSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, err := a.db.Exec(r.Context(), `
		update identity_sessions
		set revoked_at = coalesce(revoked_at, now())
		where person_id = $1 and revoked_at is null
	`, personID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke sessions")
		return
	}
	_ = a.recordAuthEvent(r.Context(), a.db, personID, "", "all_sessions_revoked")
	a.clearSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleRequestRecovery(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
		return
	}
	var req emailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	var personID, identityID string
	var ciphertext []byte
	var verifiedAt sql.NullTime
	err := a.db.QueryRow(r.Context(), `
		select person_id, id, email_ciphertext, verified_at
		from email_identities
		where email_lookup_hash = $1
	`, a.identity.emailLookupHash(req.Email)).Scan(&personID, &identityID, &ciphertext, &verifiedAt)
	if err == nil && verifiedAt.Valid {
		email, revealErr := a.identity.revealEmail(ciphertext)
		if revealErr == nil {
			tx, beginErr := a.db.Begin(r.Context())
			if beginErr == nil {
				defer func() { _ = tx.Rollback(r.Context()) }()
				if challengeErr := a.createIdentityChallenge(r.Context(), tx, personID, identityID, email, identityRecoverPurpose, recoveryLifetime); challengeErr == nil {
					_ = a.recordAuthEvent(r.Context(), tx, personID, "", "recovery_requested")
					_ = tx.Commit(r.Context())
				}
			}
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (a *App) handleCompleteRecovery(w http.ResponseWriter, r *http.Request) {
	if !a.identityAvailable(w) {
		return
	}
	var req recoveryCompleteRequest
	if err := decodeJSON(r, &req); err != nil || len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "valid token and password are required")
		return
	}
	passwordHash, err := hashPassword(req.NewPassword)
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
	personID, _, err := consumeIdentityChallenge(r.Context(), tx, req.Token, identityRecoverPurpose)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired recovery")
		return
	}
	if _, err := tx.Exec(r.Context(), `update people set password_hash = $2 where id = $1`, personID, passwordHash); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update identity_sessions set revoked_at = coalesce(revoked_at, now())
		where person_id = $1 and revoked_at is null
	`, personID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke sessions")
		return
	}
	if err := a.recordAuthEvent(r.Context(), tx, personID, "", "password_recovered"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record authentication event")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save password")
		return
	}
	a.clearSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := a.loadCurrentUser(r.Context(), personID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleMobileAuthDebug(w http.ResponseWriter, r *http.Request) {
	accessToken := sessionTokenFromRequest(r)
	refreshToken := refreshTokenFromRequest(r)
	personID, recognized := a.requirePersonID(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"hasAccessToken":  accessToken != "",
		"hasRefreshToken": refreshToken != "",
		"recognized":      recognized,
		"personId":        personID,
	})
}

func (a *App) requirePersonID(r *http.Request) (string, bool) {
	if person, ok := r.Context().Value(operatorPersonKey{}).(string); ok && person != "" {
		return person, true
	}
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
		from identity_sessions
		where access_token_hash = $1
		  and access_expires_at > now()
		  and rotated_at is null
		  and revoked_at is null
	`, tokenHash(token)).Scan(&personID)
	if err != nil {
		return "", false
	}
	_, _ = a.db.Exec(r.Context(), `
		update identity_sessions set last_seen_at = now()
		where access_token_hash = $1
	`, tokenHash(token))
	return personID, true
}

func (a *App) identityAvailable(w http.ResponseWriter) bool {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return false
	}
	if a.identityErr != nil || a.identity == nil {
		writeError(w, http.StatusInternalServerError, "identity protection unavailable")
		return false
	}
	return true
}

func (a *App) createIdentityChallenge(ctx context.Context, tx pgx.Tx, personID, identityID, email, purpose string, lifetime time.Duration) error {
	if _, err := tx.Exec(ctx, `
		update identity_challenges set consumed_at = now()
		where email_identity_id = $1 and purpose = $2 and consumed_at is null
	`, identityID, purpose); err != nil {
		return fmt.Errorf("expire prior identity challenge: %w", err)
	}
	token, hashed, err := newToken()
	if err != nil {
		return err
	}
	var challengeID string
	if err := tx.QueryRow(ctx, `
		insert into identity_challenges (person_id, email_identity_id, purpose, token_hash, expires_at)
		values ($1, $2, $3, $4, $5)
		returning id
	`, personID, identityID, purpose, hashed, time.Now().UTC().Add(lifetime)).Scan(&challengeID); err != nil {
		return fmt.Errorf("insert identity challenge: %w", err)
	}
	path := "/verify-email?token=" + token
	subject := "Verify your Subcult account"
	relatedType := "identity_verification"
	if purpose == identityRecoverPurpose {
		path = "/recover-password?token=" + token
		subject = "Recover your Subcult account"
		relatedType = "identity_recovery"
	}
	body := subject + "\n\n" + strings.TrimRight(a.config.PublicWebURL, "/") + path
	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := a.enqueueEmail(txCtx, email, subject, body, relatedType, challengeID); err != nil {
		return fmt.Errorf("enqueue identity email: %w", err)
	}
	return nil
}

func consumeIdentityChallenge(ctx context.Context, tx pgx.Tx, token, purpose string) (string, string, error) {
	var challengeID, personID, identityID string
	err := tx.QueryRow(ctx, `
		select id, person_id, email_identity_id
		from identity_challenges
		where token_hash = $1 and purpose = $2 and consumed_at is null and expires_at > now()
		for update
	`, tokenHash(token), purpose).Scan(&challengeID, &personID, &identityID)
	if err != nil {
		return "", "", errInvalidSession
	}
	result, err := tx.Exec(ctx, `
		update identity_challenges set consumed_at = now()
		where id = $1 and consumed_at is null
	`, challengeID)
	if err != nil || result.RowsAffected() != 1 {
		return "", "", errInvalidSession
	}
	return personID, identityID, nil
}

func issueSession(ctx context.Context, db identityQueryer, personID string) (sessionTokens, string, error) {
	accessToken, accessHash, err := newToken()
	if err != nil {
		return sessionTokens{}, "", err
	}
	refreshToken, refreshHash, err := newToken()
	if err != nil {
		return sessionTokens{}, "", err
	}
	now := time.Now().UTC()
	tokens := sessionTokens{
		AccessToken: accessToken, RefreshToken: refreshToken,
		AccessExpiresAt: now.Add(accessTokenLifetime), RefreshExpiresAt: now.Add(refreshTokenLifetime),
	}
	var familyID string
	err = db.QueryRow(ctx, `
		insert into identity_sessions (
			person_id, family_id, generation, access_token_hash, refresh_token_hash,
			access_expires_at, refresh_expires_at
		)
		values ($1, gen_random_uuid(), 0, $2, $3, $4, $5)
		returning family_id
	`, personID, accessHash, refreshHash, tokens.AccessExpiresAt, tokens.RefreshExpiresAt).Scan(&familyID)
	if err != nil {
		return sessionTokens{}, "", fmt.Errorf("insert identity session: %w", err)
	}
	return tokens, familyID, nil
}

func (a *App) rotateSession(ctx context.Context, refreshToken string) (sessionTokens, error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return sessionTokens{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sessionID, personID, familyID string
	var generation int
	var refreshExpiresAt time.Time
	var rotatedAt, revokedAt sql.NullTime
	err = tx.QueryRow(ctx, `
		select id, person_id, family_id, generation, refresh_expires_at, rotated_at, revoked_at
		from identity_sessions
		where refresh_token_hash = $1
		for update
	`, tokenHash(refreshToken)).Scan(&sessionID, &personID, &familyID, &generation, &refreshExpiresAt, &rotatedAt, &revokedAt)
	if err != nil || revokedAt.Valid || time.Now().UTC().After(refreshExpiresAt) {
		return sessionTokens{}, errInvalidSession
	}
	if rotatedAt.Valid {
		if _, err := tx.Exec(ctx, `
			update identity_sessions
			set revoked_at = coalesce(revoked_at, now()), reuse_detected_at = coalesce(reuse_detected_at, now())
			where family_id = $1
		`, familyID); err != nil {
			return sessionTokens{}, err
		}
		if err := a.recordAuthEvent(ctx, tx, personID, familyID, "session_reuse_detected"); err != nil {
			return sessionTokens{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return sessionTokens{}, err
		}
		return sessionTokens{}, errInvalidSession
	}

	accessToken, accessHash, err := newToken()
	if err != nil {
		return sessionTokens{}, err
	}
	newRefreshToken, refreshHash, err := newToken()
	if err != nil {
		return sessionTokens{}, err
	}
	now := time.Now().UTC()
	tokens := sessionTokens{
		AccessToken: accessToken, RefreshToken: newRefreshToken,
		AccessExpiresAt: now.Add(accessTokenLifetime), RefreshExpiresAt: now.Add(refreshTokenLifetime),
	}
	var replacementID string
	if err := tx.QueryRow(ctx, `
		insert into identity_sessions (
			person_id, family_id, generation, access_token_hash, refresh_token_hash,
			access_expires_at, refresh_expires_at
		)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id
	`, personID, familyID, generation+1, accessHash, refreshHash, tokens.AccessExpiresAt, tokens.RefreshExpiresAt).Scan(&replacementID); err != nil {
		return sessionTokens{}, err
	}
	if _, err := tx.Exec(ctx, `
		update identity_sessions
		set rotated_at = now(), replaced_by_session_id = $2
		where id = $1 and rotated_at is null
	`, sessionID, replacementID); err != nil {
		return sessionTokens{}, err
	}
	if err := a.recordAuthEvent(ctx, tx, personID, familyID, "session_rotated"); err != nil {
		return sessionTokens{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sessionTokens{}, err
	}
	return tokens, nil
}

func (a *App) recordAuthEvent(ctx context.Context, db identityQueryer, personID, familyID, eventType string) error {
	var person, family any
	if personID != "" {
		person = personID
	}
	if familyID != "" {
		family = familyID
	}
	_, err := db.Exec(ctx, `
		insert into auth_audit_events (person_id, session_family_id, event_type)
		values ($1, $2, $3)
	`, person, family, eventType)
	return err
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s$%s", passwordScheme, string(hash)), nil
}

func verifyPassword(password, hash string) (bool, string) {
	if !strings.HasPrefix(hash, passwordScheme+"$") {
		return false, ""
	}
	stored := strings.TrimPrefix(hash, passwordScheme+"$")
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil, ""
}

func newToken() (string, string, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, tokenHash(token), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func normalizeDisplayName(displayName *string) *string {
	if displayName == nil {
		return nil
	}
	value := strings.TrimSpace(*displayName)
	if value == "" {
		return nil
	}
	return &value
}

func validSignup(email, password string) bool {
	return email != "" && strings.Contains(email, "@") && len(password) >= 8
}

func decodeJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (a *App) loadCurrentUser(ctx context.Context, personID string) (CurrentUserDTO, error) {
	var user userRow
	var ciphertext []byte
	err := a.db.QueryRow(ctx, `
		select p.id, p.display_name, ei.email_ciphertext
		from people p
		join email_identities ei on ei.person_id = p.id
		where p.id = $1 and ei.verified_at is not null
	`, personID).Scan(&user.ID, &user.DisplayName, &ciphertext)
	if err != nil {
		return CurrentUserDTO{}, err
	}
	user.Email, err = a.identity.revealEmail(ciphertext)
	if err != nil {
		return CurrentUserDTO{}, err
	}

	result := CurrentUserDTO{ID: user.ID, Email: user.Email, Workspaces: []WorkspaceSummaryDTO{}}
	if user.DisplayName.Valid {
		result.DisplayName = &user.DisplayName.String
	}
	rows, err := a.db.Query(ctx, `
		select w.id, w.name, wm.role
		from workspace_members wm
		join workspaces w on w.id = wm.workspace_id
		where wm.person_id = $1 and wm.removed_at is null
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
		result.Workspaces = append(result.Workspaces, WorkspaceSummaryDTO{ID: row.ID, Name: row.Name, Role: row.Role})
	}
	return result, rows.Err()
}

func (a *App) cookieSecure() bool {
	environment := strings.ToLower(strings.TrimSpace(a.config.AppEnv))
	return environment != "development" && environment != "test"
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
