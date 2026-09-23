package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

const testWebhookSecret = "whsec_plJ3nmyCDGBKInavdOK15jsl"
const testProviderID = "56761188-7520-42d8-8898-ff6fc54ce618"

func postEmailFeedback(a *App, id, body string, valid bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/resend/webhook", strings.NewReader(body))
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	key, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(testWebhookSecret, "whsec_"))
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "." + body))
	sig := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !valid {
		sig = "v1,invalid"
	}
	req.Header.Set("svix-id", id)
	req.Header.Set("svix-timestamp", timestamp)
	req.Header.Set("svix-signature", sig)
	res := httptest.NewRecorder()
	New(a.config, a.db).Handler().ServeHTTP(res, req)
	return res
}

func feedbackBody(kind, id string) string {
	return fmt.Sprintf(`{"type":%q,"data":{"email_id":%q,"to":["untrusted@example.test"],"subject":"DO NOT RETAIN"}}`, kind, id)
}

func TestEmailFeedbackRejectsInvalidAndDisabled(t *testing.T) {
	a := emailFixture(t)
	body := feedbackBody("email.bounced", testProviderID)
	if got := postEmailFeedback(a, "msg_disabled", body, true).Code; got != 404 {
		t.Fatalf("disabled=%d", got)
	}
	a.config.ResendWebhookSecret = testWebhookSecret
	for _, tt := range []struct {
		name, body string
		valid      bool
		want       int
	}{
		{"forged", body, false, 400}, {"oversized", strings.Repeat("x", 65537), true, 413},
		{"bad json", "{", true, 400}, {"bad uuid", feedbackBody("email.bounced", "invalid"), true, 400},
		{"ignored event", feedbackBody("email.opened", testProviderID), true, 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := postEmailFeedback(a, "msg_invalid", tt.body, tt.valid).Code; got != tt.want {
				t.Fatalf("status=%d want %d", got, tt.want)
			}
		})
	}
	var count int
	if err := a.db.QueryRow(t.Context(), `select count(*) from email_provider_events`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("persisted invalid event %d %v", count, err)
	}
}

func TestEmailFeedbackEarlyBounceAndOutOfOrderDelivery(t *testing.T) {
	a := emailFixture(t)
	a.config.ResendWebhookSecret = testWebhookSecret
	id := queueTestEmail(t, a)
	body := feedbackBody("email.bounced", testProviderID)
	for range 2 {
		if got := postEmailFeedback(a, "msg_bounce", body, true).Code; got != 204 {
			t.Fatalf("early receipt=%d", got)
		}
	}
	var count int
	if err := a.db.QueryRow(t.Context(), `select count(*) from email_suppressions`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("unknown provider suppressed: %d %v", count, err)
	}
	result, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) { return testProviderID, nil }, 1)
	if err != nil || result.Accepted != 1 {
		t.Fatalf("accept=%+v %v", result, err)
	}
	if got := postEmailFeedback(a, "msg_delivered", feedbackBody("email.delivered", testProviderID), true).Code; got != 204 {
		t.Fatalf("delivered=%d", got)
	}
	if got := postEmailFeedback(a, "msg_bounce", feedbackBody("email.complained", testProviderID), true).Code; got != 409 {
		t.Fatalf("conflict=%d", got)
	}
	var rank int
	var state, recipient string
	if err := a.db.QueryRow(t.Context(), `select feedback_rank,delivery_status from email_outbox where id=$1`, id).Scan(&rank, &state); err != nil || rank != 4 || state != "accepted" {
		t.Fatalf("outcome=%d %s %v", rank, state, err)
	}
	if err := a.db.QueryRow(t.Context(), `select recipient_email from email_suppressions`).Scan(&recipient); err != nil || recipient != "guest@example.test" {
		t.Fatalf("suppression=%s %v", recipient, err)
	}
	if err := a.db.QueryRow(t.Context(), `select count(*) from email_provider_events`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("dedup=%d %v", count, err)
	}
	next := queueTestEmail(t, a)
	report, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Fatal("suppressed address sent")
		return "", nil
	}, 10)
	if err != nil || report.Suppressed != 1 || report.Attempted != 0 {
		t.Fatalf("suppression report=%+v %v", report, err)
	}
	var text string
	if err := a.db.QueryRow(t.Context(), `select delivery_status,body from email_outbox where id=$1`, next).Scan(&state, &text); err != nil || state != "suppressed" || text != "" {
		t.Fatalf("suppressed row=%s bodylen=%d %v", state, len(text), err)
	}
}

func TestEmailFeedbackOutcomePolicy(t *testing.T) {
	for _, kind := range []string{"email.delivered", "email.failed", "email.suppressed", "email.complained"} {
		t.Run(kind, func(t *testing.T) {
			a := emailFixture(t)
			a.config.ResendWebhookSecret = testWebhookSecret
			id := queueTestEmail(t, a)
			if _, err := a.db.Exec(t.Context(), `update email_outbox set delivery_status='accepted',provider_message_id=$2 where id=$1`, id, testProviderID); err != nil {
				t.Fatal(err)
			}
			if got := postEmailFeedback(a, "msg_policy", feedbackBody(kind, testProviderID), true).Code; got != 204 {
				t.Fatalf("status=%d", got)
			}
			var rank, count int
			if err := a.db.QueryRow(t.Context(), `select feedback_rank from email_outbox where id=$1`, id).Scan(&rank); err != nil || rank != emailFeedbackRank(kind) {
				t.Fatalf("rank=%d %v", rank, err)
			}
			if err := a.db.QueryRow(t.Context(), `select count(*) from email_suppressions`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 0
			if emailFeedbackRank(kind) >= 3 {
				want = 1
			}
			if count != want {
				t.Fatalf("suppression=%d want%d", count, want)
			}
		})
	}
}

func TestEmailFeedbackBacklogFailsClosed(t *testing.T) {
	a := emailFixture(t)
	prior := queueTestEmail(t, a)
	if _, err := a.db.Exec(t.Context(), `update email_outbox set delivery_status='accepted',provider_message_id=$2 where id=$1`, prior, testProviderID); err != nil {
		t.Fatal(err)
	}
	// Older harmless receipts fill the bounded reconciliation batch. The pending
	// adverse receipt must still prevent a new attempt for the same recipient.
	if _, err := a.db.Exec(t.Context(), `insert into email_provider_events(event_id,provider_message_id,event_type,received_at)
	 select 'msg_backlog_'||n,$1::uuid,'email.delivered',now()-interval '1 minute' from generate_series(1,100) n`, testProviderID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(t.Context(), `insert into email_provider_events(event_id,provider_message_id,event_type) values('msg_late_bounce',$1,'email.bounced')`, testProviderID); err != nil {
		t.Fatal(err)
	}
	queueTestEmail(t, a)
	for range 2 {
		report, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
			t.Fatal("backlogged adverse receipt allowed send")
			return "", nil
		}, 1)
		if err != nil || report.Attempted != 0 {
			t.Fatalf("report=%+v %v", report, err)
		}
	}
}

func TestEmailWebhookConfiguration(t *testing.T) {
	c := Config{Addr: ":8080", MailDeliveryEnabled: true, MailFrom: "notice@example.test", ResendAPIKey: "synthetic"}
	if c.Validate() == nil {
		t.Fatal("sending without feedback secret accepted")
	}
	c.MailDeliveryEnabled = false
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.ResendWebhookSecret = "invalid-secret-value"
	if err := c.Validate(); err == nil || strings.Contains(err.Error(), c.ResendWebhookSecret) {
		t.Fatal("invalid secret accepted or leaked")
	}
	c.ResendWebhookSecret = testWebhookSecret
	if err := c.Validate(); err != nil {
		t.Fatal("disabled sending must still accept valid webhook config")
	}
}
