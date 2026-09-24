// Package app: anonymous cultural discovery over the AT record projection
// (UX-01 / Issue #18). See docs/development/discovery-ux.md.
//
// This file never reads cultural_* (the operator's own private write path)
// and never reads any private table (contacts, staffing, tickets,
// cultural_place_protected_details). It only reads at_projection_records
// (an untrusted external mirror; see docs/development/projection.md) and,
// for reservation handoff resolution only, the public-safe columns already
// exposed by the existing anonymous /api/public/events* routes
// (status/public_slug) through event_public_links (an operator-private
// table read here only to resolve a mapping, never to expose its own rows).
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	discoveryOccurrenceCollection = "tv.subcult.event.occurrence"
	discoveryPlaceCollection      = "tv.subcult.place"

	discoveryDefaultLimit = 20
	discoveryMaxLimit     = 100
)

// discoverySourceDTO identifies the projection record backing a discovery
// item: the authority DID, the at:// URI, and a handle when the projection
// happens to carry one. The current at_projection_records schema stores no
// handle column (see migration 000011), so Handle is always nil today; the
// field is kept so a later projection enrichment does not need a contract
// change.
type discoverySourceDTO struct {
	DID    string  `json:"did"`
	URI    string  `json:"uri"`
	Handle *string `json:"handle,omitempty"`
}

// discoveryLocationDTO carries only the coarse public location fields the
// admitted tv.subcult.place Lexicon can express: no street address, no
// access notes, no protected-table field can appear here because this type
// is built only from the projected place record's own JSON, never from
// cultural_places/cultural_place_protected_details.
type discoveryLocationDTO struct {
	Name      string  `json:"name"`
	Locality  string  `json:"locality,omitempty"`
	Region    string  `json:"region,omitempty"`
	Country   string  `json:"country,omitempty"`
	Latitude  *string `json:"latitude,omitempty"`
	Longitude *string `json:"longitude,omitempty"`
}

// discoveryHandoffDTO is the reservation/ticket destination resolved for one
// occurrence. kind "none" always carries a reason and never falls back to
// routing toward a different event.
type discoveryHandoffDTO struct {
	Kind            string  `json:"kind"` // "local" | "none"
	Reason          string  `json:"reason,omitempty"`
	EventSlug       *string `json:"eventSlug,omitempty"`
	ReservationPath *string `json:"reservationPath,omitempty"`
}

// discoveryOccurrenceDTO is the public, anonymous discovery record for one
// projected tv.subcult.event.occurrence.
type discoveryOccurrenceDTO struct {
	URI              string                `json:"uri"`
	Source           discoverySourceDTO    `json:"source"`
	Name             string                `json:"name"`
	Description      string                `json:"description,omitempty"`
	StartsAt         string                `json:"startsAt"`
	EndsAt           string                `json:"endsAt,omitempty"`
	Timezone         string                `json:"timezone,omitempty"`
	Status           string                `json:"status"`           // the record's own lifecycle status
	ProjectionStatus string                `json:"projectionStatus"` // active | deleted | unavailable
	Location         *discoveryLocationDTO `json:"location,omitempty"`
	Handoff          discoveryHandoffDTO   `json:"handoff"`
}

// projectionStrongRef mirrors tv.subcult.event.occurrence#strongRef.
type projectionStrongRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// projectionOccurrenceRecord parses the JSON stored in
// at_projection_records.record for a tv.subcult.event.occurrence row. This
// is the *projected* record shape (independently authored elsewhere), not
// the operator's own cultural_* row; a field this struct doesn't declare is
// silently dropped by encoding/json, so an adversarial or unexpected extra
// field in the raw JSON can never reach discoveryOccurrenceDTO.
type projectionOccurrenceRecord struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Profile     *projectionStrongRef `json:"profile"`
	Place       *projectionStrongRef `json:"place"`
	StartsAt    string               `json:"startsAt"`
	EndsAt      string               `json:"endsAt"`
	AllDay      bool                 `json:"allDay"`
	Timezone    string               `json:"timezone"`
	Status      string               `json:"status"`
	CreatedAt   string               `json:"createdAt"`
}

// projectionPlaceRecord parses the JSON stored in at_projection_records for
// a tv.subcult.place row. Only the fields the admitted Lexicon declares
// exist here (see contracts/lexicons/tv.subcult.place.json); there is no
// street address or access-notes field in this schema at all.
type projectionPlaceRecord struct {
	Name        string `json:"name"`
	Locality    string `json:"locality"`
	Region      string `json:"region"`
	Country     string `json:"country"`
	Coordinates *struct {
		Latitude  string `json:"latitude"`
		Longitude string `json:"longitude"`
	} `json:"coordinates"`
}

type projectionRecordRow struct {
	URI    string
	DID    string
	Status string
	Record []byte // nil for a tombstone
}

func scanProjectionRecordRow(row pgx.Row) (projectionRecordRow, error) {
	var out projectionRecordRow
	var record sql.NullString // jsonb scanned as text for simplicity
	if err := row.Scan(&out.URI, &out.DID, &out.Status, &record); err != nil {
		return projectionRecordRow{}, err
	}
	if record.Valid {
		out.Record = []byte(record.String)
	}
	return out, nil
}

func (a *App) loadProjectionRecord(ctx context.Context, uri string) (projectionRecordRow, error) {
	return scanProjectionRecordRow(a.db.QueryRow(ctx, `
		select uri, did, status, record::text from at_projection_records where uri = $1
	`, uri))
}

// buildDiscoveryOccurrenceDTO assembles the public discovery item for one
// projection row. It never returns an error for a malformed/missing nested
// place: a place that cannot be resolved is simply omitted, since the
// occurrence itself is still a valid, presentable discovery item.
func (a *App) buildDiscoveryOccurrenceDTO(ctx context.Context, row projectionRecordRow) (discoveryOccurrenceDTO, bool) {
	dto := discoveryOccurrenceDTO{
		URI:              row.URI,
		Source:           discoverySourceDTO{DID: row.DID, URI: row.URI},
		ProjectionStatus: row.Status,
	}

	if len(row.Record) == 0 {
		// Tombstone (deleted before any create was seen, or a delete with
		// no retained body): still identifiable, still returned on the
		// detail route with its projection status, never on the list.
		dto.Status = "unknown"
		dto.Handoff = a.resolveDiscoveryHandoff(ctx, row.URI)
		return dto, true
	}

	var record projectionOccurrenceRecord
	if err := json.Unmarshal(row.Record, &record); err != nil {
		return discoveryOccurrenceDTO{}, false
	}

	dto.Name = record.Name
	dto.Description = record.Description
	dto.StartsAt = record.StartsAt
	dto.EndsAt = record.EndsAt
	dto.Timezone = record.Timezone
	dto.Status = record.Status
	if dto.Status == "" {
		dto.Status = "scheduled"
	}

	if record.Place != nil && record.Place.URI != "" {
		if placeRow, err := a.loadProjectionRecord(ctx, record.Place.URI); err == nil && placeRow.Status == "active" && len(placeRow.Record) > 0 {
			var place projectionPlaceRecord
			if err := json.Unmarshal(placeRow.Record, &place); err == nil {
				location := &discoveryLocationDTO{
					Name:     place.Name,
					Locality: place.Locality,
					Region:   place.Region,
					Country:  place.Country,
				}
				if place.Coordinates != nil {
					lat := place.Coordinates.Latitude
					lon := place.Coordinates.Longitude
					location.Latitude = &lat
					location.Longitude = &lon
				}
				dto.Location = location
			}
		}
	}

	dto.Handoff = a.resolveDiscoveryHandoff(ctx, row.URI)
	return dto, true
}

// resolveDiscoveryHandoff resolves exactly one occurrence URI to a
// reservation destination. A missing, stale (not fresh/changed), invalid,
// unavailable or deleted mapping always returns kind "none" with a reason;
// it never falls back to a different event_public_links row or a different
// event.
func (a *App) resolveDiscoveryHandoff(ctx context.Context, occurrenceURI string) discoveryHandoffDTO {
	var eventID, status string
	err := a.db.QueryRow(ctx, `
		select event_id, status from event_public_links
		where public_uri = $1
		order by observed_at desc
		limit 1
	`, occurrenceURI).Scan(&eventID, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return discoveryHandoffDTO{Kind: "none", Reason: "no_mapping"}
		}
		return discoveryHandoffDTO{Kind: "none", Reason: "mapping_lookup_failed"}
	}
	if status != publicLinkStatusFresh && status != publicLinkStatusChanged {
		return discoveryHandoffDTO{Kind: "none", Reason: "mapping_" + status}
	}

	var eventStatus string
	var publicSlug sql.NullString
	err = a.db.QueryRow(ctx, `select status, public_slug from events where id = $1`, eventID).Scan(&eventStatus, &publicSlug)
	if err != nil {
		return discoveryHandoffDTO{Kind: "none", Reason: "event_not_found"}
	}
	if eventStatus != "published" || !publicSlug.Valid || publicSlug.String == "" {
		return discoveryHandoffDTO{Kind: "none", Reason: "event_not_published"}
	}

	slug := publicSlug.String
	reservationPath := "/api/public/events/" + slug + "/reservations"
	return discoveryHandoffDTO{Kind: "local", EventSlug: &slug, ReservationPath: &reservationPath}
}

func discoveryPaginationParams(r *http.Request) (limit, offset int) {
	limit = discoveryDefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > discoveryMaxLimit {
		limit = discoveryMaxLimit
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}

// handleListPublicDiscoveryOccurrences lists active (not deleted/
// unavailable) projected occurrences, newest-updated first, with optional
// locality filtering and pagination. Guest access: no session required.
func (a *App) handleListPublicDiscoveryOccurrences(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeJSON(w, http.StatusOK, []discoveryOccurrenceDTO{})
		return
	}

	limit, offset := discoveryPaginationParams(r)
	locality := strings.TrimSpace(r.URL.Query().Get("locality"))

	rows, err := a.db.Query(r.Context(), `
		select o.uri, o.did, o.status, o.record::text
		from at_projection_records o
		where o.collection = $1 and o.status = 'active'
		  and (
		    $2 = ''
		    or exists (
		      select 1 from at_projection_records p
		      where p.collection = $3
		        and p.status = 'active'
		        and p.uri = (o.record -> 'place' ->> 'uri')
		        and lower(coalesce(p.record ->> 'locality', '')) = lower($2)
		    )
		    or lower(coalesce(o.record ->> 'name', '')) like '%' || lower($2) || '%'
		  )
		order by o.updated_at desc
		limit $4 offset $5
	`, discoveryOccurrenceCollection, locality, discoveryPlaceCollection, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list discovery occurrences")
		return
	}
	defer rows.Close()

	items := make([]discoveryOccurrenceDTO, 0)
	for rows.Next() {
		row, err := scanProjectionRecordRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read discovery occurrence")
			return
		}
		dto, ok := a.buildDiscoveryOccurrenceDTO(r.Context(), row)
		if !ok {
			continue
		}
		items = append(items, dto)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not list discovery occurrences")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// handleGetPublicDiscoveryOccurrence returns one occurrence by its at://
// URI. The route captures everything after the "occurrences/" segment as a
// wildcard tail (net/http's {name...} pattern keeps embedded slashes
// literal), and the client is expected to send the URI with its "at://"
// scheme stripped (e.g. "did:plc:xyz/tv.subcult.event.occurrence/abc"); this
// avoids needing to percent-encode the URI's embedded slashes. A record
// whose projection status is deleted or unavailable is still returned here
// (with that status flagged), unlike the list route, which excludes them.
func (a *App) handleGetPublicDiscoveryOccurrence(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusNotFound, "occurrence not found")
		return
	}
	tail := r.PathValue("uri")
	if tail == "" {
		writeError(w, http.StatusBadRequest, "missing occurrence uri")
		return
	}
	uri := tail
	if !strings.HasPrefix(uri, "at://") {
		uri = "at://" + uri
	}

	row, err := a.loadProjectionRecord(r.Context(), uri)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "occurrence not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load occurrence")
		return
	}
	if row.URI == "" {
		writeError(w, http.StatusNotFound, "occurrence not found")
		return
	}

	dto, ok := a.buildDiscoveryOccurrenceDTO(r.Context(), row)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "occurrence record could not be read")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
