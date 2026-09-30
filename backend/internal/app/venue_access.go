package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type venueAccessPlaceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type venueAccessPlaceIndexDTO struct {
	Places    []venueAccessPlaceDTO `json:"places"`
	NextAfter *string               `json:"nextAfter,omitempty"`
}

func (a *App) venueAccessWorkspace(w http.ResponseWriter, r *http.Request) (string, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return "", false
	}
	id, err := uuid.Parse(r.PathValue("workspaceID"))
	if err != nil {
		writeError(w, 400, "invalid workspace ID")
		return "", false
	}
	if _, _, ok := a.requireWorkspaceRole(r, id.String(), roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return "", false
	}
	return id.String(), true
}
func (a *App) handleListVenueAccessPlaces(w http.ResponseWriter, r *http.Request) {
	workspace, ok := a.venueAccessWorkspace(w, r)
	if !ok {
		return
	}
	var cursor any
	if raw := r.URL.Query().Get("after"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, 400, "invalid venue cursor")
			return
		}
		cursor = id.String()
	}
	rows, err := a.db.Query(r.Context(), `select id,name from cultural_places where workspace_id=$1 and ($2::uuid is null or id>$2) order by id limit 101`, workspace, cursor)
	if err != nil {
		writeError(w, 500, "could not load venue references")
		return
	}
	out := venueAccessPlaceIndexDTO{Places: []venueAccessPlaceDTO{}}
	for rows.Next() {
		var x venueAccessPlaceDTO
		if err = rows.Scan(&x.ID, &x.Name); err != nil {
			rows.Close()
			writeError(w, 500, "could not load venue references")
			return
		}
		out.Places = append(out.Places, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "could not load venue references")
		return
	}
	if len(out.Places) > 100 {
		out.Places = out.Places[:100]
		next := out.Places[99].ID
		out.NextAfter = &next
	}
	if _, _, ok := a.requireWorkspaceRole(r, workspace, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 200, out)
}
func (a *App) handleCreateVenueAccessPlace(w http.ResponseWriter, r *http.Request) {
	workspace, ok := a.venueAccessWorkspace(w, r)
	if !ok {
		return
	}
	var req struct {
		Name       string `json:"name"`
		RequestKey string `json:"requestKey"`
	}
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	key, err := uuid.Parse(req.RequestKey)
	if err != nil || key == uuid.Nil || req.Name == "" || !accessTextValid(req.Name, 600) {
		writeError(w, 400, "a bounded venue name and nonzero requestKey are required")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not create venue reference")
		return
	}
	defer tx.Rollback(r.Context())
	ctx := context.WithValue(r.Context(), txContextKey{}, tx)
	var locked string
	if err = tx.QueryRow(ctx, "select id from workspaces where id=$1 for update", workspace).Scan(&locked); err != nil {
		writeError(w, 409, "workspace changed")
		return
	}
	actor, err := activeOwnerTx(ctx, tx, workspace, r.Context().Value(operatorPersonKey{}))
	if err != nil {
		writeError(w, 403, "forbidden")
		return
	}
	var priorWorkspace, priorActor, priorName, priorPlace string
	err = tx.QueryRow(ctx, `select workspace_id,created_by_person_id,name,place_id from venue_access_place_requests where request_key=$1`, key.String()).Scan(&priorWorkspace, &priorActor, &priorName, &priorPlace)
	if err == nil {
		if priorWorkspace != workspace || priorActor != actor || priorName != req.Name {
			writeError(w, 409, "requestKey already used for a different venue reference")
			return
		}
		var out venueAccessPlaceDTO
		if err = tx.QueryRow(ctx, "select id,name from cultural_places where id=$1 and workspace_id=$2", priorPlace, workspace).Scan(&out.ID, &out.Name); err != nil {
			writeError(w, 500, "could not recover venue reference")
			return
		}
		tx.Rollback(ctx)
		if _, _, ok := a.requireWorkspaceRole(r, workspace, roleOwner); !ok {
			writeError(w, 403, "forbidden")
			return
		}
		writeJSON(w, 200, out)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not create venue reference")
		return
	}
	out := venueAccessPlaceDTO{Name: req.Name}
	if err = tx.QueryRow(ctx, `insert into cultural_places(workspace_id,name,created_by_person_id) values($1,$2,$3) returning id`, workspace, req.Name, actor).Scan(&out.ID); err != nil {
		writeError(w, 500, "could not create venue reference")
		return
	}
	if _, err = tx.Exec(ctx, `insert into cultural_place_protected_details(place_id) values($1)`, out.ID); err != nil {
		writeError(w, 500, "could not create venue reference")
		return
	}
	if _, err = tx.Exec(ctx, `insert into venue_access_place_requests(request_key,workspace_id,created_by_person_id,name,place_id) values($1,$2,$3,$4,$5)`, key.String(), workspace, actor, req.Name, out.ID); err != nil {
		var constraint *pgconn.PgError
		if errors.As(err, &constraint) && constraint.Code == "23505" {
			writeError(w, 409, "requestKey already used")
			return
		}
		writeError(w, 500, "could not create venue reference")
		return
	}
	if err = a.audit(ctx, actor, "place_access.reference_created", "cultural_place", out.ID, map[string]any{"workspaceId": workspace}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(w, 500, "could not create venue reference")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, workspace, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, 201, out)
}
