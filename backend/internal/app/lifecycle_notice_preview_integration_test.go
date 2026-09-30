package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func noticePreviewFixture(t *testing.T) (lifecycleFixture, string, map[string]any, string) {
	t.Helper()
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Notice event", 600), "id")
	occurrence := createLifecycleIntentOccurrence(t, fx, eventID)
	occurrence = mustObject(t, patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrence["id"].(string), map[string]any{"status": "cancelled", "expectedUpdatedAt": occurrence["updatedAt"], "expectedPublicCid": ""}, 200).JSON)
	payload := lifecycleIntentPayload(occurrence)
	payload["reason"] = "PRIVATE_REASON_DO_NOT_SEND"
	intent := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, 201).JSON)
	path := "/api/events/" + eventID + "/lifecycle-intents/" + intent["id"].(string) + "/notice-preview"
	return fx, eventID, occurrence, path
}

func TestLifecycleNoticePreviewRecipientsPrivacyAndDigest(t *testing.T) {
	fx, eventID, _, path := noticePreviewFixture(t)
	ctx := t.Context()
	insertTicket := func(email, payment string) {
		t.Helper()
		if _, err := fx.app.db.Exec(ctx, `insert into tickets(event_id,email,code,payment_status) values($1,$2,$3,$4)`, eventID, email, uuid.NewString(), payment); err != nil {
			t.Fatal(err)
		}
	}
	insertTicket(" Guest@Example.test ", "free")
	insertTicket("guest@example.test", "pending")
	insertTicket("cancelled@example.test", "cancelled")
	insertTicket("blocked@example.test", "paid")
	if _, err := fx.app.db.Exec(ctx, `insert into email_suppressions(recipient_email,reason) values('blocked@example.test','email.bounced')`); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := fx.app.db.QueryRow(ctx, `select count(*) from email_outbox`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	req := map[string]any{"audiences": []string{"ticket_holders"}}
	response := postJSON(t, fx.app, fx.ownerCookie, path, req, 200)
	if response.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatal("private preview must not be cached")
	}
	preview := mustObject(t, response.JSON)
	recipients := preview["recipients"].([]any)
	if len(recipients) != 2 {
		t.Fatalf("recipients=%#v", recipients)
	}
	first := mustObject(t, recipients[0])
	second := mustObject(t, recipients[1])
	if first["email"] != "blocked@example.test" || first["suppressed"] != true || second["email"] != "guest@example.test" || second["suppressed"] != false {
		t.Fatalf("recipients=%#v", recipients)
	}
	raw, _ := json.Marshal(preview)
	for _, private := range []string{"PRIVATE_REASON_DO_NOT_SEND", "providerReference", "ticketCode", "sourceId", "decisionSnapshot"} {
		if strings.Contains(string(raw), private) {
			t.Fatalf("private field in preview: %s", private)
		}
	}
	if !strings.Contains(preview["body"].(string), "does not change your ticket") || !strings.Contains(preview["subject"].(string), "cancelled") {
		t.Fatalf("content=%#v", preview)
	}
	repeat := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 200).JSON)
	if repeat["previewHash"] != preview["previewHash"] || len(preview["previewHash"].(string)) != 64 {
		t.Fatal("unchanged review must have a stable digest")
	}
	insertTicket("new@example.test", "free")
	next := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 200).JSON)
	if next["previewHash"] == preview["previewHash"] {
		t.Fatal("recipient changes must invalidate digest")
	}
	if _, err := fx.app.db.Exec(ctx, `insert into email_suppressions(recipient_email,reason) values('guest@example.test','email.complained')`); err != nil {
		t.Fatal(err)
	}
	suppressed := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 200).JSON)
	if suppressed["previewHash"] == next["previewHash"] {
		t.Fatal("suppression changes must invalidate digest")
	}
	var after, attempts int
	if err := fx.app.db.QueryRow(ctx, `select count(*) from email_outbox`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(ctx, `select sum(attempt_count) from event_lifecycle_actions`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if after != before || attempts != 0 {
		t.Fatalf("preview enqueued mail or dispatched action: %d->%d attempts=%d", before, after, attempts)
	}
	if _, err := fx.app.db.Exec(ctx, `update event_lifecycle_changes set decision_snapshot=jsonb_set(decision_snapshot,'{expectedPublicCid}','null'::jsonb) where event_id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
}

func TestLifecycleNoticePreviewAuthorityRevisionAndLimits(t *testing.T) {
	fx, eventID, occurrence, path := noticePreviewFixture(t)
	req := map[string]any{"audiences": []string{"ticket_holders"}}
	postJSON(t, fx.app, fx.memberCookie, path, req, http.StatusForbidden)
	other := newLifecycleFixture(t, fx.app)
	postJSON(t, fx.app, other.ownerCookie, path, req, http.StatusForbidden)
	for _, audiences := range [][]string{nil, {}, {"contact_list"}, {"ticket_holders", "ticket_holders"}} {
		postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{"audiences": audiences}, 400)
	}
	ctx := t.Context()
	if _, err := fx.app.db.Exec(ctx, `insert into tickets(event_id,email,code) values($1,'invalid mailbox',$2)`, eventID, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
	if _, err := fx.app.db.Exec(ctx, `delete from tickets where event_id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(ctx, `insert into tickets(event_id,email,code) select $1,'guest'||g||'@example.test',gen_random_uuid()::text from generate_series(1,501) g`, eventID); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
	if _, err := fx.app.db.Exec(ctx, `delete from tickets where event_id=$1 and email='guest501@example.test'`, eventID); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 200)
	if _, err := fx.app.db.Exec(ctx, `update event_occurrences set public_cid='new-public-revision' where id=$1`, occurrence["id"]); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
	if _, err := fx.app.db.Exec(ctx, `update event_occurrences set public_cid=null,updated_at=updated_at+interval '1 second' where id=$1`, occurrence["id"]); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
	if _, err := fx.app.db.Exec(ctx, `update workspace_members set revoked_at=clock_timestamp() where workspace_id=$1 and person_id=$2`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 403)
	if _, err := fx.app.db.Exec(ctx, `update workspace_members set role='owner' where workspace_id=$1 and person_id=(select id from people where email=$2)`, fx.workspaceID, fx.email("member")); err != nil {
		t.Fatal(err)
	}
	response := postJSON(t, fx.app, fx.memberCookie, path, req, 409)
	if !strings.Contains(mustObject(t, response.JSON)["error"].(string), "active owner decision") {
		t.Fatal("a new owner must not preview an earlier revoked owner's decision")
	}
}

func TestLifecycleNoticePreviewAssignedCrewAndSavedStatus(t *testing.T) {
	fx, eventID, occurrence, path := noticePreviewFixture(t)
	ctx := t.Context()
	ownerID := ownerPersonID(t, fx)
	var memberID string
	if err := fx.app.db.QueryRow(ctx, `select id from people where email=$1`, fx.email("member")).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(ctx, `insert into event_staffing_items(event_id,title,kind,status,assigned_person_id,created_by_person_id,notes) values($1,'Door','task','assigned',$2,$3,'PRIVATE_STAFFING_NOTES')`, eventID, memberID, ownerID); err != nil {
		t.Fatal(err)
	}
	var roleID, applicationID string
	if err := fx.app.db.QueryRow(ctx, `insert into event_roles(event_id,name,created_by_person_id) values($1,'Crew',$2) returning id`, eventID, ownerID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(ctx, `insert into event_role_applications(event_id,role_id,applicant_name,applicant_email,status,message) values($1,$2,'Applicant','crew@example.test','accepted','PRIVATE_APPLICATION') returning id`, eventID, roleID).Scan(&applicationID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(ctx, `insert into event_staffing_items(event_id,title,kind,status,assigned_application_id,created_by_person_id) values($1,'Sound','task','assigned',$2,$3)`, eventID, applicationID, ownerID); err != nil {
		t.Fatal(err)
	}
	req := map[string]any{"audiences": []string{"assigned_crew"}}
	preview := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 200).JSON)
	if len(preview["recipients"].([]any)) != 2 {
		t.Fatalf("crew=%#v", preview)
	}
	if _, err := fx.app.db.Exec(ctx, `update event_role_applications set status='withdrawn' where id=$1`, applicationID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(ctx, `update workspace_members set expires_at=clock_timestamp()-interval '1 second' where workspace_id=$1 and person_id=$2`, fx.workspaceID, memberID); err != nil {
		t.Fatal(err)
	}
	preview = mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 200).JSON)
	if len(preview["recipients"].([]any)) != 0 {
		t.Fatal("withdrawn applicants and expired members must be excluded")
	}
	if _, err := fx.app.db.Exec(ctx, `update event_occurrences set status='scheduled' where id=$1`, occurrence["id"]); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
	if _, err := fx.app.db.Exec(ctx, `update event_occurrences set status='rescheduled',updated_at=clock_timestamp() where id=$1`, occurrence["id"]); err != nil {
		t.Fatal(err)
	}
	current := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", 200).JSON.([]any)[0])
	payload := lifecycleIntentPayload(current)
	payload["kind"] = "reschedule"
	intent := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, 201).JSON)
	reschedulePath := "/api/events/" + eventID + "/lifecycle-intents/" + intent["id"].(string) + "/notice-preview"
	preview = mustObject(t, postJSON(t, fx.app, fx.ownerCookie, reschedulePath, req, 200).JSON)
	if !strings.Contains(preview["body"].(string), "Listed start:") || !strings.Contains(preview["body"].(string), "America/Chicago") {
		t.Fatalf("reschedule content=%#v", preview)
	}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents/"+intent["id"].(string)+"/supersede", map[string]any{}, 200)
	postJSON(t, fx.app, fx.ownerCookie, reschedulePath, req, 409)
}
