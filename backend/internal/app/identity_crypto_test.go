package app

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestIdentityProtectorRoundTripAndLookup(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	protector, err := newIdentityProtector(key, "")
	if err != nil {
		t.Fatal(err)
	}

	ciphertext, lookup, err := protector.protectEmail("  Person@Example.TEST ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext), "person@example.test") {
		t.Fatal("ciphertext contains plaintext email")
	}
	if lookup != protector.emailLookupHash("person@example.test") {
		t.Fatal("normalized lookup hash is not deterministic")
	}
	revealed, err := protector.revealEmail(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if revealed != "person@example.test" {
		t.Fatalf("revealed email = %q", revealed)
	}
}

func TestIdentityProtectorRejectsTamperAndWrongKey(t *testing.T) {
	first, err := newIdentityProtector(base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")), "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := newIdentityProtector(base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789")), "")
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, _, err := first.protectEmail("person@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.revealEmail(ciphertext); err == nil {
		t.Fatal("wrong key decrypted email")
	}
	ciphertext[len(ciphertext)-1] ^= 1
	if _, err := first.revealEmail(ciphertext); err == nil {
		t.Fatal("tampered ciphertext decrypted")
	}
}

func TestIdentityProtectionKeyValidation(t *testing.T) {
	for _, value := range []string{"", "not-base64", base64.StdEncoding.EncodeToString([]byte("short"))} {
		if _, err := decodeIdentityProtectionKey(value); err == nil {
			t.Fatalf("decodeIdentityProtectionKey(%q) succeeded", value)
		}
	}
}
