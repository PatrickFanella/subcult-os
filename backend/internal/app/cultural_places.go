package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// culturalPlaceDTO carries only the fields the operator API needs to manage
// a place. protectedDetail is included here because this DTO is only ever
// returned to an authorized workspace member over the operator API; the
// separate public projection (cultural_public_projection.go) never reads
// the protected table at all, so a public payload can never carry these
// fields by construction rather than by an easy-to-miss field-level check.
type culturalPlaceDTO struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspaceId"`
	Name              string  `json:"name"`
	Locality          string  `json:"locality"`
	Region            string  `json:"region"`
	Country           *string `json:"country,omitempty"`
	CoordinatesPublic bool    `json:"coordinatesPublic"`
	PublicLatitude    *string `json:"publicLatitude,omitempty"`
	PublicLongitude   *string `json:"publicLongitude,omitempty"`
	PublicURI         *string `json:"publicUri,omitempty"`
	PublicCID         *string `json:"publicCid,omitempty"`
	StreetAddress     string  `json:"streetAddress"`
	AccessNotes       string  `json:"accessNotes"`
	CreatedByPersonID string  `json:"createdByPersonId"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type culturalPlaceRow struct {
	ID                string
	WorkspaceID       string
	Name              string
	Locality          string
	Region            string
	Country           sql.NullString
	CoordinatesPublic bool
	PublicLatitude    sql.NullString
	PublicLongitude   sql.NullString
	PublicURI         sql.NullString
	PublicCID         sql.NullString
	StreetAddress     string
	AccessNotes       string
	CreatedByPersonID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func culturalPlaceDTOFromRow(row culturalPlaceRow) culturalPlaceDTO {
	return culturalPlaceDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Name:              row.Name,
		Locality:          row.Locality,
		Region:            row.Region,
		Country:           nullableString(row.Country),
		CoordinatesPublic: row.CoordinatesPublic,
		PublicLatitude:    nullableString(row.PublicLatitude),
		PublicLongitude:   nullableString(row.PublicLongitude),
		PublicURI:         nullableString(row.PublicURI),
		PublicCID:         nullableString(row.PublicCID),
		StreetAddress:     row.StreetAddress,
		AccessNotes:       row.AccessNotes,
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type createCulturalPlaceRequest struct {
	Name              string  `json:"name"`
	Locality          string  `json:"locality"`
	Region            string  `json:"region"`
	Country           *string `json:"country"`
	CoordinatesPublic bool    `json:"coordinatesPublic"`
	PublicLatitude    *string `json:"publicLatitude"`
	PublicLongitude   *string `json:"publicLongitude"`
	StreetAddress     string  `json:"streetAddress"`
	AccessNotes       string  `json:"accessNotes"`
}

type updateCulturalPlaceRequest struct {
	Name              *string `json:"name"`
	Locality          *string `json:"locality"`
	Region            *string `json:"region"`
	Country           *string `json:"country"`
	ClearCountry      bool    `json:"clearCountry"`
	CoordinatesPublic *bool   `json:"coordinatesPublic"`
	PublicLatitude    *string `json:"publicLatitude"`
	PublicLongitude   *string `json:"publicLongitude"`
	StreetAddress     *string `json:"streetAddress"`
	AccessNotes       *string `json:"accessNotes"`
}

func validatePlaceCoordinates(coordinatesPublic bool, lat, lon *string) (sql.NullString, sql.NullString, error) {
	if !coordinatesPublic {
		return sql.NullString{}, sql.NullString{}, nil
	}
	if lat == nil || lon == nil || strings.TrimSpace(*lat) == "" || strings.TrimSpace(*lon) == "" {
		return sql.NullString{}, sql.NullString{}, errors.New("publicLatitude and publicLongitude are required when coordinatesPublic is true")
	}
	latitude := strings.TrimSpace(*lat)
	longitude := strings.TrimSpace(*lon)
	if len(latitude) > 20 || len(longitude) > 20 {
		return sql.NullString{}, sql.NullString{}, errors.New("publicLatitude and publicLongitude must be 20 characters or fewer")
	}
	return sql.NullString{String: latitude, Valid: true}, sql.NullString{String: longitude, Valid: true}, nil
}

func (a *App) loadCulturalPlace(ctx context.Context, placeID string) (culturalPlaceRow, error) {
	var row culturalPlaceRow
	if placeID == "" {
		return row, pgx.ErrNoRows
	}
	err := a.db.QueryRow(ctx, `
		select p.id, p.workspace_id, p.name, p.locality, p.region, p.country, p.coordinates_public,
		       p.public_latitude, p.public_longitude, p.public_uri, p.public_cid,
		       coalesce(d.street_address, ''), coalesce(d.access_notes, ''),
		       p.created_by_person_id, p.created_at, p.updated_at
		from cultural_places p
		left join cultural_place_protected_details d on d.place_id = p.id
		where p.id = $1
	`, placeID).Scan(&row.ID, &row.WorkspaceID, &row.Name, &row.Locality, &row.Region, &row.Country,
		&row.CoordinatesPublic, &row.PublicLatitude, &row.PublicLongitude, &row.PublicURI, &row.PublicCID,
		&row.StreetAddress, &row.AccessNotes, &row.CreatedByPersonID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return culturalPlaceRow{}, err
	}
	return row, nil
}

func (a *App) handleListCulturalPlaces(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	if _, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		select p.id, p.workspace_id, p.name, p.locality, p.region, p.country, p.coordinates_public,
		       p.public_latitude, p.public_longitude, p.public_uri, p.public_cid,
		       coalesce(d.street_address, ''), coalesce(d.access_notes, ''),
		       p.created_by_person_id, p.created_at, p.updated_at
		from cultural_places p
		left join cultural_place_protected_details d on d.place_id = p.id
		where p.workspace_id = $1
		order by p.created_at desc, p.id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load places")
		return
	}
	defer rows.Close()

	items := make([]culturalPlaceDTO, 0)
	for rows.Next() {
		var row culturalPlaceRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.Name, &row.Locality, &row.Region, &row.Country,
			&row.CoordinatesPublic, &row.PublicLatitude, &row.PublicLongitude, &row.PublicURI, &row.PublicCID,
			&row.StreetAddress, &row.AccessNotes, &row.CreatedByPersonID, &row.CreatedAt, &row.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load places")
			return
		}
		items = append(items, culturalPlaceDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load places")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleCreateCulturalPlace(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createCulturalPlaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 600 {
		writeError(w, http.StatusBadRequest, "name is required and must be 1-600 characters")
		return
	}
	locality := strings.TrimSpace(req.Locality)
	region := strings.TrimSpace(req.Region)
	if len(locality) > 400 || len(region) > 400 {
		writeError(w, http.StatusBadRequest, "locality and region must be 400 characters or fewer")
		return
	}
	var country sql.NullString
	if req.Country != nil {
		trimmed := strings.TrimSpace(*req.Country)
		if trimmed != "" {
			if len(trimmed) != 2 {
				writeError(w, http.StatusBadRequest, "country must be a 2-character ISO 3166-1 alpha-2 code")
				return
			}
			country = sql.NullString{String: strings.ToUpper(trimmed), Valid: true}
		}
	}
	latitude, longitude, err := validatePlaceCoordinates(req.CoordinatesPublic, req.PublicLatitude, req.PublicLongitude)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	streetAddress := strings.TrimSpace(req.StreetAddress)
	accessNotes := strings.TrimSpace(req.AccessNotes)

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var placeID string
	if err := tx.QueryRow(r.Context(), `
		insert into cultural_places (workspace_id, name, locality, region, country, coordinates_public,
		                              public_latitude, public_longitude, created_by_person_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning id
	`, workspaceID, name, locality, region, country, req.CoordinatesPublic, latitude, longitude, actorID).Scan(&placeID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create place")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		insert into cultural_place_protected_details (place_id, street_address, access_notes)
		values ($1, $2, $3)
	`, placeID, streetAddress, accessNotes); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create place")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "cultural_place.created", "cultural_place", placeID, map[string]any{
		"workspaceId":       workspaceID,
		"name":              name,
		"coordinatesPublic": req.CoordinatesPublic,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save place")
		return
	}

	place, err := a.loadCulturalPlace(r.Context(), placeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load place")
		return
	}
	writeJSON(w, http.StatusOK, culturalPlaceDTOFromRow(place))
}

func (a *App) handleUpdateCulturalPlace(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	place, err := a.loadCulturalPlace(r.Context(), r.PathValue("placeID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "place not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load place")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, place.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req updateCulturalPlaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	name := place.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 600 {
			writeError(w, http.StatusBadRequest, "name must be 1-600 characters")
			return
		}
	}
	locality := place.Locality
	if req.Locality != nil {
		locality = strings.TrimSpace(*req.Locality)
	}
	region := place.Region
	if req.Region != nil {
		region = strings.TrimSpace(*req.Region)
	}
	if len(locality) > 400 || len(region) > 400 {
		writeError(w, http.StatusBadRequest, "locality and region must be 400 characters or fewer")
		return
	}
	country := place.Country
	if req.ClearCountry {
		country = sql.NullString{}
	} else if req.Country != nil {
		trimmed := strings.TrimSpace(*req.Country)
		if len(trimmed) != 2 {
			writeError(w, http.StatusBadRequest, "country must be a 2-character ISO 3166-1 alpha-2 code")
			return
		}
		country = sql.NullString{String: strings.ToUpper(trimmed), Valid: true}
	}
	coordinatesPublic := place.CoordinatesPublic
	if req.CoordinatesPublic != nil {
		coordinatesPublic = *req.CoordinatesPublic
	}
	latPtr, lonPtr := req.PublicLatitude, req.PublicLongitude
	if latPtr == nil && place.PublicLatitude.Valid {
		latPtr = &place.PublicLatitude.String
	}
	if lonPtr == nil && place.PublicLongitude.Valid {
		lonPtr = &place.PublicLongitude.String
	}
	latitude, longitude, err := validatePlaceCoordinates(coordinatesPublic, latPtr, lonPtr)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	streetAddress := place.StreetAddress
	if req.StreetAddress != nil {
		streetAddress = strings.TrimSpace(*req.StreetAddress)
	}
	accessNotes := place.AccessNotes
	if req.AccessNotes != nil {
		accessNotes = strings.TrimSpace(*req.AccessNotes)
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if _, err := tx.Exec(r.Context(), `
		update cultural_places
		set name = $2, locality = $3, region = $4, country = $5, coordinates_public = $6,
		    public_latitude = $7, public_longitude = $8, updated_at = now()
		where id = $1
	`, place.ID, name, locality, region, country, coordinatesPublic, latitude, longitude); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update place")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update cultural_place_protected_details
		set street_address = $2, access_notes = $3, updated_at = now()
		where place_id = $1
	`, place.ID, streetAddress, accessNotes); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update place")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "cultural_place.updated", "cultural_place", place.ID, map[string]any{
		"workspaceId":       place.WorkspaceID,
		"name":              name,
		"coordinatesPublic": coordinatesPublic,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save place")
		return
	}

	updated, err := a.loadCulturalPlace(r.Context(), place.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load place")
		return
	}
	writeJSON(w, http.StatusOK, culturalPlaceDTOFromRow(updated))
}
