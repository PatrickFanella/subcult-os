package app

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

const eventSettlementExportFilename = "event-settlement.csv"

type eventSettlementExportSnapshot struct {
	ReportID              string
	SettlementID          string
	EventID               string
	EventTitle            string
	StartsAt              time.Time
	Currency              string
	GrossPaidRevenueCents int
	PaidTicketCount       int
	PendingTicketCount    int
	CancelledTicketCount  int
	FreeTicketCount       int
	ReservedCount         int
	Status                string
	GeneratedAt           time.Time
	FinalizedAt           sql.NullTime
	FinalizedByPersonID   sql.NullString
	Adjustments           []eventSettlementAdjustmentRow
}

// handleGetSettlementCSV exports the closed event's stored settlement and
// append-only corrections. It deliberately does not read tickets, contacts,
// archive participants, staffing, or notes: this is a finance-only snapshot,
// not a general event-data export.
func (a *App) handleGetSettlementCSV(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	if _, ok := a.requirePermission(r, event.WorkspaceID, permFinance); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	snapshot, err := a.loadEventSettlementExport(r.Context(), event.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "settlement export unavailable")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load settlement export")
		return
	}
	// The snapshot can take long enough for a membership to be revoked. Check
	// the finance capability again immediately before any private bytes or
	// response headers are emitted.
	if _, ok := a.requirePermission(r, event.WorkspaceID, permFinance); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+eventSettlementExportFilename+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	writer := csv.NewWriter(w)
	if err := writeSettlementCSV(writer, snapshot); err != nil {
		return
	}
	writer.Flush()
}

// loadEventSettlementExport uses one read-only repeatable-read transaction so
// the settlement row and correction rows describe one consistent snapshot.
func (a *App) loadEventSettlementExport(ctx context.Context, eventID string) (eventSettlementExportSnapshot, error) {
	tx, err := a.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return eventSettlementExportSnapshot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var snapshot eventSettlementExportSnapshot
	err = tx.QueryRow(ctx, `
		select r.id, s.id, e.id, e.title, e.starts_at, s.currency,
		       s.gross_paid_revenue_cents, s.paid_ticket_count, s.pending_ticket_count,
		       s.cancelled_ticket_count, s.free_ticket_count, s.reserved_count,
		       s.status, s.generated_at, s.finalized_at, s.finalized_by_person_id
		from events e
		join event_reports r on r.event_id = e.id
		join event_settlements s on s.event_id = e.id
		where e.id = $1
	`, eventID).Scan(
		&snapshot.ReportID, &snapshot.SettlementID, &snapshot.EventID, &snapshot.EventTitle,
		&snapshot.StartsAt, &snapshot.Currency, &snapshot.GrossPaidRevenueCents,
		&snapshot.PaidTicketCount, &snapshot.PendingTicketCount, &snapshot.CancelledTicketCount,
		&snapshot.FreeTicketCount, &snapshot.ReservedCount, &snapshot.Status, &snapshot.GeneratedAt,
		&snapshot.FinalizedAt, &snapshot.FinalizedByPersonID,
	)
	if err != nil {
		return eventSettlementExportSnapshot{}, err
	}

	rows, err := tx.Query(ctx, `
		select id, settlement_id, amount_cents, label, reason, created_by_person_id, created_at
		from event_settlement_adjustments
		where settlement_id = $1
		order by created_at asc, id asc
	`, snapshot.SettlementID)
	if err != nil {
		return eventSettlementExportSnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var adjustment eventSettlementAdjustmentRow
		if err := rows.Scan(&adjustment.ID, &adjustment.SettlementID, &adjustment.AmountCents,
			&adjustment.Label, &adjustment.Reason, &adjustment.CreatedByPersonID, &adjustment.CreatedAt); err != nil {
			return eventSettlementExportSnapshot{}, err
		}
		snapshot.Adjustments = append(snapshot.Adjustments, adjustment)
	}
	if err := rows.Err(); err != nil {
		return eventSettlementExportSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return eventSettlementExportSnapshot{}, err
	}
	return snapshot, nil
}

func writeSettlementCSV(writer *csv.Writer, snapshot eventSettlementExportSnapshot) error {
	adjustmentTotalCents := 0
	for _, adjustment := range snapshot.Adjustments {
		adjustmentTotalCents += adjustment.AmountCents
	}
	netTotalCents := settlementNetTotalCents(snapshot.GrossPaidRevenueCents, adjustmentTotalCents)
	if err := writer.Write([]string{
		"record_type", "event_id", "settlement_id", "report_id", "event_title", "event_starts_at",
		"settlement_generated_at", "settlement_status", "finalized_at", "finalized_by_person_id",
		"currency", "gross_paid_revenue_cents", "adjustment_total_cents", "net_total_cents",
		"paid_ticket_count", "pending_ticket_count", "cancelled_ticket_count", "free_ticket_count", "reserved_count", "adjustment_id",
		"adjustment_amount_cents", "adjustment_label", "adjustment_reason",
		"adjustment_created_by_person_id", "adjustment_created_at",
	}); err != nil {
		return err
	}

	summary := settlementCSVBaseRow(snapshot, adjustmentTotalCents, netTotalCents)
	summary[0] = "settlement"
	if err := writer.Write(summary); err != nil {
		return err
	}
	for _, adjustment := range snapshot.Adjustments {
		row := settlementCSVBaseRow(snapshot, adjustmentTotalCents, netTotalCents)
		row[0] = "adjustment"
		// Summary money is intentionally blank on correction rows. A spreadsheet
		// that sums a column therefore cannot count gross or net revenue once per
		// adjustment; adjustment_amount_cents is the only row-level money value.
		row[11], row[12], row[13] = "", "", ""
		row[19] = csvFormulaSafe(adjustment.ID)
		row[20] = strconv.Itoa(adjustment.AmountCents)
		row[21] = csvFormulaSafe(adjustment.Label)
		row[22] = csvFormulaSafe(adjustment.Reason)
		row[23] = csvFormulaSafe(adjustment.CreatedByPersonID)
		row[24] = adjustment.CreatedAt.UTC().Format(time.RFC3339Nano)
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return writer.Error()
}

func settlementCSVBaseRow(snapshot eventSettlementExportSnapshot, adjustmentTotalCents, netTotalCents int) []string {
	finalizedAt := ""
	if snapshot.FinalizedAt.Valid {
		finalizedAt = snapshot.FinalizedAt.Time.UTC().Format(time.RFC3339Nano)
	}
	finalizedBy := ""
	if snapshot.FinalizedByPersonID.Valid {
		finalizedBy = csvFormulaSafe(snapshot.FinalizedByPersonID.String)
	}
	return []string{
		"", csvFormulaSafe(snapshot.EventID), csvFormulaSafe(snapshot.SettlementID), csvFormulaSafe(snapshot.ReportID),
		csvFormulaSafe(snapshot.EventTitle), snapshot.StartsAt.UTC().Format(time.RFC3339Nano),
		snapshot.GeneratedAt.UTC().Format(time.RFC3339Nano), csvFormulaSafe(snapshot.Status), finalizedAt, finalizedBy,
		csvFormulaSafe(snapshot.Currency), strconv.Itoa(snapshot.GrossPaidRevenueCents), strconv.Itoa(adjustmentTotalCents), strconv.Itoa(netTotalCents),
		strconv.Itoa(snapshot.PaidTicketCount), strconv.Itoa(snapshot.PendingTicketCount), strconv.Itoa(snapshot.CancelledTicketCount),
		strconv.Itoa(snapshot.FreeTicketCount), strconv.Itoa(snapshot.ReservedCount), "", "", "", "", "", "",
	}
}

// csvFormulaSafe prefixes a literal apostrophe when spreadsheet software may
// interpret untrusted text as a formula, including after leading whitespace or
// control characters. The source value remains visible and CSV quoting is
// still handled by encoding/csv.
func csvFormulaSafe(value string) string {
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			continue
		}
		if r == '=' || r == '+' || r == '-' || r == '@' {
			return "'" + value
		}
		return value
	}
	return value
}
