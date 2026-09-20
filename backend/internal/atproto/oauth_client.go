package atproto

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	atprotocoloauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
)

type OAuthClientSettings struct {
	ClientID          string
	CallbackURL       string
	JWKSURL           string
	PrivateKey        string
	KeyID             string
	UserAgent         string
	ClientName        string
	ClientHomepageURL string
	PolicyURL         string
}

// OAuthClient owns the unstable Indigo client configuration and exposes only
// encoded public documents to the HTTP application boundary.
type OAuthClient struct {
	config   atprotocoloauth.ClientConfig
	metadata []byte
	jwks     []byte
}

// GenerateOAuthClientPrivateKey returns an Indigo-compatible confidential
// client P-256 key. Callers must place it directly into a secret manager.
func GenerateOAuthClientPrivateKey() (string, error) {
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		return "", fmt.Errorf("generate AT OAuth client key: %w", err)
	}
	return key.Multibase(), nil
}

func NewOAuthClient(settings OAuthClientSettings) (*OAuthClient, error) {
	if err := ValidateOAuthClientSettings(settings); err != nil {
		return nil, err
	}
	privateKey, err := atcrypto.ParsePrivateMultibase(settings.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse AT OAuth client key: %w", err)
	}
	config := atprotocoloauth.NewPublicConfig(settings.ClientID, settings.CallbackURL, []string{"atproto"})
	config.UserAgent = strings.TrimSpace(settings.UserAgent)
	if err := config.SetClientSecret(privateKey, settings.KeyID); err != nil {
		return nil, fmt.Errorf("configure AT OAuth confidential client: %w", err)
	}

	metadata := config.ClientMetadata()
	metadata.JWKSURI = stringPtr(settings.JWKSURL)
	if value := strings.TrimSpace(settings.ClientName); value != "" {
		metadata.ClientName = &value
	}
	if value := strings.TrimSpace(settings.ClientHomepageURL); value != "" {
		metadata.ClientURI = &value
	}
	if value := strings.TrimSpace(settings.PolicyURL); value != "" {
		metadata.PolicyURI = &value
	}
	if err := metadata.Validate(settings.ClientID); err != nil {
		return nil, fmt.Errorf("validate AT OAuth client metadata: %w", err)
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode AT OAuth client metadata: %w", err)
	}
	jwksJSON, err := json.Marshal(config.PublicJWKS())
	if err != nil {
		return nil, fmt.Errorf("encode AT OAuth JWKS: %w", err)
	}
	return &OAuthClient{config: config, metadata: metadataJSON, jwks: jwksJSON}, nil
}

func ValidateOAuthClientSettings(settings OAuthClientSettings) error {
	for name, value := range map[string]string{
		"client ID": settings.ClientID, "callback URL": settings.CallbackURL,
		"JWKS URL": settings.JWKSURL, "private key": settings.PrivateKey, "key ID": settings.KeyID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("AT OAuth %s is required", name)
		}
	}
	clientID, err := validateOAuthHTTPSURL(settings.ClientID, "client ID")
	if err != nil {
		return err
	}
	callback, err := validateOAuthHTTPSURL(settings.CallbackURL, "callback URL")
	if err != nil {
		return err
	}
	jwks, err := validateOAuthHTTPSURL(settings.JWKSURL, "JWKS URL")
	if err != nil {
		return err
	}
	if !strings.EqualFold(clientID.Host, callback.Host) || !strings.EqualFold(clientID.Host, jwks.Host) {
		return errors.New("AT OAuth client ID, callback, and JWKS URLs must share one HTTPS origin")
	}
	privateKey, err := atcrypto.ParsePrivateMultibase(settings.PrivateKey)
	if err != nil {
		return fmt.Errorf("parse AT OAuth client key: %w", err)
	}
	probe := atprotocoloauth.NewPublicConfig(settings.ClientID, settings.CallbackURL, []string{"atproto"})
	if err := probe.SetClientSecret(privateKey, settings.KeyID); err != nil {
		return fmt.Errorf("validate AT OAuth confidential client key: %w", err)
	}
	return nil
}

func (c *OAuthClient) MetadataJSON() []byte { return append([]byte(nil), c.metadata...) }

func (c *OAuthClient) JWKSJSON() []byte { return append([]byte(nil), c.jwks...) }

func validateOAuthHTTPSURL(raw, name string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("AT OAuth %s must be an absolute HTTPS URL without user info, query, or fragment", name)
	}
	return parsed, nil
}

func stringPtr(value string) *string { return &value }
