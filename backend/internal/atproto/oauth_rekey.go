package atproto

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// RekeyReport contains aggregate counts only: no DID, session identifier or
// ciphertext is ever included.
type RekeyReport struct {
	Scanned int `json:"scanned"`
	Rekeyed int `json:"rekeyed"`
}

// RekeySessions rewrites atproto_oauth_sessions payloads still sealed under
// the previous IDENTITY_PROTECTION_KEY so they are sealed under the current
// key. It processes at most limit rows per call, is safe to call repeatedly,
// and is a no-op (report all zero) once every row is already current or no
// previous key is configured.
func (s *OAuthStore) RekeySessions(ctx context.Context, limit int) (RekeyReport, error) {
	return s.rekeyPayloadTable(ctx, "atproto_oauth_sessions", "", limit)
}

// RekeyRevocations rewrites pending atproto_oauth_revocations payloads still
// sealed under the previous key. Leased rows are left alone so a concurrent
// revocation worker's in-flight lease is never disturbed; the next sweep
// picks them up once they return to pending.
func (s *OAuthStore) RekeyRevocations(ctx context.Context, limit int) (RekeyReport, error) {
	return s.rekeyPayloadTable(ctx, "atproto_oauth_revocations", "status = 'pending'", limit)
}

// PendingRekeyCounts reports, without decrypting anything, how many rows in
// each key-protected table still carry a payload (a rekey sweep decides for
// itself which of those need rewriting). It gives an operator a bound on
// remaining work before removing the previous key.
func (s *OAuthStore) PendingRekeyCounts(ctx context.Context) (map[string]int, error) {
	counts := map[string]int{}
	for _, table := range []string{"atproto_oauth_sessions", "atproto_oauth_revocations"} {
		var count int
		if err := s.db.QueryRow(ctx, fmt.Sprintf(`select count(*) from %s where payload_ciphertext is not null`, table)).Scan(&count); err != nil {
			return nil, fmt.Errorf("count AT OAuth payload rows: %w", err)
		}
		counts[table] = count
	}
	return counts, nil
}

func (s *OAuthStore) rekeyPayloadTable(ctx context.Context, table, extraWhere string, limit int) (RekeyReport, error) {
	report := RekeyReport{}
	if limit < 1 || limit > 1000 {
		return report, errors.New("rekey batch limit must be between 1 and 1000")
	}
	if s.prevAEAD == nil {
		return report, nil
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return report, fmt.Errorf("begin AT OAuth rekey: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	where := "payload_ciphertext is not null"
	if extraWhere != "" {
		where += " and " + extraWhere
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		select id, did, session_id, payload_ciphertext
		from %s
		where %s
		order by id
		limit $1
		for update skip locked
	`, table, where), limit)
	if err != nil {
		return report, fmt.Errorf("scan AT OAuth rekey candidates: %w", err)
	}
	type candidate struct {
		id, did, sessionID string
		payload            []byte
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.did, &c.sessionID, &c.payload); err != nil {
			rows.Close()
			return report, fmt.Errorf("read AT OAuth rekey candidate: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return report, fmt.Errorf("read AT OAuth rekey candidates: %w", err)
	}
	rows.Close()

	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		report.Scanned++
		aad := sessionAAD(syntax.DID(c.did), c.sessionID)
		if s.sealedWithCurrentKey(aad, c.payload) {
			continue
		}
		plaintext, err := openWithOAuthAEAD(s.prevAEAD, aad, c.payload)
		if err != nil {
			return report, errors.New("decrypt AT OAuth secret for rekey")
		}
		nonce := make([]byte, s.aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return report, fmt.Errorf("generate AT OAuth rekey nonce: %w", err)
		}
		newPayload := s.aead.Seal(nonce, nonce, plaintext, []byte(aad))
		if _, err := tx.Exec(ctx, fmt.Sprintf(`update %s set payload_ciphertext = $2, updated_at = $3 where id = $1`, table),
			c.id, newPayload, s.now()); err != nil {
			return report, fmt.Errorf("update AT OAuth rekeyed row: %w", err)
		}
		report.Rekeyed++
	}
	if err := tx.Commit(ctx); err != nil {
		return report, fmt.Errorf("commit AT OAuth rekey: %w", err)
	}
	return report, nil
}
