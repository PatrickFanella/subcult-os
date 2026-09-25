package spaceslab

import (
	"errors"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	endsAt = start.Add(time.Hour)
)

func TestCrewMembershipExpiresAndRemovalWinsReplayedGrant(t *testing.T) {
	lab := New()
	if err := lab.GrantMember("synthetic:crew", RoleCrew, endsAt); err != nil {
		t.Fatalf("grant crew membership: %v", err)
	}
	record := Record{ID: "synthetic-crew-run-sheet", Scope: ScopeCrew, Content: "fixture only"}
	if _, err := lab.Read("synthetic:crew", record, start); err != nil {
		t.Fatalf("read before expiry: %v", err)
	}
	if _, err := lab.Read("synthetic:crew", record, endsAt); !errors.Is(err, ErrDenied) {
		t.Fatalf("read at expiry = %v, want ErrDenied", err)
	}

	lab.RemoveMember("synthetic:crew", start.Add(2*time.Hour))
	if err := lab.GrantMember("synthetic:crew", RoleCrew, endsAt.Add(24*time.Hour)); !errors.Is(err, ErrRemoved) {
		t.Fatalf("replayed ordinary grant = %v, want ErrRemoved", err)
	}
	if _, err := lab.Read("synthetic:crew", record, start.Add(2*time.Hour)); !errors.Is(err, ErrDenied) {
		t.Fatalf("read after removal = %v, want ErrDenied", err)
	}
}

func TestApplicationWithdrawalDoesNotGrantOrRevokeMembership(t *testing.T) {
	lab := New()
	record := Record{ID: "synthetic-crew-call", Scope: ScopeCrew, Content: "fixture only"}
	lab.SubmitApplication("synthetic:applicant")
	lab.WithdrawApplication("synthetic:applicant")
	if lab.ApplicationActive("synthetic:applicant") {
		t.Fatal("withdrawn synthetic application remains active")
	}
	if _, err := lab.Read("synthetic:applicant", record, start); !errors.Is(err, ErrDenied) {
		t.Fatalf("withdrawn application read = %v, want ErrDenied", err)
	}

	if err := lab.GrantMember("synthetic:crew", RoleCrew, endsAt); err != nil {
		t.Fatalf("grant crew membership: %v", err)
	}
	lab.SubmitApplication("synthetic:crew")
	lab.WithdrawApplication("synthetic:crew")
	if _, err := lab.Read("synthetic:crew", record, start); err != nil {
		t.Fatalf("application withdrawal changed separate membership: %v", err)
	}
}

func TestFinanceScopeAndLocalCacheResidual(t *testing.T) {
	lab := New()
	if err := lab.GrantMember("synthetic:crew", RoleCrew, endsAt); err != nil {
		t.Fatalf("grant crew membership: %v", err)
	}
	if err := lab.GrantMember("synthetic:finance", RoleFinance, endsAt); err != nil {
		t.Fatalf("grant finance membership: %v", err)
	}
	crewRecord := Record{ID: "synthetic-crew-plan", Scope: ScopeCrew, Content: "fixture only"}
	financeRecord := Record{ID: "synthetic-finance-closeout", Scope: ScopeFinance, Content: "fixture only"}
	if _, err := lab.Read("synthetic:crew", financeRecord, start); !errors.Is(err, ErrDenied) {
		t.Fatalf("crew finance read = %v, want ErrDenied", err)
	}
	if _, err := lab.Read("synthetic:finance", financeRecord, start); err != nil {
		t.Fatalf("finance scoped read: %v", err)
	}
	if _, err := lab.Read("synthetic:crew", crewRecord, start); err != nil {
		t.Fatalf("crew scoped read: %v", err)
	}

	lab.RemoveMember("synthetic:crew", start.Add(time.Minute))
	if _, err := lab.Read("synthetic:crew", crewRecord, start.Add(time.Minute)); !errors.Is(err, ErrDenied) {
		t.Fatalf("future read after removal = %v, want ErrDenied", err)
	}
	if copied, ok := lab.Cached("synthetic:crew", crewRecord.ID); !ok || copied != crewRecord {
		t.Fatalf("removed member cache = %#v, %v; want retained local copy", copied, ok)
	}
	lab.ClearCached("synthetic:crew")
	if _, ok := lab.Cached("synthetic:crew", crewRecord.ID); ok {
		t.Fatal("cleared local cache still contains copied record")
	}
}

func TestRecoveryRequiresExplicitSyntheticDecision(t *testing.T) {
	lab := New()
	if err := lab.GrantMember("synthetic:crew", RoleCrew, endsAt); err != nil {
		t.Fatalf("grant crew membership: %v", err)
	}
	lab.RemoveMember("synthetic:crew", start)
	if err := lab.RecoverMember("synthetic:crew", RoleCrew, endsAt, false); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("unapproved recovery = %v, want ErrRecoveryRequired", err)
	}
	if err := lab.RecoverMember("synthetic:crew", RoleCrew, endsAt, true); err != nil {
		t.Fatalf("approved recovery: %v", err)
	}
	if _, err := lab.Read("synthetic:crew", Record{ID: "synthetic-recovered", Scope: ScopeCrew}, start); err != nil {
		t.Fatalf("read after approved recovery: %v", err)
	}
}

func TestRemovalAndUnknownAccessInputsFailClosed(t *testing.T) {
	lab := New()
	if err := lab.GrantMember("synthetic:owner", RoleOwner, endsAt); err != nil {
		t.Fatalf("grant owner membership: %v", err)
	}
	if _, err := lab.Read("synthetic:owner", Record{ID: "unknown", Scope: "unknown"}, start); !errors.Is(err, ErrDenied) {
		t.Fatalf("owner unknown-scope read = %v, want ErrDenied", err)
	}
	if err := lab.GrantMember("synthetic:unknown-role", Role("unknown"), endsAt); err != nil {
		t.Fatalf("grant unknown role fixture: %v", err)
	}
	if _, err := lab.Read("synthetic:unknown-role", Record{ID: "crew", Scope: ScopeCrew}, start); !errors.Is(err, ErrDenied) {
		t.Fatalf("unknown-role read = %v, want ErrDenied", err)
	}
	if _, err := lab.Read("synthetic:unknown-principal", Record{ID: "crew", Scope: ScopeCrew}, start); !errors.Is(err, ErrDenied) {
		t.Fatalf("unknown-principal read = %v, want ErrDenied", err)
	}

	lab.RemoveMember("synthetic:owner", time.Time{})
	if err := lab.GrantMember("synthetic:owner", RoleOwner, endsAt); !errors.Is(err, ErrRemoved) {
		t.Fatalf("zero-time removal replay = %v, want ErrRemoved", err)
	}
}
