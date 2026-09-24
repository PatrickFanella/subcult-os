package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LINK-01: a private, operator-side relationship between a local `events`
// row and a public tv.subcult.event.occurrence AT record. This file never
// projects anything it reads or writes into a public/anonymous endpoint; a
// stray "select *" from event editing, occurrences, tickets or staffing
// never touches event_public_links, and the reverse never happens either.
// See docs/development/public-links.md.

const publicLinkOccurrenceCollection = "tv.subcult.event.occurrence"

const (
	publicLinkStatusFresh       = "fresh"
	publicLinkStatusChanged     = "changed"
	publicLinkStatusUnavailable = "unavailable"
	publicLinkStatusDeleted     = "deleted"
	publicLinkStatusInvalid     = "invalid"
)

type eventPublicLinkDTO struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspaceId"`
	EventID           string  `json:"eventId"`
	PublicURI         string  `json:"publicUri"`
	ObservedCID       string  `json:"observedCid"`
	AuthorityDID      string  `json:"authorityDid"`
	Status            string  `json:"status"`
	LastError         *string `json:"lastError,omitempty"`
	ObservedAt        string  `json:"observedAt"`
	LastCheckedAt     string  `json:"lastCheckedAt"`
	CreatedByPersonID string  `json:"createdByPersonId"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type eventPublicLinkRow struct {
	ID                string
	WorkspaceID       string
	EventID           string
	PublicURI         string
	ObservedCID       string
	AuthorityDID      string
	Status            string
	LastError         sql.NullString
	ObservedAt        time.Time
	LastCheckedAt     time.Time
	CreatedByPersonID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

const publicLinkSelectColumns = `
	id, workspace_id, event_id, public_uri, observed_cid, authority_did, status, last_error,
	observed_at, last_checked_at, created_by_person_id, created_at, updated_at
`

func scanPublicLinkRow(row pgx.Row) (eventPublicLinkRow, error) {
	var link eventPublicLinkRow
	err := row.Scan(&link.ID, &link.WorkspaceID, &link.EventID, &link.PublicURI, &link.ObservedCID,
		&link.AuthorityDID, &link.Status, &link.LastError, &link.ObservedAt, &link.LastCheckedAt,
		&link.CreatedByPersonID, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return eventPublicLinkRow{}, err
	}
	return link, nil
}

func publicLinkDTOFromRow(row eventPublicLinkRow) eventPublicLinkDTO {
	return eventPublicLinkDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		EventID:           row.EventID,
		PublicURI:         row.PublicURI,
		ObservedCID:       row.ObservedCID,
		AuthorityDID:      row.AuthorityDID,
		Status:            row.Status,
		LastError:         nullableString(row.LastError),
		ObservedAt:        row.ObservedAt.UTC().Format(time.RFC3339Nano),
		LastCheckedAt:     row.LastCheckedAt.UTC().Format(time.RFC3339Nano),
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// validLinkID reports whether a path value can be a link id at all, so a
// malformed value is answered with 404 instead of a database error.
func validLinkID(linkID string) bool {
	var id pgtype.UUID
	return linkID != "" && id.Scan(linkID) == nil
}

func (a *App) loadEventPublicLink(ctx context.Context, eventID, linkID string) (eventPublicLinkRow, error) {
	if !validLinkID(linkID) {
		return eventPublicLinkRow{}, pgx.ErrNoRows
	}
	row := a.db.QueryRow(ctx, `select `+publicLinkSelectColumns+` from event_public_links where id = $1 and event_id = $2`, linkID, eventID)
	return scanPublicLinkRow(row)
}

// occurrenceRecordFields decodes only the tv.subcult.event.occurrence fields
// this feature surfaces in a preview; it is intentionally a strict subset,
// not a copy of the admitted Lexicon's full field set, and is never fed
// back into ValidateAdmittedRecord (which runs against the raw record bytes
// beforehand).
type occurrenceRecordFields struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StartsAt    string `json:"startsAt"`
	EndsAt      string `json:"endsAt"`
	AllDay      bool   `json:"allDay"`
	Timezone    string `json:"timezone"`
	Status      string `json:"status"`
}

type publicLinkSourceIdentityDTO struct {
	DID    string `json:"did"`
	Handle string `json:"handle,omitempty"`
}

type publicLinkEventFieldsDTO struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	StartsAt    string `json:"startsAt"`
	EndsAt      string `json:"endsAt,omitempty"`
	AllDay      bool   `json:"allDay,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Status      string `json:"status,omitempty"`
}

type publicLinkPreviewDTO struct {
	SourceIdentity publicLinkSourceIdentityDTO `json:"sourceIdentity"`
	PublicURI      string                      `json:"publicUri"`
	CID            string                      `json:"cid"`
	Event          publicLinkEventFieldsDTO    `json:"event"`
}

// resolvePublicLink parses rawURI, resolves the authority behind it and
// fetches + Lexicon-validates the record it names through the injectable
// a.recordFetcher, applying the same outbound policy as every other AT
// network call in this codebase (backend/internal/atproto). It never
// persists anything; both preview and attach use it, and attach persists
// only after this succeeds.
func (a *App) resolvePublicLink(ctx context.Context, rawURI string) (publicLinkPreviewDTO, atprotocol.ResolvedRecord, error) {
	trimmed := strings.TrimSpace(rawURI)
	if trimmed == "" {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errors.New("publicUri is required")
	}
	ref, err := atprotocol.ParseRecordRef(trimmed)
	if err != nil {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errors.New("publicUri is not a valid AT URI")
	}
	if ref.Authority.Kind != atprotocol.IdentifierDID {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errors.New("publicUri authority must be a DID, not a handle")
	}
	if ref.Collection != publicLinkOccurrenceCollection {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errors.New("publicUri must reference a " + publicLinkOccurrenceCollection + " record")
	}
	if a.recordFetcher == nil {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errPublicLinkFetcherUnavailable
	}

	record, err := a.recordFetcher.FetchRecord(ctx, ref)
	if err != nil {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, err
	}
	if err := validatePublicRecord(a.lexiconCatalog, publicLinkOccurrenceCollection, record.Value); err != nil {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errPublicLinkValidation{err: err}
	}

	var fields occurrenceRecordFields
	if err := json.Unmarshal(record.Value, &fields); err != nil {
		return publicLinkPreviewDTO{}, atprotocol.ResolvedRecord{}, errPublicLinkValidation{err: err}
	}

	preview := publicLinkPreviewDTO{
		SourceIdentity: publicLinkSourceIdentityDTO{DID: record.DID, Handle: record.Handle},
		PublicURI:      ref.URI,
		CID:            record.CID,
		Event: publicLinkEventFieldsDTO{
			Name:        fields.Name,
			Description: fields.Description,
			StartsAt:    fields.StartsAt,
			EndsAt:      fields.EndsAt,
			AllDay:      fields.AllDay,
			Timezone:    fields.Timezone,
			Status:      fields.Status,
		},
	}
	return preview, record, nil
}

var errPublicLinkFetcherUnavailable = errors.New("public record fetcher is unavailable")

// errPublicLinkValidation marks a record that was fetched successfully but
// failed admitted-Lexicon validation, distinct from a fetch/resolution
// failure, so handlers can return 422 instead of 502.
type errPublicLinkValidation struct{ err error }

func (e errPublicLinkValidation) Error() string { return e.err.Error() }
func (e errPublicLinkValidation) Unwrap() error { return e.err }

// writePublicLinkResolveError classifies a resolvePublicLink failure into
// the right HTTP status: bad input, a validation failure in an otherwise
// reachable record, a definitively deleted record, or an unavailable
// upstream (network/identity failure).
func writePublicLinkResolveError(w http.ResponseWriter, err error) {
	var notFound *atprotocol.RecordNotFoundError
	var validationErr errPublicLinkValidation
	switch {
	case errors.As(err, &notFound):
		writeError(w, http.StatusNotFound, "public record not found")
	case errors.As(err, &validationErr):
		writeError(w, http.StatusUnprocessableEntity, "public record failed validation: "+validationErr.Error())
	case errors.Is(err, errPublicLinkFetcherUnavailable):
		writeError(w, http.StatusServiceUnavailable, "public record fetcher is unavailable")
	case err != nil && (strings.Contains(err.Error(), "publicUri") || strings.Contains(err.Error(), "AT URI")):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadGateway, "could not fetch public record")
	}
}

type publicLinkPreviewRequest struct {
	PublicURI string `json:"publicUri"`
}

func (a *App) handlePreviewEventPublicLink(w http.ResponseWriter, r *http.Request) {
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
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req publicLinkPreviewRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	preview, _, err := a.resolvePublicLink(r.Context(), req.PublicURI)
	if err != nil {
		writePublicLinkResolveError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (a *App) handleListEventPublicLinks(w http.ResponseWriter, r *http.Request) {
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
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	rows, err := a.db.Query(r.Context(), `select `+publicLinkSelectColumns+` from event_public_links where event_id = $1 order by created_at desc, id`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load public links")
		return
	}
	defer rows.Close()

	items := make([]eventPublicLinkDTO, 0)
	for rows.Next() {
		row, err := scanPublicLinkRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load public links")
			return
		}
		items = append(items, publicLinkDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load public links")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type attachPublicLinkRequest struct {
	PublicURI string `json:"publicUri"`
}

// handleAttachEventPublicLink persists the private link. Re-attaching the
// same (event, publicUri) pair is idempotent: the ON CONFLICT DO UPDATE
// no-op below returns the pre-existing row unchanged (observed_cid is left
// as it was at first attach) rather than inserting a duplicate or silently
// re-observing a new CID; use the refresh endpoint to update freshness.
func (a *App) handleAttachEventPublicLink(w http.ResponseWriter, r *http.Request) {
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
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req attachPublicLinkRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	preview, record, err := a.resolvePublicLink(r.Context(), req.PublicURI)
	if err != nil {
		writePublicLinkResolveError(w, err)
		return
	}

	row := a.db.QueryRow(r.Context(), `
		insert into event_public_links (workspace_id, event_id, public_uri, observed_cid, authority_did, status, observed_at, last_checked_at, created_by_person_id)
		values ($1, $2, $3, $4, $5, $6, now(), now(), $7)
		on conflict (event_id, public_uri) do update set updated_at = event_public_links.updated_at
		returning `+publicLinkSelectColumns, event.WorkspaceID, event.ID, preview.PublicURI, record.CID, record.DID, publicLinkStatusFresh, actorID)
	link, err := scanPublicLinkRow(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not attach public link")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_public_link.attached", "event_public_link", link.ID, map[string]any{
		"eventId":     event.ID,
		"workspaceId": event.WorkspaceID,
		"publicUri":   link.PublicURI,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, publicLinkDTOFromRow(link))
}

func (a *App) handleDetachEventPublicLink(w http.ResponseWriter, r *http.Request) {
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
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	linkID := r.PathValue("linkID")
	if !validLinkID(linkID) {
		writeError(w, http.StatusNotFound, "public link not found")
		return
	}
	tag, err := a.db.Exec(r.Context(), `delete from event_public_links where id = $1 and event_id = $2`, linkID, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not detach public link")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "public link not found")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_public_link.detached", "event_public_link", linkID, map[string]any{
		"eventId":     event.ID,
		"workspaceId": event.WorkspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "detached"})
}

// handleRefreshEventPublicLink re-fetches the linked public record and
// updates only this row's status/freshness columns. It never touches the
// event, its occurrences, tickets or staffing: a changed, unavailable or
// deleted public record is an operator-review signal on this link, never a
// silent rewrite of local operational facts (docs/development/data-boundaries.md).
func (a *App) handleRefreshEventPublicLink(w http.ResponseWriter, r *http.Request) {
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
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	link, err := a.loadEventPublicLink(r.Context(), event.ID, r.PathValue("linkID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "public link not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load public link")
		return
	}

	ref, err := atprotocol.ParseRecordRef(link.PublicURI)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "stored public link URI is invalid")
		return
	}
	if a.recordFetcher == nil {
		writeError(w, http.StatusServiceUnavailable, "public record fetcher is unavailable")
		return
	}

	newStatus := publicLinkStatusFresh
	var newObservedCID sql.NullString
	var lastError sql.NullString
	record, fetchErr := a.recordFetcher.FetchRecord(r.Context(), ref)
	var notFound *atprotocol.RecordNotFoundError
	var validationErr error
	if fetchErr == nil {
		validationErr = validatePublicRecord(a.lexiconCatalog, publicLinkOccurrenceCollection, record.Value)
	}
	switch {
	case fetchErr == nil && validationErr != nil:
		// The record still exists but no longer matches the admitted
		// Lexicon; record what was observed and flag it for review.
		newStatus = publicLinkStatusInvalid
		newObservedCID = sql.NullString{String: record.CID, Valid: true}
		lastError = sql.NullString{String: validationErr.Error(), Valid: true}
	case fetchErr == nil && record.CID == link.ObservedCID:
		newStatus = publicLinkStatusFresh
	case fetchErr == nil:
		newStatus = publicLinkStatusChanged
		newObservedCID = sql.NullString{String: record.CID, Valid: true}
	case errors.As(fetchErr, &notFound):
		newStatus = publicLinkStatusDeleted
	default:
		newStatus = publicLinkStatusUnavailable
		lastError = sql.NullString{String: fetchErr.Error(), Valid: true}
	}

	var row pgx.Row
	if newObservedCID.Valid {
		row = a.db.QueryRow(r.Context(), `
			update event_public_links
			set status = $3, observed_cid = $4, observed_at = now(), last_checked_at = now(), last_error = $5, updated_at = now()
			where id = $1 and event_id = $2
			returning `+publicLinkSelectColumns, link.ID, event.ID, newStatus, newObservedCID.String, lastError)
	} else {
		row = a.db.QueryRow(r.Context(), `
			update event_public_links
			set status = $3, last_checked_at = now(), last_error = $4, updated_at = now()
			where id = $1 and event_id = $2
			returning `+publicLinkSelectColumns, link.ID, event.ID, newStatus, lastError)
	}
	updated, err := scanPublicLinkRow(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update public link")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_public_link.refreshed", "event_public_link", updated.ID, map[string]any{
		"eventId":     event.ID,
		"workspaceId": event.WorkspaceID,
		"status":      updated.Status,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, publicLinkDTOFromRow(updated))
}
