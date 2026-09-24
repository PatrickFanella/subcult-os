package app

import (
	"errors"
	"fmt"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
)

func newProjectionTestProcessor(t *testing.T) *ProjectionProcessor {
	t.Helper()
	db := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	catalog, err := atprotocol.LoadLexiconCatalog(lexiconContractDirForTest)
	if err != nil {
		t.Fatal(err)
	}
	return NewProjectionProcessor(db, catalog)
}

func validProfileRecord(displayName string) []byte {
	return []byte(fmt.Sprintf(`{"$type":"tv.subcult.profile","displayName":%q,"createdAt":"2026-09-23T00:00:00Z"}`, displayName))
}

func countRows(t *testing.T, p *ProjectionProcessor, table string) int {
	t.Helper()
	var count int
	if err := p.db.QueryRow(t.Context(), "select count(*) from "+table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestProjectionAdmitsOnlyAllowlistedCollections(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	admitted := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafyprofile1", Rev: "3jz1",
		Record: validProfileRecord("Alice"),
	}
	outcome, err := p.ProcessEvent(ctx, admitted, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeStored {
		t.Fatalf("outcome = %s, want stored", outcome)
	}

	notAdmitted := StreamEvent{
		Cursor: "2", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "app.bsky.feed.post", RKey: "abc",
		CID: "bafypost1", Rev: "3jz2",
		Record: []byte(`{"$type":"app.bsky.feed.post","text":"hello"}`),
	}
	outcome, err = p.ProcessEvent(ctx, notAdmitted, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeSkippedKind {
		t.Fatalf("outcome = %s, want skipped_collection", outcome)
	}

	if got := countRows(t, p, "at_projection_records"); got != 1 {
		t.Fatalf("at_projection_records rows = %d, want 1 (non-admitted collection not stored)", got)
	}
	cursor, err := p.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "2" {
		t.Fatalf("cursor = %q, want %q (advances even for skipped collections)", cursor, "2")
	}
}

func TestProjectionRejectsRecordFailingLexiconValidation(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	event := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafybad1", Rev: "3jz1",
		Record: []byte(`{"$type":"tv.subcult.profile","createdAt":"2026-09-23T00:00:00Z"}`), // missing required displayName
	}
	outcome, err := p.ProcessEvent(ctx, event, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeQuarantined {
		t.Fatalf("outcome = %s, want quarantined", outcome)
	}
	if got := countRows(t, p, "at_projection_records"); got != 0 {
		t.Fatalf("at_projection_records rows = %d, want 0", got)
	}
	if got := countRows(t, p, "at_projection_quarantine"); got != 1 {
		t.Fatalf("at_projection_quarantine rows = %d, want 1", got)
	}
	cursor, err := p.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "1" {
		t.Fatal("cursor must still advance past a quarantined event")
	}
}

func TestProjectionRejectsPrivateFieldLeak(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	event := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafyleak1", Rev: "3jz1",
		Record: []byte(`{"$type":"tv.subcult.profile","displayName":"Alice","createdAt":"2026-09-23T00:00:00Z","internalStaffNote":"do not publish"}`),
	}
	outcome, err := p.ProcessEvent(ctx, event, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeQuarantined {
		t.Fatalf("outcome = %s, want quarantined", outcome)
	}
}

func TestProjectionRejectsOversizeRecord(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	oversizeDescription := make([]byte, projectionMaxRecordBytes+1)
	for i := range oversizeDescription {
		oversizeDescription[i] = 'a'
	}
	record := append([]byte(`{"$type":"tv.subcult.profile","displayName":"Alice","createdAt":"2026-09-23T00:00:00Z","description":"`), oversizeDescription...)
	record = append(record, []byte(`"}`)...)

	outcome, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafybig1", Rev: "3jz1", Record: record,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeQuarantined {
		t.Fatalf("outcome = %s, want quarantined", outcome)
	}
	// The stored quarantine payload must itself be bounded, independent of
	// the rejected record's size.
	var payloadLen int
	if err := p.db.QueryRow(ctx, "select length(payload) from at_projection_quarantine limit 1").Scan(&payloadLen); err != nil {
		t.Fatal(err)
	}
	if payloadLen > projectionQuarantinePayloadBytes+64 {
		t.Fatalf("quarantine payload length = %d, want bounded near %d", payloadLen, projectionQuarantinePayloadBytes)
	}
}

func TestProjectionRejectsMalformedJSON(t *testing.T) {
	p := newProjectionTestProcessor(t)
	outcome, err := p.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafymalformed1", Rev: "3jz1", Record: []byte(`{not json`),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeQuarantined {
		t.Fatalf("outcome = %s, want quarantined", outcome)
	}
}

func TestProjectionDuplicateReplayIsNoOp(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	event := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafysame1", Rev: "3jz1", Record: validProfileRecord("Alice"),
	}
	if _, err := p.ProcessEvent(ctx, event, nil); err != nil {
		t.Fatal(err)
	}

	replay := event
	replay.Cursor = "2"
	outcome, err := p.ProcessEvent(ctx, replay, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeDuplicate {
		t.Fatalf("outcome = %s, want duplicate", outcome)
	}
	if got := countRows(t, p, "at_projection_records"); got != 1 {
		t.Fatalf("at_projection_records rows = %d, want 1", got)
	}
	cursor, err := p.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "2" {
		t.Fatal("duplicate replay must still advance the cursor")
	}
}

func TestProjectionOutOfOrderUpdateIsIgnored(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	newer := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafynewer1", Rev: "3jz9", Record: validProfileRecord("Newer"),
	}
	if _, err := p.ProcessEvent(ctx, newer, nil); err != nil {
		t.Fatal(err)
	}

	older := StreamEvent{
		Cursor: "2", Kind: "commit", Operation: "update",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafyolder1", Rev: "3jz1", Record: validProfileRecord("Older"),
	}
	outcome, err := p.ProcessEvent(ctx, older, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeOutOfOrder {
		t.Fatalf("outcome = %s, want out_of_order", outcome)
	}

	var storedCID string
	if err := p.db.QueryRow(ctx, "select cid from at_projection_records where uri = $1", "at://did:plc:a/tv.subcult.profile/self").Scan(&storedCID); err != nil {
		t.Fatal(err)
	}
	if storedCID != "bafynewer1" {
		t.Fatalf("stored cid = %q, want the newer record to remain (out-of-order update rejected)", storedCID)
	}
	cursor, err := p.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "2" {
		t.Fatal("an out-of-order event must still advance the cursor")
	}
}

func TestProjectionDeleteMarksDeletedAndPreservesProvenance(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	create := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafycreate1", Rev: "3jz1", Record: validProfileRecord("Alice"),
	}
	if _, err := p.ProcessEvent(ctx, create, nil); err != nil {
		t.Fatal(err)
	}

	del := StreamEvent{
		Cursor: "2", Kind: "commit", Operation: "delete",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self", Rev: "3jz2",
	}
	outcome, err := p.ProcessEvent(ctx, del, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeDeleted {
		t.Fatalf("outcome = %s, want deleted", outcome)
	}

	var status, did, collection, rkey string
	if err := p.db.QueryRow(ctx, "select status, did, collection, rkey from at_projection_records where uri = $1",
		"at://did:plc:a/tv.subcult.profile/self").Scan(&status, &did, &collection, &rkey); err != nil {
		t.Fatal(err)
	}
	if status != "deleted" || did != "did:plc:a" || collection != "tv.subcult.profile" || rkey != "self" {
		t.Fatalf("row = (%q, %q, %q, %q), want deleted with preserved provenance", status, did, collection, rkey)
	}
}

func TestProjectionDeleteBeforeCreatePreservesTombstone(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	del := StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "delete",
		DID: "did:plc:a", Collection: "tv.subcult.place", RKey: "3jzabc", Rev: "3jz1",
	}
	outcome, err := p.ProcessEvent(ctx, del, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeDeleted {
		t.Fatalf("outcome = %s, want deleted", outcome)
	}
	if got := countRows(t, p, "at_projection_records"); got != 1 {
		t.Fatalf("at_projection_records rows = %d, want 1 (tombstone row created)", got)
	}
}

func TestProjectionAccountUnavailableAndReactivation(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	for i, rkey := range []string{"self"} {
		_ = i
		if _, err := p.ProcessEvent(ctx, StreamEvent{
			Cursor: "1", Kind: "commit", Operation: "create",
			DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: rkey,
			CID: "bafyacct1", Rev: "3jz1", Record: validProfileRecord("Alice"),
		}, nil); err != nil {
			t.Fatal(err)
		}
	}

	outcome, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "2", Kind: "account", DID: "did:plc:a", AccountStatus: "suspended",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != ProjectionOutcomeAccount {
		t.Fatalf("outcome = %s, want account", outcome)
	}
	var status string
	if err := p.db.QueryRow(ctx, "select status from at_projection_records where uri = $1", "at://did:plc:a/tv.subcult.profile/self").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "unavailable" {
		t.Fatalf("status = %q, want unavailable after account suspension", status)
	}

	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "3", Kind: "account", DID: "did:plc:a", AccountStatus: "active",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.db.QueryRow(ctx, "select status from at_projection_records where uri = $1", "at://did:plc:a/tv.subcult.profile/self").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("status = %q, want active after reactivation", status)
	}
}

func TestProjectionAccountDeletionIsTerminalAndSurvivesReactivation(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create",
		DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: "self",
		CID: "bafydel1", Rev: "3jz1", Record: validProfileRecord("Alice"),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "2", Kind: "account", DID: "did:plc:a", AccountStatus: "deleted",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "3", Kind: "account", DID: "did:plc:a", AccountStatus: "active",
	}, nil); err != nil {
		t.Fatal(err)
	}

	var status string
	if err := p.db.QueryRow(ctx, "select status from at_projection_records where uri = $1", "at://did:plc:a/tv.subcult.profile/self").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "deleted" {
		t.Fatalf("status = %q, want deleted to remain terminal across a later reactivation event", status)
	}
}

// --- Crash and replay (no cursor loss, no duplicate/missing rows) ----------

func TestProjectionCrashMidBatchThenReplayFromStoredCursor(t *testing.T) {
	p := newProjectionTestProcessor(t)
	ctx := t.Context()

	events := make([]StreamEvent, 0, 6)
	for i := 1; i <= 6; i++ {
		events = append(events, StreamEvent{
			Cursor: fmt.Sprintf("%d", i), Kind: "commit", Operation: "create",
			DID: "did:plc:a", Collection: "tv.subcult.profile", RKey: fmt.Sprintf("self%d", i),
			CID: fmt.Sprintf("bafy%d", i), Rev: fmt.Sprintf("3jz%02d", i),
			Record: validProfileRecord(fmt.Sprintf("Person %d", i)),
		})
	}
	// Use distinct rkeys (rather than the literal:self record key) purely so
	// each event maps to its own row; this test is about crash/replay
	// mechanics, not the profile record-key convention.

	crashInjected := errors.New("simulated crash before commit")
	var processed int
	for _, event := range events {
		injectCrash := event.Cursor == "4"
		_, err := p.ProcessEvent(ctx, event, func() error {
			if injectCrash {
				return crashInjected
			}
			return nil
		})
		if injectCrash {
			if !errors.Is(err, crashInjected) {
				t.Fatalf("expected simulated crash error, got %v", err)
			}
			break // the "batch" stops here, as if the process died
		}
		if err != nil {
			t.Fatal(err)
		}
		processed++
	}
	if processed != 3 {
		t.Fatalf("processed = %d, want 3 events committed before the simulated crash", processed)
	}

	cursorAfterCrash, err := p.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cursorAfterCrash != "3" {
		t.Fatalf("cursor after crash = %q, want %q (event 4's transaction rolled back entirely)", cursorAfterCrash, "3")
	}
	if got := countRows(t, p, "at_projection_records"); got != 3 {
		t.Fatalf("at_projection_records rows after crash = %d, want 3 (event 4's row must not exist)", got)
	}

	// "Restart": a fresh processor over the same database resumes from the
	// stored cursor and replays every event whose cursor is > stored,
	// exactly like a real reconnect would.
	restarted := NewProjectionProcessor(p.db, p.catalog)
	resumeCursor, err := restarted.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumeCursor != cursorAfterCrash {
		t.Fatal("restarted processor must resume from exactly the stored cursor")
	}
	for _, event := range events {
		if event.Cursor <= resumeCursor {
			continue // already durably committed before the crash
		}
		if _, err := restarted.ProcessEvent(ctx, event, nil); err != nil {
			t.Fatal(err)
		}
	}

	finalCursor, err := restarted.LoadCursor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if finalCursor != "6" {
		t.Fatalf("final cursor = %q, want %q", finalCursor, "6")
	}
	if got := countRows(t, p, "at_projection_records"); got != 6 {
		t.Fatalf("at_projection_records rows after replay = %d, want 6 (no duplicate, no missing row)", got)
	}
}
