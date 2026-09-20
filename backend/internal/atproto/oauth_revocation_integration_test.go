package atproto_test

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestOAuthStoreUnlinkOutboxAtomicityAndLateRotation(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "oauth-store-test")
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

func TestOAuthStoreConcurrentUnlinkAndRefreshCannotResurrect(t *testing.T) {
	db := newOAuthStoreTestPool(t)
	store, err := atprotocol.NewOAuthStore(db, "", "oauth-store-test")
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
