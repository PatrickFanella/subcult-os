package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5"
)

// publicPreviewPlaceholderURI and publicPreviewPlaceholderCID stand in for
// the AT-URI/CID a record will carry once PUB-01 actually publishes it.
// tv.subcult.event.occurrence's `profile`/`place` fields are required
// strong references at the wire-format level (see
// contracts/lexicons/tv.subcult.event.occurrence.json), but this slice
// writes nothing to a PDS and event_occurrences.public_uri/public_cid are
// still null for every row. The syntax below is a real, validator-passing
// at-uri/cid (reused from contracts/atproto-lexicon.fixtures.json) used only
// so /public-preview can prove the projection is schema-shaped; it is never
// treated as a live reference to a published record, and PublicURI/PublicCID
// on the returned DTOs stay nil unless the row itself has been published.
const (
	publicPreviewPlaceholderURI = "at://did:plc:vwzwgnygau7ed7b7wt5ux7y2/tv.subcult.profile/self"
	publicPreviewPlaceholderCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
)

var errNoPublicProfile = errors.New("occurrence has no credited profile to publish")

type strongRefRecord struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type publicGeoRecord struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// publicPlaceRecord mirrors tv.subcult.place exactly. Fields not declared
// here (street address, access notes, any internal identifier) cannot leak
// through this type by construction, and building it only from
// culturalPlaceRow's public columns means the protected columns never even
// reach this function.
type publicPlaceRecord struct {
	Type        string           `json:"$type"`
	Name        string           `json:"name"`
	Locality    string           `json:"locality,omitempty"`
	Region      string           `json:"region,omitempty"`
	Country     string           `json:"country,omitempty"`
	Coordinates *publicGeoRecord `json:"coordinates,omitempty"`
	CreatedAt   string           `json:"createdAt"`
}

// publicProfileRecord mirrors tv.subcult.profile exactly.
type publicProfileRecord struct {
	Type        string `json:"$type"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// publicOccurrenceRecord mirrors tv.subcult.event.occurrence exactly.
type publicOccurrenceRecord struct {
	Type        string           `json:"$type"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Profile     *strongRefRecord `json:"profile"`
	Place       *strongRefRecord `json:"place,omitempty"`
	StartsAt    string           `json:"startsAt"`
	EndsAt      string           `json:"endsAt,omitempty"`
	AllDay      bool             `json:"allDay,omitempty"`
	Timezone    string           `json:"timezone,omitempty"`
	Status      string           `json:"status,omitempty"`
	CreatedAt   string           `json:"createdAt"`
}

func buildPublicProfileRecord(row culturalProfileRow) publicProfileRecord {
	return publicProfileRecord{
		Type:        "tv.subcult.profile",
		DisplayName: row.DisplayName,
		Description: row.Description,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// buildPublicPlaceRecord only ever reads the public-column subset of
// culturalPlaceRow (name/locality/region/country/coordinates); it never
// receives or reads StreetAddress/AccessNotes, so there is no field to
// accidentally forget to strip.
func buildPublicPlaceRecord(row culturalPlaceRow) publicPlaceRecord {
	record := publicPlaceRecord{
		Type:      "tv.subcult.place",
		Name:      row.Name,
		Locality:  row.Locality,
		Region:    row.Region,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.Country.Valid {
		record.Country = row.Country.String
	}
	if row.CoordinatesPublic && row.PublicLatitude.Valid && row.PublicLongitude.Valid {
		record.Coordinates = &publicGeoRecord{
			Latitude:  row.PublicLatitude.String,
			Longitude: row.PublicLongitude.String,
		}
	}
	return record
}

func strongRefFor(publicURI, publicCID string) *strongRefRecord {
	uri := publicURI
	cid := publicCID
	if uri == "" {
		uri = publicPreviewPlaceholderURI
	}
	if cid == "" {
		cid = publicPreviewPlaceholderCID
	}
	return &strongRefRecord{URI: uri, CID: cid}
}

func buildPublicOccurrenceRecord(occurrence eventOccurrenceRow, profile *strongRefRecord, place *strongRefRecord) publicOccurrenceRecord {
	record := publicOccurrenceRecord{
		Type:      "tv.subcult.event.occurrence",
		Name:      occurrence.Name,
		Profile:   profile,
		Place:     place,
		StartsAt:  occurrence.StartsAt.UTC().Format(time.RFC3339),
		AllDay:    occurrence.AllDay,
		Status:    occurrence.Status,
		CreatedAt: occurrence.CreatedAt.UTC().Format(time.RFC3339),
	}
	if occurrence.Description != "" {
		record.Description = occurrence.Description
	}
	if occurrence.EndsAt.Valid {
		record.EndsAt = occurrence.EndsAt.Time.UTC().Format(time.RFC3339)
	}
	if occurrence.Timezone.Valid {
		record.Timezone = occurrence.Timezone.String
	}
	return record
}

type publicOccurrencePreviewDTO struct {
	Occurrence publicOccurrenceRecord `json:"occurrence"`
	Profile    publicProfileRecord    `json:"profile"`
	Place      *publicPlaceRecord     `json:"place,omitempty"`
}

// buildOccurrencePublicPreview assembles and validates the exact
// tv.subcult.event.occurrence / tv.subcult.profile / tv.subcult.place
// record shapes for one occurrence, running each through
// atproto.ValidateAdmittedRecord so a structural or allowlist regression
// fails loudly instead of silently drifting from the admitted Lexicon.
func (a *App) buildOccurrencePublicPreview(ctx context.Context, occurrence eventOccurrenceRow) (publicOccurrencePreviewDTO, error) {
	credits, err := a.loadOccurrenceCredits(ctx, occurrence.ID)
	if err != nil {
		return publicOccurrencePreviewDTO{}, err
	}
	if len(credits) == 0 {
		return publicOccurrencePreviewDTO{}, errNoPublicProfile
	}
	primary := credits[0]
	profileRow, err := a.loadCulturalProfile(ctx, primary.ProfileID)
	if err != nil {
		return publicOccurrencePreviewDTO{}, err
	}
	profileRecord := buildPublicProfileRecord(profileRow)
	profileRef := strongRefFor(nullStringValue(profileRow.PublicURI), nullStringValue(profileRow.PublicCID))

	var placeRecordPtr *publicPlaceRecord
	var placeRef *strongRefRecord
	if occurrence.PlaceID.Valid {
		placeRow, err := a.loadCulturalPlace(ctx, occurrence.PlaceID.String)
		if err != nil {
			return publicOccurrencePreviewDTO{}, err
		}
		record := buildPublicPlaceRecord(placeRow)
		placeRecordPtr = &record
		placeRef = strongRefFor(nullStringValue(placeRow.PublicURI), nullStringValue(placeRow.PublicCID))
	}

	occurrenceRecord := buildPublicOccurrenceRecord(occurrence, profileRef, placeRef)

	if err := validatePublicRecord(a.lexiconCatalog, "tv.subcult.profile", profileRecord); err != nil {
		return publicOccurrencePreviewDTO{}, err
	}
	if placeRecordPtr != nil {
		if err := validatePublicRecord(a.lexiconCatalog, "tv.subcult.place", *placeRecordPtr); err != nil {
			return publicOccurrencePreviewDTO{}, err
		}
	}
	if err := validatePublicRecord(a.lexiconCatalog, "tv.subcult.event.occurrence", occurrenceRecord); err != nil {
		return publicOccurrencePreviewDTO{}, err
	}

	return publicOccurrencePreviewDTO{
		Occurrence: occurrenceRecord,
		Profile:    profileRecord,
		Place:      placeRecordPtr,
	}, nil
}

func validatePublicRecord(catalog *atprotocol.LexiconCatalog, nsid string, record any) error {
	if catalog == nil {
		return errors.New("lexicon catalog unavailable")
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return atprotocol.ValidateAdmittedRecord(catalog, nsid, encoded)
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func (a *App) handleOccurrencePublicPreview(w http.ResponseWriter, r *http.Request) {
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
	if _, _, ok := a.requireWorkspaceRole(r, occurrence.WorkspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	preview, err := a.buildOccurrencePublicPreview(r.Context(), occurrence)
	if err != nil {
		if errors.Is(err, errNoPublicProfile) {
			writeError(w, http.StatusBadRequest, "occurrence has no credited profile")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "public preview failed validation: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}
