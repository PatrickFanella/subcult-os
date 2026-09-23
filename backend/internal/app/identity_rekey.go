package app

import (
	"context"
	"errors"
	"fmt"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdentityRekeyReport contains aggregate counts only: no email, ciphertext,
// DID or session identifier is ever included, matching the operational
// status conventions used by the AT OAuth revocation worker.
type IdentityRekeyReport struct {
	EmailIdentitiesScanned  int `json:"emailIdentitiesScanned"`
	EmailIdentitiesRekeyed  int `json:"emailIdentitiesRekeyed"`
	OAuthSessionsScanned    int `json:"oauthSessionsScanned"`
	OAuthSessionsRekeyed    int `json:"oauthSessionsRekeyed"`
	OAuthRevocationsScanned int `json:"oauthRevocationsScanned"`
	OAuthRevocationsRekeyed int `json:"oauthRevocationsRekeyed"`
}

// IdentityRekeyStatus reports how many key-protected rows remain, without
// decrypting or rewriting anything.
type IdentityRekeyStatus struct {
	PreviousKeyConfigured    bool `json:"previousKeyConfigured"`
	EmailIdentitiesTotal     int  `json:"emailIdentitiesTotal"`
	OAuthSessionsWithPayload int  `json:"oauthSessionsWithPayload"`
	OAuthRevocationsPending  int  `json:"oauthRevocationsPending"`
}

// RunIdentityRekey is the explicit, resumable IDENTITY_PROTECTION_KEY
// rotation command. It requires both IDENTITY_PROTECTION_KEY (current) and,
// to do any rewriting, IDENTITY_PROTECTION_KEY_PREVIOUS (the retiring key).
// It processes at most limit rows per table per call and is idempotent: a
// row already sealed under the current key is left untouched, so rerunning
// it (for example under -watch) is always safe. It never logs or returns an
// email, DID, session identifier or any ciphertext -- only counts.
func RunIdentityRekey(ctx context.Context, config Config, db *pgxpool.Pool, limit int, statusOnly bool) (any, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, errors.New("identity rekey command requires a database")
	}
	if limit < 1 || limit > 1000 {
		return nil, errors.New("identity rekey batch limit must be between 1 and 1000")
	}
	identity, err := newIdentityProtector(config.IdentityProtectionKey, config.IdentityProtectionKeyPrevious, config.SessionSecret)
	if err != nil {
		return nil, err
	}
	oauthStore, err := atprotocol.NewOAuthStore(db, config.IdentityProtectionKey, config.IdentityProtectionKeyPrevious, config.SessionSecret)
	if err != nil {
		return nil, err
	}

	if statusOnly {
		status := IdentityRekeyStatus{PreviousKeyConfigured: identity.prevAEAD != nil}
		if err := db.QueryRow(ctx, `select count(*) from email_identities`).Scan(&status.EmailIdentitiesTotal); err != nil {
			return nil, fmt.Errorf("count email identities: %w", err)
		}
		oauthCounts, err := oauthStore.PendingRekeyCounts(ctx)
		if err != nil {
			return nil, err
		}
		status.OAuthSessionsWithPayload = oauthCounts["atproto_oauth_sessions"]
		status.OAuthRevocationsPending = oauthCounts["atproto_oauth_revocations"]
		return status, nil
	}

	report := IdentityRekeyReport{}
	scanned, rekeyed, err := rekeyEmailIdentities(ctx, db, identity, limit)
	if err != nil {
		return nil, err
	}
	report.EmailIdentitiesScanned, report.EmailIdentitiesRekeyed = scanned, rekeyed

	sessions, err := oauthStore.RekeySessions(ctx, limit)
	if err != nil {
		return nil, err
	}
	report.OAuthSessionsScanned, report.OAuthSessionsRekeyed = sessions.Scanned, sessions.Rekeyed

	revocations, err := oauthStore.RekeyRevocations(ctx, limit)
	if err != nil {
		return nil, err
	}
	report.OAuthRevocationsScanned, report.OAuthRevocationsRekeyed = revocations.Scanned, revocations.Rekeyed
	return report, nil
}

// rekeyEmailIdentities claims a bounded batch of email_identities rows with
// FOR UPDATE SKIP LOCKED (safe to run concurrently with normal traffic and
// with another rekey run), decrypts each with the current key first, then
// the previous key, and rewrites only the rows that were not already
// current. It never returns or logs plaintext email or ciphertext.
func rekeyEmailIdentities(ctx context.Context, db *pgxpool.Pool, identity *identityProtector, limit int) (scanned, rekeyed int, err error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("begin identity rekey: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		select id, email_ciphertext
		from email_identities
		order by id
		limit $1
		for update skip locked
	`, limit)
	if err != nil {
		return 0, 0, fmt.Errorf("scan identity rekey candidates: %w", err)
	}
	type candidate struct {
		id         string
		ciphertext []byte
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.ciphertext); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("read identity rekey candidate: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("read identity rekey candidates: %w", err)
	}
	rows.Close()

	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return scanned, rekeyed, err
		}
		scanned++
		newCiphertext, newLookupHash, rotated, err := identity.reencryptEmail(c.ciphertext)
		if err != nil {
			return scanned, rekeyed, errors.New("decrypt identity secret for rekey")
		}
		if !rotated {
			continue
		}
		if _, err := tx.Exec(ctx, `
			update email_identities
			set email_ciphertext = $2, email_lookup_hash = $3, updated_at = now()
			where id = $1
		`, c.id, newCiphertext, newLookupHash); err != nil {
			return scanned, rekeyed, fmt.Errorf("update rekeyed identity: %w", err)
		}
		rekeyed++
	}
	if err := tx.Commit(ctx); err != nil {
		return scanned, rekeyed, fmt.Errorf("commit identity rekey: %w", err)
	}
	return scanned, rekeyed, nil
}
