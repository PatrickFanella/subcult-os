package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	stdmail "net/mail"
	"strings"
	"time"
)

type Brevo struct {
	key    string
	client *http.Client
}

func NewBrevo(key string) (*Brevo, error) {
	if key == "" || key != strings.TrimSpace(key) || len(key) > 1024 || strings.ContainsAny(key, "\r\n") {
		return nil, errors.New("invalid Brevo credential configuration")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &Brevo{key: key, client: &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func ValidBrevoWebhookToken(token string) bool {
	return len(token) >= 32 && len(token) <= 256 && token == strings.TrimSpace(token) && !strings.ContainsAny(token, "\r\n")
}

func BrevoMessageID(value string) (string, bool) {
	value = strings.Trim(value, "<>")
	if value == "" || len(value) > 480 || strings.ContainsAny(value, "\r\n\t ") || !strings.Contains(value, "@") {
		return "", false
	}
	return "brevo:" + value, true
}

func (c *Brevo) Send(ctx context.Context, message Message) (string, error) {
	from, fromErr := stdmail.ParseAddress(message.From)
	to, toErr := stdmail.ParseAddress(message.To)
	if !uuid.MatchString(message.ID) || fromErr != nil || toErr != nil || to.Address != message.To || !ValidAddress(message.From) ||
		(message.ReplyTo != "" && !ValidAddress(message.ReplyTo)) || strings.TrimSpace(message.Subject) == "" || len(message.Subject) > 998 || strings.ContainsAny(message.Subject, "\r\n") || message.Text == "" || len(message.Text) > 256*1024 {
		return "", &Failure{Code: "invalid_message"}
	}
	payload := map[string]any{"sender": brevoContact(from), "subject": message.Subject, "textContent": message.Text,
		"messageVersions": []any{map[string]any{"to": []any{map[string]string{"email": message.To}}}},
		"headers":         map[string]string{"idempotencyKey": message.ID}, "tags": []string{"subcult-os"}}
	if message.ReplyTo != "" {
		reply, _ := stdmail.ParseAddress(message.ReplyTo)
		payload["replyTo"] = brevoContact(reply)
	}
	body, _ := json.Marshal(payload)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return "", &Failure{Code: "invalid_message"}
	}
	request.Header.Set("api-key", c.key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return "", &Failure{Code: "transport_failed", Temporary: true}
	}
	defer response.Body.Close()
	body, err = io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(body) > 65536 {
		return "", &Failure{Code: "invalid_response", Temporary: true}
	}
	if response.StatusCode == http.StatusCreated {
		var result struct {
			MessageID  string   `json:"messageId"`
			MessageIDs []string `json:"messageIds"`
		}
		if json.Unmarshal(body, &result) != nil {
			return "", &Failure{Code: "invalid_response", Temporary: true}
		}
		if result.MessageID == "" && len(result.MessageIDs) == 1 {
			result.MessageID = result.MessageIDs[0]
		}
		id, ok := BrevoMessageID(result.MessageID)
		if !ok {
			return "", &Failure{Code: "invalid_response", Temporary: true}
		}
		return id, nil
	}
	var result struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &result)
	if result.Code == "duplicate_parameter" {
		return "", &Failure{Code: "acceptance_unconfirmed"}
	}
	temporary := response.StatusCode == 408 || response.StatusCode == 429 || response.StatusCode >= 500
	return "", &Failure{Code: "provider_rejected", Temporary: temporary, RetryAfter: retryAfter(response.Header.Get("Retry-After"), time.Now())}
}

// brevoContact omits an empty display name: Brevo rejects "name": "" with
// missing_parameter, so a bare address such as MAIL_REPLY_TO=info@subcult.tv
// would otherwise fail every send.
func brevoContact(address *stdmail.Address) map[string]string {
	contact := map[string]string{"email": address.Address}
	if address.Name != "" {
		contact["name"] = address.Name
	}
	return contact
}
