package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// Sentinel errors returned by checkSendPermission. Callers that only need
// an allow/deny decision can treat any non-nil error as deny; callers that
// need to distinguish the reason (for example to pick a delivery status
// or an HTTP status code) can use errors.Is.
var (
	// ErrConsentSuppressed means the recipient address is present in
	// email_suppressions (a prior bounce, complaint or explicit
	// suppression) and must never receive another message on this
	// channel, regardless of any grant.
	ErrConsentSuppressed = errors.New("recipient is suppressed and may not receive channel messages")
	// ErrConsentGrantRequired means the purpose requires a verified,
	// unwithdrawn consent grant for this workspace/channel/recipient and
	// none exists — because no grant was ever recorded, the recorded
	// grant is for a different purpose, it is not yet verified, or it has
	// been withdrawn. checkSendPermission deliberately does not
	// distinguish those cases further: an unverified or withdrawn grant
	// authorizes nothing, exactly like no grant at all.
	ErrConsentGrantRequired = errors.New("no verified, unwithdrawn consent grant authorizes this purpose")
)

// purposeTransactional never requires a grant: identity verification and
// recovery, ticket confirmations and workspace invitations are all
// transactional today (see docs/development/consent.md) and must keep
// sending unchanged. purposeAnnouncement requires a verified, unwithdrawn
// grant matching workspace, channel, recipient and purpose.
const (
	consentPurposeTransactional = "transactional"
	consentPurposeAnnouncement  = "announcement"
)

var consentChannels = map[string]bool{"email": true}
var consentSources = map[string]bool{"explicit_form": true, "operator_recorded": true}

// checkSendPermission is the single place that decides whether a message
// may be sent to recipient over channel for purpose, on behalf of
// workspaceID. It must be consulted both when a message is first
// requested and again, unchanged, immediately before a queued message
// actually leaves the system (processEmailDeliveries calls it per row),
// because consent can be withdrawn or a recipient suppressed after a
// message is enqueued and before it is delivered.
//
// It intentionally never joins against tickets, contacts,
// event_role_applications, atproto identity links or workspace_members:
// holding a ticket, being a contact, or being a workspace member never
// implies permission to send an announcement, no matter how well the
// workspace otherwise knows the address. See docs/development/consent.md.
func (a *App) checkSendPermission(ctx context.Context, workspaceID, channel, recipient, purpose string) error {
	recipient = normalizeEmail(recipient)

	var suppressed bool
	if err := a.db.QueryRow(ctx, `
		select exists(select 1 from email_suppressions where recipient_email = $1)
	`, recipient).Scan(&suppressed); err != nil {
		return err
	}
	if suppressed {
		return ErrConsentSuppressed
	}

	if purpose == consentPurposeTransactional {
		return nil
	}

	// An announcement can never be authorized without a workspace to scope
	// the grant to. Return the typed denial explicitly rather than letting
	// the query below fail on an empty workspaceID (which pgx would report
	// as a raw driver/encode error indistinguishable from an infra fault).
	if workspaceID == "" {
		return ErrConsentGrantRequired
	}

	var granted bool
	if err := a.db.QueryRow(ctx, `
		select exists(
			select 1 from consent_grants
			where workspace_id = $1
			  and channel = $2
			  and recipient_address = $3
			  and purpose = $4
			  and verified_at is not null
			  and withdrawn_at is null
		)
	`, workspaceID, channel, recipient, purpose).Scan(&granted); err != nil {
		return err
	}
	if !granted {
		return ErrConsentGrantRequired
	}
	return nil
}

// withdrawTokenHMACInfo is a fixed domain-separation label, not a secret:
// it only prevents this derivation from colliding with any other HMAC
// keyed on the same session secret elsewhere in this codebase.
const withdrawTokenHMACInfo = "consent-withdraw-token:v1:"

// deriveWithdrawToken computes the raw, per-grant public withdraw token
// used to build an announcement's withdraw link (see
// docs/development/announcements.md, "Withdraw link"). It is deliberately
// deterministic — HMAC-SHA256 keyed on the server's session secret, over
// the grant id — rather than a randomly drawn value, because
// verification_token_hash's raw token is a one-time secret that is never
// retained after the confirm/unsubscribe email is sent, and so cannot be
// reconstructed later to embed in a future announcement. A deterministic
// derivation lets the application recompute the same raw token whenever
// it needs one (at verification time, to mint withdraw_token_hash, and at
// announcement dispatch/send time, to build the link) without ever
// persisting the raw value itself. Only its SHA-256 hash
// (consent_grants.withdraw_token_hash) is stored, using the same
// hash-and-look-up pattern verification_token_hash already uses, so an
// attacker who reads the database still cannot derive a usable token
// without the session secret.
func deriveWithdrawToken(sessionSecret, grantID string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte(withdrawTokenHMACInfo + grantID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// mintWithdrawToken derives the deterministic withdraw token for grantID
// and returns both the raw token (to build a link) and its hash (to
// store in consent_grants.withdraw_token_hash for lookup by the public
// withdraw route).
func (a *App) mintWithdrawToken(grantID string) (raw, hashed string) {
	raw = deriveWithdrawToken(a.config.SessionSecret, grantID)
	return raw, tokenHash(raw)
}
