package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type accessComparisonOccurrenceDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	StartsAt  string  `json:"startsAt"`
	Status    string  `json:"status"`
	UpdatedAt string  `json:"updatedAt"`
	PlaceID   *string `json:"placeId,omitempty"`
}
type occurrenceAccessComparisonDTO struct {
	EventID     string                        `json:"eventId"`
	WorkspaceID string                        `json:"workspaceId"`
	EvaluatedAt string                        `json:"evaluatedAt"`
	Occurrence  accessComparisonOccurrenceDTO `json:"occurrence"`
	Event       eventAccessWorksheetDTO       `json:"event"`
	Venue       *eventAccessWorksheetDTO      `json:"venue"`
}

func (a *App) handleGetOccurrenceAccessComparison(w http.ResponseWriter, r *http.Request) {
	resource, ok := a.accessResourceOwner(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("occurrenceID"))
	if err != nil {
		writeError(w, 400, "invalid occurrence ID")
		return
	}
	// All information comes from one snapshot, including the occurrence's place.
	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		writeError(w, 500, "could not load access comparison")
		return
	}
	defer tx.Rollback(r.Context())
	out := occurrenceAccessComparisonDTO{EventID: resource.ID, WorkspaceID: resource.WorkspaceID}
	var starts, updated, now time.Time
	err = tx.QueryRow(r.Context(), `select id,name,starts_at,status,updated_at,place_id from event_occurrences where id=$1 and event_id=$2 and workspace_id=$3`, id.String(), resource.ID, resource.WorkspaceID).Scan(&out.Occurrence.ID, &out.Occurrence.Name, &starts, &out.Occurrence.Status, &updated, &out.Occurrence.PlaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "occurrence not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load access comparison")
		return
	}
	out.Occurrence.StartsAt = starts.UTC().Format(time.RFC3339Nano)
	out.Occurrence.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
	if err = tx.QueryRow(r.Context(), "select clock_timestamp()").Scan(&now); err != nil {
		writeError(w, 500, "could not load access comparison")
		return
	}
	out.EvaluatedAt = now.UTC().Format(time.RFC3339Nano)
	out.Event, err = readAccessWorksheet(r.Context(), tx, resource, now)
	if err != nil {
		writeError(w, 500, "could not load access comparison")
		return
	}
	if out.Occurrence.PlaceID != nil {
		venue := accessResource{ID: *out.Occurrence.PlaceID, WorkspaceID: resource.WorkspaceID, Scope: "venue", Column: "place_id"}
		if err = tx.QueryRow(r.Context(), "select name from cultural_places where id=$1 and workspace_id=$2", venue.ID, venue.WorkspaceID).Scan(&venue.Name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, 409, "linked venue unavailable; refresh the occurrence")
			} else {
				writeError(w, 500, "could not load access comparison")
			}
			return
		}
		worksheet, readErr := readAccessWorksheet(r.Context(), tx, venue, now)
		if readErr != nil {
			writeError(w, 500, "could not load access comparison")
			return
		}
		out.Venue = &worksheet
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not load access comparison")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, resource.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) handleListAccessComparisonOccurrences(w http.ResponseWriter, r *http.Request) {
	resource, ok := a.accessResourceOwner(w, r)
	if !ok {
		return
	}
	rows, err := a.db.Query(r.Context(), `select id,name,starts_at,status,updated_at,place_id from event_occurrences where event_id=$1 and workspace_id=$2 order by starts_at,id`, resource.ID, resource.WorkspaceID)
	if err != nil {
		writeError(w, 500, "could not load comparison occurrences")
		return
	}
	out := []accessComparisonOccurrenceDTO{}
	for rows.Next() {
		var x accessComparisonOccurrenceDTO
		var starts, updated time.Time
		if err = rows.Scan(&x.ID, &x.Name, &starts, &x.Status, &updated, &x.PlaceID); err != nil {
			rows.Close()
			writeError(w, 500, "could not load comparison occurrences")
			return
		}
		x.StartsAt = starts.UTC().Format(time.RFC3339Nano)
		x.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
		out = append(out, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "could not load comparison occurrences")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, resource.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}
