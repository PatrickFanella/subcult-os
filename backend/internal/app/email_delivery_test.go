package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

func emailFixture(t *testing.T) *App {
	t.Helper()
	db := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return &App{db: db, config: Config{MailDeliveryEnabled: true, MailFrom: "notify@example.test", MailReplyTo: "reply@example.test"}}
}

func queueTestEmail(t *testing.T, a *App) string {
	t.Helper()
	if err := a.enqueueEmail(t.Context(), "guest@example.test", "Synthetic notice", "private token", "test", ""); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := a.db.QueryRow(t.Context(), `select id::text from email_outbox order by created_at desc limit 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestEmailMigrationHoldsLegacyAndDisabledMessages(t *testing.T) {
	db := newMigrationTestPool(t)
	migrations, err := loadMigrations(migrationFS)
	if err != nil {
		t.Fatal(err)
	}
	// Build a genuine version-five fixture without weakening the current
	// binary's minimum-version guard. Only this private test schema is touched.
	if _, err := db.Exec(t.Context(), `create table schema_migrations(version integer primary key,name text not null,checksum char(64) not null,applied_at timestamptz not null default now())`); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations[:5] {
		if _, err := db.Exec(t.Context(), m.SQL); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(t.Context(), `insert into schema_migrations(version,name,checksum) values($1,$2,$3)`, m.Version, m.Name, m.Checksum); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(t.Context(), `insert into email_outbox(recipient_email,subject,body,related_type) values('old@example.test','old','old token','test')`); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	a := &App{db: db}
	queueTestEmail(t, a)
	report, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) { t.Fatal("held mail sent"); return "", nil }, 10)
	if err != nil || report.Attempted != 0 {
		t.Fatalf("held=%+v err=%v", report, err)
	}
	if _, err := RunEmailDeliveries(t.Context(), Config{}, db, 1, false); err == nil {
		t.Fatal("disabled worker accepted sending")
	}
	status, err := RunEmailDeliveries(t.Context(), Config{}, db, 1, true)
	if err != nil || status.(map[string]int)["held"] != 2 {
		t.Fatalf("status=%v err=%v", status, err)
	}
}

func TestEmailRetriesPreserveIdentityAndFrozenPayload(t *testing.T) {
	a := emailFixture(t)
	id := queueTestEmail(t, a)
	a.config.MailFrom = "changed@example.test"
	report, err := a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		if m.ID != id || m.From != "notify@example.test" || m.Text != "private token" {
			t.Fatal("payload changed")
		}
		return "", &mailprovider.Failure{Code: "provider_retryable", Temporary: true, RetryAfter: time.Hour}
	}, 1)
	if err != nil || report.Retried != 1 {
		t.Fatalf("retry=%+v %v", report, err)
	}
	var delayed bool
	if err := a.db.QueryRow(t.Context(), `select next_attempt_at>now()+interval '59 minutes' from email_outbox`).Scan(&delayed); err != nil || !delayed {
		t.Fatal("retry-after lost")
	}
	_, err = a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) { t.Fatal("early retry"); return "", nil }, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(t.Context(), `update email_outbox set next_attempt_at=now()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	report, err = a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		if m.ID != id || m.From != "notify@example.test" {
			t.Fatal("retry identity changed")
		}
		return m.ID, nil
	}, 1)
	if err != nil || report.Accepted != 1 {
		t.Fatalf("accept=%+v %v", report, err)
	}
	var body, state string
	if err := a.db.QueryRow(t.Context(), `select body,delivery_status from email_outbox`).Scan(&body, &state); err != nil || body != "" || state != "accepted" {
		t.Fatalf("terminal payload=%q status=%s err=%v", body, state, err)
	}
}

func TestEmailTerminalOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, sql, want string
		failure         error
	}{
		{"permanent", "", "failed", &mailprovider.Failure{Code: "untrusted text"}},
		{"exhausted", `update email_outbox set attempts=7`, "quarantined", errors.New("private recipient")},
		{"expired", `update email_outbox set expires_at=now()-interval '1 second'`, "quarantined", nil},
		{"idempotency-horizon", `update email_outbox set first_attempt_at=now()-interval '23 hours'`, "quarantined", nil},
		{"final-crash", `update email_outbox set delivery_status='leased',attempts=8,lease_token=gen_random_uuid(),lease_until=now()-interval '1 second'`, "quarantined", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := emailFixture(t)
			queueTestEmail(t, a)
			if tc.sql != "" {
				if _, err := a.db.Exec(t.Context(), tc.sql); err != nil {
					t.Fatal(err)
				}
			}
			report, err := a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
				if tc.failure == nil {
					t.Fatal("expired job sent")
				}
				return "", tc.failure
			}, 1)
			if err != nil {
				t.Fatal(err)
			}
			var state, body, code string
			if err := a.db.QueryRow(t.Context(), `select delivery_status,body,last_error_code from email_outbox`).Scan(&state, &body, &code); err != nil {
				t.Fatal(err)
			}
			if state != tc.want || body != "" || code == "private recipient" || code == "untrusted text" {
				t.Fatalf("terminal=%s %q %s %+v", state, body, code, report)
			}
		})
	}
}

func TestEmailConcurrentClaimsAndLeaseFencing(t *testing.T) {
	a := emailFixture(t)
	queueTestEmail(t, a)
	entered, release := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	go func() {
		r, err := a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
			close(entered)
			<-release
			return m.ID, nil
		}, 1)
		if err == nil && r.Superseded != 1 {
			err = errors.New("stale acknowledgement accepted")
		}
		result <- err
	}()
	<-entered
	r, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Error("duplicate live lease")
		return "", nil
	}, 1)
	if err != nil || r.Attempted != 0 {
		close(release)
		t.Fatalf("claim=%+v %v", r, err)
	}
	if _, err := a.db.Exec(t.Context(), `update email_outbox set lease_until=now()-interval '1 second'`); err != nil {
		close(release)
		t.Fatal(err)
	}
	r, err = a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) { return m.ID, nil }, 1)
	close(release)
	if err != nil || r.Accepted != 1 {
		t.Fatalf("recovery=%+v %v", r, err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestEmailCrashAfterProviderAcceptanceKeepsStableRetry(t *testing.T) {
	a := emailFixture(t)
	id := queueTestEmail(t, a)
	ctx, cancel := context.WithCancel(t.Context())
	_, err := a.processEmailDeliveries(ctx, func(_ context.Context, m mailprovider.Message) (string, error) { cancel(); return m.ID, nil }, 1)
	if err == nil {
		t.Fatal("cancelled acknowledgement succeeded")
	}
	if _, err := a.db.Exec(t.Context(), `update email_outbox set lease_until=now()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	r, err := a.processEmailDeliveries(t.Context(), func(_ context.Context, m mailprovider.Message) (string, error) {
		if m.ID != id {
			t.Fatal("crash changed idempotency identity")
		}
		return m.ID, nil
	}, 1)
	if err != nil || r.Accepted != 1 {
		t.Fatalf("crash recovery=%+v %v", r, err)
	}
}

func TestEmailDeliveryConfiguration(t *testing.T) {
	t.Setenv("MAIL_DELIVERY_ENABLED", "not-a-boolean")
	if LoadConfig().Validate() == nil {
		t.Fatal("invalid enable flag accepted")
	}
	c := Config{Addr: ":8080", MailDeliveryEnabled: true, MailFrom: "notify@example.test", ResendAPIKey: "synthetic"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.MailFrom = ""
	if c.Validate() == nil {
		t.Fatal("missing sender accepted")
	}
}

func TestIdentityEmailDeadlineAndConsumedChallenge(t *testing.T) {
	a := newIdentityTestApp(t)
	a.config.MailDeliveryEnabled = true
	a.config.MailFrom = "notify@example.test"
	authJSON(t, a, http.MethodPost, "/api/auth/signup", map[string]any{"email": "deadline@example.test", "password": "secret1234"}, nil, http.StatusAccepted)
	var bounded bool
	if err := a.db.QueryRow(t.Context(), `select e.expires_at<=c.expires_at from email_outbox e join identity_challenges c on c.id=e.related_id`).Scan(&bounded); err != nil || !bounded {
		t.Fatalf("challenge deadline missing: %v", err)
	}
	if _, err := a.db.Exec(t.Context(), `update identity_challenges set consumed_at=now()`); err != nil {
		t.Fatal(err)
	}
	r, err := a.processEmailDeliveries(t.Context(), func(context.Context, mailprovider.Message) (string, error) {
		t.Fatal("consumed challenge sent")
		return "", nil
	}, 1)
	if err != nil || r.Quarantined != 1 || r.Attempted != 0 {
		t.Fatalf("consumed=%+v %v", r, err)
	}
}
