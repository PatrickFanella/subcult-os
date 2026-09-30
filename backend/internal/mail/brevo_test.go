package mail

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBrevoNativeContractAndFailurePrivacy(t *testing.T) {
	for _, tc := range []struct {
		status     int
		body, code string
		temporary  bool
	}{
		{201, `{"messageIds":["<123@relay.test>"]}`, "", false},
		{201, `{"messageId":"bad identifier"}`, "invalid_response", true},
		{403, `{"message":"secret"}`, "provider_rejected", false},
		{429, `{}`, "provider_rejected", true},
		{400, `{"code":"duplicate_parameter"}`, "acceptance_unconfirmed", false},
	} {
		c, _ := NewBrevo("synthetic-key")
		c.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.brevo.com/v3/smtp/email" || r.Header.Get("api-key") != "synthetic-key" {
				t.Fatal("wrong Brevo authentication")
			}
			var v struct {
				Sender   struct{ Email, Name string }
				Headers  map[string]string
				Versions []struct{ To []struct{ Email string } } `json:"messageVersions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				t.Fatal(err)
			}
			if v.Sender.Email != "notify@example.test" || v.Sender.Name != "Subcult" || v.Headers["idempotencyKey"] != testMessage().ID || v.Versions[0].To[0].Email != testMessage().To {
				t.Fatal("wrong payload")
			}
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{}}, nil
		})
		id, err := c.Send(t.Context(), testMessage())
		if tc.code == "" {
			if err != nil || id != "brevo:123@relay.test" {
				t.Fatalf("acceptance=%s %v", id, err)
			}
		} else {
			var failure *Failure
			if !errors.As(err, &failure) || failure.Code != tc.code || failure.Temporary != tc.temporary {
				t.Fatalf("error=%v", err)
			}
		}
	}
}

func TestBrevoCredentialsAndMessageIDs(t *testing.T) {
	if _, err := NewBrevo("key\nsecret"); err == nil {
		t.Fatal("header injection allowed")
	}
	if _, ok := BrevoMessageID("<123@relay.test>"); !ok {
		t.Fatal("valid ID rejected")
	}
	if _, ok := BrevoMessageID("evil\r\n@relay.test"); ok {
		t.Fatal("invalid ID allowed")
	}
	if ValidBrevoWebhookToken("short") {
		t.Fatal("weak webhook token allowed")
	}
}
