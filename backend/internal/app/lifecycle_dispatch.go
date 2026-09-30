package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Adapters must validate their approved payload and use the claim's stable
// IdempotencyKey for every provider attempt. There are no runtime adapters yet.
// In particular, HTTP worklist drafts are not dispatch-approved.
type lifecycleAdapter struct {
	Kind, Destination string
	MaxAttempts       int
	Validate          func(json.RawMessage) error
	Execute           func(context.Context, lifecycleActionClaim) (lifecycleDispatchOutcome, error)
}

type lifecycleDispatchOutcome struct {
	Status, FailureCategory, ProviderReference string
	RetryAt                                    *time.Time
}

// dispatchLifecycleAction processes at most one action for one adapter. An
// adapter error means acceptance is uncertain, never permission to retry.
func dispatchLifecycleAction(ctx context.Context, db *pgxpool.Pool, adapter lifecycleAdapter) (*lifecycleActionClaim, error) {
	if lifecycleDestination(adapter.Kind) == "" || !validLifecycleText(adapter.Destination, lifecycleMaxDestinationBytes) || adapter.Destination != strings.TrimSpace(adapter.Destination) || adapter.MaxAttempts < 1 || adapter.MaxAttempts > 10 || adapter.Validate == nil || adapter.Execute == nil {
		return nil, errors.New("invalid lifecycle adapter")
	}
	claim, err := claimLifecycleActionForDestination(ctx, db, adapter.Kind, adapter.Destination)
	if err != nil || claim == nil {
		return claim, err
	}
	category := ""
	if claim.AttemptCount > adapter.MaxAttempts || adapter.Validate(claim.Payload) != nil {
		category = "validation"
	}
	if category == "" {
		category, err = lifecycleDispatchPreflight(ctx, db, claim)
	}
	if err != nil {
		// Leave the running lease for expiry/reconciliation when the authority
		// check itself cannot be completed. No external call has been made.
		return claim, err
	}
	outcome := lifecycleDispatchOutcome{Status: "failed", FailureCategory: category}
	if category == "" {
		// Keep the provider deadline below the two-minute lease. A provider
		// adapter must honor context cancellation and never expose raw errors.
		executeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		outcome, err = adapter.Execute(executeCtx, *claim)
		cancel()
		// Preserve a bounded adapter-supplied receipt for reconciliation even
		// when acceptance is uncertain. Never persist the adapter's error text.
		reference := strings.TrimSpace(outcome.ProviderReference)
		if len(reference) > lifecycleMaxProviderRefBytes {
			reference = ""
		}
		if err != nil {
			outcome = lifecycleDispatchOutcome{Status: "unknown", FailureCategory: "transport", ProviderReference: reference}
		} else if !validLifecycleDispatchOutcome(outcome) {
			outcome = lifecycleDispatchOutcome{Status: "unknown", FailureCategory: "internal", ProviderReference: reference}
		} else if outcome.Status == "retryable" && claim.AttemptCount >= adapter.MaxAttempts {
			outcome.Status = "failed"
			outcome.RetryAt = nil
		}
	}
	// Even when the caller cancels after provider acceptance, try to retain
	// the outcome. Fencing still prevents an expired lease from completing.
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return claim, finishLifecycleAction(finishCtx, db, claim.ID, claim.LeaseToken, outcome.Status, outcome.FailureCategory, outcome.ProviderReference, outcome.RetryAt)
}

func validLifecycleDispatchOutcome(out lifecycleDispatchOutcome) bool {
	if !validLifecycleOutcome(out.Status) || !validLifecycleFailureCategory(out.FailureCategory) || len(strings.TrimSpace(out.ProviderReference)) > lifecycleMaxProviderRefBytes {
		return false
	}
	if out.Status == "succeeded" {
		return out.FailureCategory == "" && out.RetryAt == nil
	}
	if out.FailureCategory == "" {
		return false
	}
	if out.Status == "retryable" {
		return out.RetryAt != nil && out.RetryAt.After(time.Now()) && out.ProviderReference == ""
	}
	return out.RetryAt == nil
}

// Recheck persisted lease, decision, current owner and exact occurrence binding
// immediately before invoking an adapter. This is a local authorization check,
// not a remote CID observation or a lock on a provider's state. Future adapters
// must also apply their destination's conditional-write/authority checks.
func lifecycleDispatchPreflight(ctx context.Context, db *pgxpool.Pool, claim *lifecycleActionClaim) (string, error) {
	var revision, status string
	var snapshot json.RawMessage
	var owner bool
	var updated *time.Time
	var cid *string
	err := db.QueryRow(ctx, `
		select c.target_revision,c.status,c.decision_snapshot,
		  exists(select 1 from workspace_members wm
		    where wm.workspace_id=c.workspace_id and wm.person_id=c.approved_by_person_id
		      and wm.role='owner' and wm.removed_at is null and wm.revoked_at is null
		      and (wm.expires_at is null or wm.expires_at > clock_timestamp())),
		  o.updated_at,o.public_cid
		from event_lifecycle_changes c join event_lifecycle_actions a on a.change_id=c.id
		left join event_occurrences o on o.id::text=c.decision_snapshot->>'occurrenceId'
		  and o.event_id=c.event_id and o.workspace_id=c.workspace_id
		where a.id=$1 and a.change_id=$2 and a.status='running' and a.dispatch_approved
		  and a.lease_token=$3 and a.lease_expires_at > clock_timestamp()
	`, claim.ID, claim.ChangeID, claim.LeaseToken).Scan(&revision, &status, &snapshot, &owner, &updated, &cid)
	if err != nil {
		return "", err
	}
	if status != "approved" || !owner {
		return "permission", nil
	}
	var decision struct {
		OccurrenceID      string  `json:"occurrenceId"`
		ExpectedUpdatedAt string  `json:"expectedUpdatedAt"`
		ExpectedPublicCID *string `json:"expectedPublicCid"`
	}
	if json.Unmarshal(snapshot, &decision) != nil || decision.ExpectedPublicCID == nil || decision.OccurrenceID == "" || decision.ExpectedUpdatedAt == "" || decision.ExpectedUpdatedAt != revision {
		return "validation", nil
	}
	expected, err := time.Parse(time.RFC3339Nano, revision)
	if err != nil {
		return "validation", nil
	}
	publicCID := ""
	if cid != nil {
		publicCID = *cid
	}
	if updated == nil || !expected.Equal(*updated) || *decision.ExpectedPublicCID != publicCID {
		return "validation", nil
	}
	return "", nil
}
