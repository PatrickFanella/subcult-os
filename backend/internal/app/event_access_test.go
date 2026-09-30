package app

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEventAccessValidation(t *testing.T) {
	zero := 0
	reviewed := "2026-09-01T12:00:00-05:00"
	expires := "2026-10-01T00:00:00Z"
	valid := eventAccessRequest{RequestKey: uuid.NewString(), ExpectedRevision: &zero, Value: "yes", SourceKind: "organizer_assertion", SourceReference: "Synthetic site worksheet", ReviewedAt: &reviewed, ExpiresAt: &expires, CorrectionReason: "Initial source entry"}
	cases := []struct {
		name  string
		topic string
		edit  func(*eventAccessRequest)
	}{
		{"missing revision", "entry", func(r *eventAccessRequest) { r.ExpectedRevision = nil }},
		{"wrong topic value", "seating", func(r *eventAccessRequest) {}},
		{"missing provenance", "entry", func(r *eventAccessRequest) { r.SourceReference = "" }},
		{"unknown retains assertion", "entry", func(r *eventAccessRequest) { r.Value = "unknown" }},
		{"missing review", "entry", func(r *eventAccessRequest) { r.ReviewedAt = nil }},
		{"bad review", "entry", func(r *eventAccessRequest) { s := "yesterday"; r.ReviewedAt = &s }},
		{"expiry before review", "entry", func(r *eventAccessRequest) { s := "2026-01-01T00:00:00Z"; r.ExpiresAt = &s }},
		{"missing text details", "sensory", func(r *eventAccessRequest) { r.Value = "known" }},
		{"unsupported source", "entry", func(r *eventAccessRequest) { r.SourceKind = "verified" }},
		{"oversized detail", "entry", func(r *eventAccessRequest) { r.Details = strings.Repeat("x", 1001) }},
		{"invalid unicode", "entry", func(r *eventAccessRequest) { r.Details = string([]byte{0xff}) }},
		{"control characters", "entry", func(r *eventAccessRequest) { r.Details = "bad\x00text" }},
		{"missing reason", "entry", func(r *eventAccessRequest) { r.CorrectionReason = " " }},
		{"zero key", "entry", func(r *eventAccessRequest) { r.RequestKey = uuid.Nil.String() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := valid
			tc.edit(&r)
			if _, _, err := validateAccessRequest(tc.topic, &r); err == nil {
				t.Fatal("invalid assertion admitted")
			}
		})
	}
	r := valid
	if _, _, err := validateAccessRequest("entry", &r); err != nil {
		t.Fatal(err)
	}
	if *r.ReviewedAt != "2026-09-01T17:00:00Z" {
		t.Fatal("review time not canonicalized")
	}
	unknown := eventAccessRequest{RequestKey: uuid.NewString(), ExpectedRevision: &zero, Value: "unknown", SourceKind: "unknown", CorrectionReason: "Disputed; withdraw current assertion"}
	for _, topic := range accessTopics {
		if _, _, err := validateAccessRequest(topic, &unknown); err != nil {
			t.Fatal(err)
		}
	}
}
