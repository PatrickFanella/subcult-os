package app

import (
	"database/sql"
	"mime/multipart"
	"net/http"
	"strings"
)

const maxEventImageBytes = 12 << 20

func (a *App) handleUploadEventImage(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	if a.mediaErr != nil || a.media == nil {
		message := "media storage is not configured"
		if a.mediaErr != nil {
			message = a.mediaErr.Error()
		}
		writeError(w, http.StatusServiceUnavailable, message)
		return
	}

	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	_ = actorID

	r.Body = http.MaxBytesReader(w, r.Body, maxEventImageBytes)
	if err := r.ParseMultipartForm(maxEventImageBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid image upload")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	contentType := imageContentType(header)
	if contentType == "" {
		writeError(w, http.StatusBadRequest, "image must be jpeg, png, webp, or gif")
		return
	}
	if header.Size <= 0 || header.Size > maxEventImageBytes {
		writeError(w, http.StatusBadRequest, "image is too large")
		return
	}

	imageURL, err := a.media.UploadEventImage(r.Context(), event.ID, header.Filename, contentType, file, header.Size)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not upload image")
		return
	}

	if _, err := a.db.Exec(r.Context(), `update events set image_url = $2, updated_at = now() where id = $1`, event.ID, imageURL); err != nil {
		writeError(w, http.StatusInternalServerError, "could not attach image")
		return
	}
	event.ImageURL = sql.NullString{String: imageURL, Valid: true}
	writeJSON(w, http.StatusOK, a.eventDTOFromRow(event))
}

func imageContentType(header *multipart.FileHeader) string {
	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return contentType
	default:
		return ""
	}
}
