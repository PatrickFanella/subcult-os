// Backfill, rebuild and reconciliation for the AT record projection
// (AT-01 recovery). See docs/development/projection.md.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// backfillSourceName is the at_projection_cursor source row backfill writes
// its own audit cursor under. It is intentionally distinct from
// ProjectionSourceName ("jetstream") so a backfill run can never advance or
// overwrite the live stream's resume cursor.
const backfillSourceName = "backfill"

// backfillMaxRecordsPerAuthority bounds the total number of records a single
// backfill invocation admits for one authority across all three collections
// combined, mirroring the stream path's bias toward small bounded units of
// work.
const backfillMaxRecordsPerAuthority = 5000

// backfillMaxPagesPerCollection bounds pagination so a misbehaving or
// adversarial PDS cannot make a single backfill invocation loop forever.
const backfillMaxPagesPerCollection = 100

// backfillMaxListAttempts bounds retries against a single listRecords page
// before a backfill run gives up and is recorded as failed.
const backfillMaxListAttempts = 3

// ErrProjectionAuthorityNotApproved is returned when a backfill, rebuild or
// reconcile call names a DID that is not a currently-approved authority
// (never approved, or approved then revoked).
var ErrProjectionAuthorityNotApproved = errors.New("AT projection authority is not approved")

// ProjectionAuthority is one approved backfill source.
type ProjectionAuthority struct {
	DID                string     `json:"did"`
	ApprovedByPersonID string     `json:"approved_by_person_id"`
	ApprovedAt         time.Time  `json:"approved_at"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	Note               string     `json:"note,omitempty"`
}

// ApproveProjectionAuthority records that approvedByPersonID has approved
// did as a backfill source, or re-approves a previously revoked one.
func ApproveProjectionAuthority(ctx context.Context, db *pgxpool.Pool, did, approvedByPersonID, note string) error {
	if did == "" || approvedByPersonID == "" {
		return errors.New("approving a projection authority requires a did and an approver person id")
	}
	_, err := db.Exec(ctx, `
		insert into at_projection_authorities (did, approved_by_person_id, approved_at, revoked_at, note)
		values ($1, $2, now(), null, $3)
		on conflict (did) do update set
			approved_by_person_id = excluded.approved_by_person_id,
			approved_at = now(),
			revoked_at = null,
			note = excluded.note
	`, did, approvedByPersonID, note)
	if err != nil {
		return fmt.Errorf("approve projection authority: %w", err)
	}
	return nil
}

// RevokeProjectionAuthority stops did from being an eligible backfill
// source. Records already stored from it are not retroactively removed.
func RevokeProjectionAuthority(ctx context.Context, db *pgxpool.Pool, did string) error {
	_, err := db.Exec(ctx, `update at_projection_authorities set revoked_at = now() where did = $1 and revoked_at is null`, did)
	if err != nil {
		return fmt.Errorf("revoke projection authority: %w", err)
	}
	return nil
}

// ListApprovedProjectionAuthorities returns every DID currently approved
// (approved and not revoked), sorted for deterministic iteration.
func ListApprovedProjectionAuthorities(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
	rows, err := db.Query(ctx, `select did from at_projection_authorities where revoked_at is null order by did`)
	if err != nil {
		return nil, fmt.Errorf("list approved projection authorities: %w", err)
	}
	defer rows.Close()
	var dids []string
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			return nil, err
		}
		dids = append(dids, did)
	}
	return dids, rows.Err()
}

func isApprovedProjectionAuthority(ctx context.Context, db *pgxpool.Pool, did string) (bool, error) {
	var count int
	err := db.QueryRow(ctx, `select count(*) from at_projection_authorities where did = $1 and revoked_at is null`, did).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check projection authority approval: %w", err)
	}
	return count > 0, nil
}

// ProjectionRunOutcome classifies a completed run row.
type ProjectionRunOutcome string

const (
	ProjectionRunRunning   ProjectionRunOutcome = "running"
	ProjectionRunCompleted ProjectionRunOutcome = "completed"
	ProjectionRunFailed    ProjectionRunOutcome = "failed"
	ProjectionRunGap       ProjectionRunOutcome = "gap"
	// ProjectionRunBounded marks a run that finished without error but was
	// cut off by a page/record bound before the source's listing was
	// exhausted, so it must not be treated as an authoritative full pass:
	// any deletion sweep or extra-record conclusion for the affected
	// DID/collection was skipped rather than risking a false deletion.
	ProjectionRunBounded ProjectionRunOutcome = "bounded"
)

// ProjectionRunSummary is the secret-free shape of one at_projection_runs
// row: counts are small integer aggregates (record/URI counts), never
// record bodies.
type ProjectionRunSummary struct {
	ID         string               `json:"id"`
	Kind       string               `json:"kind"`
	Authority  string               `json:"authority,omitempty"`
	StartedAt  time.Time            `json:"started_at"`
	FinishedAt *time.Time           `json:"finished_at,omitempty"`
	Outcome    ProjectionRunOutcome `json:"outcome"`
	Counts     map[string]int       `json:"counts"`
	Error      string               `json:"error,omitempty"`
}

func startProjectionRun(ctx context.Context, db *pgxpool.Pool, kind, authority string) (string, error) {
	var id string
	var authorityArg any
	if authority != "" {
		authorityArg = authority
	}
	err := db.QueryRow(ctx, `
		insert into at_projection_runs (kind, authority, started_at, outcome, counts)
		values ($1, $2, now(), 'running', '{}'::jsonb)
		returning id
	`, kind, authorityArg).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("start projection run: %w", err)
	}
	return id, nil
}

func finishProjectionRun(ctx context.Context, db *pgxpool.Pool, id string, outcome ProjectionRunOutcome, counts map[string]int, runErr error) error {
	countsJSON, err := json.Marshal(counts)
	if err != nil {
		return fmt.Errorf("encode projection run counts: %w", err)
	}
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	_, err = db.Exec(ctx, `
		update at_projection_runs
		set finished_at = now(), outcome = $2, counts = $3, error = $4
		where id = $1
	`, id, string(outcome), countsJSON, errText)
	if err != nil {
		return fmt.Errorf("finish projection run: %w", err)
	}
	return nil
}

// LastProjectionRuns returns the most recent run row for each of the three
// run kinds, in kind order. A kind with no runs yet is omitted.
func LastProjectionRuns(ctx context.Context, db *pgxpool.Pool) ([]ProjectionRunSummary, error) {
	var summaries []ProjectionRunSummary
	for _, kind := range []string{"backfill", "rebuild", "reconcile"} {
		var s ProjectionRunSummary
		var authority *string
		var countsJSON []byte
		var errText string
		err := db.QueryRow(ctx, `
			select id, kind, authority, started_at, finished_at, outcome, counts, error
			from at_projection_runs
			where kind = $1
			order by started_at desc
			limit 1
		`, kind).Scan(&s.ID, &s.Kind, &authority, &s.StartedAt, &s.FinishedAt, &s.Outcome, &countsJSON, &errText)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("load last projection run for %s: %w", kind, err)
		}
		if authority != nil {
			s.Authority = *authority
		}
		s.Error = errText
		if len(countsJSON) > 0 {
			_ = json.Unmarshal(countsJSON, &s.Counts)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// ErrProjectionListOutage marks a listRecords failure that exhausted its
// bounded retries, so a backfill caller can tell "the source is down" apart
// from a programming error.
var ErrProjectionListOutage = errors.New("AT record list source outage")

func listRecordsWithRetry(ctx context.Context, lister atprotocol.RecordLister, did, collection, cursor string, maxAttempts int) (atprotocol.RecordListPage, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		page, err := lister.ListRecords(ctx, did, collection, cursor)
		if err == nil {
			return page, nil
		}
		if errors.Is(err, atprotocol.ErrProjectionCursorGap) {
			return atprotocol.RecordListPage{}, err
		}
		lastErr = err
	}
	return atprotocol.RecordListPage{}, fmt.Errorf("%w: %v", ErrProjectionListOutage, lastErr)
}

// BackfillResult is the secret-free outcome of one RunProjectionBackfill call.
type BackfillResult struct {
	RunID   string               `json:"run_id"`
	Outcome ProjectionRunOutcome `json:"outcome"`
	Stats   ProjectionStats      `json:"stats"`
	Deleted int                  `json:"deleted"`
}

// RunProjectionBackfill lists every admitted collection from did's PDS
// (resolved through the hardened identity directory each call, so a PDS
// host migration is transparent) and feeds every listed record through
// ProjectionProcessor.ProcessEvent, so validation, the collection
// allowlist and quarantine are shared with the live stream path. It then
// marks any locally stored active record for did/collection that the
// authority no longer lists as deleted, preserving provenance. did must be
// a currently-approved authority.
//
// A source that reports ErrProjectionCursorGap for a resumed listing
// records the run as "gap" and immediately retries as a fresh, full
// listing (cursor reset) instead of silently continuing from a resume
// point the source can no longer honor; this always happens as an
// automatic second run in this implementation, since backfill does not
// otherwise persist an inter-run resume cursor.
func RunProjectionBackfill(ctx context.Context, db *pgxpool.Pool, catalog *atprotocol.LexiconCatalog, lister atprotocol.RecordLister, did string) (BackfillResult, error) {
	approved, err := isApprovedProjectionAuthority(ctx, db, did)
	if err != nil {
		return BackfillResult{}, err
	}
	if !approved {
		return BackfillResult{}, fmt.Errorf("%w: %s", ErrProjectionAuthorityNotApproved, did)
	}

	runID, err := startProjectionRun(ctx, db, "backfill", did)
	if err != nil {
		return BackfillResult{}, err
	}

	stats, deleted, gapErr, bounded, err := backfillOnce(ctx, db, catalog, lister, did)
	if gapErr {
		counts := map[string]int{"stored": stats.Stored, "deleted": stats.Deleted, "updated": stats.Stored}
		if fErr := finishProjectionRun(ctx, db, runID, ProjectionRunGap, counts, err); fErr != nil {
			return BackfillResult{}, fErr
		}
		// Retry once as a fresh run: a gap means the prior resume point
		// cannot be honored, not that the authority itself is unavailable.
		retryRunID, rErr := startProjectionRun(ctx, db, "backfill", did)
		if rErr != nil {
			return BackfillResult{}, rErr
		}
		stats, deleted, _, bounded, err = backfillOnce(ctx, db, catalog, lister, did)
		outcome := backfillOutcome(bounded, err)
		counts = map[string]int{"stored": stats.Stored, "deleted": deleted, "quarantined": stats.Quarantined}
		if fErr := finishProjectionRun(ctx, db, retryRunID, outcome, counts, err); fErr != nil {
			return BackfillResult{}, fErr
		}
		return BackfillResult{RunID: retryRunID, Outcome: outcome, Stats: stats, Deleted: deleted}, err
	}

	outcome := backfillOutcome(bounded, err)
	counts := map[string]int{"stored": stats.Stored, "deleted": deleted, "quarantined": stats.Quarantined}
	if fErr := finishProjectionRun(ctx, db, runID, outcome, counts, err); fErr != nil {
		return BackfillResult{}, fErr
	}
	return BackfillResult{RunID: runID, Outcome: outcome, Stats: stats, Deleted: deleted}, err
}

// backfillOutcome classifies a completed backfillOnce call: a hard error
// always wins, then a bounded (page/record-capped, listing not exhausted)
// pass is reported as such rather than "completed" so an operator does not
// mistake a partial pass — one whose deletion sweep was skipped — for an
// authoritative full one.
func backfillOutcome(bounded bool, err error) ProjectionRunOutcome {
	if err != nil {
		return ProjectionRunFailed
	}
	if bounded {
		return ProjectionRunBounded
	}
	return ProjectionRunCompleted
}

// backfillOnce performs exactly one full (cursor-reset) listing pass across
// the three admitted collections. The third return value reports whether
// the listing failed because of a reported cursor gap. The fourth return
// value reports whether any collection's listing was cut off by a bound
// (backfillMaxRecordsPerAuthority or backfillMaxPagesPerCollection) while
// the source still had more to list (a non-empty cursor remained); in that
// case the deletion sweep for the affected collection is skipped, since an
// incomplete listing cannot tell a genuinely deleted record apart from one
// simply not reached yet.
func backfillOnce(ctx context.Context, db *pgxpool.Pool, catalog *atprotocol.LexiconCatalog, lister atprotocol.RecordLister, did string) (ProjectionStats, int, bool, bool, error) {
	processor := NewProjectionProcessorWithSource(db, catalog, backfillSourceName)
	stats := ProjectionStats{}
	deletedTotal := 0
	sequence := 0
	anyBounded := false

	collections := admittedCollectionList()
	sort.Strings(collections)

	for _, collection := range collections {
		seen := map[string]bool{}
		cursor := ""
		totalThisCollection := 0
		for page := 0; page < backfillMaxPagesPerCollection; page++ {
			listed, err := listRecordsWithRetry(ctx, lister, did, collection, cursor, backfillMaxListAttempts)
			if err != nil {
				if errors.Is(err, atprotocol.ErrProjectionCursorGap) {
					return stats, deletedTotal, true, anyBounded, err
				}
				return stats, deletedTotal, false, anyBounded, err
			}
			for _, record := range listed.Records {
				ref, err := atprotocol.ParseRecordRef(record.URI)
				if err != nil || ref.Collection != collection {
					continue
				}
				seen[record.URI] = true
				sequence++
				event := StreamEvent{
					Cursor:     fmt.Sprintf("backfill:%s:%d", did, sequence),
					DID:        did,
					Kind:       "commit",
					Operation:  "create",
					Collection: collection,
					RKey:       ref.RecordKey,
					CID:        record.CID,
					Record:     record.Value,
				}
				outcome, err := processor.ProcessEvent(ctx, event, nil)
				if err != nil {
					return stats, deletedTotal, false, anyBounded, fmt.Errorf("apply backfilled record %s: %w", record.URI, err)
				}
				stats.Record(outcome)
				totalThisCollection++
				if totalThisCollection >= backfillMaxRecordsPerAuthority {
					break
				}
			}
			cursor = listed.Cursor
			if cursor == "" || totalThisCollection >= backfillMaxRecordsPerAuthority {
				break
			}
		}

		// cursor is non-empty here only when the loop above exited because
		// of a bound (record cap or page cap) while the source still had
		// more to list, never on natural exhaustion (which always leaves
		// cursor == ""). Skip the deletion sweep in that case: a record
		// this pass never reached is indistinguishable from one genuinely
		// deleted, and marking it deleted would silently drop a public row
		// on every bounded backfill of a large authority.
		if cursor != "" {
			anyBounded = true
			continue
		}

		deleted, err := markProjectionRecordsDeletedExcept(ctx, db, did, collection, seen)
		if err != nil {
			return stats, deletedTotal, false, anyBounded, err
		}
		deletedTotal += deleted
	}
	return stats, deletedTotal, false, anyBounded, nil
}

// markProjectionRecordsDeletedExcept marks every currently-active projection
// record for (did, collection) whose URI is not in seen as deleted,
// preserving did/collection/rkey provenance. Callers must only invoke this
// after a full (non-bounded, cursor-exhausted) listing pass for that
// collection — backfillOnce enforces this by skipping the call whenever its
// loop exited with a non-empty cursor — so absence from seen means the
// authority's repo no longer has the record.
func markProjectionRecordsDeletedExcept(ctx context.Context, db *pgxpool.Pool, did, collection string, seen map[string]bool) (int, error) {
	uris := make([]string, 0, len(seen))
	for uri := range seen {
		uris = append(uris, uri)
	}
	tag, err := db.Exec(ctx, `
		update at_projection_records
		set status = 'deleted', updated_at = now()
		where did = $1 and collection = $2 and status = 'active' and not (uri = any($3))
	`, did, collection, uris)
	if err != nil {
		return 0, fmt.Errorf("mark missing projection records deleted: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// shadowRecord is one record observed while building a rebuild/reconcile
// shadow state; it is never itself part of the JSON output of this package's
// aggregate reporting functions.
type shadowRecord struct {
	DID        string
	Collection string
	RKey       string
	CID        string
	Value      json.RawMessage
}

// buildProjectionShadowState lists every admitted collection from every
// given authority's PDS and returns the resulting shadow record set,
// together with the set of DIDs whose listing was cut off by
// backfillMaxPagesPerCollection before the source's cursor was exhausted.
// A caller must not conclude that an active record for a bounded DID is
// "extra" (absent from the authority) — the listing simply never reached
// it — mirroring backfillOnce's own deletion-sweep guard.
func buildProjectionShadowState(ctx context.Context, lister atprotocol.RecordLister, authorities []string) (map[string]shadowRecord, map[string]bool, error) {
	shadow := make(map[string]shadowRecord)
	bounded := make(map[string]bool)
	collections := admittedCollectionList()
	sort.Strings(collections)
	for _, did := range authorities {
		for _, collection := range collections {
			cursor := ""
			for page := 0; page < backfillMaxPagesPerCollection; page++ {
				listed, err := listRecordsWithRetry(ctx, lister, did, collection, cursor, backfillMaxListAttempts)
				if err != nil {
					return nil, nil, err
				}
				for _, record := range listed.Records {
					ref, err := atprotocol.ParseRecordRef(record.URI)
					if err != nil || ref.Collection != collection {
						continue
					}
					shadow[record.URI] = shadowRecord{DID: did, Collection: collection, RKey: ref.RecordKey, CID: record.CID, Value: record.Value}
				}
				cursor = listed.Cursor
				if cursor == "" {
					break
				}
			}
			if cursor != "" {
				bounded[did] = true
			}
		}
	}
	return shadow, bounded, nil
}

// ProjectionDiff reports per-record differences between the rebuilt shadow
// state and at_projection_records as counts plus URIs only; record bodies
// are never included.
type ProjectionDiff struct {
	MissingURIs        []string `json:"missing_uris"`
	ExtraURIs          []string `json:"extra_uris"`
	CIDMismatchURIs    []string `json:"cid_mismatch_uris"`
	StatusMismatchURIs []string `json:"status_mismatch_uris"`
}

// Counts summarizes the diff as small integers for run-row storage.
func (d ProjectionDiff) Counts() map[string]int {
	return map[string]int{
		"missing":         len(d.MissingURIs),
		"extra":           len(d.ExtraURIs),
		"cid_mismatch":    len(d.CIDMismatchURIs),
		"status_mismatch": len(d.StatusMismatchURIs),
	}
}

type projectionActualRecord struct {
	DID    string
	CID    string
	Status string
}

func loadActiveProjectionState(ctx context.Context, db *pgxpool.Pool, authorities []string) (map[string]projectionActualRecord, error) {
	rows, err := db.Query(ctx, `select uri, did, cid, status from at_projection_records where did = any($1)`, authorities)
	if err != nil {
		return nil, fmt.Errorf("load projection state for diff: %w", err)
	}
	defer rows.Close()
	actual := make(map[string]projectionActualRecord)
	for rows.Next() {
		var uri, did, cid, status string
		if err := rows.Scan(&uri, &did, &cid, &status); err != nil {
			return nil, err
		}
		actual[uri] = projectionActualRecord{DID: did, CID: cid, Status: status}
	}
	return actual, rows.Err()
}

// computeProjectionDiff compares a rebuilt shadow state against the actual
// at_projection_records rows. bounded is the set of DIDs whose shadow
// listing was cut off before exhaustion (see buildProjectionShadowState):
// an active actual record for a bounded DID that is absent from shadow is
// never reported as "extra", since the listing simply never reached it and
// treating it as extra would let reconcile delete a record that is still
// present at the authority.
func computeProjectionDiff(shadow map[string]shadowRecord, actual map[string]projectionActualRecord, bounded map[string]bool) ProjectionDiff {
	var diff ProjectionDiff
	for uri, sr := range shadow {
		a, ok := actual[uri]
		if !ok {
			diff.MissingURIs = append(diff.MissingURIs, uri)
			continue
		}
		if a.Status != "active" {
			diff.StatusMismatchURIs = append(diff.StatusMismatchURIs, uri)
			continue
		}
		if a.CID != sr.CID {
			diff.CIDMismatchURIs = append(diff.CIDMismatchURIs, uri)
		}
	}
	for uri, a := range actual {
		if a.Status != "active" {
			continue
		}
		if _, ok := shadow[uri]; !ok {
			if bounded[a.DID] {
				continue
			}
			diff.ExtraURIs = append(diff.ExtraURIs, uri)
		}
	}
	sort.Strings(diff.MissingURIs)
	sort.Strings(diff.ExtraURIs)
	sort.Strings(diff.CIDMismatchURIs)
	sort.Strings(diff.StatusMismatchURIs)
	return diff
}

// RunProjectionRebuild rebuilds a shadow state from every approved,
// non-revoked authority's PDS and reports how it differs from
// at_projection_records, without writing anything. Approved authorities
// with no records list as such; a diff against zero shadow authorities
// (none approved yet) is an empty diff, not an error.
func RunProjectionRebuild(ctx context.Context, db *pgxpool.Pool, lister atprotocol.RecordLister) (ProjectionDiff, error) {
	authorities, err := ListApprovedProjectionAuthorities(ctx, db)
	if err != nil {
		return ProjectionDiff{}, err
	}
	runID, err := startProjectionRun(ctx, db, "rebuild", "")
	if err != nil {
		return ProjectionDiff{}, err
	}
	shadow, bounded, err := buildProjectionShadowState(ctx, lister, authorities)
	if err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, nil, err)
		return ProjectionDiff{}, err
	}
	actual, err := loadActiveProjectionState(ctx, db, authorities)
	if err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, nil, err)
		return ProjectionDiff{}, err
	}
	diff := computeProjectionDiff(shadow, actual, bounded)
	outcome := ProjectionRunCompleted
	if len(bounded) > 0 {
		outcome = ProjectionRunBounded
	}
	if err := finishProjectionRun(ctx, db, runID, outcome, diff.Counts(), nil); err != nil {
		return ProjectionDiff{}, err
	}
	return diff, nil
}

// RunProjectionReconcile rebuilds the shadow state exactly like
// RunProjectionRebuild, then applies it: a missing, CID-mismatched or
// status-mismatched record is written through ProcessEvent (so validation
// and the collection allowlist still apply), and an extra active record
// (one the authority no longer lists) is marked deleted. It only ever
// touches records for currently-approved authorities.
func RunProjectionReconcile(ctx context.Context, db *pgxpool.Pool, catalog *atprotocol.LexiconCatalog, lister atprotocol.RecordLister) (ProjectionStats, ProjectionDiff, error) {
	authorities, err := ListApprovedProjectionAuthorities(ctx, db)
	if err != nil {
		return ProjectionStats{}, ProjectionDiff{}, err
	}
	runID, err := startProjectionRun(ctx, db, "reconcile", "")
	if err != nil {
		return ProjectionStats{}, ProjectionDiff{}, err
	}
	shadow, bounded, err := buildProjectionShadowState(ctx, lister, authorities)
	if err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, nil, err)
		return ProjectionStats{}, ProjectionDiff{}, err
	}
	actual, err := loadActiveProjectionState(ctx, db, authorities)
	if err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, nil, err)
		return ProjectionStats{}, ProjectionDiff{}, err
	}
	diff := computeProjectionDiff(shadow, actual, bounded)

	processor := NewProjectionProcessorWithSource(db, catalog, "reconcile")
	stats := ProjectionStats{}
	sequence := 0
	apply := func(uris []string) error {
		for _, uri := range uris {
			sr, ok := shadow[uri]
			if !ok {
				continue
			}
			sequence++
			event := StreamEvent{
				Cursor:     fmt.Sprintf("reconcile:%d", sequence),
				DID:        sr.DID,
				Kind:       "commit",
				Operation:  "create",
				Collection: sr.Collection,
				RKey:       sr.RKey,
				CID:        sr.CID,
				Record:     sr.Value,
			}
			outcome, err := processor.ProcessEvent(ctx, event, nil)
			if err != nil {
				return fmt.Errorf("apply reconciled record %s: %w", uri, err)
			}
			stats.Record(outcome)
		}
		return nil
	}
	if err := apply(diff.MissingURIs); err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, diff.Counts(), err)
		return stats, diff, err
	}
	if err := apply(diff.CIDMismatchURIs); err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, diff.Counts(), err)
		return stats, diff, err
	}
	if err := apply(diff.StatusMismatchURIs); err != nil {
		_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, diff.Counts(), err)
		return stats, diff, err
	}
	if len(diff.ExtraURIs) > 0 {
		if _, err := db.Exec(ctx, `update at_projection_records set status = 'deleted', updated_at = now() where uri = any($1) and status = 'active'`, diff.ExtraURIs); err != nil {
			_ = finishProjectionRun(ctx, db, runID, ProjectionRunFailed, diff.Counts(), err)
			return stats, diff, err
		}
	}

	outcome := ProjectionRunCompleted
	if len(bounded) > 0 {
		outcome = ProjectionRunBounded
	}
	if err := finishProjectionRun(ctx, db, runID, outcome, diff.Counts(), nil); err != nil {
		return stats, diff, err
	}
	return stats, diff, nil
}

// ProjectionMetrics is the aggregate, secret-free status the atproto-project
// command reports: no record bodies, DIDs, handles or other private data
// beyond what the existing ProjectionStatus already exposes (cursor and
// counts).
type ProjectionMetrics struct {
	Cursor            string                 `json:"cursor"`
	RecordCount       int                    `json:"record_count"`
	QuarantinedCount  int64                  `json:"quarantined_count"`
	RecordsByStatus   map[string]int         `json:"records_by_status"`
	StreamLagSeconds  *float64               `json:"stream_lag_seconds,omitempty"`
	LastRuns          []ProjectionRunSummary `json:"last_runs,omitempty"`
	ApprovedAuthority int                    `json:"approved_authority_count"`
}

// RunProjectionMetrics reports stream lag (derived from the jetstream
// time_us cursor), quarantine count, records grouped by status, the last
// run per kind and how many authorities are currently approved. Nothing
// here is derived from a record body, DID-scoped record content, or an
// email address.
func RunProjectionMetrics(ctx context.Context, db *pgxpool.Pool) (ProjectionMetrics, error) {
	status, err := RunProjectionStatus(ctx, db)
	if err != nil {
		return ProjectionMetrics{}, err
	}
	metrics := ProjectionMetrics{
		Cursor:           status.Cursor,
		RecordCount:      status.RecordCount,
		QuarantinedCount: status.Quarantined,
		RecordsByStatus:  map[string]int{},
	}

	rows, err := db.Query(ctx, `select status, count(*) from at_projection_records group by status`)
	if err != nil {
		return ProjectionMetrics{}, fmt.Errorf("count projection records by status: %w", err)
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return ProjectionMetrics{}, err
		}
		metrics.RecordsByStatus[status] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return ProjectionMetrics{}, err
	}

	if lag := projectionStreamLagSeconds(status.Cursor); lag != nil {
		metrics.StreamLagSeconds = lag
	}

	runs, err := LastProjectionRuns(ctx, db)
	if err != nil {
		return ProjectionMetrics{}, err
	}
	metrics.LastRuns = runs

	authorities, err := ListApprovedProjectionAuthorities(ctx, db)
	if err != nil {
		return ProjectionMetrics{}, err
	}
	metrics.ApprovedAuthority = len(authorities)

	return metrics, nil
}

// projectionStreamLagSeconds interprets cursor as a Jetstream time_us
// (microsecond Unix timestamp) decimal string and returns now minus that
// instant, in seconds. Returns nil for an empty or unparseable cursor
// rather than guessing.
func projectionStreamLagSeconds(cursor string) *float64 {
	if cursor == "" {
		return nil
	}
	var timeUS int64
	if _, err := fmt.Sscanf(cursor, "%d", &timeUS); err != nil {
		return nil
	}
	if timeUS <= 0 {
		return nil
	}
	lag := time.Since(time.UnixMicro(timeUS)).Seconds()
	return &lag
}
