package mail

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testMessage() Message {
	return Message{ID: "11111111-1111-4111-8111-111111111111", From: "Subcult <notify@example.test>", ReplyTo: "reply@example.test", To: "guest@example.test", Subject: "Ticket", Text: "private ticket code"}
}

func TestResendRequestAndBoundedClassifications(t *testing.T) {
	for _, tc := range []struct {
		status    int
		body      string
		code      string
		temporary bool
	}{
		{200, `{"id":"22222222-2222-4222-8222-222222222222"}`, "", false},
		{200, `{"id":"provider-controlled garbage"}`, "invalid_response", true},
		{200, strings.Repeat("x", 65537), "invalid_response", true},
		{400, `{"message":"private ticket code"}`, "provider_rejected", false},
		{401, `{"message":"secret key"}`, "provider_rejected", false},
		{409, `{"name":"invalid_idempotent_request"}`, "provider_rejected", false},
		{409, `{"name":"concurrent_idempotent_requests"}`, "provider_retryable", true},
		{429, `{}`, "provider_retryable", true},
		{503, `{}`, "provider_retryable", true},
		{302, `{}`, "provider_rejected", false},
	} {
		client, _ := NewResend("synthetic-key")
		client.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.resend.com/emails" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer synthetic-key" || r.Header.Get("Idempotency-Key") != "subcult-email/"+testMessage().ID {
				t.Fatal("wrong provider request contract")
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"reply_to":"reply@example.test"`) || !strings.Contains(string(body), `"text":"private ticket code"`) {
				t.Fatal("missing payload")
			}
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{"Retry-After": []string{"120"}}}, nil
		})
		id, err := client.Send(t.Context(), testMessage())
		if tc.code == "" {
			if err != nil || id != "22222222-2222-4222-8222-222222222222" {
				t.Fatalf("acceptance: %q %v", id, err)
			}
		} else {
			var failure *Failure
			if !errors.As(err, &failure) || failure.Code != tc.code || failure.Temporary != tc.temporary {
				t.Fatalf("status=%d error=%v", tc.status, err)
			}
		}
	}
}

func TestResendTransportSafetyAndValidation(t *testing.T) {
	client, _ := NewResend("synthetic-key")
	if client.client.Timeout != 10*time.Second || client.client.Transport.(*http.Transport).Proxy != nil || client.client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("unsafe transport policy")
	}
	client.client.Transport = transportFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("private recipient and key") })
	_, err := client.Send(context.Background(), testMessage())
	if err == nil || err.Error() != "transport_failed" {
		t.Fatalf("unsafe error=%v", err)
	}
	for _, mutate := range []func(*Message){
		func(m *Message) { m.ID = "" }, func(m *Message) { m.To = "one@example.test,two@example.test" },
		func(m *Message) { m.Subject = "hello\r\nBcc: other@example.test" }, func(m *Message) { m.From = "bad" },
		func(m *Message) { m.ReplyTo = "bad" }, func(m *Message) { m.Text = "" },
	} {
		m := testMessage()
		mutate(&m)
		_, err := client.Send(t.Context(), m)
		if err == nil || err.Error() != "invalid_message" {
			t.Fatalf("validation=%v", err)
		}
	}
	if _, err := NewResend("key\nsecret"); err == nil {
		t.Fatal("accepted header injection")
	}
	if retryAfter("999999999", time.Now()) != time.Hour || retryAfter("-1", time.Now()) != 0 {
		t.Fatal("unbounded retry delay")
	}
}
