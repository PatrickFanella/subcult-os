package atproto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	atprotocoloauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	oauthProtectionKeyBytes = 32
	oauthRequestLifetime    = 10 * time.Minute
)

var (
	ErrOAuthLinkContext     = errors.New("AT OAuth link requires an authenticated local person")
	ErrOAuthRequestNotFound = errors.New("AT OAuth request is missing, expired, or already claimed")
	ErrOAuthSessionNotFound = errors.New("AT OAuth session not found")
	ErrOAuthScopeRejected   = errors.New("AT OAuth session has non-identity scope")
)

type oauthLinkPersonContextKey struct{}

type OAuthStore struct {
	db        *pgxpool.Pool
	aead      cipher.AEAD
	lookupKey []byte
	now       func() time.Time
}

// WithOAuthLinkPerson binds an authenticated local account to one OAuth start
// request. It never infers account equality from email, handle, or DID.
func WithOAuthLinkPerson(ctx context.Context, personID string) context.Context {
	return context.WithValue(ctx, oauthLinkPersonContextKey{}, strings.TrimSpace(personID))
}

// NewOAuthStore constructs the encrypted persistence adapter required by
// Indigo's ClientAuthStore. Production uses the same deployment root secret as
// canonical identity storage, with a distinct derived key domain.
func NewOAuthStore(db *pgxpool.Pool, encodedKey, developmentSeed string) (*OAuthStore, error) {
	if db == nil {
		return nil, errors.New("AT OAuth store requires a database")
	}
	key, err := decodeOAuthProtectionKey(encodedKey)
	if err != nil {
		if encodedKey != "" {
			return nil, err
		}
		keySum := sha256.Sum256([]byte("subcult-os/development-atproto/" + developmentSeed))
		key = keySum[:]
	}
	block, err := aes.NewCipher(deriveOAuthKey(key, "payload-encryption"))
	if err != nil {
		return nil, fmt.Errorf("create AT OAuth cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AT OAuth AEAD: %w", err)
	}
	return &OAuthStore{
		db: db, aead: aead,
		lookupKey: deriveOAuthKey(key, "state-lookup"),
		now:       func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *OAuthStore) SaveAuthRequestInfo(ctx context.Context, info atprotocoloauth.AuthRequestData) error {
	personID, _ := ctx.Value(oauthLinkPersonContextKey{}).(string)
	if personID == "" {
		return ErrOAuthLinkContext
	}
	if !identityOnlyScopes(info.Scopes) {
		return ErrOAuthScopeRejected
	}
	if strings.TrimSpace(info.State) == "" {
		return errors.New("AT OAuth state is required")
	}
	payload, err := s.seal("request:"+info.State, info)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `
		insert into atproto_oauth_requests (
			state_hash, person_id, payload_ciphertext, expires_at
		) values ($1, $2, $3, $4)
	`, s.stateHash(info.State), personID, payload, s.now().Add(oauthRequestLifetime)); err != nil {
		return fmt.Errorf("save AT OAuth request: %w", err)
	}
	return nil
}

// GetAuthRequestInfo atomically claims a request. A callback state can be read
// only once, even if concurrent callbacks race. Expiry is enforced by SQL.
func (s *OAuthStore) GetAuthRequestInfo(ctx context.Context, state string) (*atprotocoloauth.AuthRequestData, error) {
	var payload []byte
	err := s.db.QueryRow(ctx, `
		update atproto_oauth_requests
		set claimed_at = $2
		where state_hash = $1
		  and claimed_at is null
		  and expires_at > $2
		returning payload_ciphertext
	`, s.stateHash(state), s.now()).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthRequestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("claim AT OAuth request: %w", err)
	}
	var info atprotocoloauth.AuthRequestData
	if err := s.open("request:"+state, payload, &info); err != nil {
		return nil, err
	}
	if info.State != state {
		return nil, errors.New("AT OAuth request state mismatch")
	}
	return &info, nil
}

func (s *OAuthStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	if _, err := s.db.Exec(ctx, `delete from atproto_oauth_requests where state_hash = $1`, s.stateHash(state)); err != nil {
		return fmt.Errorf("delete AT OAuth request: %w", err)
	}
	return nil
}

// DeleteExpiredRequests removes request secrets that can no longer be claimed.
// It is safe to call concurrently with callback processing because expiry and
// claiming are decided by the database.
func (s *OAuthStore) DeleteExpiredRequests(ctx context.Context) (int64, error) {
	result, err := s.db.Exec(ctx, `delete from atproto_oauth_requests where expires_at <= $1`, s.now())
	if err != nil {
		return 0, fmt.Errorf("delete expired AT OAuth requests: %w", err)
	}
	return result.RowsAffected(), nil
}

func (s *OAuthStore) SaveSession(ctx context.Context, session atprotocoloauth.ClientSessionData) error {
	if !identityOnlyScopes(session.Scopes) {
		return ErrOAuthScopeRejected
	}
	if session.AccountDID == "" || strings.TrimSpace(session.SessionID) == "" {
		return errors.New("AT OAuth session DID and ID are required")
	}
	parsedDID, err := syntax.ParseDID(session.AccountDID.String())
	if err != nil || parsedDID != session.AccountDID {
		return errors.New("AT OAuth session DID is invalid")
	}
	payload, err := s.seal(sessionAAD(session.AccountDID, session.SessionID), session)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin AT OAuth session save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var personID string
	existingSession := true
	err = tx.QueryRow(ctx, `
		select person_id
		from atproto_oauth_sessions
		where did = $1 and session_id = $2
	`, session.AccountDID.String(), session.SessionID).Scan(&personID)
	if errors.Is(err, pgx.ErrNoRows) {
		existingSession = false
		err = tx.QueryRow(ctx, `
			select person_id
			from atproto_oauth_requests
			where state_hash = $1 and claimed_at is not null and expires_at > $2
			for update
		`, s.stateHash(session.SessionID), s.now()).Scan(&personID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOAuthLinkContext
	}
	if err != nil {
		return fmt.Errorf("resolve AT OAuth session person: %w", err)
	}

	if existingSession {
		var active bool
		if err := tx.QueryRow(ctx, `
			select person_id = $1 and status = 'active'
			from did_links where did = $2
		`, personID, session.AccountDID.String()).Scan(&active); err != nil {
			return fmt.Errorf("verify AT OAuth DID link: %w", err)
		}
		if !active {
			return errors.New("AT OAuth DID link is not active")
		}
	} else {
		result, err := tx.Exec(ctx, `
			insert into did_links (person_id, did, status, verified_at)
			values ($1, $2, 'active', $3)
			on conflict (did) do update
			set status = 'active', verified_at = excluded.verified_at,
				revoked_at = null, updated_at = excluded.verified_at
			where did_links.person_id = excluded.person_id
		`, personID, session.AccountDID.String(), s.now())
		if err != nil {
			return fmt.Errorf("link AT OAuth DID: %w", err)
		}
		if result.RowsAffected() != 1 {
			return errors.New("AT OAuth DID is linked to another account")
		}
	}

	if _, err := tx.Exec(ctx, `
		insert into atproto_oauth_sessions (
			person_id, did, session_id, payload_ciphertext
		) values ($1, $2, $3, $4)
		on conflict (did, session_id) do update
		set payload_ciphertext = excluded.payload_ciphertext, updated_at = $5
		where atproto_oauth_sessions.person_id = excluded.person_id
	`, personID, session.AccountDID.String(), session.SessionID, payload, s.now()); err != nil {
		return fmt.Errorf("save AT OAuth session: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from atproto_oauth_requests where state_hash = $1`, s.stateHash(session.SessionID)); err != nil {
		return fmt.Errorf("consume AT OAuth request: %w", err)
	}
	if !existingSession {
		if _, err := tx.Exec(ctx, `
			insert into auth_audit_events (person_id, event_type)
			values ($1, 'atproto_did_linked')
		`, personID); err != nil {
			return fmt.Errorf("audit AT OAuth DID link: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit AT OAuth session: %w", err)
	}
	return nil
}

func (s *OAuthStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*atprotocoloauth.ClientSessionData, error) {
	var payload []byte
	err := s.db.QueryRow(ctx, `
		select payload_ciphertext
		from atproto_oauth_sessions
		where did = $1 and session_id = $2
	`, did.String(), sessionID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load AT OAuth session: %w", err)
	}
	var session atprotocoloauth.ClientSessionData
	if err := s.open(sessionAAD(did, sessionID), payload, &session); err != nil {
		return nil, err
	}
	if session.AccountDID != did || session.SessionID != sessionID {
		return nil, errors.New("AT OAuth session identity mismatch")
	}
	return &session, nil
}

func (s *OAuthStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	if _, err := s.db.Exec(ctx, `
		delete from atproto_oauth_sessions where did = $1 and session_id = $2
	`, did.String(), sessionID); err != nil {
		return fmt.Errorf("delete AT OAuth session: %w", err)
	}
	return nil
}

func identityOnlyScopes(scopes []string) bool {
	return len(scopes) == 1 && scopes[0] == "atproto"
}

func sessionAAD(did syntax.DID, sessionID string) string {
	return "session:" + did.String() + ":" + sessionID
}

func (s *OAuthStore) seal(aad string, value any) ([]byte, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode AT OAuth secret: %w", err)
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate AT OAuth nonce: %w", err)
	}
	return s.aead.Seal(nonce, nonce, plaintext, []byte(aad)), nil
}

func (s *OAuthStore) open(aad string, ciphertext []byte, value any) error {
	nonceSize := s.aead.NonceSize()
	if len(ciphertext) <= nonceSize {
		return errors.New("invalid encrypted AT OAuth secret")
	}
	plaintext, err := s.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], []byte(aad))
	if err != nil {
		return errors.New("decrypt AT OAuth secret")
	}
	if err := json.Unmarshal(plaintext, value); err != nil {
		return fmt.Errorf("decode AT OAuth secret: %w", err)
	}
	return nil
}

func (s *OAuthStore) stateHash(state string) string {
	mac := hmac.New(sha256.New, s.lookupKey)
	_, _ = mac.Write([]byte(state))
	return hex.EncodeToString(mac.Sum(nil))
}

func deriveOAuthKey(master []byte, purpose string) []byte {
	mac := hmac.New(sha256.New, master)
	_, _ = mac.Write([]byte("subcult-os/atproto-oauth/" + purpose))
	return mac.Sum(nil)
}

func decodeOAuthProtectionKey(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, errors.New("AT OAuth protection key is required")
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		decoded, err := encoding.DecodeString(encoded)
		if err == nil && len(decoded) == oauthProtectionKeyBytes {
			return decoded, nil
		}
	}
	return nil, errors.New("AT OAuth protection key must decode to 32 bytes")
}

var _ atprotocoloauth.ClientAuthStore = (*OAuthStore)(nil)
