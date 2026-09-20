package app

import (
	"context"
	"errors"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunATProtoRevocations is an explicit operator command, independent of whether
// new browser OAuth links are enabled. Disabling new links must not prevent
// revocation of outstanding credentials. It never applies migrations.
func RunATProtoRevocations(ctx context.Context, config Config, db *pgxpool.Pool, limit int, statusOnly bool) (any, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, errors.New("revocation command requires a database")
	}
	store, err := atprotocol.NewOAuthStore(db, config.IdentityProtectionKey, config.SessionSecret)
	if err != nil {
		return nil, err
	}
	if statusOnly {
		return store.RevocationCounts(ctx)
	}
	client, err := atprotocol.NewOAuthClient(config.atprotoOAuthSettings())
	if err != nil {
		return nil, errors.New("revocation command requires valid AT OAuth client configuration")
	}
	return store.ProcessRevocations(ctx, client.RevokeSession, limit)
}
