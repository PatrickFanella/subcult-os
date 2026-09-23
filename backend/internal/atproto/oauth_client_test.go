package atproto

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
)

func TestOAuthClientPublishesConfidentialIdentityOnlyMetadata(t *testing.T) {
	settings := testOAuthClientSettings(t)
	client, err := NewOAuthClient(settings)
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(client.MetadataJSON(), &metadata); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"client_id": settings.ClientID, "scope": "atproto",
		"token_endpoint_auth_method":      "private_key_jwt",
		"token_endpoint_auth_signing_alg": "ES256", "dpop_bound_access_tokens": true,
		"jwks_uri": settings.JWKSURL,
	} {
		if got := metadata[key]; got != want {
			t.Fatalf("metadata[%q] = %#v, want %#v", key, got, want)
		}
	}
	if _, exposed := metadata["private_key"]; exposed {
		t.Fatal("client metadata exposed private key")
	}
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(client.JWKSJSON(), &jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0]["kid"] != settings.KeyID || jwks.Keys[0]["kty"] != "EC" || jwks.Keys[0]["crv"] != "P-256" {
		t.Fatalf("unexpected public JWKS: %#v", jwks.Keys)
	}
	if _, exposed := jwks.Keys[0]["d"]; exposed {
		t.Fatal("JWKS exposed private key material")
	}
	if bytes.Contains(client.JWKSJSON(), []byte(settings.PrivateKey)) {
		t.Fatal("JWKS contained the configured private key")
	}
}

func TestOAuthClientRejectsUnsafeOrIncompatibleSettings(t *testing.T) {
	valid := testOAuthClientSettings(t)
	tests := []struct {
		name   string
		change func(*OAuthClientSettings)
	}{
		{name: "HTTP client ID", change: func(s *OAuthClientSettings) { s.ClientID = "http://subcults.subcult.tv/client" }},
		{name: "cross-origin callback", change: func(s *OAuthClientSettings) { s.CallbackURL = "https://evil.example/callback" }},
		{name: "query in JWKS", change: func(s *OAuthClientSettings) { s.JWKSURL += "?version=1" }},
		{name: "missing key ID", change: func(s *OAuthClientSettings) { s.KeyID = "" }},
		{name: "invalid private key", change: func(s *OAuthClientSettings) { s.PrivateKey = "not-a-key" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := valid
			test.change(&settings)
			if _, err := NewOAuthClient(settings); err == nil {
				t.Fatal("NewOAuthClient() unexpectedly succeeded")
			}
		})
	}
	k256, err := atcrypto.GeneratePrivateKeyK256()
	if err != nil {
		t.Fatal(err)
	}
	valid.PrivateKey = k256.Multibase()
	if _, err := NewOAuthClient(valid); err == nil {
		t.Fatal("K-256 confidential client key unexpectedly succeeded")
	}
}

func TestOAuthClientJWKSIncludesBothKeysDuringTransition(t *testing.T) {
	settings := testOAuthClientSettings(t)
	previousKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	settings.PreviousPrivateKey = previousKey.Multibase()
	settings.PreviousKeyID = "subcults-0"

	client, err := NewOAuthClient(settings)
	if err != nil {
		t.Fatal(err)
	}
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(client.JWKSJSON(), &jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 2 {
		t.Fatalf("transition JWKS key count = %d, want 2: %#v", len(jwks.Keys), jwks.Keys)
	}
	kids := map[string]bool{}
	for _, key := range jwks.Keys {
		kids[key["kid"].(string)] = true
		if _, exposed := key["d"]; exposed {
			t.Fatal("transition JWKS exposed private key material")
		}
	}
	if !kids[settings.KeyID] || !kids[settings.PreviousKeyID] {
		t.Fatalf("transition JWKS key IDs = %#v, want both %q and %q", kids, settings.KeyID, settings.PreviousKeyID)
	}
	if bytes.Contains(client.JWKSJSON(), []byte(settings.PreviousPrivateKey)) {
		t.Fatal("JWKS contained the previous private key")
	}

	// Outside the transition window, only the current key is published.
	settings.PreviousPrivateKey = ""
	settings.PreviousKeyID = ""
	after, err := NewOAuthClient(settings)
	if err != nil {
		t.Fatal(err)
	}
	var afterJWKS struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(after.JWKSJSON(), &afterJWKS); err != nil {
		t.Fatal(err)
	}
	if len(afterJWKS.Keys) != 1 || afterJWKS.Keys[0]["kid"] != settings.KeyID {
		t.Fatalf("post-transition JWKS = %#v, want only %q", afterJWKS.Keys, settings.KeyID)
	}
}

func TestOAuthClientRejectsInvalidTransitionSettings(t *testing.T) {
	valid := testOAuthClientSettings(t)
	previousKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*OAuthClientSettings)
	}{
		{name: "previous key without previous key ID", change: func(s *OAuthClientSettings) { s.PreviousPrivateKey = previousKey.Multibase() }},
		{name: "previous key ID without previous key", change: func(s *OAuthClientSettings) { s.PreviousKeyID = "subcults-0" }},
		{name: "previous key ID collides with current", change: func(s *OAuthClientSettings) {
			s.PreviousPrivateKey = previousKey.Multibase()
			s.PreviousKeyID = s.KeyID
		}},
		{name: "invalid previous key", change: func(s *OAuthClientSettings) {
			s.PreviousPrivateKey = "not-a-key"
			s.PreviousKeyID = "subcults-0"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := valid
			test.change(&settings)
			if _, err := NewOAuthClient(settings); err == nil {
				t.Fatal("NewOAuthClient() unexpectedly succeeded")
			}
		})
	}
}

func testOAuthClientSettings(t *testing.T) OAuthClientSettings {
	t.Helper()
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	return OAuthClientSettings{
		ClientID:          "https://subcults.subcult.tv/api/v1/auth/atproto/client-metadata",
		CallbackURL:       "https://subcults.subcult.tv/api/v1/auth/atproto/callback",
		JWKSURL:           "https://subcults.subcult.tv/api/v1/auth/atproto/jwks",
		PrivateKey:        key.Multibase(),
		KeyID:             "subcults-1",
		UserAgent:         "subcult-os/test",
		ClientName:        "Subcult OS",
		ClientHomepageURL: "https://subcults.subcult.tv",
		PolicyURL:         "https://subcults.subcult.tv/privacy",
	}
}
