package app

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

func TestBrevoConfigRequiresAuthenticatedFeedback(t *testing.T) {
	c := Config{Addr: ":8080", MailProvider: "brevo", MailDeliveryEnabled: true, MailFrom: "subcult-os@subcult.tv", BrevoAPIKey: "synthetic"}
	if c.Validate() == nil {
		t.Fatal("sending without feedback authentication enabled")
	}
	c.BrevoWebhookToken = strings.Repeat("x", 32)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.MailProvider = "unknown"
	if c.Validate() == nil {
		t.Fatal("unknown provider allowed")
	}
}

func TestBrevoFeedbackAuthenticatesWithoutDatabase(t *testing.T) {
	a := New(Config{Addr: ":8080", BrevoWebhookToken: strings.Repeat("x", 32)}, nil)
	for _, tc := range []struct {
		header, body string
		status       int
	}{
		{"", `{}`, 401}, {"Bearer wrong", `{}`, 401},
		{"Bearer " + strings.Repeat("x", 32), `{"event":"delivered","message-id":"bad","ts_event":1}`, 400},
		{"Bearer " + strings.Repeat("x", 32), `{"event":"opened"}`, 204},
		{"Bearer " + strings.Repeat("x", 32), `{"event":"delivered","message-id":"<valid@relay.test>","ts_event":1}`, 429},
	} {
		req := httptest.NewRequest("POST", "/api/brevo/webhook", strings.NewReader(tc.body))
		req.Header.Set("Authorization", tc.header)
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("status=%d want=%d", w.Code, tc.status)
		}
	}
}

func TestBrevoOutboxAndEarlyFeedbackPreserveSuppression(t *testing.T) {
	a := emailFixture(t)
	legacy := queueTestEmail(t, a)
	a.config.MailProvider = "brevo"
	a.config.BrevoWebhookToken = strings.Repeat("x", 32)
	id := queueTestEmail(t, a)
	body := `{"event":"hard_bounce","message-id":"<123@relay.test>","ts_event":1,"email":"untrusted@example.test"}`
	for range 2 {
		r := httptest.NewRequest("POST", "/api/brevo/webhook", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+a.config.BrevoWebhookToken)
		w := httptest.NewRecorder()
		New(a.config, a.db).Handler().ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatal(w.Code)
		}
	}
	report, err := a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		if m.ID != id {
			t.Fatalf("wrong-provider row sent: %s", m.ID)
		}
		return "brevo:123@relay.test", nil
	}, 10)
	if err != nil || report.Accepted != 1 {
		t.Fatalf("report=%+v %v", report, err)
	}
	var recipient, state string
	if err := a.db.QueryRow(t.Context(), `select recipient_email from email_suppressions`).Scan(&recipient); err != nil || recipient != "guest@example.test" {
		t.Fatalf("suppression=%s %v", recipient, err)
	}
	if err := a.db.QueryRow(t.Context(), `select delivery_status from email_outbox where id=$1`, legacy).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "pending" {
		t.Fatal(fmt.Sprintf("legacy row modified: %s", state))
	}
}

func TestBrevoRetryWindowQuarantinesBeforeSend(t *testing.T) {
	a := emailFixture(t)
	a.config.MailProvider = "brevo"
	id := queueTestEmail(t, a)
	if _, err := a.db.Exec(t.Context(), `update email_outbox set first_attempt_at=now()-interval '30 minutes',attempts=1 where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	report, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Fatal("expired retry sent")
		return "", nil
	}, 1)
	if err != nil || report.Quarantined != 1 || report.Attempted != 0 {
		t.Fatalf("report=%+v %v", report, err)
	}
}
