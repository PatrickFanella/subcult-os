package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"testing"
	"time"
)

func TestPublicEventDTOExcludesOperationalFields(t *testing.T) {
	t.Parallel()

	application := &App{config: Config{PublicWebURL: "https://events.example.test"}}
	payload, err := json.Marshal(application.publicEventDTOFromRow(eventRow{
		ID:                     "event-public",
		WorkspaceID:            "private-workspace-sentinel",
		Title:                  "Public Event",
		StartsAt:               time.Date(2026, time.July, 1, 20, 0, 0, 0, time.UTC),
		PublicDescription:      "Public description",
		LocationDisplay:        "Public venue",
		ImageURL:               sql.NullString{String: "https://media.example.test/event.jpg", Valid: true},
		TicketAllocation:       99,
		PricingMode:            "free",
		TicketCurrency:         "usd",
		ReservedCount:          11,
		CheckedInCount:         7,
		StaffingOpenCount:      5,
		StaffingAssignedCount:  4,
		StaffingCompletedCount: 3,
		StaffingCancelledCount: 2,
		Status:                 eventStatusPublished,
		PublicSlug:             sql.NullString{String: "public-event", Valid: true},
	}))
	if err != nil {
		t.Fatal(err)
	}

	var public map[string]any
	if err := json.Unmarshal(payload, &public); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"workspaceId",
		"ticketAllocation",
		"reservedCount",
		"checkedInCount",
		"staffingOpenCount",
		"staffingAssignedCount",
		"staffingCompletedCount",
		"staffingCancelledCount",
	} {
		if _, ok := public[forbidden]; ok {
			t.Fatalf("public event payload exposed %q: %s", forbidden, payload)
		}
	}
	for _, sentinel := range [][]byte{
		[]byte("private-workspace-sentinel"),
		[]byte(`"reservedCount":11`),
		[]byte(`"checkedInCount":7`),
		[]byte(`"staffingOpenCount":5`),
	} {
		if bytes.Contains(payload, sentinel) {
			t.Fatalf("public event payload exposed private sentinel %q: %s", sentinel, payload)
		}
	}
	if public["remainingTickets"] != float64(88) || public["publicUrl"] != "https://events.example.test/e/public-event" {
		t.Fatalf("unexpected public event payload: %s", payload)
	}
}
