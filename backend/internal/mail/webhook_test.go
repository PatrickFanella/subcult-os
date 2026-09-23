package mail

import (
	"strings"
	"testing"
	"time"
)

func TestWebhookPublishedVector(t *testing.T) {
	// Independent vector: https://docs.svix.com/receiving/verifying-payloads/how-manual
	secret := "whsec_plJ3nmyCDGBKInavdOK15jsl"
	id, timestamp := "msg_loFOjxBNrRLzqYUf", "1731705121"
	body := []byte(`{"event_type":"ping","data":{"success":true}}`)
	signature := "v1,rAvfW3dJ/X/qxhsaXPOyyCGmRKsaKWcsNccKXlIktD0="
	now := time.Unix(1731705121, 0)
	if !VerifyWebhook(secret, id, timestamp, signature, body, now) {
		t.Fatal("published signature rejected")
	}
	if !VerifyWebhook(secret, id, timestamp, "v2,ignored v1,invalid "+signature, body, now) {
		t.Fatal("rotation signature list rejected")
	}
	for _, tt := range []struct {
		name, secret, id, stamp, sig string
		body                         []byte
		now                          time.Time
	}{
		{"tampered", secret, id, timestamp, signature, append(body, ' '), now},
		{"expired", secret, id, timestamp, signature, body, now.Add(301 * time.Second)},
		{"future", secret, id, timestamp, signature, body, now.Add(-301 * time.Second)},
		{"overflow", secret, id, "9223372036854775807", signature, body, now},
		{"invalid timestamp", secret, id, "x", signature, body, now},
		{"invalid secret", "whsec_bad", id, timestamp, signature, body, now},
		{"missing secret", "", id, timestamp, signature, body, now},
		{"wrong signature", secret, id, timestamp, "v1,YmFk", body, now},
		{"wrong version", secret, id, timestamp, strings.Replace(signature, "v1,", "v2,", 1), body, now},
		{"missing id", secret, "", timestamp, signature, body, now},
		{"invalid id", secret, "msg.invalid", timestamp, signature, body, now},
		{"oversized", secret, id, timestamp, signature, make([]byte, 65537), now},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if VerifyWebhook(tt.secret, tt.id, tt.stamp, tt.sig, tt.body, tt.now) {
				t.Fatal("invalid signature accepted")
			}
		})
	}
}
