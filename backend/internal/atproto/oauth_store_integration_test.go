package atproto_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	appcore "git.subcult.tv/PatrickFanella/subcult-os/internal/app"
	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	atprotocoloauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOAuthStoreClaimsEncryptsLinksAndRotates(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "oauth-store-test")
	if err != nil {
		t.Fatal(err)
	}
	personID := insertOAuthStorePerson(t, db, "oauth-owner@example.test")
	request := oauthRequest("state-one")

	if err := store.SaveAuthRequestInfo(t.Context(), request); !errors.Is(err, atprotocol.ErrOAuthLinkContext) {
		t.Fatalf("SaveAuthRequestInfo() error = %v, want link context", err)
	}
	ctx := atprotocol.WithOAuthLinkPerson(t.Context(), personID)
	if err := store.SaveAuthRequestInfo(ctx, request); err != nil {
		t.Fatal(err)
	}

	var storedState string
	var requestCiphertext []byte
	if err := db.QueryRow(t.Context(), `select state_hash, payload_ciphertext from atproto_oauth_requests`).Scan(&storedState, &requestCiphertext); err != nil {
		t.Fatal(err)
	}
	if storedState == request.State || strings.Contains(string(requestCiphertext), request.PKCEVerifier) || strings.Contains(string(requestCiphertext), request.DPoPPrivateKeyMultibase) {
		t.Fatal("OAuth request secret was stored in recognizable plaintext")
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, claimErr := store.GetAuthRequestInfo(context.Background(), request.State)
			if claimErr == nil && claimed.PKCEVerifier != request.PKCEVerifier {
				claimErr = errors.New("claimed request did not round-trip")
			}
			results <- claimErr
		}()
	}
	wg.Wait()
	close(results)
	successes, replayRejects := 0, 0
	for claimErr := range results {
		switch {
		case claimErr == nil:
			successes++
		case errors.Is(claimErr, atprotocol.ErrOAuthRequestNotFound):
			replayRejects++
		default:
			t.Fatal(claimErr)
		}
	}
	if successes != 1 || replayRejects != 1 {
		t.Fatalf("concurrent claims = %d success, %d replay rejects", successes, replayRejects)
	}

	did := syntax.DID("did:plc:oauthstoreowner")
	session := oauthSession(did, request.State, "access-one", "refresh-one")
	if err := store.SaveSession(t.Context(), session); err != nil {
		t.Fatal(err)
	}
	var linkedPerson string
	if err := db.QueryRow(t.Context(), `select person_id from did_links where did = $1 and status = 'active'`, did.String()).Scan(&linkedPerson); err != nil {
		t.Fatal(err)
	}
	if linkedPerson != personID {
		t.Fatalf("linked person = %q, want %q", linkedPerson, personID)
	}
	var auditCount int
	if err := db.QueryRow(t.Context(), `
		select count(*) from auth_audit_events
		where person_id = $1 and event_type = 'atproto_did_linked'
	`, personID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("AT OAuth link audit count = %d, error = %v", auditCount, err)
	}
	var sessionCiphertext []byte
	if err := db.QueryRow(t.Context(), `select payload_ciphertext from atproto_oauth_sessions where did = $1`, did.String()).Scan(&sessionCiphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sessionCiphertext, []byte(session.AccessToken)) || bytes.Contains(sessionCiphertext, []byte(session.RefreshToken)) || bytes.Contains(sessionCiphertext, []byte(session.DPoPPrivateKeyMultibase)) {
		t.Fatal("OAuth session secret was stored in recognizable plaintext")
	}
	var requestCount int
	if err := db.QueryRow(t.Context(), `select count(*) from atproto_oauth_requests`).Scan(&requestCount); err != nil || requestCount != 0 {
		t.Fatalf("consumed request count = %d, error = %v", requestCount, err)
	}

	loaded, err := store.GetSession(t.Context(), did, request.State)
	if err != nil || loaded.AccessToken != session.AccessToken || loaded.RefreshToken != session.RefreshToken {
		t.Fatalf("GetSession() = %#v, %v", loaded, err)
	}
	session.AccessToken = "access-two"
	session.RefreshToken = "refresh-two"
	if err := store.SaveSession(t.Context(), session); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.GetSession(t.Context(), did, request.State)
	if err != nil || loaded.AccessToken != "access-two" || loaded.RefreshToken != "refresh-two" {
		t.Fatalf("rotated GetSession() = %#v, %v", loaded, err)
	}
	var rotatedCiphertext []byte
	if err := db.QueryRow(t.Context(), `
		select payload_ciphertext from atproto_oauth_sessions where did = $1
	`, did.String()).Scan(&rotatedCiphertext); err != nil {
		t.Fatal(err)
	}
	tamperedCiphertext := append([]byte(nil), rotatedCiphertext...)
	tamperedCiphertext[len(tamperedCiphertext)-1] ^= 0xff
	if _, err := db.Exec(t.Context(), `
		update atproto_oauth_sessions set payload_ciphertext = $1 where did = $2
	`, tamperedCiphertext, did.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetSession(t.Context(), did, request.State); err == nil || !strings.Contains(err.Error(), "decrypt") {
		t.Fatalf("GetSession() for tampered ciphertext error = %v", err)
	}
	if _, err := db.Exec(t.Context(), `
		update atproto_oauth_sessions set payload_ciphertext = $1 where did = $2
	`, rotatedCiphertext, did.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(t.Context(), `
		update did_links set status = 'revoked', revoked_at = now() where did = $1
	`, did.String()); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSession(t.Context(), session); err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("SaveSession() for revoked link error = %v", err)
	}
	if err := store.DeleteSession(t.Context(), did, request.State); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetSession(t.Context(), did, request.State); !errors.Is(err, atprotocol.ErrOAuthSessionNotFound) {
		t.Fatalf("GetSession() after delete error = %v", err)
	}
}

func TestOAuthStoreRejectsExpiredScopeAndCrossAccountLink(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "oauth-store-test")
	if err != nil {
		t.Fatal(err)
	}
	firstPerson := insertOAuthStorePerson(t, db, "oauth-first@example.test")
	secondPerson := insertOAuthStorePerson(t, db, "oauth-second@example.test")
	overScopedRequest := oauthRequest("over-scoped-request")
	overScopedRequest.Scopes = []string{"atproto", "repo:tv.subcult.event?action=create"}
	if err := store.SaveAuthRequestInfo(atprotocol.WithOAuthLinkPerson(t.Context(), firstPerson), overScopedRequest); !errors.Is(err, atprotocol.ErrOAuthScopeRejected) {
		t.Fatalf("over-scoped SaveAuthRequestInfo() error = %v", err)
	}

	expired := oauthRequest("expired-state")
	if err := store.SaveAuthRequestInfo(atprotocol.WithOAuthLinkPerson(t.Context(), firstPerson), expired); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(t.Context(), `update atproto_oauth_requests set expires_at = now() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAuthRequestInfo(t.Context(), expired.State); !errors.Is(err, atprotocol.ErrOAuthRequestNotFound) {
		t.Fatalf("expired GetAuthRequestInfo() error = %v", err)
	}
	deleted, err := store.DeleteExpiredRequests(t.Context())
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteExpiredRequests() = %d, %v", deleted, err)
	}

	did := syntax.DID("did:plc:sharedoauthdid")
	firstRequest := oauthRequest("first-link-state")
	saveAndClaimOAuthRequest(t, store, firstPerson, firstRequest)
	if err := store.SaveSession(t.Context(), oauthSession(did, firstRequest.State, "first-access", "first-refresh")); err != nil {
		t.Fatal(err)
	}

	secondRequest := oauthRequest("second-link-state")
	saveAndClaimOAuthRequest(t, store, secondPerson, secondRequest)
	if err := store.SaveSession(t.Context(), oauthSession(did, secondRequest.State, "second-access", "second-refresh")); err == nil || !strings.Contains(err.Error(), "another account") {
		t.Fatalf("cross-account SaveSession() error = %v", err)
	}
	var sessionCount int
	if err := db.QueryRow(t.Context(), `select count(*) from atproto_oauth_sessions where did = $1`, did.String()).Scan(&sessionCount); err != nil || sessionCount != 1 {
		t.Fatalf("session count = %d, error = %v", sessionCount, err)
	}

	scopeRequest := oauthRequest("scope-state")
	saveAndClaimOAuthRequest(t, store, secondPerson, scopeRequest)
	overScoped := oauthSession(syntax.DID("did:plc:overscoped"), scopeRequest.State, "scope-access", "scope-refresh")
	overScoped.Scopes = []string{"atproto", "repo:tv.subcult.event?action=create"}
	if err := store.SaveSession(t.Context(), overScoped); !errors.Is(err, atprotocol.ErrOAuthScopeRejected) {
		t.Fatalf("over-scoped SaveSession() error = %v", err)
	}
	invalidDID := oauthSession(syntax.DID("not-a-did"), scopeRequest.State, "invalid-access", "invalid-refresh")
	if err := store.SaveSession(t.Context(), invalidDID); err == nil || !strings.Contains(err.Error(), "DID is invalid") {
		t.Fatalf("invalid-DID SaveSession() error = %v", err)
	}
}

func TestOAuthStoreListsAndLocallyRevokesLink(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "oauth-store-test")
	if err != nil {
		t.Fatal(err)
	}
	personID := insertOAuthStorePerson(t, db, "oauth-unlink@example.test")
	request := oauthRequest("unlink-state")
	saveAndClaimOAuthRequest(t, store, personID, request)
	did := syntax.DID("did:plc:oauthunlinkowner")
	if err := store.SaveSession(t.Context(), oauthSession(did, request.State, "unlink-access", "unlink-refresh")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(t.Context(), `update did_links set handle = 'unlink.example.com' where did = $1`, did.String()); err != nil {
		t.Fatal(err)
	}
	links, err := store.ListActiveLinks(t.Context(), personID)
	if err != nil || len(links) != 1 || links[0].DID != did.String() || links[0].Handle != "unlink.example.com" {
		t.Fatalf("ListActiveLinks() = %#v, %v", links, err)
	}
	if err := store.RevokeLocalLink(t.Context(), personID, did.String()); err != nil {
		t.Fatal(err)
	}
	links, err = store.ListActiveLinks(t.Context(), personID)
	if err != nil || len(links) != 0 {
		t.Fatalf("links after revoke = %#v, %v", links, err)
	}
	var sessionCount, auditCount int
	if err := db.QueryRow(t.Context(), `select count(*) from atproto_oauth_sessions where did = $1`, did.String()).Scan(&sessionCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(t.Context(), `select count(*) from auth_audit_events where person_id = $1 and event_type = 'atproto_did_unlinked'`, personID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if sessionCount != 0 || auditCount != 1 {
		t.Fatalf("local revoke sessions=%d audit=%d, want 0/1", sessionCount, auditCount)
	}
	if err := store.RevokeLocalLink(t.Context(), personID, did.String()); !errors.Is(err, atprotocol.ErrOAuthLinkNotFound) {
		t.Fatalf("second revoke error = %v", err)
	}
}

func saveAndClaimOAuthRequest(t *testing.T, store *atprotocol.OAuthStore, personID string, request atprotocoloauth.AuthRequestData) {
	t.Helper()
	if err := store.SaveAuthRequestInfo(atprotocol.WithOAuthLinkPerson(t.Context(), personID), request); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAuthRequestInfo(t.Context(), request.State); err != nil {
		t.Fatal(err)
	}
}

func oauthRequest(state string) atprotocoloauth.AuthRequestData {
	return atprotocoloauth.AuthRequestData{
		State:                        state,
		AuthServerURL:                "https://auth.example.test",
		Scopes:                       []string{"atproto"},
		RequestURI:                   "urn:ietf:params:oauth:request_uri:" + state,
		AuthServerTokenEndpoint:      "https://auth.example.test/oauth/token",
		AuthServerRevocationEndpoint: "https://auth.example.test/oauth/revoke",
		PKCEVerifier:                 "pkce-verifier-" + state,
		DPoPAuthServerNonce:          "dpop-nonce-" + state,
		DPoPPrivateKeyMultibase:      "private-key-" + state,
	}
}

func oauthSession(did syntax.DID, sessionID, accessToken, refreshToken string) atprotocoloauth.ClientSessionData {
	return atprotocoloauth.ClientSessionData{
		AccountDID:                   did,
		SessionID:                    sessionID,
		HostURL:                      "https://pds.example.test",
		AuthServerURL:                "https://auth.example.test",
		AuthServerTokenEndpoint:      "https://auth.example.test/oauth/token",
		AuthServerRevocationEndpoint: "https://auth.example.test/oauth/revoke",
		Scopes:                       []string{"atproto"},
		AccessToken:                  accessToken,
		RefreshToken:                 refreshToken,
		DPoPAuthServerNonce:          "auth-nonce",
		DPoPHostNonce:                "host-nonce",
		DPoPPrivateKeyMultibase:      "session-private-key",
	}
}

func insertOAuthStorePerson(t *testing.T, db *pgxpool.Pool, email string) string {
	t.Helper()
	var personID string
	if err := db.QueryRow(t.Context(), `
		insert into people (email, password_hash) values ($1, 'not-a-login-hash') returning id
	`, email).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	return personID
}

func newOAuthStoreTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run AT OAuth store integration tests")
	}
	base, err := pgxpool.New(t.Context(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("atproto_oauth_test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := base.Exec(t.Context(), "create schema "+identifier); err != nil {
		base.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	if err := appcore.RunMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := base.Exec(cleanupCtx, "drop schema "+identifier+" cascade"); err != nil {
			t.Errorf("drop AT OAuth test schema: %v", err)
		}
		base.Close()
	})
	return db
}
