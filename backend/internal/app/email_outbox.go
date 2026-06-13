package app

import (
	"context"
	"database/sql"

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
