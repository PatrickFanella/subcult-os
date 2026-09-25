package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	lifecycleMaxReasonBytes      = 1000
	lifecycleMaxRevisionBytes    = 256
	lifecycleMaxDocumentBytes    = 16 * 1024
	lifecycleMaxDestinationBytes = 200
	lifecycleMaxIdempotencyBytes = 256
	lifecycleMaxProviderRefBytes = 200
)

type lifecycleChangeInput struct {
	WorkspaceID, EventID, Kind, Reason, TargetRevision, ApprovedByPersonID string
	DecisionSnapshot                                                       json.RawMessage
}

type lifecycleActionInput struct {
	ChangeID, ActionKind, Destination, IdempotencyKey string
	Payload                                           json.RawMessage
}

type lifecycleActionClaim struct {
	ID, ChangeID, ActionKind, Destination, IdempotencyKey, LeaseToken string
	Payload                                                           json.RawMessage
	AttemptCount                                                      int
}

func createLifecycleChange(ctx context.Context, db *pgxpool.Pool, in lifecycleChangeInput) (string, error) {
	if db == nil || !validLifecycleChange(in) {
		return "", errors.New("invalid lifecycle change")
	}
	var id string
	err := db.QueryRow(ctx, `
		insert into event_lifecycle_changes (workspace_id,event_id,kind,reason,target_revision,decision_snapshot,approved_by_person_id)
		select $1,$2,$3,$4,$5,$6,$7
		where exists (select 1 from events where id=$2 and workspace_id=$1)
		  and exists (
			  select 1 from workspace_members
			  where workspace_id=$1 and person_id=$7 and role='owner'
			    and removed_at is null and revoked_at is null
			    and (expires_at is null or expires_at > clock_timestamp())
		  )
		returning id
	`, in.WorkspaceID, in.EventID, in.Kind, strings.TrimSpace(in.Reason), strings.TrimSpace(in.TargetRevision), in.DecisionSnapshot, in.ApprovedByPersonID).Scan(&id)
	return id, err
}

func createLifecycleAction(ctx context.Context, db *pgxpool.Pool, in lifecycleActionInput) (string, error) {
	if db == nil || !validLifecycleAction(in) {
		return "", errors.New("invalid lifecycle action")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `select status from event_lifecycle_changes where id=$1 for update`, in.ChangeID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("lifecycle change not found")
		}
		return "", err
	}
	if status != "approved" {
		return "", errors.New("lifecycle change is superseded")
	}
	var id string
	err = tx.QueryRow(ctx, `
		insert into event_lifecycle_actions (change_id,action_kind,destination,idempotency_key,payload)
		values ($1,$2,$3,$4,$5)
		on conflict (idempotency_key) do update set idempotency_key=excluded.idempotency_key
		where event_lifecycle_actions.change_id=excluded.change_id
		  and event_lifecycle_actions.action_kind=excluded.action_kind
		  and event_lifecycle_actions.destination=excluded.destination
		  and event_lifecycle_actions.payload=excluded.payload
		returning id
	`, in.ChangeID, in.ActionKind, strings.TrimSpace(in.Destination), strings.TrimSpace(in.IdempotencyKey), in.Payload).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("lifecycle action idempotency conflict")
	}
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// claimLifecycleAction first seals expired running work as unknown. A caller
// must reconcile unknown work; it is intentionally excluded from due claims.
func claimLifecycleAction(ctx context.Context, db *pgxpool.Pool) (*lifecycleActionClaim, error) {
	if db == nil {
		return nil, errors.New("lifecycle actions require a database")
	}
	// Seal uncertain work before taking a change lock. All later lifecycle
	// transitions take the parent change lock before touching an action row.
	if _, err := db.Exec(ctx, `
		update event_lifecycle_actions
		set status='unknown', finished_at=clock_timestamp(), updated_at=clock_timestamp(),
		    failure_category='lease_expired', lease_token=null, lease_expires_at=null
		where status='running' and lease_expires_at < clock_timestamp()
	`); err != nil {
		return nil, err
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	claim := &lifecycleActionClaim{LeaseToken: uuid.NewString()}
	err = tx.QueryRow(ctx, `
		with due as (select a.id from event_lifecycle_actions a join event_lifecycle_changes c on c.id=a.change_id where c.status='approved' and a.status in ('pending','retryable') and (a.next_attempt_at is null or a.next_attempt_at <= clock_timestamp()) order by a.created_at for update of c skip locked limit 1)
		update event_lifecycle_actions a set status='running', attempt_count=attempt_count+1, lease_token=$1, lease_expires_at=clock_timestamp() + interval '2 minutes', updated_at=clock_timestamp()
		from due
		where a.id=due.id and a.status in ('pending','retryable')
		  and (a.next_attempt_at is null or a.next_attempt_at <= clock_timestamp())
		returning a.id,a.change_id,a.action_kind,a.destination,a.idempotency_key,a.payload,a.attempt_count
	`, claim.LeaseToken).Scan(&claim.ID, &claim.ChangeID, &claim.ActionKind, &claim.Destination, &claim.IdempotencyKey, &claim.Payload, &claim.AttemptCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}
	return claim, tx.Commit(ctx)
}

func finishLifecycleAction(ctx context.Context, db *pgxpool.Pool, actionID, leaseToken, outcome, failureCategory, providerRef string, retryAt *time.Time) error {
	if db == nil || !validLifecycleOutcome(outcome) || !validLifecycleFailureCategory(failureCategory) || len(strings.TrimSpace(providerRef)) > lifecycleMaxProviderRefBytes || strings.TrimSpace(actionID) == "" || strings.TrimSpace(leaseToken) == "" {
		return errors.New("invalid lifecycle action completion")
	}
	if outcome == "retryable" && retryAt == nil {
		return errors.New("retryable lifecycle action needs next attempt")
	}
	if outcome != "retryable" {
		retryAt = nil
	}
	var updated string
	err := db.QueryRow(ctx, `
		update event_lifecycle_actions set status=$3,next_attempt_at=$4,failure_category=nullif($5,''),provider_reference=nullif($6,''),lease_token=null,lease_expires_at=null,updated_at=clock_timestamp(),finished_at=case when $3 in ('succeeded','unknown','failed','superseded') then clock_timestamp() else null end
		where id=$1 and status='running' and lease_token=$2 and lease_expires_at >= clock_timestamp()
		returning id
	`, actionID, leaseToken, outcome, retryAt, strings.TrimSpace(failureCategory), strings.TrimSpace(providerRef)).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("lifecycle action lease is no longer current")
	}
	return err
}

func supersedeLifecycleActions(ctx context.Context, db *pgxpool.Pool, changeID string) error {
	if db == nil || strings.TrimSpace(changeID) == "" {
		return errors.New("invalid lifecycle change")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select 1 from event_lifecycle_changes where id=$1 for update`, changeID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update event_lifecycle_changes set status='superseded',superseded_at=now() where id=$1 and status='approved'`, changeID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update event_lifecycle_actions set status='superseded',finished_at=now(),updated_at=now() where change_id=$1 and status in ('pending','retryable')`, changeID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func validLifecycleChange(in lifecycleChangeInput) bool {
	return strings.TrimSpace(in.WorkspaceID) != "" && strings.TrimSpace(in.EventID) != "" && (in.Kind == "cancellation" || in.Kind == "reschedule") && validLifecycleText(in.Reason, lifecycleMaxReasonBytes) && validLifecycleText(in.TargetRevision, lifecycleMaxRevisionBytes) && strings.TrimSpace(in.ApprovedByPersonID) != "" && validLifecycleDocument(in.DecisionSnapshot)
}
func validLifecycleAction(in lifecycleActionInput) bool {
	return strings.TrimSpace(in.ChangeID) != "" && (in.ActionKind == "public_record" || in.ActionKind == "provider_ticket" || in.ActionKind == "operational_notice" || in.ActionKind == "refund") && validLifecycleText(in.Destination, lifecycleMaxDestinationBytes) && validLifecycleText(in.IdempotencyKey, lifecycleMaxIdempotencyBytes) && validLifecycleDocument(in.Payload)
}
func validLifecycleOutcome(value string) bool {
	return value == "succeeded" || value == "retryable" || value == "unknown" || value == "failed"
}

func validLifecycleFailureCategory(value string) bool {
	switch value {
	case "", "lease_expired", "transport", "provider_rejected", "provider_unavailable", "validation", "permission", "reconciliation_required", "internal":
		return true
	default:
		return false
	}
}

func validLifecycleText(value string, maximum int) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && len(trimmed) <= maximum
}

func validLifecycleDocument(value json.RawMessage) bool {
	if len(value) == 0 || len(value) > lifecycleMaxDocumentBytes || !json.Valid(value) {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(value, &object) == nil && object != nil
}
