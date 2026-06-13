package app

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func (a *App) audit(ctx context.Context, actorPersonID, action, subjectType, subjectID string, metadata map[string]any) error {
	var actor sql.NullString
	if actorPersonID != "" {
		actor = sql.NullString{String: actorPersonID, Valid: true}
	}
	var subject sql.NullString
	if subjectID != "" {
		subject = sql.NullString{String: subjectID, Valid: true}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok && tx != nil {
		_, err = tx.Exec(ctx, `
			insert into audit_entries (actor_person_id, action, subject_type, subject_id, metadata)
			values ($1, $2, $3, $4, $5)
		`, actor, action, subjectType, subject, encoded)
		return err
	}

	_, err = a.db.Exec(ctx, `
		insert into audit_entries (actor_person_id, action, subject_type, subject_id, metadata)
		values ($1, $2, $3, $4, $5)
	`, actor, action, subjectType, subject, encoded)
	return err
}
