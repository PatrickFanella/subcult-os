package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
	"strings"
	"time"
	"unicode"
)

type financeLineDTO struct {
	ID                string  `json:"id"`
	EventID           string  `json:"eventId"`
	EntryType         string  `json:"entryType"`
	Direction         string  `json:"direction"`
	AmountCents       int64   `json:"amountCents"`
	Currency          string  `json:"currency"`
	Label             string  `json:"label"`
	Reason            string  `json:"reason"`
	DueAt             *string `json:"dueAt,omitempty"`
	OccurredAt        *string `json:"occurredAt,omitempty"`
	PayableLineID     *string `json:"payableLineId,omitempty"`
	CorrectsLineID    *string `json:"correctsLineId,omitempty"`
	CreatedByPersonID string  `json:"createdByPersonId"`
	CreatedAt         string  `json:"createdAt"`
}
type financeLineReq struct {
	EntryType      string  `json:"entryType"`
	Direction      string  `json:"direction"`
	AmountCents    int64   `json:"amountCents"`
	Currency       string  `json:"currency"`
	Label          string  `json:"label"`
	Reason         string  `json:"reason"`
	DueAt          *string `json:"dueAt"`
	OccurredAt     *string `json:"occurredAt"`
	PayableLineID  *string `json:"payableLineId"`
	CorrectsLineID *string `json:"correctsLineId"`
	RequestKey     string  `json:"requestKey"`
}

func nullableTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}
func scanFinanceLine(row pgx.Row) (financeLineDTO, error) {
	var x financeLineDTO
	var due, occurred *time.Time
	var payable, corrects *string
	var created time.Time
	err := row.Scan(&x.ID, &x.EventID, &x.EntryType, &x.Direction, &x.AmountCents, &x.Currency, &x.Label, &x.Reason, &due, &occurred, &payable, &corrects, &x.CreatedByPersonID, &created)
	x.DueAt = nullableTimePtr(due)
	x.OccurredAt = nullableTimePtr(occurred)
	x.PayableLineID = payable
	x.CorrectsLineID = corrects
	x.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	return x, err
}

const financeLineColumns = "id,event_id,entry_type,direction,amount_cents,currency,label,reason,due_at,occurred_at,payable_line_id,corrects_line_id,created_by_person_id,created_at"

func validFinanceUUID(value string) bool {
	var id pgtype.UUID
	return id.Scan(strings.TrimSpace(value)) == nil && id.Valid
}
func financeFingerprint(req financeLineReq) string {
	b, _ := json.Marshal(req)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func parseFinanceTime(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	v, err := parseRFC3339Time(*raw)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func validateFinanceLine(req *financeLineReq) (*time.Time, *time.Time, error) {
	req.EntryType = strings.TrimSpace(req.EntryType)
	req.Direction = strings.TrimSpace(req.Direction)
	req.Currency = strings.ToLower(strings.TrimSpace(req.Currency))
	req.Label = strings.TrimSpace(req.Label)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.EntryType != "budget" && req.EntryType != "payable" && req.EntryType != "actual_payment" {
		return nil, nil, errors.New("invalid entryType")
	}
	if req.Direction != "income" && req.Direction != "expense" {
		return nil, nil, errors.New("invalid direction")
	}
	if req.AmountCents < 0 || req.AmountCents > 1000000000 {
		return nil, nil, errors.New("amountCents must be 0..1000000000")
	}
	if req.AmountCents == 0 && req.CorrectsLineID == nil {
		return nil, nil, errors.New("original amountCents must be positive")
	}
	if len(req.Currency) != 3 || strings.IndexFunc(req.Currency, func(r rune) bool { return r < 'a' || r > 'z' }) >= 0 {
		return nil, nil, errors.New("currency must be a lowercase three-letter code")
	}
	if req.Label == "" || len(req.Label) > 240 || strings.IndexFunc(req.Label, unicode.IsControl) >= 0 {
		return nil, nil, errors.New("invalid label")
	}
	if req.Reason == "" || len(req.Reason) > 2000 || strings.IndexFunc(req.Reason, unicode.IsControl) >= 0 {
		return nil, nil, errors.New("invalid reason")
	}
	if !validFinanceUUID(req.RequestKey) {
		return nil, nil, errors.New("requestKey must be UUID")
	}
	due, err := parseFinanceTime(req.DueAt)
	if err != nil {
		return nil, nil, errors.New("invalid dueAt")
	}
	occurred, err := parseFinanceTime(req.OccurredAt)
	if err != nil {
		return nil, nil, errors.New("invalid occurredAt")
	}
	if req.EntryType == "payable" {
		if req.Direction != "expense" {
			return nil, nil, errors.New("payable must be expense")
		}
	} else if due != nil {
		return nil, nil, errors.New("dueAt only applies to payable")
	}
	if req.EntryType == "actual_payment" {
		if occurred == nil {
			return nil, nil, errors.New("occurredAt is required for actual_payment")
		}
	} else if occurred != nil {
		return nil, nil, errors.New("occurredAt only applies to actual_payment")
	}
	return due, occurred, nil
}
func (a *App) financeActor(r *http.Request, workspace string) (string, bool) {
	actor, ok := a.requirePersonID(r)
	if !ok {
		return "", false
	}
	if err := a.authorize(r.Context(), actor, workspace, permFinance); err != nil {
		return "", false
	}
	return actor, true
}
func (a *App) handleListEventFinanceLines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		writeError(w, 404, "event not found")
		return
	}
	if _, ok := a.financeActor(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	rows, err := a.db.Query(r.Context(), "select "+financeLineColumns+" from event_finance_lines where event_id=$1 order by created_at,id", event.ID)
	if err != nil {
		writeError(w, 500, "could not load finance lines")
		return
	}
	out := []financeLineDTO{}
	for rows.Next() {
		x, err := scanFinanceLine(rows)
		if err != nil {
			rows.Close()
			writeError(w, 500, "could not load finance lines")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeError(w, 500, "could not load finance lines")
		return
	}
	rows.Close()
	if _, ok := a.financeActor(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}
func (a *App) handleCreateEventFinanceLine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		writeError(w, 404, "event not found")
		return
	}
	actor, ok := a.financeActor(r, event.WorkspaceID)
	if !ok {
		writeError(w, 403, "forbidden")
		return
	}
	var req financeLineReq
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "invalid json")
		return
	}
	due, occurred, err := validateFinanceLine(&req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	fp := financeFingerprint(req)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save finance line")
		return
	}
	defer tx.Rollback(r.Context())
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	var lockedEvent string
	if err := tx.QueryRow(txCtx, "select id from events where id=$1 for update", event.ID).Scan(&lockedEvent); err != nil {
		writeError(w, 409, "event changed")
		return
	}
	// Lock the membership by identity before evaluating its status. A revoked
	// or expired member that was waiting on the event lock must not inherit the
	// authorization decision made before the wait.
	var role string
	var revokedAt, removedAt, expiresAt *time.Time
	if err := tx.QueryRow(txCtx, "select role,revoked_at,removed_at,expires_at from workspace_members where workspace_id=$1 and person_id=$2 for update", event.WorkspaceID, actor).Scan(&role, &revokedAt, &removedAt, &expiresAt); err != nil {
		writeError(w, 403, "forbidden")
		return
	}
	var now time.Time
	if err := tx.QueryRow(txCtx, "select clock_timestamp()").Scan(&now); err != nil || revokedAt != nil || removedAt != nil || (expiresAt != nil && !expiresAt.After(now)) || !roleHasPermission(role, permFinance) {
		writeError(w, 403, "forbidden")
		return
	}
	var priorFP string
	var prior financeLineDTO
	err = tx.QueryRow(txCtx, "select request_fingerprint from event_finance_lines where event_id=$1 and created_by_person_id=$2 and request_key=$3", event.ID, actor, req.RequestKey).Scan(&priorFP)
	if err == nil {
		if priorFP != fp {
			writeError(w, 409, "requestKey payload mismatch")
			return
		}
		prior, err = scanFinanceLine(tx.QueryRow(txCtx, "select "+financeLineColumns+" from event_finance_lines where event_id=$1 and created_by_person_id=$2 and request_key=$3", event.ID, actor, req.RequestKey))
		if err != nil {
			writeError(w, 500, "could not load finance line")
			return
		}
		tx.Rollback(txCtx)
		if _, ok := a.financeActor(r, event.WorkspaceID); !ok {
			writeError(w, 403, "forbidden")
			return
		}
		writeJSON(w, 200, prior)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not save finance line")
		return
	}
	correctsID := ""
	if req.CorrectsLineID != nil {
		correctsID = strings.TrimSpace(*req.CorrectsLineID)
	}
	payableID := ""
	if req.PayableLineID != nil {
		payableID = strings.TrimSpace(*req.PayableLineID)
	}
	if correctsID != "" {
		var old financeLineDTO
		old, err = scanFinanceLine(tx.QueryRow(txCtx, "select "+financeLineColumns+" from event_finance_lines where id=$1 and event_id=$2 for update", correctsID, event.ID))
		if err != nil {
			writeError(w, 409, "correction predecessor is not current")
			return
		}
		var child string
		if err = tx.QueryRow(txCtx, "select id from event_finance_lines where corrects_line_id=$1", old.ID).Scan(&child); err == nil {
			writeError(w, 409, "correction predecessor is not current")
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, 500, "could not save finance line")
			return
		}
		oldPayable := ""
		if old.PayableLineID != nil {
			oldPayable = *old.PayableLineID
		}
		if old.EntryType != req.EntryType || old.Direction != req.Direction || old.Currency != req.Currency || oldPayable != payableID {
			writeError(w, 400, "correction must retain type, direction, currency, and payable link")
			return
		}
	}
	if payableID != "" && (req.EntryType != "actual_payment" || req.Direction != "expense") {
		writeError(w, 400, "payableLineId only applies to expense actual_payment")
		return
	}
	if req.EntryType == "actual_payment" && payableID != "" {
		var typ, dir, currency string
		err = tx.QueryRow(txCtx, "select entry_type,direction,currency from event_finance_lines where id=$1 and event_id=$2 and corrects_line_id is null", payableID, event.ID).Scan(&typ, &dir, &currency)
		if err != nil || typ != "payable" || dir != "expense" || currency != req.Currency {
			writeError(w, 400, "payableLineId must reference same-event original payable with matching currency")
			return
		}
	}
	x, err := scanFinanceLine(tx.QueryRow(txCtx, "insert into event_finance_lines(workspace_id,event_id,entry_type,direction,amount_cents,currency,label,reason,due_at,occurred_at,payable_line_id,corrects_line_id,request_key,request_fingerprint,created_by_person_id) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,nullif($11,'')::uuid,nullif($12,'')::uuid,$13,$14,$15) returning "+financeLineColumns, event.WorkspaceID, event.ID, req.EntryType, req.Direction, req.AmountCents, req.Currency, req.Label, req.Reason, due, occurred, payableID, correctsID, req.RequestKey, fp, actor))
	if err != nil {
		writeError(w, 500, "could not save finance line")
		return
	}
	if err = a.audit(txCtx, actor, "event_finance_line.created", "event_finance_line", x.ID, map[string]any{"eventId": event.ID, "entryType": x.EntryType}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err = tx.Commit(txCtx); err != nil {
		writeError(w, 500, "could not save finance line")
		return
	}
	if _, ok := a.financeActor(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, x)
}
