package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
	"github.com/google/uuid"
)

func emailFeedbackRank(kind string) int {
	switch kind {
	case "email.delivered":
		return 1
	case "email.failed":
		return 2
	case "email.suppressed":
		return 3
	case "email.bounced":
		return 4
	case "email.complained":
		return 5
	}
	return 0
}

func (a *App) handleResendWebhook(w http.ResponseWriter, r *http.Request) {
	if a.config.ResendWebhookSecret == "" {
		writeError(w, 404, "not found")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	if err != nil {
		writeError(w, 413, "webhook body too large")
		return
	}
	id := r.Header.Get("svix-id")
	if !mailprovider.VerifyWebhook(a.config.ResendWebhookSecret, id, r.Header.Get("svix-timestamp"), r.Header.Get("svix-signature"), body, time.Now()) {
		writeError(w, 400, "invalid webhook")
		return
	}
	var event struct {
		Type string `json:"type"`
		Data struct {
			ID string `json:"email_id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &event) != nil {
		writeError(w, 400, "invalid webhook")
		return
	}
	if emailFeedbackRank(event.Type) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	providerID, err := uuid.Parse(event.Data.ID)
	if err != nil || len(event.Data.ID) != 36 {
		writeError(w, 400, "invalid webhook")
		return
	}
	if a.db == nil {
		writeError(w, 503, "webhook storage unavailable")
		return
	}
	result, err := a.db.Exec(r.Context(), `insert into email_provider_events(event_id,provider_message_id,event_type) values($1,$2,$3)
 on conflict(event_id) do update set event_id=excluded.event_id
 where email_provider_events.provider_message_id=excluded.provider_message_id and email_provider_events.event_type=excluded.event_type`, id, providerID.String(), event.Type)
	if err != nil {
		writeError(w, 503, "webhook storage unavailable")
		return
	}
	if result.RowsAffected() != 1 {
		writeError(w, 409, "conflicting webhook identifier")
		return
	}
	if err = a.reconcileEmailFeedback(r.Context()); err != nil {
		writeError(w, 503, "webhook processing unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Persist early receipts until the worker has recorded the provider message ID.
// Reconciliation only uses our own recipient, never an address from a webhook.
func (a *App) reconcileEmailFeedback(ctx context.Context) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return errors.New("email feedback unavailable")
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `select p.event_id,p.event_type,e.id::text,e.recipient_email
 from email_provider_events p join email_outbox e on e.provider_message_id=p.provider_message_id
 where p.processed_at is null order by p.received_at,p.event_id limit 100 for update of p skip locked`)
	if err != nil {
		return errors.New("email feedback unavailable")
	}
	type receipt struct{ id, kind, message, recipient string }
	var receipts []receipt
	for rows.Next() {
		var v receipt
		if err := rows.Scan(&v.id, &v.kind, &v.message, &v.recipient); err != nil {
			rows.Close()
			return errors.New("email feedback unavailable")
		}
		receipts = append(receipts, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return errors.New("email feedback unavailable")
	}
	for _, v := range receipts {
		rank := emailFeedbackRank(v.kind)
		if _, err = tx.Exec(ctx, `update email_outbox set feedback_rank=greatest(feedback_rank,$2) where id=$1`, v.message, rank); err != nil {
			return errors.New("email feedback unavailable")
		}
		if rank >= 3 {
			if _, err = tx.Exec(ctx, `insert into email_suppressions(recipient_email,reason) values(lower(trim($1)),$2) on conflict(recipient_email) do nothing`, v.recipient, v.kind); err != nil {
				return errors.New("email feedback unavailable")
			}
		}
		if _, err = tx.Exec(ctx, `update email_provider_events set processed_at=now() where event_id=$1`, v.id); err != nil {
			return errors.New("email feedback unavailable")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return errors.New("email feedback unavailable")
	}
	return nil
}
