package app

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func techRequest(app *App, cookie *http.Cookie, method, path string, body any) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	return rec
}
func techStatus(t *testing.T, app *App, cookie *http.Cookie, method, path string, body any, status int) *httptest.ResponseRecorder {
	t.Helper()
	rec := techRequest(app, cookie, method, path, body)
	if rec.Code != status {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, rec.Code, status, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" && status != 401 {
		t.Fatal("private tech response can be cached")
	}
	return rec
}
func techList(t *testing.T, fx lifecycleFixture, eventID string, cookie *http.Cookie) eventInputListDTO {
	t.Helper()
	rec := techStatus(t, fx.app, cookie, "GET", "/api/events/"+eventID+"/input-list", nil, 200)
	var out eventInputListDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func techAct(t *testing.T, fx lifecycleFixture, eventID, occurrenceID, name string, order int) string {
	t.Helper()
	act := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{"kind": "act", "displayName": name}, 200)
	id := mustString(t, act.JSON, "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID+"/credits", map[string]any{"profileId": id, "role": "performer", "sortOrder": order}, 200)
	return id
}
func techFixture(t *testing.T) (lifecycleFixture, string, string, string, string) {
	t.Helper()
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Private sound check", 10)
	eventID := mustString(t, event, "id")
	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{"name": "Private bill", "startsAt": "2026-12-01T01:00:00Z", "timezone": "UTC"}, 200)
	occurrenceID := mustString(t, occurrence.JSON, "id")
	a := techAct(t, fx, eventID, occurrenceID, "First act", 0)
	b := techAct(t, fx, eventID, occurrenceID, "Second act", 1)
	return fx, eventID, occurrenceID, a, b
}
func techBody(version int, label, shared string) map[string]any {
	return map[string]any{"expectedVersion": version, "monitorNotes": "PRIVATE MONITOR", "backlineBrought": "PRIVATE BROUGHT", "backlineNeeded": "PRIVATE NEEDED", "inputs": []map[string]any{{"label": label, "sourceType": "mic", "micDi": "SM57", "stand": "Short", "phantomPower": false, "notes": "PRIVATE INPUT NOTE", "sharedKey": shared}}}
}
func TestTechSheetsSharedOrderRetentionAndConfirmedDeletion(t *testing.T) {
	fx, eventID, occ, a, b := techFixture(t)
	root := "/api/events/" + eventID
	sheet := root + "/tech-sheets/"
	techStatus(t, fx.app, fx.ownerCookie, "PUT", sheet+a, techBody(0, "Kick", "shared-kick"), 200)
	techStatus(t, fx.app, fx.ownerCookie, "PUT", sheet+b, techBody(0, "Kick B", "shared-kick"), 200)
	list := techList(t, fx, eventID, fx.ownerCookie)
	if len(list.Channels) != 1 || len(list.Channels[0].Uses) != 2 || list.Acts[0].ActID != a {
		t.Fatalf("shared list=%#v", list)
	}
	techStatus(t, fx.app, fx.ownerCookie, "PUT", root+"/input-list/order", map[string]any{"expectedVersion": 0, "actIds": []string{b, a}}, 200)
	list = techList(t, fx, eventID, fx.ownerCookie)
	if list.Acts[0].ActID != b || list.Channels[0].Uses[0].ActName != "Second act" {
		t.Fatal("order did not renumber shared uses")
	}
	techStatus(t, fx.app, fx.ownerCookie, "PUT", root+"/input-list/order", map[string]any{"expectedVersion": 0, "actIds": []string{a, b}}, 409)
	techStatus(t, fx.app, fx.ownerCookie, "PUT", root+"/input-list/order", map[string]any{"expectedVersion": 1, "actIds": []string{a, a}}, 409)
	techStatus(t, fx.app, fx.ownerCookie, "DELETE", root+"/tech-bill/"+a, nil, 200)
	list = techList(t, fx, eventID, fx.ownerCookie)
	if len(list.Acts) != 1 || len(list.Retained) != 1 || len(list.Channels[0].Uses) != 1 {
		t.Fatal("removed act still appears or lost retained sheet")
	}
	techStatus(t, fx.app, fx.ownerCookie, "PUT", sheet+a, techBody(1, "Detached edit", ""), 409)
	promoteFixtureMember(t, fx, roleCrew)
	crew := techList(t, fx, eventID, fx.memberCookie)
	if len(crew.Retained) != 0 || crew.CanEdit {
		t.Fatal("crew received retained organizer material")
	}
	postJSON(t, fx.app, fx.ownerCookie, root+"/occurrences/"+occ+"/credits", map[string]any{"profileId": a, "role": "performer"}, 200)
	list = techList(t, fx, eventID, fx.ownerCookie)
	if len(list.Acts) != 2 || len(list.Retained) != 0 {
		t.Fatal("reattach did not restore sheet")
	}
	techStatus(t, fx.app, fx.ownerCookie, "DELETE", sheet+a, map[string]any{"expectedVersion": 1}, 400)
	techStatus(t, fx.app, fx.ownerCookie, "DELETE", sheet+a, map[string]any{"expectedVersion": 2, "confirmed": true}, 409)
	techStatus(t, fx.app, fx.ownerCookie, "DELETE", sheet+a, map[string]any{"expectedVersion": 1, "confirmed": true}, 200)
	list = techList(t, fx, eventID, fx.ownerCookie)
	for _, act := range list.Acts {
		if act.ActID == a && (act.Version != 0 || len(act.Inputs) != 0) {
			t.Fatal("confirmed delete retained inputs")
		}
	}
}
func TestTechSheetsAuthorityScopeAndImmediateRevocation(t *testing.T) {
	fx, eventID, _, a, _ := techFixture(t)
	root := "/api/events/" + eventID
	path := root + "/tech-sheets/" + a
	techStatus(t, fx.app, nil, "GET", root+"/input-list", nil, 401)
	for _, role := range []string{roleCrew, roleDoor, roleFinance, roleMember, roleParticipant} {
		promoteFixtureMember(t, fx, role)
		want := 403
		if roleHasPermission(role, permOperate) {
			want = 200
		}
		techStatus(t, fx.app, fx.memberCookie, "GET", root+"/input-list", nil, want)
		techStatus(t, fx.app, fx.memberCookie, "PUT", path, techBody(0, "Kick", ""), 403)
		techStatus(t, fx.app, fx.memberCookie, "DELETE", path, map[string]any{"confirmed": true, "expectedVersion": 1}, 403)
		techStatus(t, fx.app, fx.memberCookie, "POST", root+"/tech-sheets/copy", map[string]any{}, 403)
	}
	promoteFixtureMember(t, fx, roleOrganizer)
	techStatus(t, fx.app, fx.memberCookie, "PUT", path, techBody(0, "Kick", ""), 200)
	_, memberID := memberIdentity(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	techStatus(t, fx.app, fx.memberCookie, "GET", root+"/input-list", nil, 403)
	techStatus(t, fx.app, fx.memberCookie, "PUT", path, techBody(1, "Kick", ""), 403)
	other := newCulturalFixture(t, fx.app)
	techStatus(t, fx.app, other.ownerCookie, "GET", root+"/input-list", nil, 403)
	foreignEvent := mustString(t, createEvent(t, other, "Other", 1), "id")
	if _, err := fx.app.db.Exec(t.Context(), `insert into event_act_tech_sheets(event_id,workspace_id,act_profile_id,updated_by_person_id) select $1,$2,$3,created_by_person_id from events where id=$1`, foreignEvent, other.workspaceID, a); err == nil {
		t.Fatal("cross-workspace act accepted")
	}
	var sheetID string
	if err := fx.app.db.QueryRow(t.Context(), `select id from event_act_tech_sheets where event_id=$1`, eventID).Scan(&sheetID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `insert into event_act_inputs(sheet_id,event_id,workspace_id,sort_order,label,source_type) values($1,$2,$3,0,'Foreign','mic')`, sheetID, foreignEvent, other.workspaceID); err == nil {
		t.Fatal("cross-workspace input accepted")
	}
}
func TestTechSheetsConcurrentCreationAndEditCompareVersions(t *testing.T) {
	fx, eventID, _, a, _ := techFixture(t)
	path := "/api/events/" + eventID + "/tech-sheets/" + a
	for _, version := range []int{0, 1} {
		var wg sync.WaitGroup
		responses := make(chan int, 2)
		start := make(chan struct{})
		for _, label := range []string{"First save", "Second save"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				responses <- techRequest(fx.app, fx.ownerCookie, "PUT", path, techBody(version, label, "")).Code
			}()
		}
		close(start)
		wg.Wait()
		close(responses)
		counts := map[int]int{}
		for code := range responses {
			counts[code]++
		}
		if counts[200] != 1 || counts[409] != 1 {
			t.Fatalf("concurrent version %d statuses=%v", version, counts)
		}
	}
	list := techList(t, fx, eventID, fx.ownerCookie)
	if list.Acts[0].Version != 2 || len(list.Acts[0].Inputs) != 1 {
		t.Fatal("lost write/version")
	}
}
func TestTechSheetsExplicitCopyDoesNotOverwriteOrCopyPlots(t *testing.T) {
	fx, eventID, _, a, b := techFixture(t)
	root := "/api/events/" + eventID
	techStatus(t, fx.app, fx.ownerCookie, "PUT", root+"/tech-sheets/"+a, techBody(0, "Kick", ""), 200)
	if _, err := fx.app.db.Exec(t.Context(), `update event_act_tech_sheets set plot_object_key='private-plot',plot_content_type='image/png' where event_id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	target := mustString(t, createEvent(t, fx, "Next draft", 5), "id")
	targetRoot := "/api/events/" + target
	occurrence := postJSON(t, fx.app, fx.ownerCookie, targetRoot+"/occurrences", map[string]any{"name": "Next", "startsAt": "2026-12-02T01:00:00Z", "timezone": "UTC"}, 200)
	for _, id := range []string{a, b} {
		postJSON(t, fx.app, fx.ownerCookie, targetRoot+"/occurrences/"+mustString(t, occurrence.JSON, "id")+"/credits", map[string]any{"profileId": id, "role": "performer"}, 200)
	}
	before := techList(t, fx, target, fx.ownerCookie)
	for _, act := range before.Acts {
		if act.Version != 0 {
			t.Fatal("new event implicitly copied tech sheets")
		}
	}
	techStatus(t, fx.app, fx.ownerCookie, "POST", targetRoot+"/tech-sheets/copy", map[string]any{"sourceEventId": eventID}, 400)
	techStatus(t, fx.app, fx.ownerCookie, "POST", targetRoot+"/tech-sheets/copy", map[string]any{"sourceEventId": eventID, "confirmed": true}, 200)
	copied := techList(t, fx, target, fx.ownerCookie)
	for _, act := range copied.Acts {
		if act.ActID == a && (act.Version != 1 || len(act.Inputs) != 1 || act.StagePlotURL != nil) {
			t.Fatal("copy incomplete or copied plot")
		}
	}
	techStatus(t, fx.app, fx.ownerCookie, "PUT", targetRoot+"/tech-sheets/"+a, techBody(1, "Edited copy", ""), 200)
	techStatus(t, fx.app, fx.ownerCookie, "POST", targetRoot+"/tech-sheets/copy", map[string]any{"sourceEventId": eventID, "confirmed": true}, 200)
	after := techList(t, fx, target, fx.ownerCookie)
	for _, act := range after.Acts {
		if act.ActID == a && act.Inputs[0].Label != "Edited copy" {
			t.Fatal("copy overwrote existing sheet")
		}
	}
}

type techPrivateMedia struct {
	fakeMediaStorage
	data     []byte
	removed  []string
	onUpload func()
	onRead   func()
}

func (m *techPrivateMedia) UploadStagePlot(_ context.Context, eventID, actID, _ string, body io.Reader, _ int64) (string, error) {
	m.data, _ = io.ReadAll(body)
	if m.onUpload != nil {
		m.onUpload()
	}
	return "events/" + eventID + "/tech/" + actID + "/" + uuid.NewString() + ".png", nil
}
func (m *techPrivateMedia) ReadStagePlot(context.Context, string) (io.ReadCloser, error) {
	if m.onRead != nil {
		m.onRead()
	}
	return io.NopCloser(bytes.NewReader(m.data)), nil
}
func (m *techPrivateMedia) RemoveStagePlot(_ context.Context, key string) error {
	m.removed = append(m.removed, key)
	return nil
}
func techPlotRequest(t *testing.T, fx lifecycleFixture, eventID, actID string, version int, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("expectedVersion", strconv.Itoa(version))
	part, err := writer.CreateFormFile("image", "plot.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = writer.Close()
	req := httptest.NewRequest("POST", "/api/events/"+eventID+"/tech-sheets/"+actID+"/stage-plot", &body)
	req.AddCookie(fx.ownerCookie)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(rec, req)
	return rec
}
func TestTechSheetsPrivatePlotsVersionCleanupAndRevocation(t *testing.T) {
	fx, eventID, _, a, _ := techFixture(t)
	path := "/api/events/" + eventID + "/tech-sheets/" + a
	techStatus(t, fx.app, fx.ownerCookie, "PUT", path, techBody(0, "Kick", ""), 200)
	media := &techPrivateMedia{}
	fx.app.media = media
	fx.app.mediaErr = nil
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 20, 20))); err != nil {
		t.Fatal(err)
	}
	if rec := techPlotRequest(t, fx, eventID, a, 1, []byte("<svg>unsafe</svg>")); rec.Code != 400 {
		t.Fatalf("svg=%d", rec.Code)
	}
	if rec := techPlotRequest(t, fx, eventID, a, 1, pngData.Bytes()); rec.Code != 200 {
		t.Fatalf("plot=%d %s", rec.Code, rec.Body.String())
	}
	promoteFixtureMember(t, fx, roleCrew)
	rec := techStatus(t, fx.app, fx.memberCookie, "GET", path+"/stage-plot", nil, 200)
	if !bytes.Equal(rec.Body.Bytes(), pngData.Bytes()) || rec.Header().Get("Content-Type") != "image/png" || rec.Header().Get("Location") != "" {
		t.Fatal("private plot not served inline")
	}
	_, memberID := memberIdentity(t, fx)
	media.onRead = func() {
		if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where id=$1`, memberID); err != nil {
			t.Error(err)
		}
	}
	techStatus(t, fx.app, fx.memberCookie, "GET", path+"/stage-plot", nil, 403)
	media.onRead = nil
	media.onUpload = func() { techStatus(t, fx.app, fx.ownerCookie, "PUT", path, techBody(2, "Concurrent edit", ""), 200) }
	if rec := techPlotRequest(t, fx, eventID, a, 2, pngData.Bytes()); rec.Code != 409 {
		t.Fatalf("upload overwrote concurrent edit=%d", rec.Code)
	}
	if len(media.removed) != 1 {
		t.Fatal("failed upload not cleaned up")
	}
	media.onUpload = nil
	techStatus(t, fx.app, fx.ownerCookie, "DELETE", "/api/events/"+eventID+"/tech-bill/"+a, nil, 200)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=null where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	promoteFixtureMember(t, fx, roleCrew)
	techStatus(t, fx.app, fx.memberCookie, "GET", path+"/stage-plot", nil, 404) // retained plots require coordination
	techStatus(t, fx.app, fx.ownerCookie, "GET", path+"/stage-plot", nil, 200)
	techStatus(t, fx.app, fx.ownerCookie, "DELETE", path, map[string]any{"expectedVersion": 3, "confirmed": true}, 200)
	if len(media.removed) != 2 {
		t.Fatal("deleted plot not cleaned up")
	}
}
func TestTechSheetsPrintEscapesNotesAndIncludes24Channels(t *testing.T) {
	list := eventInputListDTO{EventTitle: "<script>alert(1)</script>", Acts: []actTechSheetDTO{}}
	for i := range 3 {
		act := actTechSheetDTO{ActName: "Act " + strconv.Itoa(i+1), MonitorNotes: "Two wedges", BacklineBrought: "Amps", BacklineNeeded: "Drums"}
		for j := range 8 {
			act.Inputs = append(act.Inputs, actInputDTO{ID: uuid.NewString(), Label: "Input " + strconv.Itoa(j+1), SourceType: "mic", MicDI: "SM57", Stand: "Short", Notes: "<img src=x onerror=alert(1)>"})
		}
		list.Acts = append(list.Acts, act)
	}
	list.Channels = combineActInputs(list.Acts)
	rec := httptest.NewRecorder()
	(&App{}).writeInputListPrint(rec, httptest.NewRequest("GET", "/", nil), list)
	body := rec.Body.String()
	if strings.Contains(body, "<script>") || strings.Contains(body, "<img src=x") || strings.Count(body, "<tr>") != 25 || !strings.Contains(body, "size:A4 portrait") {
		t.Fatal("unsafe or incomplete print document")
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatal("print content not isolated")
	}
}

func TestTechSheetsTemplateAndArchiveReuseStartWithoutPrivateSheets(t *testing.T) {
	fx, eventID, _, actID, _ := techFixture(t)
	techStatus(t, fx.app, fx.ownerCookie, "PUT", "/api/events/"+eventID+"/tech-sheets/"+actID, techBody(0, "Private input", ""), 200)
	tmpl := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", map[string]any{"name": "Sound check template", "title": "Next show", "ticketAllocation": 10, "privateNotes": "Template note"}, 200)
	targetID := mustString(t, createEvent(t, fx, "Template target", 10), "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+targetID+"/apply-template", map[string]any{"templateId": mustString(t, tmpl.JSON, "id")}, 200)
	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, 200)
	seeded := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, 200)
	for _, id := range []string{targetID, mustString(t, seeded.JSON, "id")} {
		var count int
		if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_act_tech_sheets where event_id=$1`, id).Scan(&count); err != nil || count != 0 {
			t.Fatalf("reuse copied private tech data: count=%d err=%v", count, err)
		}
	}
}
