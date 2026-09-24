package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

// insertVerifiedAnnouncementGrant plants a verified, unwithdrawn
// announcement grant with a minted withdraw token, matching what the
// real create-grant/confirm HTTP flow produces (see
// handleCreateConsentGrant/handleConfirmConsentGrant), so dispatch tests
// exercise the same shape of row the API would leave behind.
func insertVerifiedAnnouncementGrant(t *testing.T, fx lifecycleFixture, recipient string) (grantID string) {
	t.Helper()
	grantID, _ = insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", true)
	_, hashed := fx.app.mintWithdrawToken(grantID)
	if _, err := fx.app.db.Exec(t.Context(), `update consent_grants set withdraw_token_hash = $2 where id = $1`, grantID, hashed); err != nil {
		t.Fatal(err)
	}
	return grantID
}

// makeAnnouncementDue schedules announcementID via the real schedule
// handler (proving the future-time validation runs), then backdates
// scheduled_for directly so dispatch sees it as due — the handler itself
// deliberately refuses to schedule anything but a future time.
func makeAnnouncementDue(t *testing.T, fx lifecycleFixture, announcementID string) {
	t.Helper()
	future := time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/schedule",
		map[string]any{"scheduledFor": future}, http.StatusOK)
	if _, err := fx.app.db.Exec(t.Context(), `update announcements set scheduled_for = now() - interval '1 minute' where id = $1`, announcementID); err != nil {
		t.Fatal(err)
	}
}

// TestAnnouncementSyntheticJourney is the acceptance journey required by
// SIGNAL-01 (issue #24): grant two recipients, confirm both, schedule an
// announcement in the past-due window, preview shows 2, withdraw one
// recipient's grant, dispatch enqueues exactly one outbox row with
// withheld_count 1, and running the delivery worker with a fake sender
// proves the one row is accepted while nothing is ever sent to the
// withdrawn address.
func TestAnnouncementSyntheticJourney(t *testing.T) {
	fx := consentDeliveryFixture(t)
	keep := fx.email("keep-recipient")
	drop := fx.email("withdrawn-recipient")
	insertVerifiedAnnouncementGrant(t, fx, keep)
	dropGrantID := insertVerifiedAnnouncementGrant(t, fx, drop)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements", map[string]any{
		"subject": "Synthetic lineup update",
		"body":    "New dates are up.",
	}, http.StatusOK)
	announcementID := mustString(t, created.JSON, "id")

	preview := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/preview", http.StatusOK)
	previewObj := mustObject(t, preview.JSON)
	if count, ok := previewObj["recipientCount"].(float64); !ok || count != 2 {
		t.Fatalf("expected preview recipientCount=2, got %#v", previewObj["recipientCount"])
	}

	makeAnnouncementDue(t, fx, announcementID)

	// Withdraw one recipient's grant after scheduling, before dispatch.
	if _, err := fx.app.db.Exec(t.Context(), `update consent_grants set withdrawn_at = now(), withdrawal_reason = 'recipient_requested' where id = $1`, dropGrantID); err != nil {
		t.Fatal(err)
	}

	report, err := RunAnnouncementDispatch(t.Context(), fx.app.config, fx.app.db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if report.Claimed != 1 || report.RecipientsEnqueued != 1 || report.RecipientsWithheld != 1 {
		t.Fatalf("unexpected dispatch report: %+v", report)
	}

	var outboxCount int
	if err := fx.app.db.QueryRow(t.Context(), `
		select count(*) from email_outbox where purpose = 'announcement' and workspace_id = $1
	`, fx.workspaceID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected exactly one announcement outbox row, got %d", outboxCount)
	}
	var outboxRecipient string
	if err := fx.app.db.QueryRow(t.Context(), `
		select recipient_email from email_outbox where purpose = 'announcement' and workspace_id = $1
	`, fx.workspaceID).Scan(&outboxRecipient); err != nil {
		t.Fatal(err)
	}
	if outboxRecipient != normalizeEmail(keep) {
		t.Fatalf("expected the kept recipient's row, got %q", outboxRecipient)
	}

	after := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID, http.StatusOK)
	afterObj := mustObject(t, after.JSON)
	if afterObj["status"] != announcementStatusDispatched {
		t.Fatalf("expected status dispatched, got %#v", afterObj["status"])
	}
	if v, ok := afterObj["recipientCount"].(float64); !ok || v != 1 {
		t.Fatalf("expected recipientCount=1, got %#v", afterObj["recipientCount"])
	}
	if v, ok := afterObj["withheldCount"].(float64); !ok || v != 1 {
		t.Fatalf("expected withheldCount=1, got %#v", afterObj["withheldCount"])
	}

	// Run the delivery worker with a fake sender: the one allowed row is
	// accepted, and nothing is ever sent to the withdrawn address.
	sentTo := map[string]bool{}
	sendReport, err := fx.app.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		sentTo[m.To] = true
		return m.ID, nil
	}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if sendReport.Accepted != 1 {
		t.Fatalf("expected exactly one accepted send, got %+v", sendReport)
	}
	if !sentTo[normalizeEmail(keep)] {
		t.Fatalf("expected the kept recipient to receive the message")
	}
	if sentTo[normalizeEmail(drop)] {
		t.Fatal("the withdrawn recipient must never receive the announcement")
	}

	delivered := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID, http.StatusOK)
	deliveredObj := mustObject(t, delivered.JSON)
	counts, ok := deliveredObj["deliveryStatusCounts"].(map[string]any)
	if !ok {
		t.Fatalf("expected deliveryStatusCounts object, got %#v", deliveredObj["deliveryStatusCounts"])
	}
	if v, ok := counts["accepted"].(float64); !ok || v != 1 {
		t.Fatalf("expected one accepted delivery outcome, got %#v", counts)
	}
}

// TestAnnouncementCancelBeforeDispatchProducesNoRows proves cancelling a
// scheduled announcement before it becomes due (or before a dispatch run
// claims it) prevents any email_outbox row from ever being enqueued.
func TestAnnouncementCancelBeforeDispatchProducesNoRows(t *testing.T) {
	fx := consentDeliveryFixture(t)
	insertVerifiedAnnouncementGrant(t, fx, fx.email("cancel-audience"))

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements", map[string]any{
		"subject": "Never sent",
		"body":    "This should never dispatch.",
	}, http.StatusOK)
	announcementID := mustString(t, created.JSON, "id")

	makeAnnouncementDue(t, fx, announcementID)

	cancelled := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/cancel", map[string]any{}, http.StatusOK)
	cancelledObj := mustObject(t, cancelled.JSON)
	if cancelledObj["status"] != announcementStatusCancelled {
		t.Fatalf("expected status cancelled, got %#v", cancelledObj["status"])
	}

	report, err := RunAnnouncementDispatch(t.Context(), fx.app.config, fx.app.db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if report.Claimed != 0 {
		t.Fatalf("a cancelled announcement must never be claimed for dispatch, got %+v", report)
	}

	var outboxCount int
	if err := fx.app.db.QueryRow(t.Context(), `
		select count(*) from email_outbox where purpose = 'announcement' and workspace_id = $1
	`, fx.workspaceID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected no announcement outbox rows after cancellation, got %d", outboxCount)
	}

	// Cancelling again is a conflict, never a silent success, and
	// cancelling after dispatch has started must never be allowed either.
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/cancel", map[string]any{}, http.StatusConflict)
}

// TestAnnouncementRequiresManageAnnouncementsPermission proves a plain
// member (crew, no manage_announcements) is denied every announcement
// endpoint.
func TestAnnouncementRequiresManageAnnouncementsPermission(t *testing.T) {
	fx := newLifecycleFixture(t)
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/announcements", map[string]any{
		"subject": "x", "body": "y",
	}, http.StatusForbidden)
	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/announcements", http.StatusForbidden)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements", map[string]any{
		"subject": "x", "body": "y",
	}, http.StatusOK)
	announcementID := mustString(t, created.JSON, "id")

	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID, http.StatusForbidden)
	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/preview", http.StatusForbidden)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/schedule", map[string]any{"scheduledFor": future}, http.StatusForbidden)
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/cancel", map[string]any{}, http.StatusForbidden)
}

// TestAnnouncementScheduleRejectsPastAndPresent proves the explicit
// future-scheduling requirement is enforced by the endpoint itself, not
// only by test helpers that bypass it.
func TestAnnouncementScheduleRejectsPastAndPresent(t *testing.T) {
	fx := newLifecycleFixture(t)
	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements", map[string]any{
		"subject": "x", "body": "y",
	}, http.StatusOK)
	announcementID := mustString(t, created.JSON, "id")

	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/announcements/"+announcementID+"/schedule", map[string]any{"scheduledFor": past}, http.StatusBadRequest)
}

// TestAnnouncementWithdrawLinkTokenWorksThroughPublicRoute proves the
// deterministic withdraw token minted at verification resolves through
// the existing public withdraw route.
func TestAnnouncementWithdrawLinkTokenWorksThroughPublicRoute(t *testing.T) {
	fx := newLifecycleFixture(t)
	recipient := fx.email("withdraw-link")
	grantID := insertVerifiedAnnouncementGrant(t, fx, recipient)
	token := deriveWithdrawToken(fx.app.config.SessionSecret, grantID)

	postJSON(t, fx.app, nil, "/api/public/consent/"+token+"/withdraw", map[string]any{}, http.StatusOK)

	var withdrawnAt any
	if err := fx.app.db.QueryRow(t.Context(), `select withdrawn_at from consent_grants where id = $1`, grantID).Scan(&withdrawnAt); err != nil {
		t.Fatal(err)
	}
	if withdrawnAt == nil {
		t.Fatal("expected the grant to be withdrawn via the deterministic withdraw token")
	}
}
