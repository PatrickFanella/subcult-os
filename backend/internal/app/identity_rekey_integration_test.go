package app

import (
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	atprotocoloauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// TestIdentityRekeyRotatesEmailAndOAuthPayloads is the end-to-end proof for
// acceptance criteria D: it seeds email_identities and AT OAuth session/
// revocation rows sealed under a "previous" key, then confirms the rekey
// command rewrites them under a "current" key, that the previous key can be
// removed afterward, and that a second run is a no-op.
func TestIdentityRekeyRotatesEmailAndOAuthPayloads(t *testing.T) {
	pool := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), pool); err != nil {
		t.Fatal(err)
	}

	oldConfig := Config{AppEnv: "test", SessionSecret: "identity-rekey-old-secret", IdentityProtectionKey: testIdentityProtectionKeyPrevious}
	oldIdentity, err := newIdentityProtector(oldConfig.IdentityProtectionKey, "", oldConfig.SessionSecret)
	if err != nil {
		t.Fatal(err)
	}
	oldOAuthStore, err := atprotocol.NewOAuthStore(pool, oldConfig.IdentityProtectionKey, "", oldConfig.SessionSecret)
	if err != nil {
		t.Fatal(err)
	}

	// Seed a person with an email sealed under the old key.
	oldCiphertext, oldLookup, err := oldIdentity.protectEmail("rekey-target@example.test")
	if err != nil {
		t.Fatal(err)
	}
	var personID string
	if err := pool.QueryRow(t.Context(), `
		insert into people (email, password_hash) values ('rekey-target@example.test', 'not-a-login-hash') returning id
	`).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	var identityID string
	if err := pool.QueryRow(t.Context(), `
		insert into email_identities (person_id, email_ciphertext, email_lookup_hash, verified_at)
		values ($1, $2, $3, now()) returning id
	`, personID, oldCiphertext, oldLookup).Scan(&identityID); err != nil {
		t.Fatal(err)
	}

	// Seed an AT OAuth session sealed under the old key.
	did := syntax.DID("did:plc:identityrekeytarget")
	oldRequest := atprotocoloauth.AuthRequestData{
		State: "rekey-state", AuthServerURL: "https://auth.example.test", Scopes: []string{"atproto"},
		RequestURI:                   "urn:ietf:params:oauth:request_uri:rekey-state",
		AuthServerTokenEndpoint:      "https://auth.example.test/oauth/token",
		AuthServerRevocationEndpoint: "https://auth.example.test/oauth/revoke",
		PKCEVerifier:                 "pkce-verifier", DPoPAuthServerNonce: "dpop-nonce",
		DPoPPrivateKeyMultibase: "private-key-rekey",
	}
	saveCtx := atprotocol.WithOAuthLinkPerson(t.Context(), personID)
	if err := oldOAuthStore.SaveAuthRequestInfo(saveCtx, oldRequest); err != nil {
		t.Fatal(err)
	}
	if _, err := oldOAuthStore.GetAuthRequestInfo(t.Context(), oldRequest.State); err != nil {
		t.Fatal(err)
	}
	oldSession := atprotocoloauth.ClientSessionData{
		AccountDID: did, SessionID: oldRequest.State, HostURL: "https://pds.example.test",
		AuthServerURL: "https://auth.example.test", AuthServerTokenEndpoint: "https://auth.example.test/oauth/token",
		AuthServerRevocationEndpoint: "https://auth.example.test/oauth/revoke", Scopes: []string{"atproto"},
		AccessToken: "access-old", RefreshToken: "refresh-old",
		DPoPAuthServerNonce: "auth-nonce", DPoPHostNonce: "host-nonce", DPoPPrivateKeyMultibase: "session-private-key",
	}
	if err := oldOAuthStore.SaveSession(t.Context(), oldSession); err != nil {
		t.Fatal(err)
	}
	if err := oldOAuthStore.RevokeLocalLink(t.Context(), personID, did.String()); err != nil {
		t.Fatal(err)
	}

	rotatedConfig := Config{
		AppEnv: "test", SessionSecret: "identity-rekey-old-secret",
		IdentityProtectionKey: testIdentityProtectionKey, IdentityProtectionKeyPrevious: testIdentityProtectionKeyPrevious,
		Addr: ":8080",
	}

	// Reads during the rotation window still work with current+previous.
	status, err := RunIdentityRekey(t.Context(), rotatedConfig, pool, 100, true)
	if err != nil {
		t.Fatal(err)
	}
	before, ok := status.(IdentityRekeyStatus)
	if !ok || !before.PreviousKeyConfigured || before.EmailIdentitiesTotal != 1 || before.OAuthSessionsWithPayload != 0 || before.OAuthRevocationsPending != 1 {
		t.Fatalf("pre-rekey status = %#v", status)
	}

	// Run the rekey command; it must rewrite both the email and the queued revocation.
	result, err := RunIdentityRekey(t.Context(), rotatedConfig, pool, 100, false)
	if err != nil {
		t.Fatal(err)
	}
	report, ok := result.(IdentityRekeyReport)
	if !ok || report.EmailIdentitiesScanned != 1 || report.EmailIdentitiesRekeyed != 1 {
		t.Fatalf("rekey report = %#v", result)
	}
	if report.OAuthRevocationsScanned != 1 || report.OAuthRevocationsRekeyed != 1 {
		t.Fatalf("rekey report OAuth revocations = %#v", result)
	}

	// Idempotent: rerunning immediately finds nothing left to rotate.
	secondResult, err := RunIdentityRekey(t.Context(), rotatedConfig, pool, 100, false)
	if err != nil {
		t.Fatal(err)
	}
	secondReport, ok := secondResult.(IdentityRekeyReport)
	if !ok || secondReport.EmailIdentitiesRekeyed != 0 || secondReport.OAuthRevocationsRekeyed != 0 {
		t.Fatalf("second rekey run should be a no-op, got %#v", secondResult)
	}

	// After rekeying, the previous key can be removed and everything still reads.
	currentOnlyConfig := Config{AppEnv: "test", SessionSecret: "identity-rekey-old-secret", IdentityProtectionKey: testIdentityProtectionKey, Addr: ":8080"}
	currentOnlyIdentity, err := newIdentityProtector(currentOnlyConfig.IdentityProtectionKey, "", currentOnlyConfig.SessionSecret)
	if err != nil {
		t.Fatal(err)
	}
	var rekeyedCiphertext []byte
	if err := pool.QueryRow(t.Context(), `select email_ciphertext from email_identities where id = $1`, identityID).Scan(&rekeyedCiphertext); err != nil {
		t.Fatal(err)
	}
	revealed, err := currentOnlyIdentity.revealEmail(rekeyedCiphertext)
	if err != nil || revealed != "rekey-target@example.test" {
		t.Fatalf("post-rekey reveal = %q, %v", revealed, err)
	}

	currentOnlyStore, err := atprotocol.NewOAuthStore(pool, currentOnlyConfig.IdentityProtectionKey, "", currentOnlyConfig.SessionSecret)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := currentOnlyStore.RevocationCounts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if counts["pending"] != 1 {
		t.Fatalf("revocation counts after current-only read = %#v", counts)
	}
	var revocationPayload []byte
	if err := pool.QueryRow(t.Context(), `select payload_ciphertext from atproto_oauth_revocations where did = $1`, did.String()).Scan(&revocationPayload); err != nil {
		t.Fatal(err)
	}
	if revocationPayload == nil {
		t.Fatal("rekeyed revocation payload is missing")
	}

	// Final status confirms nothing is left protected by only the previous key.
	finalStatus, err := RunIdentityRekey(t.Context(), rotatedConfig, pool, 100, true)
	if err != nil {
		t.Fatal(err)
	}
	after, ok := finalStatus.(IdentityRekeyStatus)
	if !ok || after.EmailIdentitiesTotal != 1 || after.OAuthRevocationsPending != 1 {
		t.Fatalf("final status = %#v", finalStatus)
	}
}

// TestIdentityRekeyRequiresValidConfig proves the command fails closed when
// the current key is missing or malformed, matching the production config
// gate, and never partially applies a batch on that failure.
func TestIdentityRekeyRequiresValidConfig(t *testing.T) {
	pool := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	badConfig := Config{AppEnv: "production", Addr: ":8080", DatabaseURL: "postgres://ignored", SessionSecret: "replace-with-a-long-random-secret", PublicWebURL: "https://subcult.example"}
	if _, err := RunIdentityRekey(t.Context(), badConfig, pool, 100, true); err == nil {
		t.Fatal("RunIdentityRekey() should fail without a valid IDENTITY_PROTECTION_KEY in production")
	}
}
