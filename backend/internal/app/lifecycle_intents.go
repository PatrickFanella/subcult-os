package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Lifecycle intents are deliberately private, draft-only records. They make
// a proposed coordinated change observable but never dispatch it.
type lifecycleIntentRequest struct {
	OccurrenceID      string   `json:"occurrenceId"`
	ExpectedUpdatedAt string   `json:"expectedUpdatedAt"`
	ExpectedPublicCID string   `json:"expectedPublicCid"`
	Kind              string   `json:"kind"`
	Reason            string   `json:"reason"`
	ActionKinds       []string `json:"actionKinds"`
	DecisionKey       string   `json:"decisionKey"`
}

type lifecycleIntentActionDTO struct {
	ID              string  `json:"id"`
	ActionKind      string  `json:"actionKind"`
	Destination     string  `json:"destination"`
	Status          string  `json:"status"`
	AttemptCount    int     `json:"attemptCount"`
	FailureCategory *string `json:"failureCategory,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type lifecycleIntentDTO struct {
	ID                 string                     `json:"id"`
	WorkspaceID        string                     `json:"workspaceId"`
	EventID            string                     `json:"eventId"`
	OccurrenceID       string                     `json:"occurrenceId"`
	Kind               string                     `json:"kind"`
	Reason             string                     `json:"reason"`
	TargetRevision     string                     `json:"targetRevision"`
	Status             string                     `json:"status"`
	ApprovedByPersonID string                     `json:"approvedByPersonId"`
	ExpectedPublicCID  string                     `json:"expectedPublicCid"`
	CreatedAt          string                     `json:"createdAt"`
	SupersededAt       *string                    `json:"supersededAt,omitempty"`
	Actions            []lifecycleIntentActionDTO `json:"actions"`
}

func lifecycleDestination(kind string) string {
	switch kind {
	case "public_record":
		return "public_record"
	case "provider_ticket":
		return "ticket_provider"
	case "operational_notice":
		return "operational_notice"
	case "refund":
		return "refund_provider"
	default:
		return ""
	}
}

func validIntentActions(kinds []string) bool {
	if len(kinds) == 0 || len(kinds) > 4 {
		return false
	}
	seen := map[string]bool{}
	for _, kind := range kinds {
		if lifecycleDestination(kind) == "" || seen[kind] {
			return false
		}
		seen[kind] = true
	}
	return true
}

func (a *App) handleCreateLifecycleIntent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	var req lifecycleIntentRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "invalid json")
		return
	}
	if !validIntentActions(req.ActionKinds) || !validLifecycleText(req.Reason, lifecycleMaxReasonBytes) || (req.Kind != "cancellation" && req.Kind != "reschedule") {
		writeError(w, 400, "invalid lifecycle intent")
		return
	}
	if _, err := uuid.Parse(req.DecisionKey); err != nil {
		writeError(w, 400, "decisionKey must be a UUID")
		return
	}
	expectedUpdatedAt, err := time.Parse(time.RFC3339Nano, req.ExpectedUpdatedAt)
	if err != nil {
		writeError(w, 400, "invalid expectedUpdatedAt")
		return
	}
	req.ExpectedUpdatedAt = expectedUpdatedAt.UTC().Format(time.RFC3339Nano)
	// Empty CID is an explicit CAS condition for an occurrence that has no
	// recorded public revision. Do not accept an omitted occurrence reference.
	if strings.TrimSpace(req.OccurrenceID) == "" {
		writeError(w, 400, "occurrenceId is required")
		return
	}
	// Reject non-owners before taking lifecycle locks or comparing revisions.
	// The transaction below repeats this check after the relevant locks.
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		lifecycleIntentEventError(w, err)
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not create lifecycle intent")
		return
	}
	defer tx.Rollback(r.Context())
	ctx := context.WithValue(r.Context(), txContextKey{}, tx)
	var eventWorkspaceID string
	if err := tx.QueryRow(ctx, `select workspace_id from events where id=$1 for update`, r.PathValue("eventID")).Scan(&eventWorkspaceID); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "event not found")
		return
	} else if err != nil {
		writeError(w, 500, "could not load event")
		return
	}
	// A replay resolves from the stored immutable decision before checking the
	// occurrence's current revision. A later edit cannot turn a completed
	// request into a false conflict or cause a second decision/action set.
	var existing lifecycleIntentDTO
	var existingSnapshot json.RawMessage
	err = tx.QueryRow(ctx, `select id,event_id,kind,reason,target_revision,decision_snapshot,approved_by_person_id from event_lifecycle_changes where workspace_id=$1 and request_key=$2 for update`, eventWorkspaceID, req.DecisionKey).Scan(&existing.ID, &existing.EventID, &existing.Kind, &existing.Reason, &existing.TargetRevision, &existingSnapshot, &existing.ApprovedByPersonID)
	if err == nil {
		actorID, ownerErr := activeOwnerTx(ctx, tx, eventWorkspaceID, r.Context().Value(operatorPersonKey{}))
		if ownerErr != nil {
			writeError(w, 403, "forbidden")
			return
		}
		snapshot, snapshotErr := lifecycleSnapshotStrings(existingSnapshot)
		if snapshotErr != nil || existing.EventID != r.PathValue("eventID") || existing.Kind != req.Kind || existing.Reason != strings.TrimSpace(req.Reason) || existing.TargetRevision != req.ExpectedUpdatedAt || existing.ApprovedByPersonID != actorID || snapshot["occurrenceId"] != req.OccurrenceID || snapshot["expectedUpdatedAt"] != req.ExpectedUpdatedAt || snapshot["expectedPublicCid"] != req.ExpectedPublicCID || verifyLifecycleIntentActions(ctx, tx, existing.ID, req.ActionKinds) != nil {
			writeError(w, http.StatusConflict, "decisionKey is already bound to a different lifecycle intent")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			writeError(w, 500, "could not record lifecycle intent")
			return
		}
		if _, _, ok := a.requireWorkspaceRole(r, eventWorkspaceID, roleOwner); !ok {
			writeError(w, 403, "forbidden")
			return
		}
		intent, err := a.loadLifecycleIntent(r.Context(), eventWorkspaceID, existing.ID)
		if err != nil {
			writeError(w, 500, "could not load lifecycle intent")
			return
		}
		writeJSON(w, http.StatusOK, intent)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not load lifecycle decision")
		return
	}
	var occurrence eventOccurrenceRow
	occurrence, err = scanOccurrenceRow(tx.QueryRow(r.Context(), `select `+occurrenceSelectColumns+` from event_occurrences where id=$1 for update`, req.OccurrenceID))
	if errors.Is(err, pgx.ErrNoRows) || occurrence.EventID != r.PathValue("eventID") || occurrence.WorkspaceID != eventWorkspaceID {
		writeError(w, 400, "occurrence does not belong to this event")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load occurrence")
		return
	}
	if !occurrence.UpdatedAt.Equal(expectedUpdatedAt) || occurrence.PublicCID.String != req.ExpectedPublicCID {
		writeError(w, http.StatusConflict, "occurrence changed; reload before recording an intent")
		return
	}
	actorID, err := activeOwnerTx(ctx, tx, eventWorkspaceID, r.Context().Value(operatorPersonKey{}))
	if err != nil {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	snapshot, _ := json.Marshal(map[string]string{"occurrenceId": occurrence.ID, "expectedUpdatedAt": occurrence.UpdatedAt.UTC().Format(time.RFC3339Nano), "expectedPublicCid": occurrence.PublicCID.String, "occurrenceStatus": occurrence.Status})
	var changeID string
	err = tx.QueryRow(ctx, `insert into event_lifecycle_changes (workspace_id,event_id,kind,reason,target_revision,decision_snapshot,approved_by_person_id,request_key)
		values ($1,$2,$3,$4,$5,$6,$7,$8) on conflict (workspace_id,request_key) do nothing returning id`, eventWorkspaceID, r.PathValue("eventID"), req.Kind, strings.TrimSpace(req.Reason), occurrence.UpdatedAt.UTC().Format(time.RFC3339Nano), snapshot, actorID, req.DecisionKey).Scan(&changeID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "decisionKey is already bound to a different lifecycle intent")
		return
	}
	if err != nil {
		writeError(w, 500, "could not record lifecycle decision")
		return
	}
	{
		for _, kind := range req.ActionKinds {
			payload, _ := json.Marshal(map[string]string{"eventId": r.PathValue("eventID"), "occurrenceId": occurrence.ID, "changeKind": req.Kind, "targetRevision": occurrence.UpdatedAt.UTC().Format(time.RFC3339Nano)})
			key := "lifecycle-intent:" + changeID + ":" + kind
			if _, err := tx.Exec(ctx, `insert into event_lifecycle_actions (change_id,action_kind,destination,idempotency_key,payload) values ($1,$2,$3,$4,$5)`, changeID, kind, lifecycleDestination(kind), key, payload); err != nil {
				writeError(w, 500, "could not record lifecycle action")
				return
			}
		}
	}
	if a.audit(ctx, actorID, "lifecycle_intent.recorded", "event_lifecycle_change", changeID, map[string]any{"workspaceId": eventWorkspaceID, "eventId": r.PathValue("eventID"), "occurrenceId": occurrence.ID, "kind": req.Kind}) != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not record lifecycle intent")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, eventWorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	intent, err := a.loadLifecycleIntent(r.Context(), eventWorkspaceID, changeID)
	if err != nil {
		writeError(w, 500, "could not load lifecycle intent")
		return
	}
	writeJSON(w, http.StatusCreated, intent)
}

func activeOwnerTx(ctx context.Context, tx pgx.Tx, workspaceID string, person any) (string, error) {
	personID, ok := person.(string)
	if !ok || personID == "" {
		return "", errors.New("no operator")
	}
	var found string
	err := tx.QueryRow(ctx, `select person_id from workspace_members where workspace_id=$1 and person_id=$2 and role='owner' and removed_at is null and revoked_at is null and (expires_at is null or expires_at > clock_timestamp()) for update`, workspaceID, personID).Scan(&found)
	return found, err
}

func verifyLifecycleIntentActions(ctx context.Context, tx pgx.Tx, changeID string, requested []string) error {
	rows, err := tx.Query(ctx, `select action_kind from event_lifecycle_actions where change_id=$1`, changeID)
	if err != nil {
		return err
	}
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			return err
		}
		got[kind] = true
	}
	if err := rows.Err(); err != nil || len(got) != len(requested) {
		return errors.New("action mismatch")
	}
	for _, kind := range requested {
		if !got[kind] {
			return errors.New("action mismatch")
		}
	}
	return nil
}

// Migration 16 permits arbitrary bounded JSON objects. Worklist records use
// four string fields, but older ledger rows can contain booleans or nested
// values; those rows must remain listable instead of becoming a 500.
func lifecycleSnapshotStrings(raw json.RawMessage) (map[string]string, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, key := range []string{"occurrenceId", "expectedUpdatedAt", "expectedPublicCid", "occurrenceStatus"} {
		if value, ok := values[key]; ok {
			var decoded string
			if json.Unmarshal(value, &decoded) == nil {
				out[key] = decoded
			}
		}
	}
	return out, nil
}

func lifecycleIntentEventError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "event not found")
	} else {
		writeError(w, 500, "could not load event")
	}
}

func (a *App) handleListLifecycleIntents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		lifecycleIntentEventError(w, err)
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	rows, err := a.db.Query(r.Context(), `select id from event_lifecycle_changes where workspace_id=$1 and event_id=$2 order by created_at desc`, event.WorkspaceID, event.ID)
	if err != nil {
		writeError(w, 500, "could not load lifecycle intents")
		return
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			writeError(w, 500, "could not load lifecycle intents")
			return
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		writeError(w, 500, "could not load lifecycle intents")
		return
	}
	if err := rows.Close(); err != nil {
		writeError(w, 500, "could not load lifecycle intents")
		return
	}
	out := make([]lifecycleIntentDTO, 0, len(ids))
	for _, id := range ids {
		item, err := a.loadLifecycleIntent(r.Context(), event.WorkspaceID, id)
		if err != nil {
			writeError(w, 500, "could not load lifecycle intents")
			return
		}
		out = append(out, item)
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) handleSupersedeLifecycleIntent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not supersede lifecycle intent")
		return
	}
	defer tx.Rollback(r.Context())
	ctx := context.WithValue(r.Context(), txContextKey{}, tx)
	var workspaceID string
	if err := tx.QueryRow(ctx, `select workspace_id from events where id=$1 for update`, r.PathValue("eventID")).Scan(&workspaceID); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "event not found")
		return
	} else if err != nil {
		writeError(w, 500, "could not load event")
		return
	}
	var status string
	if err := tx.QueryRow(ctx, `select status from event_lifecycle_changes where id=$1 and workspace_id=$2 and event_id=$3 for update`, r.PathValue("changeID"), workspaceID, r.PathValue("eventID")).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "lifecycle intent not found")
		return
	} else if err != nil {
		writeError(w, 500, "could not supersede lifecycle intent")
		return
	}
	actorID, err := activeOwnerTx(ctx, tx, workspaceID, r.Context().Value(operatorPersonKey{}))
	if err != nil {
		writeError(w, 403, "forbidden")
		return
	}
	if status == "approved" {
		if _, err := tx.Exec(ctx, `update event_lifecycle_changes set status='superseded',superseded_at=clock_timestamp() where id=$1`, r.PathValue("changeID")); err != nil {
			writeError(w, 500, "could not supersede lifecycle intent")
			return
		}
		if _, err := tx.Exec(ctx, `update event_lifecycle_actions set status='superseded',finished_at=clock_timestamp(),updated_at=clock_timestamp() where change_id=$1 and status in ('pending','retryable')`, r.PathValue("changeID")); err != nil {
			writeError(w, 500, "could not supersede lifecycle intent")
			return
		}
		if err := a.audit(ctx, actorID, "lifecycle_intent.superseded", "event_lifecycle_change", r.PathValue("changeID"), map[string]any{"workspaceId": workspaceID, "eventId": r.PathValue("eventID")}); err != nil {
			writeError(w, 500, "could not record audit")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, 500, "could not supersede lifecycle intent")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, workspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	intent, err := a.loadLifecycleIntent(r.Context(), workspaceID, r.PathValue("changeID"))
	if err != nil {
		writeError(w, 500, "could not load lifecycle intent")
		return
	}
	writeJSON(w, 200, intent)
}

func (a *App) loadLifecycleIntent(ctx context.Context, workspaceID, changeID string) (lifecycleIntentDTO, error) {
	var out lifecycleIntentDTO
	var snapshot json.RawMessage
	var created time.Time
	var superseded *time.Time
	err := a.db.QueryRow(ctx, `select id,workspace_id,event_id,kind,reason,target_revision,decision_snapshot,approved_by_person_id,status,created_at,superseded_at from event_lifecycle_changes where id=$1 and workspace_id=$2`, changeID, workspaceID).Scan(&out.ID, &out.WorkspaceID, &out.EventID, &out.Kind, &out.Reason, &out.TargetRevision, &snapshot, &out.ApprovedByPersonID, &out.Status, &created, &superseded)
	if err != nil {
		return out, err
	}
	decision, err := lifecycleSnapshotStrings(snapshot)
	if err != nil {
		return out, errors.New("invalid lifecycle snapshot")
	}
	out.OccurrenceID = decision["occurrenceId"]
	out.ExpectedPublicCID = decision["expectedPublicCid"]
	out.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	if superseded != nil {
		value := superseded.UTC().Format(time.RFC3339Nano)
		out.SupersededAt = &value
	}
	rows, err := a.db.Query(ctx, `select id,action_kind,destination,status,attempt_count,failure_category,created_at,updated_at from event_lifecycle_actions where change_id=$1 order by action_kind`, changeID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Actions = []lifecycleIntentActionDTO{}
	for rows.Next() {
		var item lifecycleIntentActionDTO
		var failure *string
		var actionCreated, actionUpdated time.Time
		if err := rows.Scan(&item.ID, &item.ActionKind, &item.Destination, &item.Status, &item.AttemptCount, &failure, &actionCreated, &actionUpdated); err != nil {
			return out, err
		}
		item.FailureCategory = failure
		item.CreatedAt = actionCreated.UTC().Format(time.RFC3339Nano)
		item.UpdatedAt = actionUpdated.UTC().Format(time.RFC3339Nano)
		out.Actions = append(out.Actions, item)
	}
	return out, rows.Err()
}
