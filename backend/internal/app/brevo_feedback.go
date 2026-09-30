package app

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	mailprovider "git.subcult.tv/PatrickFanella/subcult-os/internal/mail"
)

func (a *App) handleBrevoWebhook(w http.ResponseWriter, r *http.Request) {
	if !mailprovider.ValidBrevoWebhookToken(a.config.BrevoWebhookToken) {
		writeError(w, 404, "not found")
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+a.config.BrevoWebhookToken)) != 1 {
		writeError(w, 401, "invalid webhook")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	if err != nil {
		writeError(w, 413, "webhook body too large")
		return
	}
	var event struct {
		Event         string `json:"event"`
		MessageID     string `json:"message-id"`
		Timestamp     int64  `json:"ts_event"`
		SentTimestamp int64  `json:"ts"`
	}
	if json.Unmarshal(body, &event) != nil {
		writeError(w, 400, "invalid webhook")
		return
	}
	kind := map[string]string{"delivered": "email.delivered", "hard_bounce": "email.bounced", "hardBounce": "email.bounced", "spam": "email.complained", "blocked": "email.suppressed", "invalid": "email.suppressed", "unsubscribed": "email.suppressed"}[event.Event]
	if kind == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	providerID, ok := mailprovider.BrevoMessageID(event.MessageID)
	if event.Timestamp == 0 {
		event.Timestamp = event.SentTimestamp
	}
	if !ok || event.Timestamp <= 0 {
		writeError(w, 400, "invalid webhook")
		return
	}
	if a.db == nil {
		writeError(w, 429, "webhook storage unavailable")
		return
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s/%s/%d", providerID, kind, event.Timestamp)))
	id := "brevo:" + hex.EncodeToString(digest[:])
	if _, err = a.db.Exec(r.Context(), `insert into email_provider_events(event_id,provider_message_id,event_type) values($1,$2,$3) on conflict(event_id) do nothing`, id, providerID, kind); err != nil {
		writeError(w, 429, "webhook storage unavailable")
		return
	}
	if err = a.reconcileEmailFeedback(r.Context()); err != nil {
		writeError(w, 429, "webhook processing unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
