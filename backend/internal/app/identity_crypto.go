package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const identityProtectionKeyBytes = 32

// identityProtector encrypts and looks up verified emails under the current
// IDENTITY_PROTECTION_KEY. An optional previous key is retained only during a
// rotation window: it lets in-flight reads decrypt rows sealed before
// rotation and lets lookups match hashes computed under the retired key.
// Every write (protectEmail) always uses the current key.
type identityProtector struct {
	aead      cipher.AEAD
	lookupKey []byte

	prevAEAD      cipher.AEAD
	prevLookupKey []byte
}

func newIdentityProtector(encodedKey, previousEncodedKey, developmentSeed string) (*identityProtector, error) {
	key, err := decodeIdentityProtectionKey(encodedKey)
	if err != nil {
		if encodedKey != "" {
			return nil, err
		}
		keySum := sha256.Sum256([]byte("subcult-os/development-identity/" + developmentSeed))
		key = keySum[:]
	}

	aead, lookupKey, err := identityAEADFromMasterKey(key)
	if err != nil {
		return nil, err
	}
	protector := &identityProtector{aead: aead, lookupKey: lookupKey}

	if previous := strings.TrimSpace(previousEncodedKey); previous != "" {
		previousKey, err := decodeIdentityProtectionKey(previous)
		if err != nil {
			return nil, fmt.Errorf("previous identity protection key: %w", err)
		}
		prevAEAD, prevLookupKey, err := identityAEADFromMasterKey(previousKey)
		if err != nil {
			return nil, fmt.Errorf("previous identity protection key: %w", err)
		}
		protector.prevAEAD = prevAEAD
		protector.prevLookupKey = prevLookupKey
	}
	return protector, nil
}

func identityAEADFromMasterKey(key []byte) (cipher.AEAD, []byte, error) {
	encryptionKey := deriveIdentityKey(key, "email-encryption")
	lookupKey := deriveIdentityKey(key, "email-lookup")
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create identity cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create identity AEAD: %w", err)
	}
	return aead, lookupKey, nil
}

func decodeIdentityProtectionKey(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, errors.New("identity protection key is required")
	}
	var decoded []byte
	var err error
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		decoded, err = encoding.DecodeString(encoded)
		if err == nil {
			break
		}
	}
	if err != nil || len(decoded) != identityProtectionKeyBytes {
		return nil, errors.New("identity protection key must decode to 32 bytes")
	}
	return decoded, nil
}

func deriveIdentityKey(master []byte, purpose string) []byte {
	mac := hmac.New(sha256.New, master)
	_, _ = mac.Write([]byte("subcult-os/identity/" + purpose))
	return mac.Sum(nil)
}

func (p *identityProtector) protectEmail(email string) ([]byte, string, error) {
	normalized := normalizeEmail(email)
	if normalized == "" {
		return nil, "", errors.New("email is required")
	}
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", fmt.Errorf("generate identity nonce: %w", err)
	}
	ciphertext := p.aead.Seal(nonce, nonce, []byte(normalized), nil)
	return ciphertext, p.emailLookupHash(normalized), nil
}

// revealEmail decrypts with the current key first, then the previous key
// during a rotation window. Wrong keys and tampered ciphertext both fail.
func (p *identityProtector) revealEmail(ciphertext []byte) (string, error) {
	if plaintext, err := openWithAEAD(p.aead, ciphertext); err == nil {
		return plaintext, nil
	}
	if p.prevAEAD != nil {
		if plaintext, err := openWithAEAD(p.prevAEAD, ciphertext); err == nil {
			return plaintext, nil
		}
	}
	return "", errors.New("decrypt email")
}

func openWithAEAD(aead cipher.AEAD, ciphertext []byte) (string, error) {
	nonceSize := aead.NonceSize()
	if len(ciphertext) <= nonceSize {
		return "", errors.New("invalid encrypted email")
	}
	plaintext, err := aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", errors.New("decrypt email")
	}
	return string(plaintext), nil
}

func (p *identityProtector) emailLookupHash(email string) string {
	return hashEmailLookup(p.lookupKey, email)
}

// emailLookupHashCandidates returns every hash worth trying in a database
// lookup: the current key's hash, plus the previous key's hash while a
// rotation window is open. Rows written under the retired key still resolve
// to the same account until they are re-encrypted.
func (p *identityProtector) emailLookupHashCandidates(email string) []string {
	candidates := []string{p.emailLookupHash(email)}
	if p.prevLookupKey != nil {
		candidates = append(candidates, hashEmailLookup(p.prevLookupKey, email))
	}
	return candidates
}

func hashEmailLookup(lookupKey []byte, email string) string {
	mac := hmac.New(sha256.New, lookupKey)
	_, _ = mac.Write([]byte(normalizeEmail(email)))
	return hex.EncodeToString(mac.Sum(nil))
}

// reencryptEmail rewrites ciphertext sealed under the previous key so it is
// sealed under the current key, refreshing the lookup hash to match. It
// reports rotated=false (and touches nothing) when the row is already
// current, so a re-encryption sweep is idempotent and safe to rerun.
func (p *identityProtector) reencryptEmail(ciphertext []byte) (newCiphertext []byte, newLookupHash string, rotated bool, err error) {
	if _, err := openWithAEAD(p.aead, ciphertext); err == nil {
		return nil, "", false, nil
	}
	plaintext, err := p.revealEmail(ciphertext)
	if err != nil {
		return nil, "", false, err
	}
	newCiphertext, newLookupHash, err = p.protectEmail(plaintext)
	if err != nil {
		return nil, "", false, err
	}
	return newCiphertext, newLookupHash, true, nil
}
