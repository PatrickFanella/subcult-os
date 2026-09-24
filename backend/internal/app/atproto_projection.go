// Package app: allowlisted, restart-safe AT Protocol record projection
// (AT-01). See docs/development/projection.md for the adapter choice and
// operating model.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectionSourceName identifies the single supported stream source in the
// at_projection_cursor table. A future second source would use a distinct
// name; the schema already supports one row per source.
const ProjectionSourceName = "jetstream"

// projectionMaxRecordBytes bounds the JSON record payload admitted into
// at_projection_records. Anything larger is quarantined instead of stored.
const projectionMaxRecordBytes = 64 * 1024

// projectionQuarantinePayloadBytes bounds how much of a rejected event body
// is retained for inspection, so an adversarial or oversize record cannot
// grow the quarantine table without bound either.
const projectionQuarantinePayloadBytes = 4 * 1024

// admittedProjectionCollections is exactly the set of Lexicon NSIDs
// currently admitted per ADR 0007. An event for any other collection
// advances the cursor without being stored, quarantined, or otherwise
// inspected.
var admittedProjectionCollections = map[string]bool{
	"tv.subcult.profile":          true,
	"tv.subcult.place":            true,
	"tv.subcult.event.occurrence": true,
}

// ErrProjectionStreamEnded is returned by an in-memory StreamSource once its
// fixed event list is exhausted, distinguishing a deliberate end of test
// input from a transient connection error worth retrying.
var ErrProjectionStreamEnded = errors.New("projection stream source ended")

// StreamEvent is the OS-owned shape every stream adapter normalizes into,
// modeled on Jetstream's JSON commit-event envelope (see projection.md).
// Cursor is the opaque, monotonically increasing position to resume from;
// for the live Jetstream adapter this is a "time_us" microsecond timestamp
// serialized as a decimal string.
type StreamEvent struct {
	Cursor string
	DID    string
	Kind   string // "commit" or "account"

	// Commit fields (Kind == "commit").
	Operation  string // "create", "update", or "delete"
	Collection string
	RKey       string
	CID        string
	Rev        string
	Record     json.RawMessage

	// Account fields (Kind == "account").
	AccountStatus string // "active", "deactivated", "takendown", "suspended", or "deleted"
}

// StreamSource is the small interface every stream adapter implements.
// Production code uses the live Jetstream websocket adapter; tests use an
// in-memory implementation and never touch the network.
type StreamSource interface {
	Next(ctx context.Context) (StreamEvent, error)
}

// MemoryStreamSource is a fixed, ordered, in-memory StreamSource used by
// tests. It returns ErrProjectionStreamEnded once exhausted.
type MemoryStreamSource struct {
	events []StreamEvent
	pos    int
}

// NewMemoryStreamSource builds a MemoryStreamSource over a fixed event list.
func NewMemoryStreamSource(events []StreamEvent) *MemoryStreamSource {
	return &MemoryStreamSource{events: events}
}

// Next returns the next queued event, or ErrProjectionStreamEnded.
func (m *MemoryStreamSource) Next(ctx context.Context) (StreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return StreamEvent{}, err
	}
	if m.pos >= len(m.events) {
		return StreamEvent{}, ErrProjectionStreamEnded
	}
	event := m.events[m.pos]
	m.pos++
	return event, nil
}

// ProjectionOutcome classifies what a single ProcessEvent call did, for
// tests and for aggregate status reporting.
type ProjectionOutcome string

const (
	ProjectionOutcomeStored      ProjectionOutcome = "stored"
	ProjectionOutcomeDeleted     ProjectionOutcome = "deleted"
	ProjectionOutcomeAccount     ProjectionOutcome = "account"
	ProjectionOutcomeSkippedKind ProjectionOutcome = "skipped_collection"
	ProjectionOutcomeDuplicate   ProjectionOutcome = "duplicate"
	ProjectionOutcomeOutOfOrder  ProjectionOutcome = "out_of_order"
	ProjectionOutcomeQuarantined ProjectionOutcome = "quarantined"
)

// ProjectionProcessor applies admitted-collection filtering, Lexicon
// validation, and the public-projection allowlist to each StreamEvent, and
// commits the resulting record/cursor/quarantine write atomically.
type ProjectionProcessor struct {
	db      *pgxpool.Pool
	catalog *atprotocol.LexiconCatalog
	source  string
}

// NewProjectionProcessor builds a processor. catalog may be nil only in
// tests that exercise collection filtering without validation; production
// callers must supply the admitted Lexicon catalog.
func NewProjectionProcessor(db *pgxpool.Pool, catalog *atprotocol.LexiconCatalog) *ProjectionProcessor {
	return &ProjectionProcessor{db: db, catalog: catalog, source: ProjectionSourceName}
}

// LoadCursor returns the stored cursor for this processor's source, or ""
// if none has been committed yet (fresh start).
func (p *ProjectionProcessor) LoadCursor(ctx context.Context) (string, error) {
	var cursor string
	err := p.db.QueryRow(ctx, `select cursor from at_projection_cursor where source_name = $1`, p.source).Scan(&cursor)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load projection cursor: %w", err)
	}
	return cursor, nil
}

// ProcessEvent applies one StreamEvent inside a single database transaction
// that also advances the stored cursor, so a crash can never leave the
// cursor ahead of what was actually committed. onBeforeCommit, when set, is
// invoked after the row write but before the transaction commits; tests use
// it to simulate a crash mid-batch.
func (p *ProjectionProcessor) ProcessEvent(ctx context.Context, event StreamEvent, onBeforeCommit func() error) (ProjectionOutcome, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin projection transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	outcome, err := p.applyEvent(ctx, tx, event)
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `
		insert into at_projection_cursor (source_name, cursor, updated_at)
		values ($1, $2, now())
		on conflict (source_name) do update set cursor = excluded.cursor, updated_at = now()
	`, p.source, event.Cursor); err != nil {
		return "", fmt.Errorf("advance projection cursor: %w", err)
	}

	if onBeforeCommit != nil {
		if err := onBeforeCommit(); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit projection transaction: %w", err)
	}
	return outcome, nil
}

func (p *ProjectionProcessor) applyEvent(ctx context.Context, tx pgx.Tx, event StreamEvent) (ProjectionOutcome, error) {
	switch event.Kind {
	case "account":
		if err := p.applyAccountEvent(ctx, tx, event); err != nil {
			return "", err
		}
		return ProjectionOutcomeAccount, nil
	case "commit":
		return p.applyCommitEvent(ctx, tx, event)
	default:
		return "", fmt.Errorf("unknown stream event kind %q", event.Kind)
	}
}

func (p *ProjectionProcessor) applyAccountEvent(ctx context.Context, tx pgx.Tx, event StreamEvent) error {
	var targetStatus string
	switch event.AccountStatus {
	case "active":
		// Reactivation only restores rows this processor itself marked
		// unavailable; a record explicitly deleted by a delete commit, or
		// by a prior permanent account deletion, stays deleted.
		_, err := tx.Exec(ctx, `update at_projection_records set status = 'active', updated_at = now(), source_cursor = $2 where did = $1 and status = 'unavailable'`, event.DID, event.Cursor)
		return err
	case "deactivated", "takendown", "suspended":
		targetStatus = "unavailable"
	case "deleted":
		targetStatus = "deleted"
	default:
		return fmt.Errorf("unknown account status %q", event.AccountStatus)
	}
	_, err := tx.Exec(ctx, `update at_projection_records set status = $3, updated_at = now(), source_cursor = $2 where did = $1 and status <> $3`, event.DID, event.Cursor, targetStatus)
	return err
}

func (p *ProjectionProcessor) applyCommitEvent(ctx context.Context, tx pgx.Tx, event StreamEvent) (ProjectionOutcome, error) {
	if !admittedProjectionCollections[event.Collection] {
		return ProjectionOutcomeSkippedKind, nil
	}
	uri := fmt.Sprintf("at://%s/%s/%s", event.DID, event.Collection, event.RKey)

	var existingCID, existingRev, existingStatus string
	err := tx.QueryRow(ctx, `select cid, rev, status from at_projection_records where uri = $1 for update`, uri).Scan(&existingCID, &existingRev, &existingStatus)
	found := true
	if errors.Is(err, pgx.ErrNoRows) {
		found = false
	} else if err != nil {
		return "", fmt.Errorf("read existing projection record: %w", err)
	}

	if found && event.Operation != "delete" && event.Rev != "" && existingRev != "" && event.Rev < existingRev {
		return ProjectionOutcomeOutOfOrder, nil
	}
	if found && event.Operation != "delete" && event.CID != "" && event.CID == existingCID {
		return ProjectionOutcomeDuplicate, nil
	}

	switch event.Operation {
	case "delete":
		if _, err := tx.Exec(ctx, `
			insert into at_projection_records (uri, did, collection, rkey, cid, rev, record, size_bytes, status, source_cursor)
			values ($1, $2, $3, $4, '', $5, null, 0, 'deleted', $6)
			on conflict (uri) do update set status = 'deleted', updated_at = now(), source_cursor = excluded.source_cursor
		`, uri, event.DID, event.Collection, event.RKey, event.Rev, event.Cursor); err != nil {
			return "", fmt.Errorf("mark projection record deleted: %w", err)
		}
		return ProjectionOutcomeDeleted, nil

	case "create", "update":
		if len(event.Record) == 0 {
			return p.quarantine(ctx, tx, uri, event, "empty record body")
		}
		if len(event.Record) > projectionMaxRecordBytes {
			return p.quarantine(ctx, tx, uri, event, fmt.Sprintf("record exceeds %d byte bound", projectionMaxRecordBytes))
		}
		var probe map[string]any
		if err := json.Unmarshal(event.Record, &probe); err != nil {
			return p.quarantine(ctx, tx, uri, event, "malformed JSON record")
		}
		if p.catalog != nil {
			if err := atprotocol.ValidateAdmittedRecord(p.catalog, event.Collection, event.Record); err != nil {
				return p.quarantine(ctx, tx, uri, event, "failed admitted Lexicon validation: "+err.Error())
			}
		}
		if _, err := tx.Exec(ctx, `
			insert into at_projection_records (uri, did, collection, rkey, cid, rev, record, size_bytes, status, source_cursor)
			values ($1, $2, $3, $4, $5, $6, $7, $8, 'active', $9)
			on conflict (uri) do update set
				cid = excluded.cid,
				rev = excluded.rev,
				record = excluded.record,
				size_bytes = excluded.size_bytes,
				status = 'active',
				updated_at = now(),
				source_cursor = excluded.source_cursor
		`, uri, event.DID, event.Collection, event.RKey, event.CID, event.Rev, event.Record, len(event.Record), event.Cursor); err != nil {
			return "", fmt.Errorf("upsert projection record: %w", err)
		}
		return ProjectionOutcomeStored, nil

	default:
		return "", fmt.Errorf("unknown commit operation %q", event.Operation)
	}
}

func (p *ProjectionProcessor) quarantine(ctx context.Context, tx pgx.Tx, uri string, event StreamEvent, reason string) (ProjectionOutcome, error) {
	payload := string(event.Record)
	if len(payload) > projectionQuarantinePayloadBytes {
		payload = payload[:projectionQuarantinePayloadBytes] + "...(truncated)"
	}
	if _, err := tx.Exec(ctx, `
		insert into at_projection_quarantine (uri, reason, payload, cursor)
		values ($1, $2, $3, $4)
	`, uri, reason, payload, event.Cursor); err != nil {
		return "", fmt.Errorf("quarantine projection event: %w", err)
	}
	return ProjectionOutcomeQuarantined, nil
}

// ProjectionRunOptions bounds a live Run loop.
type ProjectionRunOptions struct {
	// MaxReconnectDelay bounds the exponential backoff applied between
	// reconnect attempts after a transient source error.
	MaxReconnectDelay time.Duration
	// MaxAttempts bounds how many consecutive transient errors Run tolerates
	// before giving up and returning an error. Zero means unbounded (the
	// live worker command relies on its own supervisor/restart instead).
	MaxAttempts int
}

// projectionBackoffDelay returns a bounded exponential backoff: 1s, 2s, 4s,
// ... capped at max. attempt is zero-based.
func projectionBackoffDelay(attempt int, max time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	delay := time.Second
	for i := 0; i < attempt && delay < max; i++ {
		delay *= 2
	}
	if delay > max {
		delay = max
	}
	return delay
}

// Run consumes a single, already-open source until ctx is done, an
// unrecoverable error occurs, or (in bounded-attempt mode) MaxAttempts
// consecutive transient errors are exceeded. It retries Next on the same
// source, which suits in-memory and test sources; the production loop is
// RunWithConnector, which re-dials at the committed cursor instead.
func (p *ProjectionProcessor) Run(ctx context.Context, source StreamSource, opts ProjectionRunOptions) (ProjectionStats, error) {
	stats := ProjectionStats{}
	maxDelay := opts.MaxReconnectDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return stats, nil
		}
		event, err := source.Next(ctx)
		if err != nil {
			if errors.Is(err, ErrProjectionStreamEnded) {
				return stats, nil
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return stats, nil
			}
			attempt++
			if opts.MaxAttempts > 0 && attempt > opts.MaxAttempts {
				return stats, fmt.Errorf("projection stream exhausted reconnect attempts: %w", err)
			}
			delay := projectionBackoffDelay(attempt-1, maxDelay)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return stats, nil
			case <-timer.C:
			}
			continue
		}
		attempt = 0
		outcome, err := p.ProcessEvent(ctx, event, nil)
		if err != nil {
			return stats, err
		}
		stats.Record(outcome)
	}
}

// StreamConnector opens a StreamSource positioned at cursor ("" means the
// live tail). RunWithConnector calls it once at start and again after every
// transient failure, so a dropped connection is re-dialed at the last
// durably committed cursor instead of being retried on a dead socket.
type StreamConnector func(ctx context.Context, cursor string) (StreamSource, error)

// RunWithConnector is the production loop: it dials through connect at the
// starting cursor, consumes events until the source fails, closes the
// failed source, waits with bounded exponential backoff, and re-dials at
// the last cursor that ProcessEvent committed. It returns when ctx ends,
// when the source reports ErrProjectionStreamEnded, when ProcessEvent
// fails (a database error is not transient), or when MaxAttempts
// consecutive connection failures are exceeded.
func (p *ProjectionProcessor) RunWithConnector(ctx context.Context, connect StreamConnector, startCursor string, opts ProjectionRunOptions) (ProjectionStats, error) {
	stats := ProjectionStats{}
	maxDelay := opts.MaxReconnectDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	cursor := startCursor
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return stats, nil
		}
		source, err := connect(ctx, cursor)
		if err == nil {
			err = p.consume(ctx, source, &stats, &cursor)
			if closer, ok := source.(io.Closer); ok {
				_ = closer.Close()
			}
		}
		if err == nil || errors.Is(err, ErrProjectionStreamEnded) {
			return stats, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return stats, nil
		}
		var permanent *projectionProcessError
		if errors.As(err, &permanent) {
			return stats, permanent.err
		}
		attempt++
		if opts.MaxAttempts > 0 && attempt > opts.MaxAttempts {
			return stats, fmt.Errorf("projection stream exhausted reconnect attempts: %w", err)
		}
		timer := time.NewTimer(projectionBackoffDelay(attempt-1, maxDelay))
		select {
		case <-ctx.Done():
			timer.Stop()
			return stats, nil
		case <-timer.C:
		}
	}
}

// projectionProcessError marks a ProcessEvent failure so RunWithConnector
// stops instead of reconnecting: the stream is fine, the database is not.
type projectionProcessError struct{ err error }

func (e *projectionProcessError) Error() string { return e.err.Error() }
func (e *projectionProcessError) Unwrap() error { return e.err }

// consume drains source until it errors, recording outcomes into stats and
// advancing cursor after every committed event.
func (p *ProjectionProcessor) consume(ctx context.Context, source StreamSource, stats *ProjectionStats, cursor *string) error {
	for {
		event, err := source.Next(ctx)
		if err != nil {
			return err
		}
		outcome, err := p.ProcessEvent(ctx, event, nil)
		if err != nil {
			return &projectionProcessError{err: err}
		}
		stats.Record(outcome)
		*cursor = event.Cursor
	}
}

// ProjectionStats is an aggregate, secret-free count of processing outcomes
// for the atproto-project command's status output.
type ProjectionStats struct {
	Stored      int `json:"stored"`
	Deleted     int `json:"deleted"`
	Account     int `json:"account"`
	Skipped     int `json:"skipped_collection"`
	Duplicate   int `json:"duplicate"`
	OutOfOrder  int `json:"out_of_order"`
	Quarantined int `json:"quarantined"`
}

// Record increments the counter matching outcome.
func (s *ProjectionStats) Record(outcome ProjectionOutcome) {
	switch outcome {
	case ProjectionOutcomeStored:
		s.Stored++
	case ProjectionOutcomeDeleted:
		s.Deleted++
	case ProjectionOutcomeAccount:
		s.Account++
	case ProjectionOutcomeSkippedKind:
		s.Skipped++
	case ProjectionOutcomeDuplicate:
		s.Duplicate++
	case ProjectionOutcomeOutOfOrder:
		s.OutOfOrder++
	case ProjectionOutcomeQuarantined:
		s.Quarantined++
	}
}

// ProjectionStatus is the aggregate, secret-free status the atproto-project
// command reports.
type ProjectionStatus struct {
	Cursor      string `json:"cursor"`
	RecordCount int    `json:"record_count"`
	Quarantined int64  `json:"quarantined_count"`
}

// RunProjectionStatus reports the current cursor and row counts without
// consuming any stream events.
func RunProjectionStatus(ctx context.Context, db *pgxpool.Pool) (ProjectionStatus, error) {
	processor := NewProjectionProcessor(db, nil)
	cursor, err := processor.LoadCursor(ctx)
	if err != nil {
		return ProjectionStatus{}, err
	}
	var status ProjectionStatus
	status.Cursor = cursor
	if err := db.QueryRow(ctx, `select count(*) from at_projection_records`).Scan(&status.RecordCount); err != nil {
		return ProjectionStatus{}, fmt.Errorf("count projection records: %w", err)
	}
	if err := db.QueryRow(ctx, `select count(*) from at_projection_quarantine`).Scan(&status.Quarantined); err != nil {
		return ProjectionStatus{}, fmt.Errorf("count projection quarantine: %w", err)
	}
	return status, nil
}

// RunProjection loads the admitted Lexicon catalog, opens the configured
// live stream source, and runs the processor to completion (until ctx is
// canceled or the source gives up). It never applies migrations.
func RunProjection(ctx context.Context, config Config, db *pgxpool.Pool) (ProjectionStats, error) {
	if db == nil {
		return ProjectionStats{}, errors.New("projection command requires a database")
	}
	if strings.TrimSpace(config.ATProjectionSourceURL) == "" {
		return ProjectionStats{}, errors.New("AT_PROJECTION_SOURCE_URL is required")
	}
	catalog, err := atprotocol.LoadEmbeddedLexiconCatalog()
	if err != nil {
		return ProjectionStats{}, fmt.Errorf("load admitted lexicon catalog: %w", err)
	}
	processor := NewProjectionProcessor(db, catalog)
	cursor, err := processor.LoadCursor(ctx)
	if err != nil {
		return ProjectionStats{}, err
	}
	connect := func(ctx context.Context, cursor string) (StreamSource, error) {
		source, err := NewJetstreamSource(config.ATProjectionSourceURL, cursor, admittedCollectionList())
		if err != nil {
			return nil, fmt.Errorf("connect projection stream source: %w", err)
		}
		return source, nil
	}
	return processor.RunWithConnector(ctx, connect, cursor, ProjectionRunOptions{})
}

func admittedCollectionList() []string {
	collections := make([]string, 0, len(admittedProjectionCollections))
	for collection := range admittedProjectionCollections {
		collections = append(collections, collection)
	}
	return collections
}

var _ io.Closer = (*JetstreamSource)(nil)
