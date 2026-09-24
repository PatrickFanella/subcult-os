package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

// insertConsentGrant plants a consent_grants row directly, bypassing the
// HTTP layer, so tests can construct exactly the verified/withdrawn/purpose
// combination under test.
func insertConsentGrant(t *testing.T, fx lifecycleFixture, workspaceID, recipient, purpose, source string, verified bool) (grantID, token string) {
	t.Helper()
	token, hashed, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	var verifiedAt any
	if verified {
		verifiedAt = time.Now().UTC()
	}
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into consent_grants (
			workspace_id, channel, recipient_address, purpose, scope,
			verification_token_hash, verified_at, disclosure_version, source, created_by_person_id
		)
		values ($1, 'email', $2, $3, 'newsletter', $4, $5, 'v1', $6, $7)
		returning id
	`, workspaceID, normalizeEmail(recipient), purpose, hashed, verifiedAt, source, ownerPersonID(t, fx)).Scan(&grantID); err != nil {
		t.Fatal(err)
	}
	return grantID, token
}

func TestCheckSendPermissionTransactionalNeverRequiresGrant(t *testing.T) {
	fx := newLifecycleFixture(t)
	if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", "nobody-ever-granted@example.test", consentPurposeTransactional); err != nil {
		t.Fatalf("transactional send should never require a grant: %v", err)
	}
}

func TestCheckSendPermissionAnnouncementDenialReasons(t *testing.T) {
	fx := newLifecycleFixture(t)

	t.Run("unknown grant", func(t *testing.T) {
		err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", "never-granted+"+fx.suffix+"@example.test", consentPurposeAnnouncement)
		if !errors.Is(err, ErrConsentGrantRequired) {
			t.Fatalf("want ErrConsentGrantRequired, got %v", err)
		}
	})

	t.Run("wrong purpose", func(t *testing.T) {
		recipient := "transactional-only+" + fx.suffix + "@example.test"
		insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeTransactional, "operator_recorded", true)
		err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement)
		if !errors.Is(err, ErrConsentGrantRequired) {
			t.Fatalf("a transactional grant must not authorize announcement: %v", err)
		}
	})

	t.Run("unverified grant", func(t *testing.T) {
		recipient := "unverified+" + fx.suffix + "@example.test"
		insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", false)
		err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement)
		if !errors.Is(err, ErrConsentGrantRequired) {
			t.Fatalf("an unverified grant must not authorize announcement: %v", err)
		}
	})

	t.Run("verified grant allows announcement", func(t *testing.T) {
		recipient := "verified+" + fx.suffix + "@example.test"
		insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", true)
		if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement); err != nil {
			t.Fatalf("verified unwithdrawn grant should authorize announcement: %v", err)
		}
	})

	t.Run("withdrawn grant denies again", func(t *testing.T) {
		recipient := "withdrawn+" + fx.suffix + "@example.test"
		grantID, _ := insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", true)
		if _, err := fx.app.db.Exec(t.Context(), `update consent_grants set withdrawn_at = now(), withdrawal_reason = 'recipient_requested' where id = $1`, grantID); err != nil {
			t.Fatal(err)
		}
		err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement)
		if !errors.Is(err, ErrConsentGrantRequired) {
			t.Fatalf("a withdrawn grant must not authorize announcement: %v", err)
		}
	})
}

// TestCheckSendPermissionNeverConsultsUnrelatedTables plants a ticket
// holder, a contact and a workspace member all sharing the same address as
// the workspace being asked about, and proves checkSendPermission still
// denies an announcement without a real consent_grants row: none of those
// relationships may imply permission (docs/development/consent.md).
func TestCheckSendPermissionNeverConsultsUnrelatedTables(t *testing.T) {
	fx := newLifecycleFixture(t)
	shared := "shared-address+" + fx.suffix + "@example.test"

	event := createEvent(t, fx, "Consent boundary check", 10)
	eventID := mustString(t, event, "id")
	insertTicketWithPaymentStatus(t, fx, eventID, shared, "Ticket Holder", "paid", 1500, "usd")

	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/contacts", map[string]any{
		"displayName": "Shared Contact", "email": shared,
	}, http.StatusOK)

	// A workspace member's own account email is also, deliberately, the
	// shared address: membership must not imply consent either.
	memberCookie := signupAndVerifyCookie(t, fx.app, shared, "Shared Member")
	invite := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": shared}, http.StatusOK)
	postJSON(t, fx.app, memberCookie, "/api/invitations/"+mustString(t, invite.JSON, "token")+"/accept", map[string]any{}, http.StatusOK)

	err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", shared, consentPurposeAnnouncement)
	if !errors.Is(err, ErrConsentGrantRequired) {
		t.Fatalf("ticket/contact/membership must never grant announcement permission: %v", err)
	}
}

func TestCheckSendPermissionSuppressionOverridesEverything(t *testing.T) {
	fx := newLifecycleFixture(t)
	recipient := "suppressed+" + fx.suffix + "@example.test"
	insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", true)
	if _, err := fx.app.db.Exec(t.Context(), `insert into email_suppressions(recipient_email, reason) values ($1, 'email.bounced')`, normalizeEmail(recipient)); err != nil {
		t.Fatal(err)
	}

	if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement); !errors.Is(err, ErrConsentSuppressed) {
		t.Fatalf("a suppressed address must be denied even with a verified grant: %v", err)
	}
	if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeTransactional); !errors.Is(err, ErrConsentSuppressed) {
		t.Fatalf("a suppressed address must be denied for transactional purpose too: %v", err)
	}
}

// --- HTTP endpoint behavior --------------------------------------------

func TestConsentGrantHTTPLifecycle(t *testing.T) {
	fx := newLifecycleFixture(t)
	recipient := "http-lifecycle+" + fx.suffix + "@example.test"

	// A baseline "crew"/legacy member cannot create or list consent grants.
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/consent-grants", map[string]any{
		"channel": "email", "recipientAddress": recipient, "purpose": "announcement",
		"disclosureVersion": "v1", "source": "explicit_form",
	}, http.StatusForbidden)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/consent-grants", map[string]any{
		"channel": "email", "recipientAddress": recipient, "purpose": "announcement",
		"disclosureVersion": "v1", "source": "explicit_form",
	}, http.StatusOK)
	if mustString(t, created.JSON, "status") != "unverified" {
		t.Fatalf("expected unverified status, got %+v", created.JSON)
	}

	// A second request for the same active tuple conflicts.
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/consent-grants", map[string]any{
		"channel": "email", "recipientAddress": recipient, "purpose": "announcement",
		"disclosureVersion": "v1", "source": "explicit_form",
	}, http.StatusConflict)

	// Not yet verified: still denied.
	if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement); !errors.Is(err, ErrConsentGrantRequired) {
		t.Fatalf("unverified grant must not authorize yet: %v", err)
	}

	token := latestIdentityToken(t, fx.app, "consent_verification")
	postJSON(t, fx.app, nil, "/api/public/consent/"+token+"/confirm", map[string]any{}, http.StatusOK)

	if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement); err != nil {
		t.Fatalf("confirmed grant should authorize announcement: %v", err)
	}

	list := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/consent-grants", http.StatusOK)
	rows, ok := list.JSON.([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected exactly one listed grant, got %+v", list.JSON)
	}
	row := mustObject(t, rows[0])
	if row["recipientAddress"] != normalizeEmail(recipient) {
		t.Fatalf("unexpected recipient in listing: %+v", row)
	}
	if _, leaked := row["verificationTokenHash"]; leaked {
		t.Fatalf("verification token hash must never be listed: %+v", row)
	}

	// Unsubscribe via the same token, without any session.
	req := postJSON(t, fx.app, nil, "/api/public/consent/"+token+"/withdraw", map[string]any{}, http.StatusOK)
	if mustString(t, req.JSON, "status") != "withdrawn" {
		t.Fatalf("expected withdrawn status: %+v", req.JSON)
	}
	// Idempotent: withdrawing again still succeeds.
	postJSON(t, fx.app, nil, "/api/public/consent/"+token+"/withdraw", map[string]any{}, http.StatusOK)

	if err := fx.app.checkSendPermission(t.Context(), fx.workspaceID, "email", recipient, consentPurposeAnnouncement); !errors.Is(err, ErrConsentGrantRequired) {
		t.Fatalf("withdrawn grant must deny announcement: %v", err)
	}

	// A confirm attempt after withdrawal reveals nothing and fails closed.
	postJSON(t, fx.app, nil, "/api/public/consent/"+token+"/confirm", map[string]any{}, http.StatusNotFound)
}

func TestConsentPublicTokenRoutesRevealNothingForUnknownTokens(t *testing.T) {
	fx := newLifecycleFixture(t)
	postJSON(t, fx.app, nil, "/api/public/consent/not-a-real-token/confirm", map[string]any{}, http.StatusNotFound)
	postJSON(t, fx.app, nil, "/api/public/consent/not-a-real-token/withdraw", map[string]any{}, http.StatusNotFound)
}

// --- Send-time recheck ---------------------------------------------------

func insertQueuedOutboxMessage(t *testing.T, fx lifecycleFixture, workspaceID, recipient, purpose string) string {
	t.Helper()
	var id string
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into email_outbox (
			recipient_email, subject, body, related_type, delivery_status,
			sender_address, reply_to_address, purpose, workspace_id
		)
		values ($1, 'Announcement', 'private announcement body', 'test', 'pending', 'notify@example.test', 'reply@example.test', $2, $3)
		returning id::text
	`, recipient, purpose, workspaceID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func consentDeliveryFixture(t *testing.T) lifecycleFixture {
	t.Helper()
	fx := newLifecycleFixture(t)
	fx.app.config.MailDeliveryEnabled = true
	fx.app.config.MailFrom = "notify@example.test"
	fx.app.config.MailReplyTo = "reply@example.test"
	return fx
}

func TestProcessEmailDeliveriesRevokeBeforeSend(t *testing.T) {
	fx := consentDeliveryFixture(t)
	recipient := "revoke-before-send+" + fx.suffix + "@example.test"
	grantID, _ := insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", true)
	insertQueuedOutboxMessage(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement)

	if _, err := fx.app.db.Exec(t.Context(), `update consent_grants set withdrawn_at = now(), withdrawal_reason = 'recipient_requested' where id = $1`, grantID); err != nil {
		t.Fatal(err)
	}

	report, err := fx.app.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Fatal("withdrawn-grant message must never reach the provider")
		return "", nil
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if report.Withheld != 1 || report.Accepted != 0 {
		t.Fatalf("expected exactly one withheld message, got %+v", report)
	}
	var status, body, code string
	if err := fx.app.db.QueryRow(t.Context(), `select delivery_status, body, last_error_code from email_outbox where recipient_email = $1`, normalizeEmail(recipient)).Scan(&status, &body, &code); err != nil {
		t.Fatal(err)
	}
	if status != "withheld_consent" || body != "" || code != "consent_required" {
		t.Fatalf("status=%q body=%q code=%q", status, body, code)
	}
}

func TestProcessEmailDeliveriesWithholdsWrongPurposeGrant(t *testing.T) {
	fx := consentDeliveryFixture(t)
	recipient := "wrong-purpose+" + fx.suffix + "@example.test"
	insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeTransactional, "operator_recorded", true)
	insertQueuedOutboxMessage(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement)

	report, err := fx.app.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Fatal("wrong-purpose grant must never authorize an announcement send")
		return "", nil
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if report.Withheld != 1 {
		t.Fatalf("expected withheld=1, got %+v", report)
	}
}

func TestProcessEmailDeliveriesWithholdsUnknownGrant(t *testing.T) {
	fx := consentDeliveryFixture(t)
	recipient := "unknown-grant+" + fx.suffix + "@example.test"
	insertQueuedOutboxMessage(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement)

	report, err := fx.app.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Fatal("an address with no grant at all must never receive an announcement")
		return "", nil
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if report.Withheld != 1 {
		t.Fatalf("expected withheld=1, got %+v", report)
	}
}

func TestProcessEmailDeliveriesSendsVerifiedAnnouncementGrant(t *testing.T) {
	fx := consentDeliveryFixture(t)
	recipient := "verified-send+" + fx.suffix + "@example.test"
	insertConsentGrant(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement, "explicit_form", true)
	insertQueuedOutboxMessage(t, fx, fx.workspaceID, recipient, consentPurposeAnnouncement)

	sent := false
	report, err := fx.app.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		sent = true
		return m.ID, nil
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !sent || report.Accepted != 1 || report.Withheld != 0 {
		t.Fatalf("expected the verified grant to authorize sending, got sent=%v report=%+v", sent, report)
	}
}

func TestProcessEmailDeliveriesUnaffectedForTransactionalRows(t *testing.T) {
	// Legacy/default rows keep sending exactly as before this slice.
	fx := consentDeliveryFixture(t)
	id := queueTestEmail(t, fx.app)
	sent := false
	report, err := fx.app.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		sent = true
		if m.ID != id {
			t.Fatalf("unexpected message id %s", m.ID)
		}
		return m.ID, nil
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !sent || report.Accepted != 1 {
		t.Fatalf("expected the transactional row to send unchanged, got sent=%v report=%+v", sent, report)
	}
}
