package atproto

import (
	"encoding/base64"
	"testing"
)

func TestOAuthProtectionKeyAndScopePolicy(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(make([]byte, oauthProtectionKeyBytes))
	key, err := decodeOAuthProtectionKey(encoded)
	if err != nil || len(key) != oauthProtectionKeyBytes {
		t.Fatalf("decodeOAuthProtectionKey() = %d bytes, %v", len(key), err)
	}
	for _, invalid := range []string{"", "not-base64", base64.StdEncoding.EncodeToString(make([]byte, 31))} {
		if _, err := decodeOAuthProtectionKey(invalid); err == nil {
			t.Fatalf("decodeOAuthProtectionKey(%q) unexpectedly succeeded", invalid)
		}
	}
	if !identityOnlyScopes([]string{"atproto"}) {
		t.Fatal("identity-only atproto scope rejected")
	}
	for _, scopes := range [][]string{nil, {}, {"atproto", "repo:tv.subcult.event?action=create"}, {"repo:tv.subcult.event?action=create"}} {
		if identityOnlyScopes(scopes) {
			t.Fatalf("non-identity scopes unexpectedly accepted: %#v", scopes)
		}
	}
}
