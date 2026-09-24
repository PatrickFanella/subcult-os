package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5/pgxpool"
)

// fixtureListerFunc adapts a closure to atprotocol.RecordLister so each test
// can script exactly the pages/errors it needs without a network call.
type fixtureListerFunc func(ctx context.Context, did, collection, cursor string) (atprotocol.RecordListPage, error)

func (f fixtureListerFunc) ListRecords(ctx context.Context, did, collection, cursor string) (atprotocol.RecordListPage, error) {
	return f(ctx, did, collection, cursor)
}

func newBackfillTestDB(t *testing.T) (*pgxpool.Pool, *atprotocol.LexiconCatalog) {
	t.Helper()
	db := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	catalog, err := atprotocol.LoadLexiconCatalog(lexiconContractDirForTest)
	if err != nil {
		t.Fatal(err)
	}
	return db, catalog
}

func createTestPersonForBackfill(t *testing.T, db *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := db.QueryRow(t.Context(), `insert into people (email, password_hash) values ($1, 'x') returning id`, fmt.Sprintf("backfill-%p@example.test", t)).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func profileRecordValue(displayName string) []byte {
	return []byte(fmt.Sprintf(`{"$type":"tv.subcult.profile","displayName":%q,"createdAt":"2026-09-23T00:00:00Z"}`, displayName))
}

// singlePageLister returns exactly one page (given records) for the named
// collection and an empty page for every other collection.
func singlePageLister(did, collection string, records []atprotocol.ListedRecord) fixtureListerFunc {
	return func(ctx context.Context, gotDID, gotCollection, cursor string) (atprotocol.RecordListPage, error) {
		if gotDID != did || gotCollection != collection || cursor != "" {
			return atprotocol.RecordListPage{}, nil
		}
		return atprotocol.RecordListPage{Records: records}, nil
	}
}

func TestRunProjectionBackfillRejectsUnapprovedAuthority(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	lister := singlePageLister("did:plc:unapproved", "tv.subcult.profile", nil)
	_, err := RunProjectionBackfill(t.Context(), db, catalog, lister, "did:plc:unapproved")
	if !errors.Is(err, ErrProjectionAuthorityNotApproved) {
		t.Fatalf("err = %v, want ErrProjectionAuthorityNotApproved", err)
	}
}

func TestRunProjectionBackfillStoresApprovedAuthorityRecords(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:backfillone"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	lister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: "at://" + did + "/tv.subcult.profile/self", CID: "bafy1", Value: profileRecordValue("Alice")},
	})

	result, err := RunProjectionBackfill(t.Context(), db, catalog, lister, did)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != ProjectionRunCompleted {
		t.Fatalf("outcome = %s, want completed", result.Outcome)
	}
	if result.Stats.Stored != 1 {
		t.Fatalf("stored = %d, want 1", result.Stats.Stored)
	}

	var count int
	if err := db.QueryRow(t.Context(), `select count(*) from at_projection_records where did = $1 and status = 'active'`, did).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("active record count = %d, want 1", count)
	}

	runs, err := LastProjectionRuns(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, run := range runs {
		if run.Kind == "backfill" && run.Outcome == ProjectionRunCompleted {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a completed backfill run, got %+v", runs)
	}
}

func TestRunProjectionBackfillMarksMissingRecordsDeletedPreservingProvenance(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:backfillmissing"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	uriA := "at://" + did + "/tv.subcult.profile/a"
	uriB := "at://" + did + "/tv.subcult.profile/b"

	firstLister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uriA, CID: "bafyA", Value: profileRecordValue("A")},
		{URI: uriB, CID: "bafyB", Value: profileRecordValue("B")},
	})
	if _, err := RunProjectionBackfill(t.Context(), db, catalog, firstLister, did); err != nil {
		t.Fatal(err)
	}

	// Second backfill: the authority no longer lists B.
	secondLister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uriA, CID: "bafyA", Value: profileRecordValue("A")},
	})
	result, err := RunProjectionBackfill(t.Context(), db, catalog, secondLister, did)
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", result.Deleted)
	}

	var status, gotDID, gotCollection, gotRKey string
	err = db.QueryRow(t.Context(), `select status, did, collection, rkey from at_projection_records where uri = $1`, uriB).
		Scan(&status, &gotDID, &gotCollection, &gotRKey)
	if err != nil {
		t.Fatal(err)
	}
	if status != "deleted" {
		t.Fatalf("status = %s, want deleted", status)
	}
	if gotDID != did || gotCollection != "tv.subcult.profile" || gotRKey != "b" {
		t.Fatalf("provenance not preserved: did=%s collection=%s rkey=%s", gotDID, gotCollection, gotRKey)
	}

	var activeStatus string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uriA).Scan(&activeStatus); err != nil {
		t.Fatal(err)
	}
	if activeStatus != "active" {
		t.Fatalf("uriA status = %s, want active", activeStatus)
	}
}

// TestRunProjectionBackfillAccountMigrationKeepsProvenance simulates a PDS
// host migration: FetchRecords resolves the authority through the identity
// directory on every call, so the URI (at://did/collection/rkey) and did
// column stay stable across a host change; only the record content or CID
// may legitimately change between two backfill passes.
func TestRunProjectionBackfillAccountMigrationKeepsProvenance(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:migrated"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}
	uri := "at://" + did + "/tv.subcult.profile/self"

	beforeLister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uri, CID: "bafybefore", Value: profileRecordValue("Before Migration")},
	})
	if _, err := RunProjectionBackfill(t.Context(), db, catalog, beforeLister, did); err != nil {
		t.Fatal(err)
	}

	// The authority's PDS host changed; the lister (which resolves per call)
	// now serves an updated record under the same DID/URI.
	afterLister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uri, CID: "bafyafter", Value: profileRecordValue("After Migration")},
	})
	if _, err := RunProjectionBackfill(t.Context(), db, catalog, afterLister, did); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(t.Context(), `select count(*) from at_projection_records where uri = $1`, uri).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want exactly 1 (no duplicate from host migration)", count)
	}
	var cid, did2 string
	if err := db.QueryRow(t.Context(), `select cid, did from at_projection_records where uri = $1`, uri).Scan(&cid, &did2); err != nil {
		t.Fatal(err)
	}
	if cid != "bafyafter" {
		t.Fatalf("cid = %s, want updated bafyafter", cid)
	}
	if did2 != did {
		t.Fatalf("did = %s, want unchanged %s", did2, did)
	}
}

func TestRunProjectionBackfillSourceOutageRecordsFailedRun(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:outage"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	calls := 0
	lister := fixtureListerFunc(func(ctx context.Context, gotDID, gotCollection, cursor string) (atprotocol.RecordListPage, error) {
		if gotCollection != "tv.subcult.profile" {
			return atprotocol.RecordListPage{}, nil
		}
		calls++
		return atprotocol.RecordListPage{}, errors.New("simulated PDS outage")
	})

	result, err := RunProjectionBackfill(t.Context(), db, catalog, lister, did)
	if err == nil {
		t.Fatal("expected outage error")
	}
	if !errors.Is(err, ErrProjectionListOutage) {
		t.Fatalf("err = %v, want ErrProjectionListOutage", err)
	}
	if result.Outcome != ProjectionRunFailed {
		t.Fatalf("outcome = %s, want failed", result.Outcome)
	}
	if calls != backfillMaxListAttempts {
		t.Fatalf("calls = %d, want bounded retries = %d", calls, backfillMaxListAttempts)
	}

	runs, err := LastProjectionRuns(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	sawFailed := false
	for _, run := range runs {
		if run.Kind == "backfill" && run.Outcome == ProjectionRunFailed {
			sawFailed = true
			if run.Error == "" {
				t.Fatal("expected a recorded error message")
			}
		}
	}
	if !sawFailed {
		t.Fatalf("expected a failed backfill run, got %+v", runs)
	}
}

func TestRunProjectionBackfillCursorGapTriggersFreshBackfill(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:gap"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}
	uri := "at://" + did + "/tv.subcult.profile/self"

	firstAttempt := true
	lister := fixtureListerFunc(func(ctx context.Context, gotDID, gotCollection, cursor string) (atprotocol.RecordListPage, error) {
		if gotCollection != "tv.subcult.profile" {
			return atprotocol.RecordListPage{}, nil
		}
		if firstAttempt {
			firstAttempt = false
			return atprotocol.RecordListPage{}, atprotocol.ErrProjectionCursorGap
		}
		return atprotocol.RecordListPage{Records: []atprotocol.ListedRecord{
			{URI: uri, CID: "bafygap", Value: profileRecordValue("Gap Recovery")},
		}}, nil
	})

	result, err := RunProjectionBackfill(t.Context(), db, catalog, lister, did)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != ProjectionRunCompleted {
		t.Fatalf("outcome = %s, want completed after gap recovery", result.Outcome)
	}

	runs, err := allProjectionRunsForTest(t, db, "backfill")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("run count = %d, want 2 (gap then completed)", len(runs))
	}
	if runs[0].Outcome != ProjectionRunGap || runs[1].Outcome != ProjectionRunCompleted {
		t.Fatalf("unexpected run outcomes: %+v", runs)
	}

	var status string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uri).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("status = %s, want active", status)
	}
}

func allProjectionRunsForTest(t *testing.T, db *pgxpool.Pool, kind string) ([]ProjectionRunSummary, error) {
	t.Helper()
	rows, err := db.Query(t.Context(), `select id, kind, authority, started_at, finished_at, outcome, counts, error from at_projection_runs where kind = $1 order by started_at asc`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectionRunSummary
	for rows.Next() {
		var s ProjectionRunSummary
		var authority *string
		var countsJSON []byte
		if err := rows.Scan(&s.ID, &s.Kind, &authority, &s.StartedAt, &s.FinishedAt, &s.Outcome, &countsJSON, &s.Error); err != nil {
			return nil, err
		}
		if authority != nil {
			s.Authority = *authority
		}
		_ = json.Unmarshal(countsJSON, &s.Counts)
		out = append(out, s)
	}
	return out, rows.Err()
}

func TestRunProjectionRebuildReportsDiff(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:rebuild"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	uriMissing := "at://" + did + "/tv.subcult.profile/missing"
	uriMismatch := "at://" + did + "/tv.subcult.profile/mismatch"
	uriExtra := "at://" + did + "/tv.subcult.profile/extra"

	// Seed at_projection_records directly: uriMismatch (stale cid) and
	// uriExtra (authority no longer has it) already stored, active.
	processor := NewProjectionProcessor(db, catalog)
	for _, seed := range []struct {
		uri, cid, name string
	}{
		{uriMismatch, "bafystale", "Stale"},
		{uriExtra, "bafyextra", "Extra"},
	} {
		ref, err := atprotocol.ParseRecordRef(seed.uri)
		if err != nil {
			t.Fatal(err)
		}
		_, err = processor.ProcessEvent(t.Context(), StreamEvent{
			Cursor: "seed:" + seed.uri, DID: did, Kind: "commit", Operation: "create",
			Collection: "tv.subcult.profile", RKey: ref.RecordKey, CID: seed.cid,
			Record: profileRecordValue(seed.name),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	lister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uriMissing, CID: "bafymissing", Value: profileRecordValue("Missing")},
		{URI: uriMismatch, CID: "bafyfresh", Value: profileRecordValue("Fresh")},
	})

	diff, err := RunProjectionRebuild(t.Context(), db, lister)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(diff.MissingURIs, uriMissing) {
		t.Fatalf("missing = %v, want to contain %s", diff.MissingURIs, uriMissing)
	}
	if !containsString(diff.CIDMismatchURIs, uriMismatch) {
		t.Fatalf("cid mismatch = %v, want to contain %s", diff.CIDMismatchURIs, uriMismatch)
	}
	if !containsString(diff.ExtraURIs, uriExtra) {
		t.Fatalf("extra = %v, want to contain %s", diff.ExtraURIs, uriExtra)
	}

	// Rebuild does not write.
	var status string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uriExtra).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("rebuild must not write; uriExtra status = %s", status)
	}
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func TestRunProjectionReconcileAppliesDiffOnlyForApprovedAuthorities(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:reconcile"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	uriMissing := "at://" + did + "/tv.subcult.profile/missing"
	uriExtra := "at://" + did + "/tv.subcult.profile/extra"

	processor := NewProjectionProcessor(db, catalog)
	ref, err := atprotocol.ParseRecordRef(uriExtra)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "seed", DID: did, Kind: "commit", Operation: "create",
		Collection: "tv.subcult.profile", RKey: ref.RecordKey, CID: "bafyextra",
		Record: profileRecordValue("Extra"),
	}, nil); err != nil {
		t.Fatal(err)
	}

	lister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uriMissing, CID: "bafymissing", Value: profileRecordValue("Missing")},
	})

	stats, diff, err := RunProjectionReconcile(t.Context(), db, catalog, lister)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Stored != 1 {
		t.Fatalf("stored = %d, want 1 (the missing record applied)", stats.Stored)
	}
	if !containsString(diff.MissingURIs, uriMissing) || !containsString(diff.ExtraURIs, uriExtra) {
		t.Fatalf("unexpected diff: %+v", diff)
	}

	var missingStatus, extraStatus string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uriMissing).Scan(&missingStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uriExtra).Scan(&extraStatus); err != nil {
		t.Fatal(err)
	}
	if missingStatus != "active" {
		t.Fatalf("missing record status = %s, want active", missingStatus)
	}
	if extraStatus != "deleted" {
		t.Fatalf("extra record status = %s, want deleted", extraStatus)
	}
}

// pagedLister serves records for one collection across multiple pages of
// pageSize each (empty page for every other collection), forever appending
// a non-empty cursor: it never reports exhaustion, simulating an authority
// with more records than any bound this package enforces.
func pagedLister(did, collection string, pageSize int) fixtureListerFunc {
	return func(ctx context.Context, gotDID, gotCollection, cursor string) (atprotocol.RecordListPage, error) {
		if gotDID != did || gotCollection != collection {
			return atprotocol.RecordListPage{}, nil
		}
		start := 0
		if cursor != "" {
			fmt.Sscanf(cursor, "%d", &start)
		}
		records := make([]atprotocol.ListedRecord, 0, pageSize)
		for i := 0; i < pageSize; i++ {
			n := start + i
			uri := fmt.Sprintf("at://%s/%s/r%d", did, collection, n)
			records = append(records, atprotocol.ListedRecord{URI: uri, CID: fmt.Sprintf("bafy%d", n), Value: profileRecordValue(fmt.Sprintf("Record %d", n))})
		}
		return atprotocol.RecordListPage{Records: records, Cursor: fmt.Sprintf("%d", start+pageSize)}, nil
	}
}

// TestRunProjectionBackfillBoundedListingSkipsDeletionSweep proves that when
// an authority's listing is cut off by backfillMaxPagesPerCollection while
// the source still reports a non-empty cursor, backfill does not run its
// deletion sweep (which would otherwise flip every not-yet-listed active
// record to deleted) and instead records the run as bounded.
func TestRunProjectionBackfillBoundedListingSkipsDeletionSweep(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:bounded"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	// Pre-store an active record that the bounded lister below will never
	// reach (it only ever serves freshly-numbered records, never this URI).
	preexistingURI := "at://" + did + "/tv.subcult.profile/preexisting"
	processor := NewProjectionProcessor(db, catalog)
	ref, err := atprotocol.ParseRecordRef(preexistingURI)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "seed", DID: did, Kind: "commit", Operation: "create",
		Collection: "tv.subcult.profile", RKey: ref.RecordKey, CID: "bafypre",
		Record: profileRecordValue("Preexisting"),
	}, nil); err != nil {
		t.Fatal(err)
	}

	// One record per page, forever: the loop always exits via the page
	// bound with a non-empty cursor remaining, never via exhaustion.
	lister := pagedLister(did, "tv.subcult.profile", 1)

	result, err := RunProjectionBackfill(t.Context(), db, catalog, lister, did)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != ProjectionRunBounded {
		t.Fatalf("outcome = %s, want bounded", result.Outcome)
	}
	if result.Deleted != 0 {
		t.Fatalf("deleted = %d, want 0 (deletion sweep must be skipped for a bounded pass)", result.Deleted)
	}

	var status string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, preexistingURI).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("preexisting record status = %s, want active (must survive a bounded backfill)", status)
	}
}

// TestRunProjectionRebuildBoundedListingDoesNotReportExtra proves the same
// truncation guard in buildProjectionShadowState: an active record for a
// DID whose shadow listing was cut off by the page bound must not be
// reported as "extra" (and therefore must not be deleted by reconcile).
func TestRunProjectionRebuildBoundedListingDoesNotReportExtra(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:reboundedrebuild"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	preexistingURI := "at://" + did + "/tv.subcult.profile/preexisting"
	processor := NewProjectionProcessor(db, catalog)
	ref, err := atprotocol.ParseRecordRef(preexistingURI)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "seed", DID: did, Kind: "commit", Operation: "create",
		Collection: "tv.subcult.profile", RKey: ref.RecordKey, CID: "bafypre",
		Record: profileRecordValue("Preexisting"),
	}, nil); err != nil {
		t.Fatal(err)
	}

	lister := pagedLister(did, "tv.subcult.profile", 1)

	diff, err := RunProjectionRebuild(t.Context(), db, lister)
	if err != nil {
		t.Fatal(err)
	}
	if containsString(diff.ExtraURIs, preexistingURI) {
		t.Fatalf("extra = %v, must not include a record the bounded listing never reached", diff.ExtraURIs)
	}

	runs, err := allProjectionRunsForTest(t, db, "rebuild")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Outcome != ProjectionRunBounded {
		t.Fatalf("rebuild runs = %+v, want a single bounded run", runs)
	}

	// Reconcile must not delete it either.
	if _, _, err := RunProjectionReconcile(t.Context(), db, catalog, lister); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, preexistingURI).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("preexisting record status after reconcile = %s, want active", status)
	}
}

// TestRunProjectionReconcileRepairsStatusMismatchBackToActive proves that a
// record deleted locally (for example, by a stream delete commit) but still
// listed with the same CID at the approved authority is repaired back to
// active by reconcile, rather than being silently reported as a duplicate
// and left deleted.
func TestRunProjectionReconcileRepairsStatusMismatchBackToActive(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:statusmismatch"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	uri := "at://" + did + "/tv.subcult.profile/self"
	ref, err := atprotocol.ParseRecordRef(uri)
	if err != nil {
		t.Fatal(err)
	}
	processor := NewProjectionProcessor(db, catalog)
	if _, err := processor.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "1", DID: did, Kind: "commit", Operation: "create",
		Collection: "tv.subcult.profile", RKey: ref.RecordKey, CID: "bafysame",
		Record: profileRecordValue("Still Here"),
	}, nil); err != nil {
		t.Fatal(err)
	}
	// A stream delete commit marks the row deleted without changing its cid.
	if _, err := processor.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "2", DID: did, Kind: "commit", Operation: "delete",
		Collection: "tv.subcult.profile", RKey: ref.RecordKey, Rev: "2",
	}, nil); err != nil {
		t.Fatal(err)
	}

	var preStatus string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uri).Scan(&preStatus); err != nil {
		t.Fatal(err)
	}
	if preStatus != "deleted" {
		t.Fatalf("precondition: status = %s, want deleted", preStatus)
	}

	// The authority still lists the same record with the same CID.
	lister := singlePageLister(did, "tv.subcult.profile", []atprotocol.ListedRecord{
		{URI: uri, CID: "bafysame", Value: profileRecordValue("Still Here")},
	})

	stats, diff, err := RunProjectionReconcile(t.Context(), db, catalog, lister)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(diff.StatusMismatchURIs, uri) {
		t.Fatalf("status mismatch = %v, want to contain %s", diff.StatusMismatchURIs, uri)
	}
	if stats.Duplicate != 0 {
		t.Fatalf("duplicate = %d, want 0 (a status-mismatched record must be repaired, not short-circuited)", stats.Duplicate)
	}

	var postStatus string
	if err := db.QueryRow(t.Context(), `select status from at_projection_records where uri = $1`, uri).Scan(&postStatus); err != nil {
		t.Fatal(err)
	}
	if postStatus != "active" {
		t.Fatalf("status after reconcile = %s, want active", postStatus)
	}

	runs, err := allProjectionRunsForTest(t, db, "reconcile")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Outcome != ProjectionRunCompleted {
		t.Fatalf("reconcile runs = %+v, want a single completed run", runs)
	}
}

func TestRunProjectionMetricsExcludesPrivateData(t *testing.T) {
	db, catalog := newBackfillTestDB(t)
	person := createTestPersonForBackfill(t, db)
	did := "did:plc:metrics"
	if err := ApproveProjectionAuthority(t.Context(), db, did, person, "test"); err != nil {
		t.Fatal(err)
	}

	processor := NewProjectionProcessor(db, catalog)
	if _, err := processor.ProcessEvent(t.Context(), StreamEvent{
		Cursor: "1700000000000000", DID: did, Kind: "commit", Operation: "create",
		Collection: "tv.subcult.profile", RKey: "self", CID: "bafymetrics",
		Record: profileRecordValue("Secret Human Name secret@example.test"),
	}, nil); err != nil {
		t.Fatal(err)
	}

	metrics, err := RunProjectionMetrics(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.StreamLagSeconds == nil {
		t.Fatal("expected a stream lag value for a numeric time_us cursor")
	}
	if metrics.RecordsByStatus["active"] < 1 {
		t.Fatalf("records by status = %+v, want at least one active", metrics.RecordsByStatus)
	}

	encoded, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	if strings.Contains(body, "secret@example.test") {
		t.Fatal("metrics JSON must not contain a record's email-shaped content")
	}
	if strings.Contains(body, "Secret Human Name") {
		t.Fatal("metrics JSON must not contain record body content")
	}
	if strings.Contains(body, `"record"`) || strings.Contains(body, `"value"`) {
		t.Fatal("metrics JSON must not expose a record/value field at all")
	}
}
