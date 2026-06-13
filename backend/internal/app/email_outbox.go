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
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok && tx != nil {
		_, err := tx.Exec(ctx, `
			insert into email_outbox (recipient_email, subject, body, related_type, related_id)
			values ($1, $2, $3, $4, $5)
		`, recipient, subject, body, relatedType, related)
		return err
	}

	_, err := a.db.Exec(ctx, `
		insert into email_outbox (recipient_email, subject, body, related_type, related_id)
		values ($1, $2, $3, $4, $5)
	`, recipient, subject, body, relatedType, related)
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
	if _, ok := a.requirePersonID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		select id, recipient_email, subject, body, related_type, related_id, created_at
		from email_outbox
		order by created_at desc
		limit 25
	`)
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
