package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailDeliveryReport struct {
	Attempted   int `json:"attempted"`
	Accepted    int `json:"accepted"`
	Retried     int `json:"retried"`
	Failed      int `json:"failed"`
	Quarantined int `json:"quarantined"`
	Superseded  int `json:"superseded"`
	Suppressed  int `json:"suppressed"`
	Withheld    int `json:"withheld"`
}

func RunEmailDeliveries(ctx context.Context, config Config, db *pgxpool.Pool, limit int, statusOnly bool) (any, error) {
	if db == nil {
		return nil, errors.New("email worker requires a database")
	}
	if version, err := CurrentSchemaVersion(ctx, db); err != nil || version != minimumSchemaVersion {
		return nil, errors.New("email worker requires current migrations")
	}
	if statusOnly {
		rows, err := db.Query(ctx, `select delivery_status,count(*) from email_outbox group by delivery_status`)
		if err != nil {
			return nil, errors.New("email status unavailable")
		}
		defer rows.Close()
		counts := map[string]int{}
		for rows.Next() {
			var state string
			var count int
			if err := rows.Scan(&state, &count); err != nil {
				return nil, errors.New("email status unavailable")
			}
			counts[state] = count
		}
		return counts, rows.Err()
	}
	if !config.MailDeliveryEnabled {
		return nil, errors.New("mail delivery is disabled")
	}
	if err := config.Validate(); err != nil {
		return nil, errors.New("invalid email worker configuration")
	}
	provider, err := mailprovider.NewResend(config.ResendAPIKey)
	if err != nil {
		return nil, err
	}
	a := &App{db: db, config: config}
	return a.processEmailDeliveries(ctx, provider.Send, limit)
}

func (a *App) processEmailDeliveries(ctx context.Context, send func(context.Context, mailprovider.Message) (string, error), limit int) (EmailDeliveryReport, error) {
	report := EmailDeliveryReport{}
	if send == nil || limit < 1 || limit > 100 {
		return report, errors.New("email batch limit must be 1..100")
	}
	if err := a.reconcileEmailFeedback(ctx); err != nil {
		return report, err
	}
	suppressed, err := a.db.Exec(ctx, `with blocked as (
	 select e.id from email_outbox e join email_suppressions s on s.recipient_email=lower(trim(e.recipient_email))
	 where e.delivery_status='pending' or (e.delivery_status='leased' and e.lease_until<=now())
	 order by e.id limit 100 for update of e skip locked)
	 update email_outbox e set delivery_status='suppressed',body='',lease_token=null,lease_until=null,last_error_code='recipient_suppressed'
	 from blocked where e.id=blocked.id`)
	if err != nil {
		return report, errors.New("email suppression unavailable")
	}
	report.Suppressed = int(suppressed.RowsAffected())
	// Never retry after the provider's 24-hour idempotency retention. The
	// 23-hour local limit leaves margin and also bounds late crash recovery.
	expired, err := a.db.Exec(ctx, `with expired as (
	 select id from email_outbox where delivery_status in ('pending','leased') and
	 (expires_at<=now() or first_attempt_at<=now()-interval '23 hours' or (attempts>=8 and lease_until<=now()) or
	 exists(select 1 from identity_challenges c where c.id=email_outbox.related_id and c.consumed_at is not null))
	 order by expires_at limit 100 for update skip locked)
	 update email_outbox e set delivery_status='quarantined',body='',lease_token=null,lease_until=null,last_error_code='retry_window_closed'
	 from expired where e.id=expired.id`)
	if err != nil {
		return report, errors.New("email expiration failed")
	}
	report.Quarantined = int(expired.RowsAffected())
	for range limit {
		var message mailprovider.Message
		var lease, purpose string
		var attempts int
		var workspaceID sql.NullString
		err := a.db.QueryRow(ctx, `with candidate as (
		 select id from email_outbox where expires_at>now() and attempts<8 and
		 not exists(select 1 from email_suppressions s where s.recipient_email=lower(trim(email_outbox.recipient_email))) and
		 not exists(select 1 from email_provider_events p join email_outbox prior on prior.provider_message_id=p.provider_message_id
		 where p.processed_at is null and p.event_type in ('email.bounced','email.complained','email.suppressed')
		 and lower(trim(prior.recipient_email))=lower(trim(email_outbox.recipient_email))) and
		 not exists(select 1 from identity_challenges c where c.id=email_outbox.related_id and c.consumed_at is not null) and
		 (first_attempt_at is null or first_attempt_at>now()-interval '23 hours') and
		 ((delivery_status='pending' and next_attempt_at<=now()) or (delivery_status='leased' and lease_until<=now()))
		 order by next_attempt_at,id limit 1 for update skip locked)
		 update email_outbox e set delivery_status='leased',attempts=e.attempts+1,
		 first_attempt_at=coalesce(e.first_attempt_at,now()),lease_token=gen_random_uuid(),lease_until=now()+interval '2 minutes'
		 from candidate where e.id=candidate.id
		 returning e.id::text,e.sender_address,e.reply_to_address,e.recipient_email,e.subject,e.body,e.lease_token::text,e.attempts,e.purpose,e.workspace_id::text`).Scan(
			&message.ID, &message.From, &message.ReplyTo, &message.To, &message.Subject, &message.Text, &lease, &attempts, &purpose, &workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return report, errors.New("email claim failed")
		}
		report.Attempted++
		// Recheck consent/suppression immediately before this queued
		// message actually leaves the system: a grant may have been
		// withdrawn, or the address suppressed, since it was enqueued.
		// See docs/development/consent.md and checkSendPermission.
		checkPermission := a.checkSendPermission
		if a.consentCheckOverride != nil {
			checkPermission = a.consentCheckOverride
		}
		if permErr := checkPermission(ctx, workspaceID.String, "email", message.To, purpose); permErr != nil {
			// Only a genuine consent decision (a typed denial) may move this
			// row to the terminal withheld_consent status. Any other error
			// (a transient database failure, a cancelled context, or a
			// driver-level fault) must not be recorded as a permanent
			// consent withholding, and must not erase the row's body: leave
			// the lease to expire so the row is retried, exactly like the
			// claim-failure path above.
			if !errors.Is(permErr, ErrConsentGrantRequired) && !errors.Is(permErr, ErrConsentSuppressed) {
				return report, errors.New("email consent check failed")
			}
			code := "consent_required"
			if errors.Is(permErr, ErrConsentSuppressed) {
				code = "recipient_suppressed"
			}
			ack, ackErr := a.db.Exec(ctx, `update email_outbox set delivery_status='withheld_consent',last_error_code=$3,
			 lease_token=null,lease_until=null,body=''
			 where id=$1 and lease_token=$2 and delivery_status='leased'`, message.ID, lease, code)
			if ackErr != nil {
				return report, errors.New("email consent withholding failed")
			}
			if ack.RowsAffected() == 0 {
				report.Superseded++
				continue
			}
			report.Withheld++
			continue
		}
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		providerID, callErr := send(callCtx, message)
		cancel()
		status, code := "accepted", ""
		delay := time.Minute * time.Duration(1<<(attempts-1))
		if callErr != nil {
			status, code = "pending", "provider_failed"
			var failure *mailprovider.Failure
			if errors.As(callErr, &failure) {
				if !failure.Temporary {
					status, code = "failed", "provider_rejected"
				}
				if failure.RetryAfter > delay {
					delay = failure.RetryAfter
				}
			}
			if status == "pending" && attempts >= 8 {
				status, code = "quarantined", "retry_exhausted"
			}
			providerID = ""
		} else if providerID == "" {
			status, code = "quarantined", "missing_provider_id"
		}
		if delay > time.Hour {
			delay = time.Hour
		}
		ack, err := a.db.Exec(ctx, `update email_outbox set delivery_status=$3,last_error_code=nullif($4,''),
		 next_attempt_at=now()+$5*interval '1 second',lease_token=null,lease_until=null,
		 provider_message_id=nullif($6,'')::uuid,accepted_at=case when $3='accepted' then now() else accepted_at end,
		 body=case when $3='pending' then body else '' end
		 where id=$1 and lease_token=$2 and delivery_status='leased'`, message.ID, lease, status, code, int(delay.Seconds()), providerID)
		if err != nil {
			return report, errors.New("email acknowledgement failed")
		}
		if ack.RowsAffected() == 0 {
			report.Superseded++
			continue
		}
		if err := a.reconcileEmailFeedback(ctx); err != nil {
			return report, err
		}
		switch status {
		case "accepted":
			report.Accepted++
		case "pending":
			report.Retried++
		case "failed":
			report.Failed++
		case "quarantined":
			report.Quarantined++
		}
	}
	return report, nil
}
