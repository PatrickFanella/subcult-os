package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata" // Keep IANA validation available in the minimal runtime image.

	"github.com/jackc/pgx/v5"
)

const (
	occurrenceStatusScheduled   = "scheduled"
	occurrenceStatusRescheduled = "rescheduled"
	occurrenceStatusPostponed   = "postponed"
	occurrenceStatusCancelled   = "cancelled"
)

func isValidOccurrenceStatus(status string) bool {
	switch status {
	case occurrenceStatusScheduled, occurrenceStatusRescheduled, occurrenceStatusPostponed, occurrenceStatusCancelled:
		return true
	default:
		return false
	}
}

type occurrenceCreditDTO struct {
	ProfileID   string `json:"profileId"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	SortOrder   int    `json:"sortOrder"`
}

type eventOccurrenceDTO struct {
	ID                string                `json:"id"`
	WorkspaceID       string                `json:"workspaceId"`
	EventID           string                `json:"eventId"`
	PlaceID           *string               `json:"placeId,omitempty"`
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	StartsAt          string                `json:"startsAt"`
	EndsAt            *string               `json:"endsAt,omitempty"`
	AllDay            bool                  `json:"allDay"`
	Timezone          *string               `json:"timezone,omitempty"`
	Status            string                `json:"status"`
	PublicURI         *string               `json:"publicUri,omitempty"`
	PublicCID         *string               `json:"publicCid,omitempty"`
	Credits           []occurrenceCreditDTO `json:"credits"`
	CreatedByPersonID string                `json:"createdByPersonId"`
	CreatedAt         string                `json:"createdAt"`
	UpdatedAt         string                `json:"updatedAt"`
}

type eventOccurrenceRow struct {
	ID                string
	WorkspaceID       string
	EventID           string
	PlaceID           sql.NullString
	Name              string
	Description       string
	StartsAt          time.Time
	EndsAt            sql.NullTime
	AllDay            bool
	Timezone          sql.NullString
	Status            string
	PublicURI         sql.NullString
	PublicCID         sql.NullString
	CreatedByPersonID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (a *App) loadOccurrenceCredits(ctx context.Context, occurrenceID string) ([]occurrenceCreditDTO, error) {
	rows, err := a.db.Query(ctx, `
		select eop.profile_id, cp.display_name, eop.role, eop.sort_order
		from event_occurrence_profiles eop
		join cultural_profiles cp on cp.id = eop.profile_id
		where eop.occurrence_id = $1
		order by eop.sort_order, eop.profile_id
	`, occurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	credits := make([]occurrenceCreditDTO, 0)
	for rows.Next() {
		var credit occurrenceCreditDTO
		if err := rows.Scan(&credit.ProfileID, &credit.DisplayName, &credit.Role, &credit.SortOrder); err != nil {
			return nil, err
		}
		credits = append(credits, credit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return credits, nil
}

func (a *App) occurrenceDTOFromRow(ctx context.Context, row eventOccurrenceRow) (eventOccurrenceDTO, error) {
	credits, err := a.loadOccurrenceCredits(ctx, row.ID)
	if err != nil {
		return eventOccurrenceDTO{}, err
	}
	return eventOccurrenceDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		EventID:           row.EventID,
		PlaceID:           nullableString(row.PlaceID),
		Name:              row.Name,
		Description:       row.Description,
		StartsAt:          row.StartsAt.UTC().Format(time.RFC3339Nano),
		EndsAt:            nullableTimeString(row.EndsAt),
		AllDay:            row.AllDay,
		Timezone:          nullableString(row.Timezone),
		Status:            row.Status,
		PublicURI:         nullableString(row.PublicURI),
		PublicCID:         nullableString(row.PublicCID),
		Credits:           credits,
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

const occurrenceSelectColumns = `
	id, workspace_id, event_id, place_id, name, description, starts_at, ends_at, all_day,
	timezone, status, public_uri, public_cid, created_by_person_id, created_at, updated_at
`

func scanOccurrenceRow(row pgx.Row) (eventOccurrenceRow, error) {
	var occurrence eventOccurrenceRow
	err := row.Scan(&occurrence.ID, &occurrence.WorkspaceID, &occurrence.EventID, &occurrence.PlaceID,
		&occurrence.Name, &occurrence.Description, &occurrence.StartsAt, &occurrence.EndsAt, &occurrence.AllDay,
		&occurrence.Timezone, &occurrence.Status, &occurrence.PublicURI, &occurrence.PublicCID,
		&occurrence.CreatedByPersonID, &occurrence.CreatedAt, &occurrence.UpdatedAt)
	if err != nil {
		return eventOccurrenceRow{}, err
	}
	return occurrence, nil
}

func (a *App) loadOccurrence(ctx context.Context, occurrenceID string) (eventOccurrenceRow, error) {
	if occurrenceID == "" {
		return eventOccurrenceRow{}, pgx.ErrNoRows
	}
	row := a.db.QueryRow(ctx, `select `+occurrenceSelectColumns+` from event_occurrences where id = $1`, occurrenceID)
	return scanOccurrenceRow(row)
}

type createOccurrenceRequest struct {
	PlaceID     *string `json:"placeId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	StartsAt    string  `json:"startsAt"`
	EndsAt      *string `json:"endsAt"`
	AllDay      bool    `json:"allDay"`
	Timezone    *string `json:"timezone"`
	Status      *string `json:"status"`
}

type updateOccurrenceRequest struct {
	ExpectedUpdatedAt *string `json:"expectedUpdatedAt"`
	ExpectedPublicCID *string `json:"expectedPublicCid"`
	PlaceID           *string `json:"placeId"`
	ClearPlace        bool    `json:"clearPlace"`
	Name              *string `json:"name"`
	Description       *string `json:"description"`
	StartsAt          *string `json:"startsAt"`
	EndsAt            *string `json:"endsAt"`
	ClearEndsAt       bool    `json:"clearEndsAt"`
	AllDay            *bool   `json:"allDay"`
	Timezone          *string `json:"timezone"`
	ClearTimezone     bool    `json:"clearTimezone"`
	Status            *string `json:"status"`
}

func validateOccurrenceSchedule(startsAt time.Time, endsAt sql.NullTime, timezone sql.NullString) error {
	if endsAt.Valid && !endsAt.Time.After(startsAt) {
		return errors.New("endsAt must be after startsAt")
	}
	if timezone.Valid {
		if timezone.String == "Local" || len(timezone.String) > 64 {
			return errors.New("timezone must be a valid IANA timezone")
		}
		if _, err := time.LoadLocation(timezone.String); err != nil {
			return errors.New("timezone must be a valid IANA timezone")
		}
	}
	return nil
}

func (a *App) handleListEventOccurrences(w http.ResponseWriter, r *http.Request) {
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

	rows, err := a.db.Query(r.Context(), `select `+occurrenceSelectColumns+` from event_occurrences where event_id = $1 order by starts_at, id`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrences")
		return
	}
	defer rows.Close()

	items := make([]eventOccurrenceDTO, 0)
	for rows.Next() {
		row, err := scanOccurrenceRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load occurrences")
			return
		}
		dto, err := a.occurrenceDTOFromRow(r.Context(), row)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load occurrences")
			return
		}
		items = append(items, dto)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrences")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleCreateEventOccurrence(w http.ResponseWriter, r *http.Request) {
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

	var req createOccurrenceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 1000 {
		writeError(w, http.StatusBadRequest, "name is required and must be 1-1000 characters")
		return
	}
	description := strings.TrimSpace(req.Description)
	if len(description) > 12000 {
		writeError(w, http.StatusBadRequest, "description must be 12000 characters or fewer")
		return
	}
	startsAt, err := parseRFC3339Time(req.StartsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startsAt")
		return
	}
	var endsAt sql.NullTime
	if req.EndsAt != nil && strings.TrimSpace(*req.EndsAt) != "" {
		parsed, err := parseRFC3339Time(*req.EndsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid endsAt")
			return
		}
		endsAt = sql.NullTime{Time: parsed, Valid: true}
	}
	var timezone sql.NullString
	if req.Timezone != nil {
		trimmed := strings.TrimSpace(*req.Timezone)
		if trimmed != "" {
			if len(trimmed) > 64 {
				writeError(w, http.StatusBadRequest, "timezone must be 64 characters or fewer")
				return
			}
			timezone = sql.NullString{String: trimmed, Valid: true}
		}
	}
	if err := validateOccurrenceSchedule(startsAt, endsAt, timezone); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := occurrenceStatusScheduled
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = strings.TrimSpace(*req.Status)
		if !isValidOccurrenceStatus(status) {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
	}

	var placeID sql.NullString
	if req.PlaceID != nil && strings.TrimSpace(*req.PlaceID) != "" {
		place, err := a.loadCulturalPlace(r.Context(), strings.TrimSpace(*req.PlaceID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusBadRequest, "place not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "could not load place")
			return
		}
		if place.WorkspaceID != event.WorkspaceID {
			writeError(w, http.StatusBadRequest, "place belongs to a different workspace")
			return
		}
		placeID = sql.NullString{String: place.ID, Valid: true}
	}

	row := a.db.QueryRow(r.Context(), `
		insert into event_occurrences (workspace_id, event_id, place_id, name, description, starts_at, ends_at,
		                                all_day, timezone, status, created_by_person_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		returning `+occurrenceSelectColumns, event.WorkspaceID, event.ID, placeID, name, description, startsAt, endsAt,
		req.AllDay, timezone, status, actorID)
	created, err := scanOccurrenceRow(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create occurrence")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_occurrence.created", "event_occurrence", created.ID, map[string]any{
		"eventId":     event.ID,
		"workspaceId": event.WorkspaceID,
		"name":        name,
		"startsAt":    startsAt.UTC().Format(time.RFC3339Nano),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	dto, err := a.occurrenceDTOFromRow(r.Context(), created)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// handleUpdateEventOccurrence updates occurrence time/place/status fields
// only. Rescheduling never touches the ticketing/reservation tables (which
// key off the private operator event, not the occurrence), so this handler
// intentionally has no code path that reaches tickets at all.
func (a *App) handleUpdateEventOccurrence(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	occurrence, err := a.loadOccurrence(r.Context(), r.PathValue("occurrenceID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "occurrence not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, occurrence.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req updateOccurrenceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ExpectedUpdatedAt != nil {
		expected, err := parseRFC3339Time(*req.ExpectedUpdatedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expectedUpdatedAt")
			return
		}
		if !expected.Equal(occurrence.UpdatedAt) {
			writeError(w, http.StatusConflict, "occurrence changed; reload before editing")
			return
		}
	}
	if req.ExpectedPublicCID != nil && *req.ExpectedPublicCID != occurrence.PublicCID.String {
		writeError(w, http.StatusConflict, "public record changed; reload before editing")
		return
	}

	name := occurrence.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 1000 {
			writeError(w, http.StatusBadRequest, "name must be 1-1000 characters")
			return
		}
	}
	description := occurrence.Description
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
		if len(description) > 12000 {
			writeError(w, http.StatusBadRequest, "description must be 12000 characters or fewer")
			return
		}
	}
	startsAt := occurrence.StartsAt
	rescheduled := false
	if req.StartsAt != nil {
		parsed, err := parseRFC3339Time(*req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid startsAt")
			return
		}
		if !parsed.Equal(startsAt) {
			rescheduled = true
		}
		startsAt = parsed
	}
	endsAt := occurrence.EndsAt
	if req.ClearEndsAt {
		rescheduled = rescheduled || endsAt.Valid
		endsAt = sql.NullTime{}
	} else if req.EndsAt != nil {
		parsed, err := parseRFC3339Time(*req.EndsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid endsAt")
			return
		}
		if !endsAt.Valid || !parsed.Equal(endsAt.Time) {
			rescheduled = true
		}
		endsAt = sql.NullTime{Time: parsed, Valid: true}
	}
	allDay := occurrence.AllDay
	if req.AllDay != nil {
		allDay = *req.AllDay
	}
	timezone := occurrence.Timezone
	if req.ClearTimezone {
		timezone = sql.NullString{}
	} else if req.Timezone != nil {
		trimmed := strings.TrimSpace(*req.Timezone)
		if len(trimmed) > 64 {
			writeError(w, http.StatusBadRequest, "timezone must be 64 characters or fewer")
			return
		}
		timezone = sql.NullString{String: trimmed, Valid: trimmed != ""}
	}
	if err := validateOccurrenceSchedule(startsAt, endsAt, timezone); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := occurrence.Status
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
		if !isValidOccurrenceStatus(status) {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
	} else if rescheduled && status == occurrenceStatusScheduled {
		status = occurrenceStatusRescheduled
	}

	placeID := occurrence.PlaceID
	if req.ClearPlace {
		placeID = sql.NullString{}
	} else if req.PlaceID != nil && strings.TrimSpace(*req.PlaceID) != "" {
		place, err := a.loadCulturalPlace(r.Context(), strings.TrimSpace(*req.PlaceID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusBadRequest, "place not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "could not load place")
			return
		}
		if place.WorkspaceID != occurrence.WorkspaceID {
			writeError(w, http.StatusBadRequest, "place belongs to a different workspace")
			return
		}
		placeID = sql.NullString{String: place.ID, Valid: true}
	}

	row := a.db.QueryRow(r.Context(), `
		update event_occurrences
		set place_id = $2, name = $3, description = $4, starts_at = $5, ends_at = $6,
		    all_day = $7, timezone = $8, status = $9,
		    updated_at = greatest(clock_timestamp(), updated_at + interval '1 microsecond')
		where id = $1 and updated_at = $10 and public_cid is not distinct from $11
		returning `+occurrenceSelectColumns, occurrence.ID, placeID, name, description, startsAt, endsAt, allDay, timezone, status, occurrence.UpdatedAt, occurrence.PublicCID)
	updated, err := scanOccurrenceRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "occurrence changed; reload before editing")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update occurrence")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_occurrence.updated", "event_occurrence", updated.ID, map[string]any{
		"workspaceId": updated.WorkspaceID,
		"eventId":     updated.EventID,
		"rescheduled": rescheduled,
		"status":      status,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	dto, err := a.occurrenceDTOFromRow(r.Context(), updated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

type attachOccurrenceCreditRequest struct {
	ProfileID string `json:"profileId"`
	Role      string `json:"role"`
	SortOrder int    `json:"sortOrder"`
}

func (a *App) handleAttachOccurrenceCredit(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	occurrence, err := a.loadOccurrence(r.Context(), r.PathValue("occurrenceID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "occurrence not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, occurrence.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req attachOccurrenceCreditRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	profileID := strings.TrimSpace(req.ProfileID)
	if profileID == "" {
		writeError(w, http.StatusBadRequest, "profileId is required")
		return
	}
	profile, err := a.loadCulturalProfile(r.Context(), profileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load profile")
		return
	}
	if profile.WorkspaceID != occurrence.WorkspaceID {
		writeError(w, http.StatusBadRequest, "profile belongs to a different workspace")
		return
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "host"
	}
	if role != "host" && role != "performer" && role != "collective" {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if _, err := a.db.Exec(r.Context(), `
		insert into event_occurrence_profiles (occurrence_id, workspace_id, profile_id, role, sort_order)
		values ($1, $2, $3, $4, $5)
		on conflict (occurrence_id, profile_id) do update set role = excluded.role, sort_order = excluded.sort_order
	`, occurrence.ID, occurrence.WorkspaceID, profile.ID, role, req.SortOrder); err != nil {
		writeError(w, http.StatusInternalServerError, "could not attach credit")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_occurrence.credit_attached", "event_occurrence", occurrence.ID, map[string]any{
		"profileId": profile.ID,
		"role":      role,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}

	updated, err := a.loadOccurrence(r.Context(), occurrence.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	dto, err := a.occurrenceDTOFromRow(r.Context(), updated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (a *App) handleDetachOccurrenceCredit(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	occurrence, err := a.loadOccurrence(r.Context(), r.PathValue("occurrenceID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "occurrence not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, occurrence.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	profileID := r.PathValue("profileID")

	tag, err := a.db.Exec(r.Context(), `
		delete from event_occurrence_profiles where occurrence_id = $1 and profile_id = $2
	`, occurrence.ID, profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not detach credit")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "credit not found")
		return
	}
	if err := a.audit(r.Context(), actorID, "event_occurrence.credit_detached", "event_occurrence", occurrence.ID, map[string]any{
		"profileId": profileID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}

	updated, err := a.loadOccurrence(r.Context(), occurrence.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	dto, err := a.occurrenceDTOFromRow(r.Context(), updated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
