// Package offlinelab is a synthetic-only model for researching disconnected
// door operations. It is not connected to the application, device storage, a
// network transport, or a production admission path.
package offlinelab

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSnapshotExpired  = errors.New("synthetic snapshot expired")
	ErrTicketIneligible = errors.New("synthetic ticket is not eligible")
	ErrTicketUnknown    = errors.New("synthetic ticket is unknown")
)

type Ticket struct {
	Code              string
	DisplayName       string
	AdmissionEligible bool
	CheckedIn         bool
}

type Snapshot struct {
	ID        string
	EventID   string
	Revision  string
	ExpiresAt time.Time
	Tickets   map[string]Ticket
}

type Operation struct {
	ID               string
	ClientID         string
	SnapshotID       string
	SnapshotRevision string
	EventID          string
	TicketCode       string
	RecordedAt       time.Time
}

type Client struct {
	snapshot Snapshot
	clientID string
}

func NewClient(snapshot Snapshot, clientID string) *Client {
	return &Client{snapshot: snapshot, clientID: clientID}
}

// CheckIn only creates a local, provisional fixture operation. It does not
// admit a person and it does not communicate with a server.
func (c *Client) CheckIn(code string, now time.Time) (Operation, error) {
	if !now.Before(c.snapshot.ExpiresAt) {
		return Operation{}, ErrSnapshotExpired
	}
	ticket, ok := c.snapshot.Tickets[code]
	if !ok {
		return Operation{}, ErrTicketUnknown
	}
	if !ticket.AdmissionEligible {
		return Operation{}, ErrTicketIneligible
	}
	return Operation{
		ID:               uuid.NewString(),
		ClientID:         c.clientID,
		SnapshotID:       c.snapshot.ID,
		SnapshotRevision: c.snapshot.Revision,
		EventID:          c.snapshot.EventID,
		TicketCode:       code,
		RecordedAt:       now,
	}, nil
}

type MergeStatus string

const (
	MergeAccepted            MergeStatus = "accepted"
	MergeDuplicateOperation  MergeStatus = "duplicate_operation"
	MergeOperationConflict   MergeStatus = "operation_conflict"
	MergeDuplicateCheckIn    MergeStatus = "duplicate_check_in"
	MergeExpiredSnapshot     MergeStatus = "expired_snapshot"
	MergeStaleSnapshot       MergeStatus = "stale_snapshot"
	MergeRevokedClientReview MergeStatus = "revoked_client_review"
	MergeIneligible          MergeStatus = "ineligible"
	MergeUnknownTicket       MergeStatus = "unknown_ticket"
)

type MergeResult struct {
	OperationID string
	Status      MergeStatus
	DuplicateOf MergeStatus
}

// Server is a synthetic server-authoritative merge fixture. It deliberately
// retains review outcomes rather than trying to reconstruct a door action.
type Server struct {
	snapshot       Snapshot
	currentRev     string
	revokedClients map[string]bool
	results        map[string]MergeResult
	operations     map[string]Operation
	checkedIn      map[string]string
}

func NewServer(snapshot Snapshot, currentRevision string) *Server {
	return &Server{
		snapshot:       snapshot,
		currentRev:     currentRevision,
		revokedClients: make(map[string]bool),
		results:        make(map[string]MergeResult),
		operations:     make(map[string]Operation),
		checkedIn:      make(map[string]string),
	}
}

func (s *Server) RevokeClient(clientID string) {
	s.revokedClients[clientID] = true
}

func (s *Server) Merge(op Operation, now time.Time) MergeResult {
	if previous, ok := s.results[op.ID]; ok {
		if original := s.operations[op.ID]; original != op {
			return MergeResult{OperationID: op.ID, Status: MergeOperationConflict}
		}
		return MergeResult{OperationID: op.ID, Status: MergeDuplicateOperation, DuplicateOf: previous.Status}
	}
	result := MergeResult{OperationID: op.ID}
	switch {
	case s.revokedClients[op.ClientID]:
		result.Status = MergeRevokedClientReview
	case op.EventID != s.snapshot.EventID || op.SnapshotID != s.snapshot.ID || op.SnapshotRevision != s.currentRev:
		result.Status = MergeStaleSnapshot
	case !now.Before(s.snapshot.ExpiresAt):
		result.Status = MergeExpiredSnapshot
	default:
		ticket, ok := s.snapshot.Tickets[op.TicketCode]
		if !ok {
			result.Status = MergeUnknownTicket
		} else if !ticket.AdmissionEligible {
			result.Status = MergeIneligible
		} else if ticket.CheckedIn {
			result.Status = MergeDuplicateCheckIn
		} else if _, duplicate := s.checkedIn[op.TicketCode]; duplicate {
			result.Status = MergeDuplicateCheckIn
		} else {
			s.checkedIn[op.TicketCode] = op.ID
			result.Status = MergeAccepted
		}
	}
	s.results[op.ID] = result
	s.operations[op.ID] = op
	return result
}
