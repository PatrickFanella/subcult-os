package app

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strings"
	"time"
)

// participantPortalDTO contains only records assigned to the authenticated
// person. An event-role application is deliberately not an identity claim and
// never appears here.
type participantPortalDTO struct {
	Assignments []participantAssignmentDTO `json:"assignments"`
	Commitments []participantCommitmentDTO `json:"commitments"`
}

type participantAssignmentDTO struct {
	EventID                 string  `json:"eventId"`
	EventTitle              string  `json:"eventTitle"`
	StaffingItemID          string  `json:"staffingItemId"`
	Title                   string  `json:"title"`
	Kind                    string  `json:"kind"`
	StartsAt                *string `json:"startsAt,omitempty"`
	EndsAt                  *string `json:"endsAt,omitempty"`
	Status                  string  `json:"status"`
	ParticipantRequirements string  `json:"participantRequirements"`

	// Calendar export fields. Unexported fields are never in the JSON body.
	locationDisplay string
	startsAt        sql.NullTime
	endsAt          sql.NullTime
	revisedAt       time.Time
}

type participantCommitmentDTO struct {
	ID         string  `json:"id"`
	EventID    string  `json:"eventId"`
	EventTitle string  `json:"eventTitle"`
	Title      string  `json:"title"`
	DueAt      *string `json:"dueAt,omitempty"`
	Status     string  `json:"status"`
}

type participantPortalLoad struct {
	portal       participantPortalDTO
	workspaceIDs map[string]struct{}
}

// handleGetParticipantPortal returns a participant's own person-backed
// staffing assignments (organizer-assigned or claimed shift places) and
// commitments. A held place reads as assigned to its holder even while other
// places on the shift remain open. It does not use application names or
// emails as an account identity, and it does not expose operator notes,
// contacts, finance, or payout data.
func (a *App) handleGetParticipantPortal(w http.ResponseWriter, r *http.Request) {
	loaded, ok := a.loadAuthorizedParticipantPortal(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, loaded.portal)
}

// handleGetParticipantShiftCalendar downloads the same assignments as the
// portal as an iCalendar attachment. Untimed assignments are omitted. A file
// already saved or imported by the participant cannot be revoked remotely.
func (a *App) handleGetParticipantShiftCalendar(w http.ResponseWriter, r *http.Request) {
	loaded, ok := a.loadAuthorizedParticipantPortal(w, r)
	if !ok {
		return
	}
	body := participantShiftCalendar(loaded.portal.Assignments)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+participantShiftCalendarFilename+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, body)
}

// loadAuthorizedParticipantPortal sets private/no-store caching, requires an
// authenticated person and rechecks every contributing membership before the
// caller writes schedule bytes. It writes the error response when it fails.
func (a *App) loadAuthorizedParticipantPortal(w http.ResponseWriter, r *http.Request) (participantPortalLoad, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return participantPortalLoad{}, false
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return participantPortalLoad{}, false
	}

	loaded, err := a.loadParticipantPortal(r.Context(), personID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load participant portal")
		return participantPortalLoad{}, false
	}
	// The loader joins active membership. Recheck immediately before emitting
	// private schedule bytes so a revocation during the queries is denied.
	for workspaceID := range loaded.workspaceIDs {
		if _, err := a.activeMembership(r.Context(), personID, workspaceID); err != nil {
			writeError(w, http.StatusForbidden, "forbidden")
			return participantPortalLoad{}, false
		}
	}
	return loaded, true
}

func (a *App) loadParticipantPortal(ctx context.Context, personID string) (participantPortalLoad, error) {
	loaded := participantPortalLoad{
		portal: participantPortalDTO{
			Assignments: []participantAssignmentDTO{},
			Commitments: []participantCommitmentDTO{},
		},
		workspaceIDs: make(map[string]struct{}),
	}
	assignments, err := a.db.Query(ctx, `
		select e.workspace_id, e.id, e.title, esi.id, esi.title, esi.kind,
		       esi.starts_at, esi.ends_at,
		       case when esi.status = 'open' then 'assigned' else esi.status end,
		       esi.participant_requirements, e.location_display,
		       -- A claimed place also revises when it is (re)claimed, so a
		       -- release followed by a new claim advances the calendar entry.
		       greatest(esi.updated_at, e.updated_at, (
				select max(c.created_at) from event_staffing_claims c
				where c.staffing_item_id = esi.id and c.person_id = $1 and c.status = 'active'
		       ))
		from event_staffing_items esi
		join events e on e.id = esi.event_id
		join workspace_members wm on wm.workspace_id = e.workspace_id
		where (esi.assigned_person_id = $1 or exists (
			select 1 from event_staffing_claims c
			where c.staffing_item_id = esi.id and c.person_id = $1 and c.status = 'active'
		  ))
		  and wm.person_id = $1
		  and wm.removed_at is null
		  and wm.revoked_at is null
		  and (wm.expires_at is null or wm.expires_at > now())
		  and esi.status <> 'cancelled'
		order by esi.starts_at nulls last, e.starts_at, esi.created_at, esi.id
	`, personID)
	if err != nil {
		return participantPortalLoad{}, err
	}
	defer assignments.Close()
	for assignments.Next() {
		var workspaceID string
		var assignment participantAssignmentDTO
		var startsAt, endsAt sql.NullTime
		if err := assignments.Scan(&workspaceID, &assignment.EventID, &assignment.EventTitle, &assignment.StaffingItemID,
			&assignment.Title, &assignment.Kind, &startsAt, &endsAt, &assignment.Status, &assignment.ParticipantRequirements,
			&assignment.locationDisplay, &assignment.revisedAt); err != nil {
			return participantPortalLoad{}, err
		}
		assignment.StartsAt = participantPortalTime(startsAt)
		assignment.EndsAt = participantPortalTime(endsAt)
		assignment.startsAt = startsAt
		assignment.endsAt = endsAt
		loaded.workspaceIDs[workspaceID] = struct{}{}
		loaded.portal.Assignments = append(loaded.portal.Assignments, assignment)
	}
	if err := assignments.Err(); err != nil {
		return participantPortalLoad{}, err
	}

	commitments, err := a.db.Query(ctx, `
		select e.workspace_id, c.id, e.id, e.title, c.title, c.due_at, c.status
		from commitments c
		join events e on e.id = c.event_id and e.workspace_id = c.workspace_id
		join workspace_members wm on wm.workspace_id = e.workspace_id
		where c.owner_person_id = $1
		  and wm.person_id = $1
		  and wm.removed_at is null
		  and wm.revoked_at is null
		  and (wm.expires_at is null or wm.expires_at > now())
		  and exists (
			select 1
			from event_staffing_items esi
			where esi.event_id = e.id
			  and (esi.assigned_person_id = $1 or exists (
				select 1 from event_staffing_claims c
				where c.staffing_item_id = esi.id and c.person_id = $1 and c.status = 'active'
			  ))
			  and esi.status <> 'cancelled'
		  )
		order by c.due_at nulls last, c.created_at, c.id
	`, personID)
	if err != nil {
		return participantPortalLoad{}, err
	}
	defer commitments.Close()
	for commitments.Next() {
		var workspaceID string
		var commitment participantCommitmentDTO
		var dueAt sql.NullTime
		if err := commitments.Scan(&workspaceID, &commitment.ID, &commitment.EventID, &commitment.EventTitle,
			&commitment.Title, &dueAt, &commitment.Status); err != nil {
			return participantPortalLoad{}, err
		}
		commitment.DueAt = participantPortalTime(dueAt)
		loaded.workspaceIDs[workspaceID] = struct{}{}
		loaded.portal.Commitments = append(loaded.portal.Commitments, commitment)
	}
	if err := commitments.Err(); err != nil {
		return participantPortalLoad{}, err
	}
	return loaded, nil
}

func participantPortalTime(value sql.NullTime) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.UTC().Format(time.RFC3339Nano)
	return &formatted
}

const (
	participantShiftCalendarFilename = "subcult-os-shifts.ics"
	participantShiftCalendarProdID   = "-//Subcult//Subcult OS participant shifts//EN"
	// participantShiftUIDDomain is fixed so UIDs stay stable across hosts and
	// configuration changes.
	participantShiftUIDDomain = "os.subcult.tv"
)

// participantShiftCalendar converts portal assignments into calendar events.
// The UID is derived from the staffing item, so a later download with a new
// time updates the same calendar entry. It includes only fields the portal
// already shows, plus the event's public location display.
func participantShiftCalendar(assignments []participantAssignmentDTO) string {
	events := make([]icalEvent, 0, len(assignments))
	for _, assignment := range assignments {
		if !assignment.startsAt.Valid {
			continue
		}
		event := icalEvent{
			UID:      "staffing-" + assignment.StaffingItemID + "@" + participantShiftUIDDomain,
			Revised:  assignment.revisedAt,
			Start:    assignment.startsAt.Time,
			Summary:  assignment.Title + " · " + assignment.EventTitle,
			Location: assignment.locationDisplay,
		}
		if assignment.endsAt.Valid {
			end := assignment.endsAt.Time
			event.End = &end
		}
		description := []string{"Event: " + assignment.EventTitle, "Assignment: " + assignment.Kind}
		if requirements := strings.TrimSpace(assignment.ParticipantRequirements); requirements != "" {
			description = append(description, "", "Requirements:", requirements)
		}
		event.Description = strings.Join(description, "\n")
		events = append(events, event)
	}
	return writeICalendar(participantShiftCalendarProdID, events)
}
