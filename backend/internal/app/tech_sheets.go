package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type actInputDTO struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	SourceType   string `json:"sourceType"`
	MicDI        string `json:"micDi"`
	Stand        string `json:"stand"`
	PhantomPower bool   `json:"phantomPower"`
	Notes        string `json:"notes"`
	SharedKey    string `json:"sharedKey"`
}
type actTechSheetDTO struct {
	ActID           string        `json:"actId"`
	ActName         string        `json:"actName"`
	Attached        bool          `json:"attached"`
	Version         int           `json:"version"`
	MonitorNotes    string        `json:"monitorNotes"`
	BacklineBrought string        `json:"backlineBrought"`
	BacklineNeeded  string        `json:"backlineNeeded"`
	StagePlotURL    *string       `json:"stagePlotUrl"`
	Inputs          []actInputDTO `json:"inputs"`
}
type inputUseDTO struct {
	ActName string `json:"actName"`
	actInputDTO
}
type inputChannelDTO struct {
	Number    int           `json:"number"`
	SharedKey string        `json:"sharedKey"`
	Uses      []inputUseDTO `json:"uses"`
}
type eventInputListDTO struct {
	EventID      string            `json:"eventId"`
	EventTitle   string            `json:"eventTitle"`
	CanEdit      bool              `json:"canEdit"`
	OrderVersion int               `json:"orderVersion"`
	Acts         []actTechSheetDTO `json:"acts"`
	Retained     []actTechSheetDTO `json:"retained,omitempty"`
	Channels     []inputChannelDTO `json:"channels"`
}
type putTechSheetRequest struct {
	ExpectedVersion *int          `json:"expectedVersion"`
	MonitorNotes    string        `json:"monitorNotes"`
	BacklineBrought string        `json:"backlineBrought"`
	BacklineNeeded  string        `json:"backlineNeeded"`
	ClearStagePlot  bool          `json:"clearStagePlot"`
	Inputs          []actInputDTO `json:"inputs"`
}

var sharedInputKey = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// Active credits, not retained sheets, are the bill authority. Duplicated
// cross-listings credit an act once, using its earliest credited position.
const techActiveCredit = `exists(select 1 from event_occurrence_profiles credit join event_occurrences occurrence on occurrence.id=credit.occurrence_id
 where occurrence.event_id=e.id and occurrence.workspace_id=e.workspace_id and credit.profile_id=cp.id and credit.role='performer')`

func (a *App) techAccess(w http.ResponseWriter, r *http.Request, edit bool) (eventRow, string, bool, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return eventRow{}, "", false, false
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		writeError(w, 404, "event not found")
		return eventRow{}, "", false, false
	}
	actor, role, ok := a.requireWorkspaceRole(r, event.WorkspaceID)
	if !ok || !roleHasPermission(role, permOperate) || edit && !roleHasPermission(role, permCoordinate) {
		writeError(w, 403, "forbidden")
		return eventRow{}, "", false, false
	}
	return event, actor, roleHasPermission(role, permCoordinate), true
}

func validateTechSheet(req *putTechSheetRequest) bool {
	if req.ExpectedVersion == nil || *req.ExpectedVersion < 0 || len(req.Inputs) > 64 {
		return false
	}
	for _, text := range []*string{&req.MonitorNotes, &req.BacklineBrought, &req.BacklineNeeded} {
		*text = strings.TrimSpace(*text)
		if !boundedTechText(*text, 2000) {
			return false
		}
	}
	shared := map[string]bool{}
	ids := map[string]bool{}
	for i := range req.Inputs {
		input := &req.Inputs[i]
		input.Label = strings.TrimSpace(input.Label)
		input.MicDI = strings.TrimSpace(input.MicDI)
		input.Stand = strings.TrimSpace(input.Stand)
		input.Notes = strings.TrimSpace(input.Notes)
		input.SharedKey = strings.ToLower(strings.TrimSpace(input.SharedKey))
		if input.Label == "" || !boundedTechText(input.Label, 100) || !boundedTechText(input.MicDI, 100) || !boundedTechText(input.Stand, 100) || !boundedTechText(input.Notes, 500) {
			return false
		}
		if input.SourceType != "mic" && input.SourceType != "di" && input.SourceType != "line" {
			return false
		}
		if input.ID == "" {
			input.ID = uuid.NewString()
		}
		if !validUUID(input.ID) || ids[input.ID] {
			return false
		}
		ids[input.ID] = true
		if input.SharedKey != "" {
			if !sharedInputKey.MatchString(input.SharedKey) || shared[input.SharedKey] {
				return false
			}
			shared[input.SharedKey] = true
		}
	}
	return true
}
func boundedTechText(value string, limit int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= limit && !strings.ContainsRune(value, 0)
}

func (a *App) readInputList(ctx context.Context, event eventRow, editable bool) (eventInputListDTO, error) {
	out := eventInputListDTO{EventID: event.ID, EventTitle: event.Title, CanEdit: editable, Acts: []actTechSheetDTO{}, Channels: []inputChannelDTO{}}
	tx, err := a.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `select cp.id,cp.display_name,`+techActiveCredit+`,coalesce(s.id::text,''),coalesce(s.version,0),coalesce(s.monitor_notes,''),coalesce(s.backline_brought,''),coalesce(s.backline_needed,''),s.plot_object_key is not null
 from events e join cultural_profiles cp on cp.workspace_id=e.workspace_id and cp.kind='act'
 left join event_act_tech_sheets s on s.event_id=e.id and s.act_profile_id=cp.id
 where e.id=$1 and (`+techActiveCredit+` or ($2 and s.id is not null))
 order by (select min(credit.sort_order) from event_occurrence_profiles credit join event_occurrences occurrence on occurrence.id=credit.occurrence_id where occurrence.event_id=e.id and credit.profile_id=cp.id and credit.role='performer') nulls last,cp.id`, event.ID, editable)
	if err != nil {
		return out, err
	}
	sheets := map[string]*actTechSheetDTO{}
	acts := []*actTechSheetDTO{}
	for rows.Next() {
		sheet := &actTechSheetDTO{Inputs: []actInputDTO{}}
		var sheetID string
		var plot bool
		if err := rows.Scan(&sheet.ActID, &sheet.ActName, &sheet.Attached, &sheetID, &sheet.Version, &sheet.MonitorNotes, &sheet.BacklineBrought, &sheet.BacklineNeeded, &plot); err != nil {
			rows.Close()
			return out, err
		}
		if plot {
			url := "/api/events/" + event.ID + "/tech-sheets/" + sheet.ActID + "/stage-plot"
			sheet.StagePlotURL = &url
		}
		acts = append(acts, sheet)
		if sheetID != "" {
			sheets[sheetID] = sheet
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.Query(ctx, `select sheet_id,id,label,source_type,mic_di,stand,phantom_power,notes,shared_key from event_act_inputs where event_id=$1 and workspace_id=$2 order by sheet_id,sort_order`, event.ID, event.WorkspaceID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var sheetID string
		var input actInputDTO
		if err := rows.Scan(&sheetID, &input.ID, &input.Label, &input.SourceType, &input.MicDI, &input.Stand, &input.PhantomPower, &input.Notes, &input.SharedKey); err != nil {
			rows.Close()
			return out, err
		}
		if sheet := sheets[sheetID]; sheet != nil {
			sheet.Inputs = append(sheet.Inputs, input)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	var orderData []byte
	err = tx.QueryRow(ctx, `select revision,act_ids from event_input_list_orders where event_id=$1`, event.ID).Scan(&out.OrderVersion, &orderData)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var order []string
	if len(orderData) > 0 {
		if err := json.Unmarshal(orderData, &order); err != nil {
			return out, err
		}
	}
	rank := map[string]int{}
	for i, id := range order {
		rank[id] = i
	}
	sort.SliceStable(acts, func(i, j int) bool {
		left, l := rank[acts[i].ActID]
		right, r := rank[acts[j].ActID]
		if l != r {
			return l
		}
		return l && left < right
	})
	for _, sheet := range acts {
		if sheet.Attached {
			out.Acts = append(out.Acts, *sheet)
		} else if editable {
			out.Retained = append(out.Retained, *sheet)
		}
	}
	out.Channels = combineActInputs(out.Acts)
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}
func combineActInputs(acts []actTechSheetDTO) []inputChannelDTO {
	channels := []inputChannelDTO{}
	shared := map[string]int{}
	for _, act := range acts {
		for _, input := range act.Inputs {
			index, exists := shared[input.SharedKey]
			if input.SharedKey == "" || !exists {
				index = len(channels)
				channels = append(channels, inputChannelDTO{Number: index + 1, SharedKey: input.SharedKey, Uses: []inputUseDTO{}})
				if input.SharedKey != "" {
					shared[input.SharedKey] = index
				}
			}
			channels[index].Uses = append(channels[index].Uses, inputUseDTO{ActName: act.ActName, actInputDTO: input})
		}
	}
	return channels
}
func (a *App) handleInputList(w http.ResponseWriter, r *http.Request) {
	event, _, editable, ok := a.techAccess(w, r, false)
	if !ok {
		return
	}
	out, err := a.readInputList(r.Context(), event, editable)
	if err != nil {
		writeError(w, 500, "could not load input list")
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
	if r.URL.Query().Get("format") == "print" {
		a.writeInputListPrint(w, r, out)
		return
	}
	writeJSON(w, 200, out)
}
func (a *App) handlePutTechSheet(w http.ResponseWriter, r *http.Request) {
	event, actor, _, ok := a.techAccess(w, r, true)
	if !ok {
		return
	}
	actID := r.PathValue("actID")
	if !validUUID(actID) {
		writeError(w, 404, "act not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	var req putTechSheetRequest
	if err := decodeJSON(r, &req); err != nil || !validateTechSheet(&req) {
		writeError(w, 400, "a current version, bounded notes and up to 64 valid inputs are required")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save tech sheet")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	// Serialize creation as well as updates, without granting detached sheets
	// authority to reintroduce an act to the combined bill.
	if _, err := a.loadEventDetailsForUpdate(r.Context(), tx, event.ID); err != nil {
		writeError(w, 404, "event not found")
		return
	}
	var attached bool
	if err := tx.QueryRow(r.Context(), `select exists(select 1 from cultural_profiles cp join events e on e.workspace_id=cp.workspace_id where e.id=$1 and cp.id=$2 and cp.kind='act' and `+techActiveCredit+`)`, event.ID, actID).Scan(&attached); err != nil || !attached {
		writeError(w, 409, "attach the act as a performer before editing its sheet")
		return
	}
	var sheetID string
	var version int
	var oldPlot sql.NullString
	err = tx.QueryRow(r.Context(), `select id,version,plot_object_key from event_act_tech_sheets where event_id=$1 and act_profile_id=$2 for update`, event.ID, actID).Scan(&sheetID, &version, &oldPlot)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not load tech sheet")
		return
	}
	if version != *req.ExpectedVersion {
		writeError(w, 409, "tech sheet changed; reload before saving")
		return
	}
	if sheetID == "" {
		err = tx.QueryRow(r.Context(), `insert into event_act_tech_sheets(event_id,workspace_id,act_profile_id,monitor_notes,backline_brought,backline_needed,updated_by_person_id) values($1,$2,$3,$4,$5,$6,$7) returning id`, event.ID, event.WorkspaceID, actID, req.MonitorNotes, req.BacklineBrought, req.BacklineNeeded, actor).Scan(&sheetID)
	} else {
		_, err = tx.Exec(r.Context(), `update event_act_tech_sheets set monitor_notes=$2,backline_brought=$3,backline_needed=$4,version=version+1,updated_by_person_id=$5,updated_at=clock_timestamp(),plot_object_key=case when $6 then null else plot_object_key end,plot_content_type=case when $6 then null else plot_content_type end where id=$1`, sheetID, req.MonitorNotes, req.BacklineBrought, req.BacklineNeeded, actor, req.ClearStagePlot)
	}
	if err != nil {
		writeError(w, 500, "could not save tech sheet")
		return
	}
	if _, err := tx.Exec(r.Context(), `delete from event_act_inputs where sheet_id=$1`, sheetID); err != nil {
		writeError(w, 500, "could not save inputs")
		return
	}
	for i, input := range req.Inputs {
		if _, err := tx.Exec(r.Context(), `insert into event_act_inputs(id,sheet_id,event_id,workspace_id,sort_order,label,source_type,mic_di,stand,phantom_power,notes,shared_key) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, input.ID, sheetID, event.ID, event.WorkspaceID, i, input.Label, input.SourceType, input.MicDI, input.Stand, input.PhantomPower, input.Notes, input.SharedKey); err != nil {
			writeError(w, 400, "input identities conflict with another sheet")
			return
		}
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "tech_sheet.saved", "tech_sheet", sheetID, map[string]any{"eventId": event.ID, "actId": actID, "version": version + 1}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not save tech sheet")
		return
	}
	if req.ClearStagePlot && oldPlot.Valid {
		a.removeStagePlotObject(oldPlot.String)
	}
	writeJSON(w, 200, map[string]int{"version": version + 1})
}
func (a *App) handleDeleteTechSheet(w http.ResponseWriter, r *http.Request) {
	event, actor, _, ok := a.techAccess(w, r, true)
	if !ok {
		return
	}
	if !validUUID(r.PathValue("actID")) {
		writeError(w, 404, "sheet not found")
		return
	}
	var req struct {
		ExpectedVersion int  `json:"expectedVersion"`
		Confirmed       bool `json:"confirmed"`
	}
	if err := decodeJSON(r, &req); err != nil || !req.Confirmed || req.ExpectedVersion < 1 {
		writeError(w, 400, "confirm deletion with the current sheet version")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not delete sheet")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var id string
	var plot sql.NullString
	err = tx.QueryRow(r.Context(), `delete from event_act_tech_sheets where event_id=$1 and act_profile_id=$2 and version=$3 returning id,plot_object_key`, event.ID, r.PathValue("actID"), req.ExpectedVersion).Scan(&id, &plot)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 409, "sheet changed or was deleted; reload first")
		return
	}
	if err != nil {
		writeError(w, 500, "could not delete sheet")
		return
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "tech_sheet.deleted", "tech_sheet", id, map[string]any{"eventId": event.ID}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not delete sheet")
		return
	}
	if plot.Valid {
		a.removeStagePlotObject(plot.String)
	}
	writeJSON(w, 200, map[string]bool{"deleted": true})
}
func (a *App) handleOrderInputList(w http.ResponseWriter, r *http.Request) {
	event, actor, _, ok := a.techAccess(w, r, true)
	if !ok {
		return
	}
	var req struct {
		ExpectedVersion *int     `json:"expectedVersion"`
		ActIDs          []string `json:"actIds"`
	}
	if err := decodeJSON(r, &req); err != nil || req.ExpectedVersion == nil || *req.ExpectedVersion < 0 || len(req.ActIDs) > 100 {
		writeError(w, 400, "a current order version and act IDs are required")
		return
	}
	current, err := a.readInputList(r.Context(), event, false)
	if err != nil {
		writeError(w, 500, "could not load bill")
		return
	}
	expected := map[string]bool{}
	for _, act := range current.Acts {
		expected[act.ActID] = true
	}
	if len(req.ActIDs) != len(expected) {
		writeError(w, 409, "bill changed; reload before reordering")
		return
	}
	for _, id := range req.ActIDs {
		if !expected[id] {
			writeError(w, 409, "bill changed; reload before reordering")
			return
		}
		delete(expected, id)
	}
	data, _ := json.Marshal(req.ActIDs)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save order")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := a.loadEventDetailsForUpdate(r.Context(), tx, event.ID); err != nil {
		writeError(w, 404, "event not found")
		return
	}
	var version int
	if *req.ExpectedVersion == 0 {
		err = tx.QueryRow(r.Context(), `insert into event_input_list_orders(event_id,workspace_id,revision,act_ids) values($1,$2,1,$3) on conflict(event_id) do nothing returning revision`, event.ID, event.WorkspaceID, data).Scan(&version)
	} else {
		err = tx.QueryRow(r.Context(), `update event_input_list_orders set revision=revision+1,act_ids=$2 where event_id=$1 and revision=$3 returning revision`, event.ID, data, *req.ExpectedVersion).Scan(&version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 409, "input-list order changed; reload first")
		return
	}
	if err != nil {
		writeError(w, 500, "could not save order")
		return
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "input_list.reordered", "event", event.ID, nil); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not save order")
		return
	}
	writeJSON(w, 200, map[string]int{"version": version})
}

func (a *App) handleCopyTechSheets(w http.ResponseWriter, r *http.Request) {
	target, actor, _, ok := a.techAccess(w, r, true)
	if !ok {
		return
	}
	var req struct {
		SourceEventID string `json:"sourceEventId"`
		Confirmed     bool   `json:"confirmed"`
	}
	if err := decodeJSON(r, &req); err != nil || !req.Confirmed || !validUUID(req.SourceEventID) || req.SourceEventID == target.ID {
		writeError(w, 400, "choose another source event and confirm the copy")
		return
	}
	source, err := a.loadEventDetails(r.Context(), req.SourceEventID)
	if err != nil || source.WorkspaceID != target.WorkspaceID {
		writeError(w, 404, "source event not found")
		return
	}
	if target.Status != eventStatusDraft {
		writeError(w, 409, "copy sheets only into a draft event")
		return
	}
	data, err := a.readInputList(r.Context(), source, true)
	if err != nil {
		writeError(w, 500, "could not read source sheets")
		return
	}
	// Each copy is an insert-only opt-in. Never overwrite an existing target
	// sheet or carry a private stage-plot object into another event implicitly.
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not copy sheets")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	locked, err := a.loadEventDetailsForUpdate(r.Context(), tx, target.ID)
	if err != nil || locked.Status != eventStatusDraft {
		writeError(w, 409, "target event changed; reload first")
		return
	}
	copied := 0
	for _, sheet := range append(data.Acts, data.Retained...) {
		if sheet.Version == 0 {
			continue
		}
		var id string
		err = tx.QueryRow(r.Context(), `insert into event_act_tech_sheets(event_id,workspace_id,act_profile_id,monitor_notes,backline_brought,backline_needed,updated_by_person_id)
   select e.id,e.workspace_id,$2,$3,$4,$5,$6 from events e join cultural_profiles cp on cp.id=$2 and cp.workspace_id=e.workspace_id and cp.kind='act' where e.id=$1 and `+techActiveCredit+` on conflict(event_id,act_profile_id) do nothing returning id`, target.ID, sheet.ActID, sheet.MonitorNotes, sheet.BacklineBrought, sheet.BacklineNeeded, actor).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			writeError(w, 500, "could not copy sheet")
			return
		}
		for i, input := range sheet.Inputs {
			if _, err := tx.Exec(r.Context(), `insert into event_act_inputs(sheet_id,event_id,workspace_id,sort_order,label,source_type,mic_di,stand,phantom_power,notes,shared_key) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, target.ID, target.WorkspaceID, i, input.Label, input.SourceType, input.MicDI, input.Stand, input.PhantomPower, input.Notes, input.SharedKey); err != nil {
				writeError(w, 500, "could not copy inputs")
				return
			}
		}
		copied++
	}
	if _, _, ok := a.requireCoordinator(r, target.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "tech_sheets.copied", "event", target.ID, map[string]any{"sourceEventId": source.ID, "copied": copied}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not copy sheets")
		return
	}
	writeJSON(w, 200, map[string]int{"copied": copied})
}

// Remove only this event's performer credits; preserve the sheet for review.
func (a *App) handleRemoveTechBillAct(w http.ResponseWriter, r *http.Request) {
	event, actor, _, ok := a.techAccess(w, r, true)
	if !ok {
		return
	}
	actID := r.PathValue("actID")
	if !validUUID(actID) {
		writeError(w, 404, "act not found")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not remove performer credit")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := tx.Exec(r.Context(), `delete from event_occurrence_profiles credit using event_occurrences occurrence where credit.occurrence_id=occurrence.id and occurrence.event_id=$1 and credit.profile_id=$2 and credit.role='performer'`, event.ID, actID); err != nil {
		writeError(w, 500, "could not remove performer credit")
		return
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "tech_bill.act_removed", "event", event.ID, map[string]any{"actId": actID}); err != nil {
		writeError(w, 500, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not remove performer credit")
		return
	}
	writeJSON(w, 200, map[string]bool{"removed": true})
}

func (a *App) removeStagePlotObject(key string) {
	storage, ok := a.media.(privateMediaStorage)
	if !ok || key == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// A failed object cleanup leaves no usable public URL; normal storage
	// retention can remove the unreferenced private object later.
	_ = storage.RemoveStagePlot(ctx, key)
}
