//go:build recovery_rehearsal

package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// scripts/qa-backup-recovery.py runs this explicitly tagged fixture. Its
// databases are disposable launcher services, never TEST_DATABASE_URL.
func TestBackupRecoveryFixture(t *testing.T) {
	phase := os.Getenv("RECOVERY_PHASE")
	u, err := url.Parse(os.Getenv("RECOVERY_DB_URL"))
	if err != nil || u.Scheme != "postgres" || u.Path != "/backup_rehearsal" ||
		(phase == "seed" && u.Host != "db:5432") ||
		(phase != "seed" && u.Host != "restored:5432") {
		t.Fatal("rehearsal requires its disposable database service")
	}
	if phase != "seed" && phase != "verify" && phase != "wrong-identity" && phase != "wrong-signing" {
		t.Fatal("unknown recovery phase")
	}
	key := os.Getenv("RECOVERY_IDENTITY_KEY")
	if _, err := decodeIdentityProtectionKey(key); err != nil {
		t.Fatal("rehearsal requires an explicit synthetic identity key")
	}
	protector, err := newIdentityProtector(key, "", "")
	if err != nil {
		t.Fatal("construct identity protector")
	}
	signingKey, err := atcrypto.ParsePrivateMultibase(os.Getenv("RECOVERY_SIGNING_KEY"))
	if err != nil {
		t.Fatal("rehearsal requires a synthetic signing key")
	}
	client, err := atprotocol.NewOAuthClient(atprotocol.OAuthClientSettings{
		ClientID: "https://recovery.example.test/client", CallbackURL: "https://recovery.example.test/callback",
		JWKSURL: "https://recovery.example.test/jwks", KeyID: "recovery-test", PrivateKey: signingKey.Multibase(),
	})
	if err != nil {
		t.Fatal("construct synthetic OAuth client")
	}
	pool, err := OpenDB(t.Context(), u.String())
	if err != nil || pool == nil {
		t.Fatal("open disposable recovery database")
	}
	defer pool.Close()
	store, err := atprotocol.NewOAuthStore(pool, key, "", "")
	if err != nil {
		t.Fatal("construct OAuth store")
	}
	const email = "backup-fixture@example.test"
	const activeDID = syntax.DID("did:plc:recoveryactive")
	const revokedDID = syntax.DID("did:plc:recoveryrevoked")
	const challenge = "subcult-os synthetic backup recovery challenge"
	if phase == "seed" {
		var tables int
		if err := pool.QueryRow(t.Context(), `select count(*) from information_schema.tables where table_schema='public'`).Scan(&tables); err != nil || tables != 0 {
			t.Fatal("refusing to seed a nonempty database")
		}
		if err := RunMigrations(t.Context(), pool); err != nil {
			t.Fatal("migrate synthetic source")
		}
		ciphertext, lookup, err := protector.protectEmail(email)
		if err != nil {
			t.Fatal("encrypt synthetic email")
		}
		var personID string
		if err := pool.QueryRow(t.Context(), `insert into people(email,password_hash) values ($1,'not-a-login-hash') returning id`, email).Scan(&personID); err != nil {
			t.Fatal("seed synthetic person")
		}
		if _, err := pool.Exec(t.Context(), `insert into email_identities(person_id,email_ciphertext,email_lookup_hash,verified_at) values ($1,$2,$3,now())`, personID, ciphertext, lookup); err != nil {
			t.Fatal("seed encrypted identity")
		}
		for i, did := range []syntax.DID{activeDID, revokedDID} {
			dpopKey, err := atcrypto.GeneratePrivateKeyP256()
			if err != nil {
				t.Fatal("generate synthetic DPoP key")
			}
			state := "active-session"
			if i == 1 {
				state = "revoked-session"
			}
			request := oauth.AuthRequestData{State: state, AuthServerURL: "https://auth.example.test", Scopes: []string{"atproto"}, PKCEVerifier: "synthetic-pkce"}
			ctx := atprotocol.WithOAuthLinkPerson(t.Context(), personID)
			if err := store.SaveAuthRequestInfo(ctx, request); err != nil {
				t.Fatal("seed encrypted OAuth request")
			}
			if _, err := store.GetAuthRequestInfo(t.Context(), state); err != nil {
				t.Fatal("claim synthetic OAuth request")
			}
			session := oauth.ClientSessionData{AccountDID: did, SessionID: state, HostURL: "https://pds.example.test", AuthServerURL: "https://auth.example.test", AuthServerRevocationEndpoint: "https://auth.example.test/revoke", Scopes: []string{"atproto"}, AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh", DPoPPrivateKeyMultibase: dpopKey.Multibase()}
			if err := store.SaveSession(t.Context(), session); err != nil {
				t.Fatal("seed encrypted OAuth session")
			}
			if i == 1 {
				if err := store.RevokeLocalLink(t.Context(), personID, did.String()); err != nil {
					t.Fatal("seed queued revocation")
				}
			}
		}
		if err := store.SaveAuthRequestInfo(atprotocol.WithOAuthLinkPerson(t.Context(), personID), oauth.AuthRequestData{State: "pending-request", AuthServerURL: "https://auth.example.test", Scopes: []string{"atproto"}, PKCEVerifier: "synthetic-pkce"}); err != nil {
			t.Fatal("seed pending request")
		}
		publicKey, err := signingKey.PublicKey()
		if err != nil {
			t.Fatal("derive public signing key")
		}
		if _, err := pool.Exec(t.Context(), `create table recovery_rehearsal_marker(public_key text not null,jwks_sha256 text not null)`); err != nil {
			t.Fatal("create fixture marker")
		}
		digest := sha256.Sum256(client.JWKSJSON())
		if _, err := pool.Exec(t.Context(), `insert into recovery_rehearsal_marker values ($1,$2)`, publicKey.Multibase(), hex.EncodeToString(digest[:])); err != nil {
			t.Fatal("record public signing identity")
		}
		t.Log("seeded encrypted identity, OAuth session, request and revocation; public signing identity recorded")
		return
	}
	version, err := CurrentSchemaVersion(t.Context(), pool)
	if err != nil || version != minimumSchemaVersion {
		t.Fatal("restored migration version mismatch")
	}
	var ciphertext []byte
	if err := pool.QueryRow(t.Context(), `select email_ciphertext from email_identities`).Scan(&ciphertext); err != nil {
		t.Fatal("load restored email ciphertext")
	}
	revealed, emailErr := protector.revealEmail(ciphertext)
	session, sessionErr := store.GetSession(t.Context(), activeDID, "active-session")
	if phase == "wrong-identity" {
		if emailErr == nil || sessionErr == nil || errors.Is(sessionErr, atprotocol.ErrOAuthSessionNotFound) {
			t.Fatal("wrong key did not reject existing protected data")
		}
		t.Log("wrong identity key rejected restored email and OAuth ciphertext")
		return
	}
	if emailErr != nil || revealed != email || sessionErr != nil || session.AccessToken != "synthetic-access" || session.RefreshToken != "synthetic-refresh" {
		t.Fatal("matching identity key did not recover fixture")
	}
	var publicKeyText, expectedJWKS string
	if err := pool.QueryRow(t.Context(), `select public_key,jwks_sha256 from recovery_rehearsal_marker`).Scan(&publicKeyText, &expectedJWKS); err != nil {
		t.Fatal("load signing identity receipt")
	}
	publicKey, err := atcrypto.ParsePublicMultibase(publicKeyText)
	if err != nil {
		t.Fatal("parse recorded public key")
	}
	sig, err := signingKey.HashAndSign([]byte(challenge))
	if err != nil {
		t.Fatal("sign recovery challenge")
	}
	verifyErr := publicKey.HashAndVerify([]byte(challenge), sig)
	digest := sha256.Sum256(client.JWKSJSON())
	matchingJWKS := hex.EncodeToString(digest[:]) == expectedJWKS
	if phase == "wrong-signing" {
		if verifyErr == nil || matchingJWKS {
			t.Fatal("wrong signing key matched restored identity")
		}
		t.Log("wrong signing key rejected by public-key challenge and JWKS identity")
		return
	}
	if verifyErr != nil || !matchingJWKS {
		t.Fatal("restored signing key continuity failed")
	}
	request, err := store.GetAuthRequestInfo(t.Context(), "pending-request")
	if err != nil || request.PKCEVerifier != "synthetic-pkce" {
		t.Fatal("recover pending OAuth request")
	}
	calls := 0
	report, err := store.ProcessRevocations(t.Context(), func(_ context.Context, data oauth.ClientSessionData) error {
		if data.AccountDID != revokedDID || data.AccessToken != "synthetic-access" || data.RefreshToken != "synthetic-refresh" {
			return errors.New("fixture mismatch")
		}
		calls++
		return nil
	}, 1)
	if err != nil || report.Completed != 1 || calls != 1 {
		t.Fatal("recover revocation payload with fake provider")
	}
	t.Log("restored schema, email, OAuth session/request/revocation, signing challenge and JWKS identity passed; no provider calls")
}
