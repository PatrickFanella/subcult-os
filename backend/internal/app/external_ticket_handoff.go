package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

// externalTicketHandoff is private operator configuration. It neither fetches
// the target nor creates a provider session, payment, import, webhook, or
// public record. The URL is only a future browser navigation destination.
type externalTicketHandoffDTO struct {
	Visibility    string `json:"visibility"`
	OccurrenceID  string `json:"occurrenceId"`
	PurchaseURL   string `json:"purchaseUrl"`
	ProviderLabel string `json:"providerLabel"`
	UpdatedAt     string `json:"updatedAt"`
}

type externalTicketHandoffRow struct {
	OccurrenceID  string
	PurchaseURL   string
	ProviderLabel string
	UpdatedAt     time.Time
}

func scanExternalTicketHandoff(row pgx.Row) (externalTicketHandoffRow, error) {
	var handoff externalTicketHandoffRow
	err := row.Scan(&handoff.OccurrenceID, &handoff.PurchaseURL, &handoff.ProviderLabel, &handoff.UpdatedAt)
	return handoff, err
}

func externalTicketHandoffDTOFromRow(row externalTicketHandoffRow) externalTicketHandoffDTO {
	return externalTicketHandoffDTO{Visibility: "private", OccurrenceID: row.OccurrenceID, PurchaseURL: row.PurchaseURL, ProviderLabel: row.ProviderLabel, UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func (a *App) loadExternalTicketHandoff(ctx context.Context, occurrenceID string) (externalTicketHandoffRow, error) {
	return scanExternalTicketHandoff(a.db.QueryRow(ctx, `
		select occurrence_id, purchase_url, provider_label, updated_at
		from occurrence_external_ticket_handoffs where occurrence_id = $1
	`, occurrenceID))
}

func containsExternalTicketHost(allowed []string, host string) bool {
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), host) {
			return true
		}
	}
	return false
}

func (a *App) validateExternalTicketHandoff(purchaseURL, providerLabel string) (string, string, error) {
	if strings.IndexFunc(purchaseURL, unicode.IsControl) >= 0 || strings.IndexFunc(providerLabel, unicode.IsControl) >= 0 {
		return "", "", errors.New("purchaseUrl and providerLabel must not contain control characters")
	}
	purchaseURL = strings.TrimSpace(purchaseURL)
	providerLabel = strings.TrimSpace(providerLabel)
	if providerLabel == "" || len(providerLabel) > 120 {
		return "", "", errors.New("providerLabel must be 1-120 visible characters")
	}
	if purchaseURL == "" || len(purchaseURL) > 2000 {
		return "", "", errors.New("purchaseUrl must be 1-2000 characters")
	}
	parsed, err := url.ParseRequestURI(purchaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Port() != "" {
		return "", "", errors.New("purchaseUrl must be an HTTPS URL without credentials")
	}
	host := strings.ToLower(parsed.Hostname())
	if !containsExternalTicketHost(a.config.ExternalTicketAllowedHosts, host) {
		return "", "", errors.New("purchaseUrl hostname is not allowed")
	}
	return parsed.String(), providerLabel, nil
}

func (a *App) loadExternalTicketOccurrence(r *http.Request) (eventOccurrenceRow, string, bool) {
	occurrence, err := a.loadOccurrence(r.Context(), r.PathValue("occurrenceID"))
	if err != nil || occurrence.EventID != r.PathValue("eventID") {
		return eventOccurrenceRow{}, "", false
	}
	actorID, _, ok := a.requireWorkspaceRole(r, occurrence.WorkspaceID, roleOwner, roleOrganizer)
	return occurrence, actorID, ok
}

func (a *App) handleGetExternalTicketHandoff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	occurrence, _, ok := a.loadExternalTicketOccurrence(r)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	handoff, err := a.loadExternalTicketHandoff(r.Context(), occurrence.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "external ticket handoff not configured")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load external ticket handoff")
		return
	}
	if _, _, err := a.validateExternalTicketHandoff(handoff.PurchaseURL, handoff.ProviderLabel); err != nil {
		writeError(w, http.StatusConflict, "external ticket handoff is no longer allowed; review configuration")
		return
	}
	// The response is private. Recheck immediately before serializing it so a
	// revocation or expiry between the initial load and the output cannot leak
	// configuration that is no longer readable.
	if _, _, ok := a.loadExternalTicketOccurrence(r); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeJSON(w, http.StatusOK, externalTicketHandoffDTOFromRow(handoff))
}

type putExternalTicketHandoffRequest struct {
	PurchaseURL   string `json:"purchaseUrl"`
	ProviderLabel string `json:"providerLabel"`
	// ExpectedUpdatedAt is empty for a first configuration; thereafter it must
	// be the exact revision returned by GET or the prior PUT.
	ExpectedUpdatedAt *string `json:"expectedUpdatedAt"`
}

func (a *App) handlePutExternalTicketHandoff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	occurrence, actorID, ok := a.loadExternalTicketOccurrence(r)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req putExternalTicketHandoffRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	purchaseURL, providerLabel, err := a.validateExternalTicketHandoff(req.PurchaseURL, req.ProviderLabel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ExpectedUpdatedAt == nil {
		writeError(w, http.StatusBadRequest, "expectedUpdatedAt is required")
		return
	}

	var expected sql.NullTime
	if strings.TrimSpace(*req.ExpectedUpdatedAt) != "" {
		parsed, err := parseRFC3339Time(*req.ExpectedUpdatedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expectedUpdatedAt")
			return
		}
		expected = sql.NullTime{Time: parsed, Valid: true}
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save external ticket handoff")
		return
	}
	defer tx.Rollback(r.Context())
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	var lockedID string
	if err := tx.QueryRow(txCtx, `select id from event_occurrences where id=$1 and event_id=$2 for update`, occurrence.ID, occurrence.EventID).Scan(&lockedID); err != nil {
		writeError(w, http.StatusConflict, "occurrence changed; reload before editing")
		return
	}
	var activeRole string
	if err := tx.QueryRow(txCtx, `select role from workspace_members where workspace_id=$1 and person_id=$2 and removed_at is null and revoked_at is null and (expires_at is null or expires_at > clock_timestamp()) for update`, occurrence.WorkspaceID, actorID).Scan(&activeRole); err != nil || (activeRole != roleOwner && activeRole != roleOrganizer) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	current, currentErr := scanExternalTicketHandoff(tx.QueryRow(txCtx, `select occurrence_id,purchase_url,provider_label,updated_at from occurrence_external_ticket_handoffs where occurrence_id=$1 for update`, occurrence.ID))
	if currentErr != nil && !errors.Is(currentErr, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "could not save external ticket handoff")
		return
	}
	var handoff externalTicketHandoffRow
	if errors.Is(currentErr, pgx.ErrNoRows) {
		if expected.Valid {
			writeError(w, http.StatusConflict, "external ticket handoff changed; reload before editing")
			return
		}
		handoff, err = scanExternalTicketHandoff(tx.QueryRow(txCtx, `insert into occurrence_external_ticket_handoffs (occurrence_id,workspace_id,event_id,purchase_url,provider_label,created_by_person_id,updated_by_person_id) values ($1,$2,$3,$4,$5,$6,$6) returning occurrence_id,purchase_url,provider_label,updated_at`, occurrence.ID, occurrence.WorkspaceID, occurrence.EventID, purchaseURL, providerLabel, actorID))
	} else {
		if !expected.Valid || !expected.Time.Equal(current.UpdatedAt) {
			writeError(w, http.StatusConflict, "external ticket handoff changed; reload before editing")
			return
		}
		handoff, err = scanExternalTicketHandoff(tx.QueryRow(txCtx, `update occurrence_external_ticket_handoffs set purchase_url=$2,provider_label=$3,updated_by_person_id=$4,updated_at=greatest(clock_timestamp(),updated_at+interval '1 microsecond') where occurrence_id=$1 returning occurrence_id,purchase_url,provider_label,updated_at`, occurrence.ID, purchaseURL, providerLabel, actorID))
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save external ticket handoff")
		return
	}
	if err := a.audit(txCtx, actorID, "external_ticket_handoff.configured", "event_occurrence", occurrence.ID, map[string]any{"eventId": occurrence.EventID, "workspaceId": occurrence.WorkspaceID}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(txCtx); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save external ticket handoff")
		return
	}
	// Commit ends the membership lock. Check once more before emitting private
	// configuration so a just-expired or just-revoked session gets no body.
	if _, _, ok := a.loadExternalTicketOccurrence(r); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeJSON(w, http.StatusOK, externalTicketHandoffDTOFromRow(handoff))
}
