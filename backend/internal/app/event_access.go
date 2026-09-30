package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var accessTopics = []string{"entry", "bathrooms", "seating", "sensory", "transit", "contact"}

type eventAccessRevisionDTO struct {
	EvaluatedAt      string  `json:"evaluatedAt"`
	ID               string  `json:"id,omitempty"`
	Topic            string  `json:"topic"`
	Scope            string  `json:"scope"`
	Revision         int     `json:"revision"`
	Value            string  `json:"value"`
	EffectiveValue   string  `json:"effectiveValue"`
	NeedsReview      bool    `json:"needsReview"`
	Details          string  `json:"details"`
	SourceKind       string  `json:"sourceKind"`
	SourceReference  string  `json:"sourceReference"`
	ReviewedAt       *string `json:"reviewedAt,omitempty"`
	ExpiresAt        *string `json:"expiresAt,omitempty"`
	CorrectionReason string  `json:"correctionReason"`
	RecordedAt       string  `json:"recordedAt,omitempty"`
}
type eventAccessWorksheetDTO struct {
	EventID     string                   `json:"eventId"`
	EvaluatedAt string                   `json:"evaluatedAt"`
	Entries     []eventAccessRevisionDTO `json:"entries"`
}
type eventAccessHistoryDTO struct {
	Topic      string                   `json:"topic"`
	Revisions  []eventAccessRevisionDTO `json:"revisions"`
	NextBefore *int                     `json:"nextBefore,omitempty"`
}
type eventAccessRequest struct {
	RequestKey       string  `json:"requestKey"`
	ExpectedRevision *int    `json:"expectedRevision"`
	Value            string  `json:"value"`
	Details          string  `json:"details"`
	SourceKind       string  `json:"sourceKind"`
	SourceReference  string  `json:"sourceReference"`
	ReviewedAt       *string `json:"reviewedAt"`
	ExpiresAt        *string `json:"expiresAt"`
	CorrectionReason string  `json:"correctionReason"`
}

func validAccessValue(topic, value string) bool {
	for _, known := range accessTopics {
		if topic == known && value == "unknown" {
			return true
		}
	}
	switch topic {
	case "entry", "bathrooms":
		return value == "yes" || value == "no"
	case "seating":
		return value == "available" || value == "limited" || value == "not_available"
	case "sensory", "transit", "contact":
		return value == "known"
	}
	return false
}
func accessTextValid(value string, max int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= max && strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) < 0
}
func validateAccessRequest(topic string, req *eventAccessRequest) (*time.Time, *time.Time, error) {
	req.Details = strings.TrimSpace(req.Details)
	req.SourceReference = strings.TrimSpace(req.SourceReference)
	req.CorrectionReason = strings.TrimSpace(req.CorrectionReason)
	key, err := uuid.Parse(req.RequestKey)
	if err != nil || key == uuid.Nil {
		return nil, nil, errors.New("requestKey must be a nonzero UUID")
	}
	req.RequestKey = key.String()
	if req.ExpectedRevision == nil || *req.ExpectedRevision < 0 || *req.ExpectedRevision >= 2147483647 {
		return nil, nil, errors.New("expectedRevision is required and must be a nonnegative revision")
	}
	if !validAccessValue(topic, req.Value) {
		return nil, nil, errors.New("invalid access topic or value")
	}
	if !accessTextValid(req.Details, 1000) || !accessTextValid(req.SourceReference, 500) || !accessTextValid(req.CorrectionReason, 1000) || req.CorrectionReason == "" {
		return nil, nil, errors.New("bounded plain-text source, details and correction reason are required")
	}
	reviewed, err := parseFinanceTime(req.ReviewedAt)
	if err != nil {
		return nil, nil, errors.New("invalid reviewedAt")
	}
	expires, err := parseFinanceTime(req.ExpiresAt)
	if err != nil {
		return nil, nil, errors.New("invalid expiresAt")
	}
	if req.Value == "unknown" {
		if req.SourceKind != "unknown" || req.SourceReference != "" || req.Details != "" || reviewed != nil || expires != nil {
			return nil, nil, errors.New("unknown must not retain an assertion or source dates")
		}
	} else {
		if (req.SourceKind != "organizer_assertion" && req.SourceKind != "event_observation" && req.SourceKind != "external_reference") || req.SourceReference == "" || reviewed == nil {
			return nil, nil, errors.New("an assertion requires source kind, reference and review date")
		}
		if req.Value == "known" && req.Details == "" {
			return nil, nil, errors.New("known text topics require details")
		}
		if expires != nil && !expires.After(*reviewed) {
			return nil, nil, errors.New("expiry must follow review")
		}
	}
	req.ReviewedAt = nullableTimePtr(reviewed)
	req.ExpiresAt = nullableTimePtr(expires)
	return reviewed, expires, nil
}

const accessRevisionColumns = "id,topic,revision,value,details,source_kind,source_reference,reviewed_at,expires_at,correction_reason,recorded_at"

func scanAccessRevision(row pgx.Row, now time.Time) (eventAccessRevisionDTO, error) {
	x := eventAccessRevisionDTO{Scope: "event", EvaluatedAt: now.UTC().Format(time.RFC3339Nano)}
	var reviewed, expires *time.Time
	var recorded time.Time
	err := row.Scan(&x.ID, &x.Topic, &x.Revision, &x.Value, &x.Details, &x.SourceKind, &x.SourceReference, &reviewed, &expires, &x.CorrectionReason, &recorded)
	x.ReviewedAt = nullableTimePtr(reviewed)
	x.ExpiresAt = nullableTimePtr(expires)
	x.RecordedAt = recorded.UTC().Format(time.RFC3339Nano)
	x.EffectiveValue = x.Value
	x.NeedsReview = x.Value == "unknown" || (expires != nil && !expires.After(now))
	if x.NeedsReview {
		x.EffectiveValue = "unknown"
	}
	return x, err
}
func (a *App) accessEventOwner(w http.ResponseWriter, r *http.Request) (eventRow, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return eventRow{}, false
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		lifecycleIntentEventError(w, err)
		return eventRow{}, false
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return eventRow{}, false
	}
	return event, true
}
func (a *App) handleGetEventAccess(w http.ResponseWriter, r *http.Request) {
	event, ok := a.accessEventOwner(w, r)
	if !ok {
		return
	}
	var now time.Time
	if err := a.db.QueryRow(r.Context(), "select clock_timestamp()").Scan(&now); err != nil {
		writeError(w, 500, "could not load access worksheet")
		return
	}
	out := eventAccessWorksheetDTO{EventID: event.ID, EvaluatedAt: now.UTC().Format(time.RFC3339Nano), Entries: []eventAccessRevisionDTO{}}
	for _, topic := range accessTopics {
		x, err := scanAccessRevision(a.db.QueryRow(r.Context(), "select "+accessRevisionColumns+" from event_access_revisions where event_id=$1 and topic=$2 order by revision desc limit 1", event.ID, topic), now)
		if errors.Is(err, pgx.ErrNoRows) {
			x = eventAccessRevisionDTO{EvaluatedAt: now.UTC().Format(time.RFC3339Nano), Topic: topic, Scope: "event", Value: "unknown", EffectiveValue: "unknown", SourceKind: "unknown", NeedsReview: true}
		} else if err != nil {
			writeError(w, 500, "could not load access worksheet")
			return
		}
		out.Entries = append(out.Entries, x)
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}
func (a *App) handleGetEventAccessHistory(w http.ResponseWriter, r *http.Request) {
	event, ok := a.accessEventOwner(w, r)
	if !ok {
		return
	}
	topic := r.PathValue("topic")
	if !validAccessValue(topic, "unknown") {
		writeError(w, 400, "invalid access topic")
		return
	}
	before := int64(2147483648)
	if raw := r.URL.Query().Get("before"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 1 || parsed > before {
			writeError(w, 400, "invalid history cursor")
			return
		}
		before = parsed
	}
	var now time.Time
	if err := a.db.QueryRow(r.Context(), "select clock_timestamp()").Scan(&now); err != nil {
		writeError(w, 500, "could not load access history")
		return
	}
	rows, err := a.db.Query(r.Context(), "select "+accessRevisionColumns+" from event_access_revisions where event_id=$1 and topic=$2 and revision::bigint<$3 order by revision desc limit 21", event.ID, topic, before)
	if err != nil {
		writeError(w, 500, "could not load access history")
		return
	}
	out := eventAccessHistoryDTO{Topic: topic, Revisions: []eventAccessRevisionDTO{}}
	for rows.Next() {
		x, err := scanAccessRevision(rows, now)
		if err != nil {
			rows.Close()
			writeError(w, 500, "could not load access history")
			return
		}
		out.Revisions = append(out.Revisions, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "could not load access history")
		return
	}
	if len(out.Revisions) > 20 {
		out.Revisions = out.Revisions[:20]
		cursor := out.Revisions[19].Revision
		out.NextBefore = &cursor
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}
func (a *App) handleCreateEventAccessRevision(w http.ResponseWriter, r *http.Request) {
	event, ok := a.accessEventOwner(w, r)
	if !ok {
		return
	}
	topic := r.PathValue("topic")
	var req eventAccessRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	reviewed, expires, err := validateAccessRequest(topic, &req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	encoded, _ := json.Marshal(req)
	hash := sha256.Sum256(encoded)
	fp := hex.EncodeToString(hash[:])
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save access revision")
		return
	}
	defer tx.Rollback(r.Context())
	ctx := context.WithValue(r.Context(), txContextKey{}, tx)
	var locked string
	if err = tx.QueryRow(ctx, "select id from events where id=$1 and workspace_id=$2 for update", event.ID, event.WorkspaceID).Scan(&locked); err != nil {
		writeError(w, 409, "event changed")
		return
	}
	actor, err := activeOwnerTx(ctx, tx, event.WorkspaceID, r.Context().Value(operatorPersonKey{}))
	if err != nil {
		writeError(w, 403, "forbidden")
		return
	}
	var now time.Time
	if err = tx.QueryRow(ctx, "select clock_timestamp()").Scan(&now); err != nil {
		writeError(w, 500, "could not save access revision")
		return
	}
	var priorID, priorFP, priorEvent, priorActor, priorTopic string
	err = tx.QueryRow(ctx, "select id,request_fingerprint,event_id,recorded_by_person_id,topic from event_access_revisions where request_key=$1", req.RequestKey).Scan(&priorID, &priorFP, &priorEvent, &priorActor, &priorTopic)
	if err == nil {
		if priorFP != fp || priorEvent != event.ID || priorActor != actor || priorTopic != topic {
			writeError(w, 409, "requestKey already used for a different revision")
			return
		}
		x, err := scanAccessRevision(tx.QueryRow(ctx, "select "+accessRevisionColumns+" from event_access_revisions where id=$1", priorID), now)
		if err != nil {
			writeError(w, 500, "could not load access revision")
			return
		}
		tx.Rollback(ctx)
		if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
			writeError(w, 403, "forbidden")
			return
		}
		writeJSON(w, 200, x)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not save access revision")
		return
	}
	if reviewed != nil && reviewed.After(now) {
		writeError(w, 400, "review date must not be in the future")
		return
	}
	var current int
	if err = tx.QueryRow(ctx, "select coalesce(max(revision),0) from event_access_revisions where event_id=$1 and topic=$2", event.ID, topic).Scan(&current); err != nil {
		writeError(w, 500, "could not save access revision")
		return
	}
	if current != *req.ExpectedRevision {
		writeError(w, 409, "access information changed; reload before correcting")
		return
	}
	x, err := scanAccessRevision(tx.QueryRow(ctx, "insert into event_access_revisions(event_id,topic,revision,value,details,source_kind,source_reference,reviewed_at,expires_at,correction_reason,recorded_by_person_id,request_key,request_fingerprint) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) returning "+accessRevisionColumns, event.ID, topic, current+1, req.Value, req.Details, req.SourceKind, req.SourceReference, reviewed, expires, req.CorrectionReason, actor, req.RequestKey, fp), now)
	if err != nil {
		var constraint *pgconn.PgError
		if errors.As(err, &constraint) && constraint.Code == "23505" {
			writeError(w, 409, "requestKey or revision already used")
			return
		}
		writeError(w, 500, "could not save access revision")
		return
	}
	if err = a.audit(ctx, actor, "event_access.revised", "event_access_revision", x.ID, map[string]any{"eventId": event.ID, "topic": topic, "revision": x.Revision}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(w, 500, "could not save access revision")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 201, x)
}
