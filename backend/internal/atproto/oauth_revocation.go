package atproto

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRevocationUnsupported = errors.New("provider does not support token revocation")
	ErrRevocationPayload     = errors.New("invalid revocation credentials")
	ErrRevocationProvider    = errors.New("provider revocation failed")
)

// RevocationReport contains counts only: no identity, token, endpoint or raw
// provider error is exposed to logs or operational status output.
type RevocationReport struct {
	Attempted   int `json:"attempted"`
	Completed   int `json:"completed"`
	Retried     int `json:"retried"`
	Quarantined int `json:"quarantined"`
	Superseded  int `json:"superseded"`
}

// RevokeSession uses the SDK's confidential-client assertions and DPoP nonce
// handling with the same public-only, no-proxy, no-redirect outbound policy.
// Raw SDK errors can contain provider-controlled data and are never returned.
func (c *OAuthClient) RevokeSession(ctx context.Context, data oauth.ClientSessionData) error {
	return c.revokeSession(ctx, data, publicOnlyHTTPClient(10*time.Second))
}

func (c *OAuthClient) revokeSession(ctx context.Context, data oauth.ClientSessionData, client *http.Client) error {
	if data.AuthServerRevocationEndpoint == "" {
		return ErrRevocationUnsupported
	}
	if _, err := validateOAuthHTTPSURL(data.AuthServerRevocationEndpoint, "revocation endpoint"); err != nil {
		return ErrRevocationPayload
	}
	if _, err := validateOAuthHTTPSURL(data.AuthServerURL, "authorization server"); err != nil {
		return ErrRevocationPayload
	}
	if c == nil || data.AccessToken == "" || data.RefreshToken == "" || !identityOnlyScopes(data.Scopes) {
		return ErrRevocationPayload
	}
	key, err := atcrypto.ParsePrivateMultibase(data.DPoPPrivateKeyMultibase)
	if err != nil {
		return ErrRevocationPayload
	}
	session := oauth.ClientSession{Config: &c.config, Client: client, Data: &data, DPoPPrivateKey: key}
	if err := session.RevokeSession(ctx); err != nil {
		return ErrRevocationProvider
	}
	return nil
}

// ProcessRevocations claims bounded work with crash-recoverable leases. A
// provider timeout may follow success: retrying revocation must be harmless.
// Lease tokens fence old responses after lease expiry or a late token rotation.
func (s *OAuthStore) ProcessRevocations(ctx context.Context, revoke func(context.Context, oauth.ClientSessionData) error, limit int) (RevocationReport, error) {
	report := RevocationReport{}
	if revoke == nil || limit < 1 || limit > 100 {
		return report, errors.New("revocation processor requires handler and limit between 1 and 100")
	}
	// Expired credentials are purged, never silently marked remotely revoked.
	result, err := s.db.Exec(ctx, `
		with expired as (
			select id from atproto_oauth_revocations
			where payload_ciphertext is not null and (expires_at <= $1 or
			 (status='leased' and lease_until <= $1 and attempts >= 8))
			order by expires_at limit 100 for update skip locked
		)
		update atproto_oauth_revocations r
		set status='quarantined', payload_ciphertext=null, lease_token=null,
		    lease_until=null, last_error_code=case when expires_at <= $1
		      then 'retention_expired' else 'retry_exhausted' end, updated_at=$1
		from expired where r.id=expired.id
	`, s.now())
	if err != nil {
		return report, errors.New("expire AT OAuth revocation credentials")
	}
	report.Quarantined += int(result.RowsAffected())
	for range limit {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		var id, did, sessionID, lease string
		var payload []byte
		var attempts int
		err := s.db.QueryRow(ctx, `
			with candidate as (
				select id from atproto_oauth_revocations
				where expires_at > $1 and attempts < 8 and
				 ((status='pending' and next_attempt_at <= $1) or
				  (status='leased' and lease_until <= $1))
				order by next_attempt_at, id limit 1 for update skip locked
			)
			update atproto_oauth_revocations r
			set status='leased', attempts=r.attempts+1, lease_token=gen_random_uuid(),
			    lease_until=$1+interval '2 minutes', updated_at=$1
			from candidate where r.id=candidate.id
			returning r.id::text, r.did, r.session_id, r.payload_ciphertext,
			          r.attempts, r.lease_token::text
		`, s.now()).Scan(&id, &did, &sessionID, &payload, &attempts, &lease)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return report, errors.New("claim AT OAuth revocation")
		}
		report.Attempted++
		var data oauth.ClientSessionData
		callErr := s.open(sessionAAD(syntax.DID(did), sessionID), payload, &data)
		if callErr != nil || data.AccountDID.String() != did || data.SessionID != sessionID || !identityOnlyScopes(data.Scopes) {
			callErr = ErrRevocationPayload
		} else {
			callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			callErr = revoke(callCtx, data)
			cancel()
		}
		status, code := "completed", ""
		next := s.now()
		switch {
		case errors.Is(callErr, ErrRevocationUnsupported):
			status, code = "quarantined", "unsupported"
		case errors.Is(callErr, ErrRevocationPayload):
			status, code = "quarantined", "invalid_payload"
		case callErr != nil && attempts >= 8:
			status, code = "quarantined", "retry_exhausted"
		case callErr != nil:
			status, code = "pending", "provider_failed"
			next = next.Add(time.Minute * time.Duration(1<<(attempts-1)))
		}
		// If shutdown canceled the acknowledgement, the lease remains durable
		// and another worker retries it after expiry.
		result, err := s.db.Exec(ctx, `
			update atproto_oauth_revocations
			set status=$3, last_error_code=nullif($4,''), next_attempt_at=$5,
			    lease_token=null, lease_until=null, updated_at=$6,
			    payload_ciphertext=case when $3='pending' then payload_ciphertext else null end
			where id=$1 and lease_token=$2 and status='leased'
		`, id, lease, status, code, next, s.now())
		if err != nil {
			return report, errors.New("acknowledge AT OAuth revocation")
		}
		if result.RowsAffected() == 0 {
			report.Superseded++
			continue
		}
		switch status {
		case "completed":
			report.Completed++
		case "pending":
			report.Retried++
		case "quarantined":
			report.Quarantined++
		}
	}
	return report, nil
}

// RevocationCounts exposes aggregate operational state only. In particular,
// quarantined means local unlink succeeded but remote revocation is unproven.
func (s *OAuthStore) RevocationCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.Query(ctx, `select status, count(*) from atproto_oauth_revocations group by status`)
	if err != nil {
		return nil, fmt.Errorf("count AT OAuth revocations: %w", err)
	}
	defer rows.Close()
	counts := map[string]int{"pending": 0, "leased": 0, "completed": 0, "quarantined": 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}
