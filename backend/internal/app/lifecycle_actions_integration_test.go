package app

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestLifecycleActionLedgerIsIdempotentFencedAndNeverBlindRetriesUnknownWork(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Lifecycle ledger", 1)
	eventID := mustString(t, event, "id")
	changeID, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "cancellation", Reason: "weather", TargetRevision: "rev-1", DecisionSnapshot: json.RawMessage(`{"scope":"event"}`), ApprovedByPersonID: ownerPersonID(t, fx)})
	if err != nil {
		t.Fatal(err)
	}
	in := lifecycleActionInput{ChangeID: changeID, ActionKind: "operational_notice", Destination: "outbox", IdempotencyKey: "lifecycle-test-" + fx.suffix, Payload: json.RawMessage(`{"template":"cancel"}`)}
	actionID, err := createLifecycleAction(t.Context(), fx.app.db, in)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := createLifecycleAction(t.Context(), fx.app.db, in); err != nil || again != actionID {
		t.Fatalf("idempotent create id=%q err=%v", again, err)
	}
	in.Payload = json.RawMessage(`{"template":"changed"}`)
	if _, err := createLifecycleAction(t.Context(), fx.app.db, in); err == nil {
		t.Fatal("changed payload reused idempotency key")
	}
	claim, err := claimLifecycleAction(t.Context(), fx.app.db)
	if err != nil || claim == nil || claim.ID != actionID {
		t.Fatalf("claim=%+v err=%v", claim, err)
	}
	if err := finishLifecycleAction(t.Context(), fx.app.db, claim.ID, claim.LeaseToken, "retryable", "transport", "", nil); err == nil {
		t.Fatal("retryable completion without schedule accepted")
	}
	retryAt := time.Now().UTC().Add(time.Minute)
	if err := finishLifecycleAction(t.Context(), fx.app.db, claim.ID, claim.LeaseToken, "retryable", "transport", "", &retryAt); err != nil {
		t.Fatal(err)
	}
	if next, err := claimLifecycleAction(t.Context(), fx.app.db); err != nil || next != nil {
		t.Fatalf("scheduled retry claimed early: %+v %v", next, err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update event_lifecycle_actions set status='running', lease_expires_at=now()-interval '1 second' where id=$1`, actionID); err != nil {
		t.Fatal(err)
	}
	if next, err := claimLifecycleAction(t.Context(), fx.app.db); err != nil || next != nil {
		t.Fatalf("expired unknown action was blindly reclaimed: %+v %v", next, err)
	}
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select status from event_lifecycle_actions where id=$1`, actionID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "unknown" {
		t.Fatalf("expired lease status=%q", status)
	}
}

func TestLifecycleChangeRejectsCrossWorkspaceEventAndSupersededActionIntent(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Bound change", 1), "id")
	other := newLifecycleFixture(t, fx.app)
	input := lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "cancellation", Reason: "weather", TargetRevision: "rev-1", DecisionSnapshot: json.RawMessage(`{"scope":"event"}`), ApprovedByPersonID: ownerPersonID(t, fx)}
	if _, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: other.workspaceID, EventID: eventID, Kind: input.Kind, Reason: input.Reason, TargetRevision: input.TargetRevision, DecisionSnapshot: input.DecisionSnapshot, ApprovedByPersonID: ownerPersonID(t, other)}); err == nil {
		t.Fatal("cross-workspace event change was accepted")
	}
	if _, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: input.Kind, Reason: input.Reason, TargetRevision: input.TargetRevision, DecisionSnapshot: input.DecisionSnapshot, ApprovedByPersonID: ownerPersonID(t, other)}); err == nil {
		t.Fatal("foreign workspace approver was accepted")
	}
	changeID, err := createLifecycleChange(t.Context(), fx.app.db, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := supersedeLifecycleActions(t.Context(), fx.app.db, changeID); err != nil {
		t.Fatal(err)
	}
	_, err = createLifecycleAction(t.Context(), fx.app.db, lifecycleActionInput{ChangeID: changeID, ActionKind: "operational_notice", Destination: "outbox", IdempotencyKey: "superseded-" + fx.suffix, Payload: json.RawMessage(`{"template":"cancel"}`)})
	if err == nil || err.Error() != "lifecycle change is superseded" {
		t.Fatalf("superseded change action err=%v", err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=clock_timestamp() where workspace_id=$1 and person_id=$2`, fx.workspaceID, input.ApprovedByPersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "reschedule", Reason: "new time", TargetRevision: "rev-2", DecisionSnapshot: json.RawMessage(`{"scope":"event"}`), ApprovedByPersonID: input.ApprovedByPersonID}); err == nil {
		t.Fatal("revoked approver was accepted")
	}
}

func TestLifecycleActionConcurrentClaimHasOneWinnerAndSupersedeStopsPending(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Concurrent ledger", 1), "id")
	changeID, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "reschedule", Reason: "weather", TargetRevision: "rev-2", DecisionSnapshot: json.RawMessage(`{"scope":"event"}`), ApprovedByPersonID: ownerPersonID(t, fx)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createLifecycleAction(t.Context(), fx.app.db, lifecycleActionInput{ChangeID: changeID, ActionKind: "operational_notice", Destination: "outbox", IdempotencyKey: "concurrent-" + fx.suffix, Payload: json.RawMessage(`{"template":"reschedule"}`)}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	claims := make(chan *lifecycleActionClaim, 2)
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			claim, claimErr := claimLifecycleAction(t.Context(), fx.app.db)
			claims <- claim
			errs <- claimErr
		}()
	}
	close(start)
	wait.Wait()
	close(claims)
	close(errs)
	winners := 0
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for claim := range claims {
		if claim != nil {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent claim winners=%d, want 1", winners)
	}

	pendingChange, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "cancellation", Reason: "venue closed", TargetRevision: "rev-3", DecisionSnapshot: json.RawMessage(`{"scope":"event"}`), ApprovedByPersonID: ownerPersonID(t, fx)})
	if err != nil {
		t.Fatal(err)
	}
	pendingID, err := createLifecycleAction(t.Context(), fx.app.db, lifecycleActionInput{ChangeID: pendingChange, ActionKind: "operational_notice", Destination: "outbox", IdempotencyKey: "pending-" + fx.suffix, Payload: json.RawMessage(`{"template":"cancel"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err := supersedeLifecycleActions(t.Context(), fx.app.db, pendingChange); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select status from event_lifecycle_actions where id=$1`, pendingID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "superseded" {
		t.Fatalf("pending action status=%q, want superseded", status)
	}
}

func TestLifecycleActionCreateThenSupersedeSerializesOnChange(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Serialized ledger", 1), "id")
	changeID, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "cancellation", Reason: "weather", TargetRevision: "rev-1", DecisionSnapshot: json.RawMessage(`{"scope":"event"}`), ApprovedByPersonID: ownerPersonID(t, fx)})
	if err != nil {
		t.Fatal(err)
	}
	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(t.Context())
	if _, err := blocker.Exec(t.Context(), `select id from event_lifecycle_changes where id=$1 for update`, changeID); err != nil {
		t.Fatal(err)
	}
	created := make(chan error, 1)
	go func() {
		_, createErr := createLifecycleAction(t.Context(), fx.app.db, lifecycleActionInput{ChangeID: changeID, ActionKind: "operational_notice", Destination: "outbox", IdempotencyKey: "serialized-" + fx.suffix, Payload: json.RawMessage(`{"template":"cancel"}`)})
		created <- createErr
	}()
	waitForLifecycleChangeLock(t, fx)
	superseded := make(chan error, 1)
	go func() { superseded <- supersedeLifecycleActions(t.Context(), fx.app.db, changeID) }()
	if err := blocker.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	var createErr error
	select {
	case createErr = <-created:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for blocked lifecycle action creation")
	}
	select {
	case err := <-superseded:
		if err != nil {
			t.Fatalf("supersede actions: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for supersession")
	}
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_lifecycle_actions where change_id=$1`, changeID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		if createErr == nil || createErr.Error() != "lifecycle change is superseded" {
			t.Fatalf("missing action with create err=%v", createErr)
		}
		return
	}
	if createErr != nil || count != 1 {
		t.Fatalf("serialized action count=%d create err=%v", count, createErr)
	}
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select status from event_lifecycle_actions where change_id=$1`, changeID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "superseded" {
		t.Fatalf("serialized action status=%q, want superseded", status)
	}
}

func waitForLifecycleChangeLock(t *testing.T, fx lifecycleFixture) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		if err := fx.app.db.QueryRow(t.Context(), `
			select count(*) from pg_stat_activity
			where datname=current_database() and wait_event_type='Lock'
			  and query like '%event_lifecycle_changes%for update%'
		`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for lifecycle action creation to lock the change")
}

func TestLifecycleActionMigrationUpgradesVersionFifteen(t *testing.T) {
	pool := newMigrationTestPool(t)
	ctx := t.Context()
	migrations, err := loadMigrations(migrationFS)
	if err != nil {
		t.Fatal(err)
	}
	if err := runMigrations(ctx, pool, migrations[:15]); err != nil {
		t.Fatal(err)
	}
	if version, err := CurrentSchemaVersion(ctx, pool); err != nil || version != 15 {
		t.Fatalf("pre-upgrade version=%d err=%v", version, err)
	}
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var actionTable, changeTable bool
	if err := pool.QueryRow(ctx, `select to_regclass('event_lifecycle_actions') is not null, to_regclass('event_lifecycle_changes') is not null`).Scan(&actionTable, &changeTable); err != nil {
		t.Fatal(err)
	}
	if !actionTable || !changeTable {
		t.Fatalf("ledger tables missing after v16 upgrade: actions=%v changes=%v", actionTable, changeTable)
	}
}
