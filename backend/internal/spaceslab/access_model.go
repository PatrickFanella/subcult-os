// Package spaceslab holds a synthetic, offline access-model experiment.
//
// It deliberately does not implement the AT Protocol Spaces proposal or any
// provider API. Its small state machine gives the research note executable
// examples for membership, scoped reads, withdrawal, recovery, and locally
// retained copies.
package spaceslab

import (
	"errors"
	"time"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleCrew    Role = "crew"
	RoleFinance Role = "finance"
)

type Scope string

const (
	ScopeCrew    Scope = "crew"
	ScopeFinance Scope = "finance"
)

var (
	ErrDenied           = errors.New("synthetic space access denied")
	ErrRemoved          = errors.New("synthetic member has been removed")
	ErrRecoveryRequired = errors.New("synthetic recovery decision is required")
)

// Record contains only synthetic test content. A locally cached record models
// a copy a client has already received; removing membership cannot erase it.
type Record struct {
	ID      string
	Scope   Scope
	Content string
}

type member struct {
	role      Role
	expiresAt time.Time
	removed   bool
}

// Lab is an intentionally in-memory, single-process model. It makes no
// protocol, storage, identity, or network claims.
type Lab struct {
	members      map[string]member
	applications map[string]bool
	cache        map[string]map[string]Record
}

func New() *Lab {
	return &Lab{
		members:      make(map[string]member),
		applications: make(map[string]bool),
		cache:        make(map[string]map[string]Record),
	}
}

// SubmitApplication records a synthetic application. It grants no access.
func (l *Lab) SubmitApplication(principal string) {
	l.applications[principal] = true
}

// WithdrawApplication marks the synthetic application withdrawn. Withdrawal
// is independent of membership, because an application is not a credential.
func (l *Lab) WithdrawApplication(principal string) {
	l.applications[principal] = false
}

func (l *Lab) ApplicationActive(principal string) bool {
	return l.applications[principal]
}

// GrantMember creates or renews a membership until it is removed. A removal
// is terminal in this model until an explicit recovery decision is made.
func (l *Lab) GrantMember(principal string, role Role, expiresAt time.Time) error {
	if previous, ok := l.members[principal]; ok && previous.removed {
		return ErrRemoved
	}
	l.members[principal] = member{role: role, expiresAt: expiresAt}
	return nil
}

// RemoveMember blocks future reads and later ordinary membership replays. The
// timestamp is accepted only as a fixture input; a zero timestamp is still an
// explicit removal rather than a way to bypass removal precedence.
func (l *Lab) RemoveMember(principal string, at time.Time) {
	current := l.members[principal]
	current.removed = true
	l.members[principal] = current
}

// RecoverMember models a separately reviewed synthetic recovery decision. It
// is deliberately explicit so a stale membership grant cannot restore access.
func (l *Lab) RecoverMember(principal string, role Role, expiresAt time.Time, recoveryApproved bool) error {
	if !recoveryApproved {
		return ErrRecoveryRequired
	}
	l.members[principal] = member{role: role, expiresAt: expiresAt}
	return nil
}

// Read decides a fresh server-side read, then stores the received synthetic
// record as a local copy for that principal.
func (l *Lab) Read(principal string, record Record, now time.Time) (Record, error) {
	if !l.canRead(principal, record.Scope, now) {
		return Record{}, ErrDenied
	}
	if l.cache[principal] == nil {
		l.cache[principal] = make(map[string]Record)
	}
	l.cache[principal][record.ID] = record
	return record, nil
}

// Cached returns a previously received local copy. It intentionally does not
// re-authorize: denial of a future read is not erasure of a copied record.
func (l *Lab) Cached(principal, recordID string) (Record, bool) {
	record, ok := l.cache[principal][recordID]
	return record, ok
}

// ClearCached simulates an application clearing its own local store.
func (l *Lab) ClearCached(principal string) {
	delete(l.cache, principal)
}

func (l *Lab) canRead(principal string, scope Scope, now time.Time) bool {
	membership, ok := l.members[principal]
	if !ok || membership.removed || !membership.expiresAt.After(now) {
		return false
	}
	if scope != ScopeCrew && scope != ScopeFinance {
		return false
	}
	if membership.role == RoleOwner {
		return true
	}
	return (membership.role == RoleCrew && scope == ScopeCrew) ||
		(membership.role == RoleFinance && scope == ScopeFinance)
}
