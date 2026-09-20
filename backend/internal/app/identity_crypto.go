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
)

const identityProtectionKeyBytes = 32

type identityProtector struct {
	aead      cipher.AEAD
	lookupKey []byte
}

func newIdentityProtector(encodedKey, developmentSeed string) (*identityProtector, error) {
	key, err := decodeIdentityProtectionKey(encodedKey)
	if err != nil {
		if encodedKey != "" {
			return nil, err
		}
		keySum := sha256.Sum256([]byte("subcult-os/development-identity/" + developmentSeed))
		key = keySum[:]
	}

	encryptionKey := deriveIdentityKey(key, "email-encryption")
	lookupKey := deriveIdentityKey(key, "email-lookup")
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create identity cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create identity AEAD: %w", err)
	}
	return &identityProtector{aead: aead, lookupKey: lookupKey}, nil
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

func (p *identityProtector) revealEmail(ciphertext []byte) (string, error) {
	nonceSize := p.aead.NonceSize()
	if len(ciphertext) <= nonceSize {
		return "", errors.New("invalid encrypted email")
	}
	plaintext, err := p.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", errors.New("decrypt email")
	}
	return string(plaintext), nil
}

func (p *identityProtector) emailLookupHash(email string) string {
	mac := hmac.New(sha256.New, p.lookupKey)
	_, _ = mac.Write([]byte(normalizeEmail(email)))
	return hex.EncodeToString(mac.Sum(nil))
}
