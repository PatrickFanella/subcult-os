package app

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

type Config struct {
	AppEnv                        string
	DatabaseURL                   string
	SessionSecret                 string
	IdentityProtectionKey         string
	IdentityProtectionKeyPrevious string
	PublicWebURL                  string
	ATProtoOAuthEnabled           bool
	ATProtoOAuthClientID          string
	ATProtoOAuthCallback          string
	ATProtoOAuthJWKSURL           string
	ATProtoOAuthKey               string
	ATProtoOAuthKeyID             string
	ATProtoOAuthKeyPrevious       string
	ATProtoOAuthKeyIDPrevious     string
	atProtoOAuthEnabledRaw        string
	Addr                          string
	StripeSecretKey               string
	StripeWebhookSecret           string
	MediaS3Endpoint               string
	MediaS3AccessKey              string
	MediaS3SecretKey              string
	MediaS3Bucket                 string
	MediaS3Region                 string
	MediaPublicBaseURL            string
	MailDeliveryEnabled           bool
	mailDeliveryEnabledRaw        string
	ResendAPIKey                  string
	ResendWebhookSecret           string
	MailFrom                      string
	MailReplyTo                   string
}

func LoadConfig() Config {
	atProtoOAuthEnabledRaw := env("ATPROTO_OAUTH_ENABLED", "false")
	mailDeliveryEnabledRaw := env("MAIL_DELIVERY_ENABLED", "false")
	return Config{
		MailDeliveryEnabled:           parseEnvBool(mailDeliveryEnabledRaw),
		mailDeliveryEnabledRaw:        mailDeliveryEnabledRaw,
		ResendAPIKey:                  env("RESEND_API_KEY", ""),
		ResendWebhookSecret:           env("RESEND_WEBHOOK_SECRET", ""),
		MailFrom:                      env("MAIL_FROM", ""),
		MailReplyTo:                   env("MAIL_REPLY_TO", ""),
		AppEnv:                        env("APP_ENV", "development"),
		DatabaseURL:                   env("DATABASE_URL", ""),
		SessionSecret:                 env("SESSION_SECRET", "dev-session-secret-change-me"),
		IdentityProtectionKey:         env("IDENTITY_PROTECTION_KEY", ""),
		IdentityProtectionKeyPrevious: env("IDENTITY_PROTECTION_KEY_PREVIOUS", ""),
		PublicWebURL:                  env("PUBLIC_WEB_URL", "http://localhost:5173"),
		ATProtoOAuthEnabled:           parseEnvBool(atProtoOAuthEnabledRaw),
		ATProtoOAuthClientID:          env("ATPROTO_OAUTH_CLIENT_ID", ""),
		ATProtoOAuthCallback:          env("ATPROTO_OAUTH_CALLBACK_URL", ""),
		ATProtoOAuthJWKSURL:           env("ATPROTO_OAUTH_JWKS_URL", ""),
		ATProtoOAuthKey:               env("ATPROTO_OAUTH_CLIENT_PRIVATE_KEY", ""),
		ATProtoOAuthKeyID:             env("ATPROTO_OAUTH_CLIENT_KEY_ID", "subcults-1"),
		ATProtoOAuthKeyPrevious:       env("ATPROTO_OAUTH_CLIENT_PRIVATE_KEY_PREVIOUS", ""),
		ATProtoOAuthKeyIDPrevious:     env("ATPROTO_OAUTH_CLIENT_KEY_ID_PREVIOUS", ""),
		atProtoOAuthEnabledRaw:        atProtoOAuthEnabledRaw,
		Addr:                          env("API_ADDR", ":8080"),
		StripeSecretKey:               env("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:           env("STRIPE_WEBHOOK_SECRET", ""),
		MediaS3Endpoint:               env("MEDIA_S3_ENDPOINT", ""),
		MediaS3AccessKey:              env("MEDIA_S3_ACCESS_KEY", ""),
		MediaS3SecretKey:              env("MEDIA_S3_SECRET_KEY", ""),
		MediaS3Bucket:                 env("MEDIA_S3_BUCKET", ""),
		MediaS3Region:                 env("MEDIA_S3_REGION", "us-east-1"),
		MediaPublicBaseURL:            env("MEDIA_PUBLIC_BASE_URL", ""),
	}
}

func (c Config) Validate() error {
	var problems []string
	if c.ResendWebhookSecret != "" && !mailprovider.ValidWebhookSecret(c.ResendWebhookSecret) {
		problems = append(problems, "RESEND_WEBHOOK_SECRET must be a valid signing secret")
	}
	if raw := c.mailDeliveryEnabledRaw; raw != "" {
		if _, err := strconv.ParseBool(raw); err != nil {
			problems = append(problems, "MAIL_DELIVERY_ENABLED must be a boolean")
		}
	}
	if c.MailDeliveryEnabled {
		if !mailprovider.ValidWebhookSecret(c.ResendWebhookSecret) {
			problems = append(problems, "RESEND_WEBHOOK_SECRET is required for mail delivery")
		}
		if _, err := mailprovider.NewResend(c.ResendAPIKey); err != nil {
			problems = append(problems, "RESEND_API_KEY is required for mail delivery")
		}
		if !mailprovider.ValidAddress(c.MailFrom) {
			problems = append(problems, "MAIL_FROM must be a valid sender address")
		}
		if c.MailReplyTo != "" && !mailprovider.ValidAddress(c.MailReplyTo) {
			problems = append(problems, "MAIL_REPLY_TO must be a valid address")
		}
	}
	if raw := strings.TrimSpace(c.atProtoOAuthEnabledRaw); raw != "" {
		if _, err := strconv.ParseBool(raw); err != nil {
			problems = append(problems, "ATPROTO_OAUTH_ENABLED must be a boolean")
		}
	}
	if strings.TrimSpace(c.Addr) == "" {
		problems = append(problems, "API_ADDR is required")
	}
	if strings.EqualFold(strings.TrimSpace(c.AppEnv), "production") {
		if strings.TrimSpace(c.DatabaseURL) == "" {
			problems = append(problems, "DATABASE_URL is required in production")
		}
		if strings.TrimSpace(c.PublicWebURL) == "" {
			problems = append(problems, "PUBLIC_WEB_URL is required in production")
		}
		secret := strings.TrimSpace(c.SessionSecret)
		if secret == "" || secret == "dev-session-secret-change-me" || len(secret) < 24 {
			problems = append(problems, "SESSION_SECRET must be a non-default value with at least 24 characters in production")
		}
		if _, err := decodeIdentityProtectionKey(c.IdentityProtectionKey); err != nil {
			problems = append(problems, "IDENTITY_PROTECTION_KEY must be base64 for exactly 32 bytes in production")
		}
		if previous := strings.TrimSpace(c.IdentityProtectionKeyPrevious); previous != "" {
			if _, err := decodeIdentityProtectionKey(previous); err != nil {
				problems = append(problems, "IDENTITY_PROTECTION_KEY_PREVIOUS must be base64 for exactly 32 bytes when set")
			}
		}
		stripeSecretKey := strings.TrimSpace(c.StripeSecretKey)
		stripeWebhookSecret := strings.TrimSpace(c.StripeWebhookSecret)
		if stripeSecretKey == "" && stripeWebhookSecret != "" {
			problems = append(problems, "STRIPE_SECRET_KEY is required when STRIPE_WEBHOOK_SECRET is set in production")
		}
		if stripeSecretKey != "" && stripeWebhookSecret == "" {
			problems = append(problems, "STRIPE_WEBHOOK_SECRET is required when STRIPE_SECRET_KEY is set in production")
		}
	}
	if c.ATProtoOAuthEnabled {
		if err := atprotocol.ValidateOAuthClientSettings(c.atprotoOAuthSettings()); err != nil {
			problems = append(problems, err.Error())
		}
		publicURL, err := url.Parse(strings.TrimSpace(c.PublicWebURL))
		if err != nil || publicURL.Scheme != "https" || publicURL.Hostname() == "" || publicURL.User != nil || publicURL.RawQuery != "" || publicURL.Fragment != "" || (publicURL.Path != "" && publicURL.Path != "/") {
			problems = append(problems, "PUBLIC_WEB_URL must be an HTTPS origin when AT OAuth is enabled")
		} else if clientURL, parseErr := url.Parse(strings.TrimSpace(c.ATProtoOAuthClientID)); parseErr == nil && !strings.EqualFold(publicURL.Host, clientURL.Host) {
			problems = append(problems, "PUBLIC_WEB_URL and ATPROTO_OAUTH_CLIENT_ID must share one origin")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(problems, "; "))
	}
	return nil
}

func (c Config) atprotoOAuthSettings() atprotocol.OAuthClientSettings {
	return atprotocol.OAuthClientSettings{
		ClientID: c.ATProtoOAuthClientID, CallbackURL: c.ATProtoOAuthCallback,
		JWKSURL: c.ATProtoOAuthJWKSURL, PrivateKey: c.ATProtoOAuthKey, KeyID: c.ATProtoOAuthKeyID,
		PreviousPrivateKey: c.ATProtoOAuthKeyPrevious, PreviousKeyID: c.ATProtoOAuthKeyIDPrevious,
		UserAgent: "subcult-os/0.1", ClientName: "Subcult OS", ClientHomepageURL: c.PublicWebURL,
		PolicyURL: strings.TrimRight(c.PublicWebURL, "/") + "/privacy",
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseEnvBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed
}
