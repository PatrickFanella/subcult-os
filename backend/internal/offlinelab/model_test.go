package offlinelab

import (
	"errors"
	"testing"
	"time"
)

var fixtureStart = time.Date(2026, time.September, 25, 18, 0, 0, 0, time.UTC)

func fixtureSnapshot() Snapshot {
	return Snapshot{
		ID: "synthetic-snapshot-1", EventID: "synthetic-event-1", Revision: "synthetic-roster-r1", ExpiresAt: fixtureStart.Add(15 * time.Minute),
		Tickets: map[string]Ticket{
			"SYNTH-ONE": {Code: "SYNTH-ONE", DisplayName: "Fixture One", AdmissionEligible: true},
			"SYNTH-NO":  {Code: "SYNTH-NO", DisplayName: "Fixture No", AdmissionEligible: false},
		},
	}
}

func TestTwoIndependentClientsResolveDuplicateOnlyAtServerMerge(t *testing.T) {
	snapshot := fixtureSnapshot()
	clientA := NewClient(snapshot, "synthetic-client-a")
	clientB := NewClient(snapshot, "synthetic-client-b")
	a, err := clientA.CheckIn("SYNTH-ONE", fixtureStart.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	b, err := clientB.CheckIn("SYNTH-ONE", fixtureStart.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatalf("independent clients reused operation ID %q", a.ID)
	}

	server := NewServer(snapshot, snapshot.Revision)
	if got := server.Merge(b, fixtureStart.Add(2*time.Minute)); got.Status != MergeAccepted {
		t.Fatalf("first reconnect = %#v", got)
	}
	if got := server.Merge(a, fixtureStart.Add(2*time.Minute)); got.Status != MergeDuplicateCheckIn {
		t.Fatalf("second reconnect = %#v", got)
	}
	if got := server.Merge(b, fixtureStart.Add(2*time.Minute)); got.Status != MergeDuplicateOperation || got.DuplicateOf != MergeAccepted {
		t.Fatalf("lost-response retry = %#v", got)
	}
}

func TestExpiredAndRevokedFixtureOperationsNeverCheckIn(t *testing.T) {
	snapshot := fixtureSnapshot()
	client := NewClient(snapshot, "synthetic-client-a")
	if _, err := client.CheckIn("SYNTH-ONE", snapshot.ExpiresAt); !errors.Is(err, ErrSnapshotExpired) {
		t.Fatalf("local expiry = %v", err)
	}
	operation, err := client.CheckIn("SYNTH-ONE", fixtureStart.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got := NewServer(snapshot, snapshot.Revision).Merge(operation, snapshot.ExpiresAt); got.Status != MergeExpiredSnapshot {
		t.Fatalf("expired reconnect = %#v", got)
	}

	operation, err = client.CheckIn("SYNTH-ONE", fixtureStart.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(snapshot, snapshot.Revision)
	server.RevokeClient(operation.ClientID)
	if got := server.Merge(operation, fixtureStart.Add(2*time.Minute)); got.Status != MergeRevokedClientReview {
		t.Fatalf("revoked reconnect = %#v", got)
	}
}

func TestFixtureFailsClosedForIneligibleAndStaleRoster(t *testing.T) {
	snapshot := fixtureSnapshot()
	client := NewClient(snapshot, "synthetic-client-a")
	if _, err := client.CheckIn("SYNTH-NO", fixtureStart); !errors.Is(err, ErrTicketIneligible) {
		t.Fatalf("ineligible local check = %v", err)
	}
	op, err := client.CheckIn("SYNTH-ONE", fixtureStart)
	if err != nil {
		t.Fatal(err)
	}
	if got := NewServer(snapshot, "synthetic-roster-r2").Merge(op, fixtureStart); got.Status != MergeStaleSnapshot {
		t.Fatalf("stale roster merge = %#v", got)
	}
}

func TestExistingCheckInAndChangedReplayNeverMutateFixtureState(t *testing.T) {
	snapshot := fixtureSnapshot()
	ticket := snapshot.Tickets["SYNTH-ONE"]
	ticket.CheckedIn = true
	snapshot.Tickets["SYNTH-ONE"] = ticket
	client := NewClient(snapshot, "synthetic-client-a")
	op, err := client.CheckIn("SYNTH-ONE", fixtureStart)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(snapshot, snapshot.Revision)
	if got := server.Merge(op, fixtureStart); got.Status != MergeDuplicateCheckIn {
		t.Fatalf("existing check-in = %#v", got)
	}
	changed := op
	changed.TicketCode = "SYNTH-NO"
	if got := server.Merge(changed, fixtureStart); got.Status != MergeOperationConflict {
		t.Fatalf("changed replay = %#v", got)
	}
}
