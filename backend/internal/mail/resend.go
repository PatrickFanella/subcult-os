// Package mail implements the single transactional email provider boundary.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	stdmail "net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	ID      string
	From    string
	ReplyTo string
	To      string
	Subject string
	Text    string
}

// Failure exposes only an application-owned code, never provider text or PII.
type Failure struct {
	Code       string
	Temporary  bool
	RetryAfter time.Duration
}

func (e *Failure) Error() string { return e.Code }

type Resend struct {
	key    string
	client *http.Client
}

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func NewResend(key string) (*Resend, error) {
	if key == "" || key != strings.TrimSpace(key) || len(key) > 1024 || strings.ContainsAny(key, "\r\n") {
		return nil, errors.New("invalid Resend credential configuration")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &Resend{key: key, client: &http.Client{
		Timeout: 10 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func ValidAddress(value string) bool {
	if value == "" || len(value) > 320 || strings.ContainsAny(value, "\r\n") {
		return false
	}
	_, err := stdmail.ParseAddress(value)
	return err == nil
}

func (c *Resend) Send(ctx context.Context, message Message) (string, error) {
	to, err := stdmail.ParseAddress(message.To)
	if !uuid.MatchString(message.ID) || err != nil || to.Address != message.To || !ValidAddress(message.From) ||
		(message.ReplyTo != "" && !ValidAddress(message.ReplyTo)) || strings.TrimSpace(message.Subject) == "" ||
		len(message.Subject) > 998 || strings.ContainsAny(message.Subject, "\r\n") || message.Text == "" || len(message.Text) > 256*1024 {
		return "", &Failure{Code: "invalid_message"}
	}
	payload, _ := json.Marshal(struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		ReplyTo string   `json:"reply_to,omitempty"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{message.From, []string{message.To}, message.ReplyTo, message.Subject, message.Text})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return "", &Failure{Code: "invalid_message"}
	}
	request.Header.Set("Authorization", "Bearer "+c.key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "subcult-email/"+message.ID)
	response, err := c.client.Do(request)
	if err != nil {
		return "", &Failure{Code: "transport_failed", Temporary: true}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil || len(body) > 64*1024 {
		return "", &Failure{Code: "invalid_response", Temporary: true}
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		var result struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(body, &result) != nil || !uuid.MatchString(result.ID) {
			return "", &Failure{Code: "invalid_response", Temporary: true}
		}
		return result.ID, nil // Accepted by the provider, not proof of delivery.
	}
	var failure struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(body, &failure)
	temporary := response.StatusCode == 408 || response.StatusCode == 429 || response.StatusCode >= 500 ||
		(response.StatusCode == 409 && failure.Name == "concurrent_idempotent_requests")
	code := "provider_rejected"
	if temporary {
		code = "provider_retryable"
	}
	return "", &Failure{Code: code, Temporary: temporary, RetryAfter: retryAfter(response.Header.Get("Retry-After"), time.Now())}
}

func retryAfter(value string, now time.Time) time.Duration {
	seconds, err := strconv.ParseInt(value, 10, 32)
	delay := time.Duration(seconds) * time.Second
	if err != nil {
		at, parseErr := http.ParseTime(value)
		if parseErr != nil {
			return 0
		}
		delay = at.Sub(now)
	}
	if delay < 0 {
		return 0
	}
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}
