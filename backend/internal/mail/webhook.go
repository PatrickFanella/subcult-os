package mail

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

func ValidWebhookSecret(secret string) bool { _, ok := webhookKey(secret); return ok }

func webhookKey(secret string) ([]byte, bool) {
	if !strings.HasPrefix(secret, "whsec_") || len(secret) > 256 {
		return nil, false
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	return key, err == nil && len(key) >= 16
}

// VerifyWebhook authenticates the exact raw bytes, not re-serialized JSON.
func VerifyWebhook(secret, id, timestamp, signatures string, body []byte, now time.Time) bool {
	key, ok := webhookKey(secret)
	if !ok || len(id) == 0 || len(id) > 256 || len(signatures) > 4096 || len(body) > 65536 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || seconds < now.Unix()-300 || seconds > now.Unix()+300 {
		return false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "."))
	mac.Write(body)
	expected := mac.Sum(nil)
	for _, value := range strings.Fields(signatures) {
		version, encoded, found := strings.Cut(value, ",")
		if !found || version != "v1" {
			continue
		}
		actual, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil && hmac.Equal(expected, actual) {
			return true
		}
	}
	return false
}
