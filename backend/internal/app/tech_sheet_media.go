package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const maxStagePlotBytes = 4 << 20

func (a *App) handleUploadStagePlot(w http.ResponseWriter, r *http.Request) {
	event, actor, _, ok := a.techAccess(w, r, true)
	if !ok {
		return
	}
	actID := r.PathValue("actID")
	if !validUUID(actID) {
		writeError(w, 404, "sheet not found")
		return
	}
	storage, ok := a.media.(privateMediaStorage)
	if !ok || a.mediaErr != nil {
		writeError(w, 503, "private media storage is not configured")
		return
	}
	if s, ok := a.media.(*s3MediaStorage); ok && !s.privateReady() {
		writeError(w, 503, "a separate private media bucket is required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxStagePlotBytes+(64<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, 400, "invalid stage plot upload")
		return
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	expected, err := strconv.Atoi(r.FormValue("expectedVersion"))
	if err != nil || expected < 1 {
		writeError(w, 400, "save the sheet first and use its current version")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, 400, "stage plot image is required")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxStagePlotBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxStagePlotBytes {
		writeError(w, 400, "stage plot must be at most 4 MiB")
		return
	}
	contentType := http.DetectContentType(data)
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") || (contentType != "image/png" && contentType != "image/jpeg") || config.Width > 8192 || config.Height > 8192 || int64(config.Width)*int64(config.Height) > 20_000_000 {
		writeError(w, 400, "use a PNG or JPEG stage plot up to 8192 pixels and 20 megapixels")
		return
	}
	// Confirm scope/version before contacting storage, then compare again at
	// commit. An upload must not overwrite a concurrent technical edit.
	var current int
	if err := a.db.QueryRow(r.Context(), `select s.version from event_act_tech_sheets s join cultural_profiles cp on cp.id=s.act_profile_id join events e on e.id=s.event_id where s.event_id=$1 and s.act_profile_id=$2 and s.workspace_id=$3 and `+techActiveCredit, event.ID, actID, event.WorkspaceID).Scan(&current); err != nil {
		writeError(w, 404, "active sheet not found")
		return
	}
	if current != expected {
		writeError(w, 409, "tech sheet changed; reload before uploading")
		return
	}
	uploadCtx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	key, err := storage.UploadStagePlot(uploadCtx, event.ID, actID, contentType, bytes.NewReader(data), int64(len(data)))
	cancel()
	if err != nil {
		writeError(w, 502, "could not store private stage plot")
		return
	}
	saved := false
	defer func() {
		if !saved {
			a.removeStagePlotObject(key)
		}
	}()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not attach stage plot")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var sheetID string
	var oldPlot sql.NullString
	if err := tx.QueryRow(r.Context(), `select id,plot_object_key from event_act_tech_sheets where event_id=$1 and act_profile_id=$2 and workspace_id=$3 and version=$4 for update`, event.ID, actID, event.WorkspaceID, expected).Scan(&sheetID, &oldPlot); err != nil {
		writeError(w, 409, "tech sheet changed; reload before uploading")
		return
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	if _, err := tx.Exec(r.Context(), `update event_act_tech_sheets set plot_object_key=$2,plot_content_type=$3,version=version+1,updated_by_person_id=$4,updated_at=clock_timestamp() where id=$1`, sheetID, key, contentType, actor); err != nil {
		writeError(w, 500, "could not attach stage plot")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "tech_sheet.stage_plot_updated", "tech_sheet", sheetID, map[string]any{"eventId": event.ID, "actId": actID, "version": expected + 1}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not attach stage plot")
		return
	}
	saved = true
	if oldPlot.Valid {
		a.removeStagePlotObject(oldPlot.String)
	}
	writeJSON(w, 200, map[string]int{"version": expected + 1})
}
func (a *App) handleReadStagePlot(w http.ResponseWriter, r *http.Request) {
	event, _, editable, ok := a.techAccess(w, r, false)
	if !ok {
		return
	}
	actID := r.PathValue("actID")
	if !validUUID(actID) {
		writeError(w, 404, "stage plot not found")
		return
	}
	var key, contentType string
	err := a.db.QueryRow(r.Context(), `select s.plot_object_key,s.plot_content_type from event_act_tech_sheets s join events e on e.id=s.event_id join cultural_profiles cp on cp.id=s.act_profile_id where s.event_id=$1 and s.act_profile_id=$2 and s.workspace_id=$3 and s.plot_object_key is not null and ($4 or `+techActiveCredit+`)`, event.ID, actID, event.WorkspaceID, editable).Scan(&key, &contentType)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "stage plot not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not read stage plot")
		return
	}
	// Stored keys, not user-supplied URLs, address only this event/act prefix.
	if !strings.HasPrefix(key, "events/"+event.ID+"/tech/"+actID+"/") {
		writeError(w, 500, "stage plot scope is invalid")
		return
	}
	storage, ok := a.media.(privateMediaStorage)
	if !ok {
		writeError(w, 503, "private media storage is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	body, err := storage.ReadStagePlot(ctx, key)
	if err != nil {
		writeError(w, 502, "could not read private stage plot")
		return
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, maxStagePlotBytes+1))
	if err != nil || len(data) > maxStagePlotBytes {
		writeError(w, 502, "could not read private stage plot")
		return
	}
	permission := permOperate
	if editable {
		permission = permCoordinate
	}
	if _, ok := a.requirePermission(r, event.WorkspaceID, permission); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Content-Disposition", `inline; filename="stage-plot`+imageExtension("", contentType)+`"`)
	_, _ = w.Write(data)
}
