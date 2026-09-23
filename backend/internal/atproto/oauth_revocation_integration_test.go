package atproto_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOAuthStoreUnlinkOutboxAtomicityAndLateRotation(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "", "oauth-store-test")
	if err != nil {
		t.Fatal(err)
	}
	person := insertOAuthStorePerson(t, db, "revocation@example.test")
	did := syntax.DID("did:plc:outboxowner")
	request := oauthRequest("outbox-state")
	saveAndClaimOAuthRequest(t, store, person, request)
	session := oauthSession(did, request.State, "secret-access-before", "secret-refresh-before")
	if err := store.SaveSession(t.Context(), session); err != nil {
		t.Fatal(err)
	}

	// Storage failure must not disable the link while losing its credentials.
	if _, err := db.Exec(t.Context(), `alter table atproto_oauth_revocations add constraint force_queue_failure check (false) not valid`); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeLocalLink(t.Context(), person, did.String()); err == nil {
		t.Fatal("unlink succeeded despite queue failure")
	}
	if _, err := store.GetSession(t.Context(), did, request.State); err != nil {
		t.Fatalf("rollback lost session: %v", err)
	}
	if _, err := db.Exec(t.Context(), `alter table atproto_oauth_revocations drop constraint force_queue_failure`); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeLocalLink(t.Context(), person, did.String()); err != nil {
		t.Fatal(err)
	}
	var payload []byte
	var status string
	if err := db.QueryRow(t.Context(), `select payload_ciphertext, status from atproto_oauth_revocations`).Scan(&payload, &status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || bytes.Contains(payload, []byte(session.AccessToken)) || bytes.Contains(payload, []byte(session.RefreshToken)) {
		t.Fatal("outbox status/encryption invariant failed")
	}
	if _, err := store.GetSession(t.Context(), did, request.State); !errors.Is(err, atprotocol.ErrOAuthSessionNotFound) {
		t.Fatalf("unlinked session remains readable: %v", err)
	}

	// Simulate a worker holding the old payload when an in-flight refresh saves.
	if _, err := db.Exec(t.Context(), `update atproto_oauth_revocations set status='leased', lease_until=now()+interval '1 minute', lease_token=gen_random_uuid()`); err != nil {
		t.Fatal(err)
	}
	session.AccessToken = "secret-access-after"
	session.RefreshToken = "secret-refresh-after"
	if err := store.SaveSession(t.Context(), session); !errors.Is(err, atprotocol.ErrOAuthLinkRevoked) {
		t.Fatalf("late refresh error = %v", err)
	}
	var after []byte
	var unleased bool
	if err := db.QueryRow(t.Context(), `select payload_ciphertext, status, lease_token is null and lease_until is null from atproto_oauth_revocations`).Scan(&after, &status, &unleased); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || !unleased || bytes.Equal(payload, after) || bytes.Contains(after, []byte(session.RefreshToken)) {
		t.Fatal("late refresh did not fence and replace encrypted revocation work")
	}
	if links, err := store.ListActiveLinks(t.Context(), person); err != nil || len(links) != 0 {
		t.Fatalf("late refresh resurrected link: %v", err)
	}
	var count int
	if err := db.QueryRow(t.Context(), `select count(*) from atproto_oauth_sessions`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("active sessions=%d error=%v", count, err)
	}
}

func revocationFixture(t *testing.T) (*pgxpool.Pool, *atprotocol.OAuthStore, oauth.ClientSessionData) {
	t.Helper()
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "", "revocation-worker-test")
	if err != nil {
		t.Fatal(err)
	}
	person := insertOAuthStorePerson(t, db, "worker@example.test")
	request := oauthRequest("worker-state")
	saveAndClaimOAuthRequest(t, store, person, request)
	session := oauthSession(syntax.DID("did:plc:workerowner"), request.State, "worker-access", "worker-refresh")
	if err := store.SaveSession(t.Context(), session); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeLocalLink(t.Context(), person, session.AccountDID.String()); err != nil {
		t.Fatal(err)
	}
	return db, store, session
}

func TestOAuthStoreRevocationRetriesThenPurgesSecrets(t *testing.T) {
	db, store, session := revocationFixture(t)
	report, err := store.ProcessRevocations(t.Context(), func(_ context.Context, got oauth.ClientSessionData) error {
		if got.RefreshToken != session.RefreshToken {
			t.Fatal("wrong decrypted revocation payload")
		}
		return errors.New("secret token and provider text must not be persisted")
	}, 1)
	if err != nil || report.Retried != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	var code string
	var retained, delayed bool
	if err := db.QueryRow(t.Context(), `select last_error_code, payload_ciphertext is not null, next_attempt_at > now() from atproto_oauth_revocations`).Scan(&code, &retained, &delayed); err != nil {
		t.Fatal(err)
	}
	if code != "provider_failed" || !retained || !delayed {
		t.Fatal("unsafe failure persistence or missing backoff")
	}
	report, err = store.ProcessRevocations(t.Context(), func(context.Context, oauth.ClientSessionData) error { t.Fatal("retried before due"); return nil }, 1)
	if err != nil || report.Attempted != 0 {
		t.Fatalf("early retry=%+v err=%v", report, err)
	}
	if _, err := db.Exec(t.Context(), `update atproto_oauth_revocations set next_attempt_at=now()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	report, err = store.ProcessRevocations(t.Context(), func(context.Context, oauth.ClientSessionData) error { return nil }, 1)
	if err != nil || report.Completed != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	counts, err := store.RevocationCounts(t.Context())
	if err != nil || counts["completed"] != 1 || counts["pending"] != 0 {
		t.Fatalf("counts=%v err=%v", counts, err)
	}
	if err := db.QueryRow(t.Context(), `select payload_ciphertext is not null from atproto_oauth_revocations`).Scan(&retained); err != nil || retained {
		t.Fatalf("secret retained after completion: %v", err)
	}
}

func TestOAuthStoreRevocationTerminalAndCrashRecovery(t *testing.T) {
	for _, tc := range []struct {
		name, prepare string
		handlerErr    error
		wantCode      string
		calls         int
	}{
		{"unsupported", "", atprotocol.ErrRevocationUnsupported, "unsupported", 1},
		{"invalid", "update atproto_oauth_revocations set payload_ciphertext='broken'::bytea", nil, "invalid_payload", 0},
		{"exhausted", "update atproto_oauth_revocations set attempts=7", errors.New("outage"), "retry_exhausted", 1},
		{"expired", "update atproto_oauth_revocations set expires_at=now()-interval '1 second'", nil, "retention_expired", 0},
		{"last-lease-crash", "update atproto_oauth_revocations set attempts=8, status='leased', lease_token=gen_random_uuid(), lease_until=now()-interval '1 second'", nil, "retry_exhausted", 0},
		{"lease-recovery", "update atproto_oauth_revocations set attempts=1, status='leased', lease_token=gen_random_uuid(), lease_until=now()-interval '1 second'", nil, "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, store, _ := revocationFixture(t)
			if tc.prepare != "" {
				if _, err := db.Exec(t.Context(), tc.prepare); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			_, err := store.ProcessRevocations(t.Context(), func(context.Context, oauth.ClientSessionData) error { calls++; return tc.handlerErr }, 1)
			if err != nil {
				t.Fatal(err)
			}
			var code, status string
			var purged bool
			if err := db.QueryRow(t.Context(), `select coalesce(last_error_code,''),status,payload_ciphertext is null from atproto_oauth_revocations`).Scan(&code, &status, &purged); err != nil {
				t.Fatal(err)
			}
			if calls != tc.calls || code != tc.wantCode || !purged {
				t.Fatalf("calls=%d code=%q purged=%t", calls, code, purged)
			}
			if (tc.wantCode == "" && status != "completed") || (tc.wantCode != "" && status != "quarantined") {
				t.Fatalf("wrong terminal state %s", status)
			}
		})
	}
}

func TestOAuthStoreRevocationFencesLateRotationAndConcurrentWorkers(t *testing.T) {
	_, store, session := revocationFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	go func() {
		report, err := store.ProcessRevocations(t.Context(), func(context.Context, oauth.ClientSessionData) error { close(entered); <-release; return nil }, 1)
		if err == nil && report.Superseded != 1 {
			err = errors.New("old worker acknowledged new credentials")
		}
		result <- err
	}()
	<-entered
	other, err := store.ProcessRevocations(t.Context(), func(context.Context, oauth.ClientSessionData) error { return errors.New("duplicate claim") }, 1)
	if err != nil || other.Attempted != 0 {
		close(release)
		t.Fatalf("concurrent claim=%+v err=%v", other, err)
	}
	session.RefreshToken = "rotated-while-leased"
	if err := store.SaveSession(t.Context(), session); !errors.Is(err, atprotocol.ErrOAuthLinkRevoked) {
		close(release)
		t.Fatalf("rotation=%v", err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	report, err := store.ProcessRevocations(t.Context(), func(_ context.Context, got oauth.ClientSessionData) error {
		if got.RefreshToken != session.RefreshToken {
			return errors.New("lost rotated credentials")
		}
		return nil
	}, 1)
	if err != nil || report.Completed != 1 {
		t.Fatalf("rotated report=%+v err=%v", report, err)
	}
}

func TestOAuthStoreConcurrentUnlinkAndRefreshCannotResurrect(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "", "oauth-store-test")
	if err != nil {
		t.Fatal(err)
	}
	person := insertOAuthStorePerson(t, db, "concurrent-revocation@example.test")
	did := syntax.DID("did:plc:concurrentoutbox")
	request := oauthRequest("concurrent-outbox-state")
	saveAndClaimOAuthRequest(t, store, person, request)
	session := oauthSession(did, request.State, "access-before", "refresh-before")
	if err := store.SaveSession(t.Context(), session); err != nil {
		t.Fatal(err)
	}
	session.RefreshToken = "refresh-after"
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); <-start; results <- store.SaveSession(t.Context(), session) }()
	go func() { defer wg.Done(); <-start; results <- store.RevokeLocalLink(t.Context(), person, did.String()) }()
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, atprotocol.ErrOAuthLinkRevoked) {
			t.Fatal(err)
		}
	}
	var active, queued int
	if err := db.QueryRow(t.Context(), `select count(*) from atproto_oauth_sessions`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(t.Context(), `select count(*) from atproto_oauth_revocations where status='pending'`).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if active != 0 || queued != 1 {
		t.Fatalf("active=%d queued=%d", active, queued)
	}
}
