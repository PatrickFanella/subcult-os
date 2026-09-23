package app

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestIdentityProtectorRoundTripAndLookup(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	protector, err := newIdentityProtector(key, "", "")
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
	first, err := newIdentityProtector(base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")), "", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := newIdentityProtector(base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789")), "", "")
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

func TestIdentityProtectorRotationReadsUnderCurrentAndPrevious(t *testing.T) {
	currentKey := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	previousKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))

	before, err := newIdentityProtector(previousKey, "", "")
	if err != nil {
		t.Fatal(err)
	}
	oldCiphertext, oldLookup, err := before.protectEmail("rotating@example.test")
	if err != nil {
		t.Fatal(err)
	}

	// Rotated protector: current key is new, previous key is what sealed the
	// existing row. Reads must still succeed and the lookup hash computed
	// under the previous key must be among the rotated lookup candidates.
	rotated, err := newIdentityProtector(currentKey, previousKey, "")
	if err != nil {
		t.Fatal(err)
	}
	revealed, err := rotated.revealEmail(oldCiphertext)
	if err != nil {
		t.Fatalf("revealEmail() during rotation window: %v", err)
	}
	if revealed != "rotating@example.test" {
		t.Fatalf("revealed email = %q", revealed)
	}
	candidates := rotated.emailLookupHashCandidates("rotating@example.test")
	found := false
	for _, candidate := range candidates {
		if candidate == oldLookup {
			found = true
		}
	}
	if !found {
		t.Fatalf("lookup candidates %v do not include previous-key hash %q", candidates, oldLookup)
	}

	// New writes always use the current key, never the previous one.
	newCiphertext, newLookup, err := rotated.protectEmail("rotating@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if newLookup == oldLookup {
		t.Fatal("new write reused the previous key's lookup hash")
	}
	currentOnly, err := newIdentityProtector(currentKey, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := currentOnly.revealEmail(newCiphertext); err != nil {
		t.Fatalf("current-only protector could not read a freshly written row: %v", err)
	}
}

func TestIdentityProtectorRotationStillRejectsWrongKey(t *testing.T) {
	currentKey := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	previousKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))
	unrelatedKey := base64.StdEncoding.EncodeToString([]byte("fedcba9876543210fedcba9876543210"))

	unrelated, err := newIdentityProtector(unrelatedKey, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, _, err := unrelated.protectEmail("stranger@example.test")
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := newIdentityProtector(currentKey, previousKey, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.revealEmail(ciphertext); err == nil {
		t.Fatal("neither current nor previous key should decrypt a row sealed by an unrelated key")
	}
}

func TestIdentityProtectorReencryptEmailIsIdempotent(t *testing.T) {
	currentKey := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	previousKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))

	before, err := newIdentityProtector(previousKey, "", "")
	if err != nil {
		t.Fatal(err)
	}
	oldCiphertext, _, err := before.protectEmail("reencrypt@example.test")
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := newIdentityProtector(currentKey, previousKey, "")
	if err != nil {
		t.Fatal(err)
	}

	newCiphertext, newLookup, rotatedFlag, err := rotated.reencryptEmail(oldCiphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !rotatedFlag {
		t.Fatal("reencryptEmail() did not report rotation for a previous-key row")
	}
	currentOnly, err := newIdentityProtector(currentKey, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := currentOnly.revealEmail(newCiphertext); err != nil {
		t.Fatalf("current-only protector could not read a rekeyed row: %v", err)
	}
	if newLookup != currentOnly.emailLookupHash("reencrypt@example.test") {
		t.Fatal("rekeyed row does not carry the current-key lookup hash")
	}

	// Running it again on the already-current ciphertext must be a no-op.
	_, _, rotatedAgain, err := rotated.reencryptEmail(newCiphertext)
	if err != nil {
		t.Fatal(err)
	}
	if rotatedAgain {
		t.Fatal("reencryptEmail() rewrote a row that was already sealed under the current key")
	}
}
