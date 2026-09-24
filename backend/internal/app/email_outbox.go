package app

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type txContextKey struct{}

func (a *App) enqueueEmail(ctx context.Context, recipient, subject, body, relatedType, relatedID string) error {
	var related sql.NullString
	if relatedID != "" {
		related = sql.NullString{String: relatedID, Valid: true}
	}
	status := "held"
	if a.config.MailDeliveryEnabled {
		status = "pending"
	}
	const insert = `insert into email_outbox
		(recipient_email, subject, body, related_type, related_id, delivery_status, sender_address, reply_to_address, expires_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,
		 least(now()+interval '23 hours', coalesce((select expires_at from identity_challenges where id=$5),now()+interval '23 hours')))`
	args := []any{recipient, subject, body, relatedType, related, status, a.config.MailFrom, a.config.MailReplyTo}
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok && tx != nil {
		_, err := tx.Exec(ctx, insert, args...)
		return err
	}
	_, err := a.db.Exec(ctx, insert, args...)
	return err
}

type emailOutboxDTO struct {
	ID             string  `json:"id"`
	RecipientEmail string  `json:"recipientEmail"`
	Subject        string  `json:"subject"`
	Body           string  `json:"body"`
	RelatedType    string  `json:"relatedType"`
	RelatedID      *string `json:"relatedId"`
	CreatedAt      string  `json:"createdAt"`
}

func (a *App) handleDevEmailOutbox(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	if strings.EqualFold(a.config.AppEnv, "production") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// The development outbox is a debugging aid, not a mail admin view. A
	// signed-in person may only see messages addressed to their own account
	// email, their own identity challenges, or messages that belong to a
	// workspace they are currently a member of (invitations and ticket
	// confirmations for that workspace's events). Nothing else is listed,
	// so one tenant cannot read another tenant's plaintext recipients.
	rows, err := a.db.Query(r.Context(), `
		with caller as (
			select lower(email) as email from people where id = $1
		), memberships as (
			select workspace_id from workspace_members
			where person_id = $1 and removed_at is null
		)
		select o.id, o.recipient_email, o.subject, o.body, o.related_type, o.related_id, o.created_at
		from email_outbox o
		where lower(o.recipient_email) = (select email from caller)
		   or (o.related_type in ('identity_verification', 'identity_recovery')
		       and o.related_id in (select id from identity_challenges where person_id = $1))
		   or (o.related_type = 'workspace_invitation'
		       and o.related_id in (select id from workspace_invitations where workspace_id in (select workspace_id from memberships)))
		   or (o.related_type = 'ticket'
		       and o.related_id in (
		         select t.id from tickets t join events e on e.id = t.event_id
		         where e.workspace_id in (select workspace_id from memberships)))
		order by o.created_at desc
		limit 25
	`, personID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load email outbox")
		return
	}
	defer rows.Close()

	out := make([]emailOutboxDTO, 0)
	for rows.Next() {
		var item emailOutboxDTO
		var relatedID sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.RecipientEmail, &item.Subject, &item.Body, &item.RelatedType, &relatedID, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not scan email outbox")
			return
		}
		if relatedID.Valid {
			item.RelatedID = &relatedID.String
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read email outbox")
		return
	}

	writeJSON(w, http.StatusOK, out)
}
