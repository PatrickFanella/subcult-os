package app

import (
	"database/sql"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// parsedICalEvent holds one VEVENT's unescaped property values by name.
type parsedICalEvent map[string]string

// parseICalendar is a minimal RFC 5545 reader for tests. It checks CRLF line
// endings, the 75-octet physical line limit and UTF-8 validity, unfolds
// continuation lines and unescapes TEXT values.
func parseICalendar(t *testing.T, body string) []parsedICalEvent {
	t.Helper()
	if !strings.HasSuffix(body, "\r\n") {
		t.Fatalf("calendar does not end with CRLF: %q", body)
	}
	physical := strings.Split(strings.TrimSuffix(body, "\r\n"), "\r\n")
	var logical []string
	for _, line := range physical {
		if strings.ContainsAny(line, "\r\n") {
			t.Fatalf("bare CR or LF in line %q", line)
		}
		if len(line) > icalMaxLineOctets {
			t.Fatalf("line exceeds %d octets (%d): %q", icalMaxLineOctets, len(line), line)
		}
		if !utf8.ValidString(line) {
			t.Fatalf("physical line is not valid UTF-8: %q", line)
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if len(logical) == 0 {
				t.Fatal("continuation line before any content line")
			}
			logical[len(logical)-1] += line[1:]
			continue
		}
		logical = append(logical, line)
	}
	if len(logical) < 2 || logical[0] != "BEGIN:VCALENDAR" || logical[len(logical)-1] != "END:VCALENDAR" {
		t.Fatalf("calendar is not wrapped in VCALENDAR: %q", logical)
	}
	var events []parsedICalEvent
	var current parsedICalEvent
	for _, line := range logical[1 : len(logical)-1] {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("content line without value: %q", line)
		}
		switch {
		case line == "BEGIN:VEVENT":
			current = parsedICalEvent{}
		case line == "END:VEVENT":
			events = append(events, current)
			current = nil
		case current != nil:
			current[name] = unescapeICalText(t, value)
		}
	}
	return events
}

func unescapeICalText(t *testing.T, value string) string {
	t.Helper()
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == ',' || c == ';' {
			t.Fatalf("unescaped %q in TEXT value %q", c, value)
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i == len(value) {
			t.Fatalf("dangling escape in %q", value)
		}
		switch value[i] {
		case '\\', ';', ',':
			b.WriteByte(value[i])
		case 'n', 'N':
			b.WriteByte('\n')
		default:
			t.Fatalf("invalid escape \\%c in %q", value[i], value)
		}
	}
	return b.String()
}

func TestICalEscapeText(t *testing.T) {
	got := icalEscapeText("a\\b;c,d\r\ne\rf\ng\th\x00i\x1bj")
	want := `a\\b\;c\,d\ne\nf\ng` + "\th" + "ij"
	if got != want {
		t.Fatalf("escape = %q, want %q", got, want)
	}
}

func TestICalFoldNeverSplitsMultibyteRunes(t *testing.T) {
	for _, value := range []string{
		strings.Repeat("é", 80),
		strings.Repeat("a", 74) + strings.Repeat("€", 30),
		strings.Repeat("🎛️", 25) + "x",
		strings.Repeat("x", 75),
		strings.Repeat("x", 76),
	} {
		folded := icalFold("SUMMARY:" + value)
		for _, line := range strings.Split(folded, "\r\n") {
			if len(line) > icalMaxLineOctets {
				t.Fatalf("line exceeds 75 octets (%d): %q", len(line), line)
			}
			if !utf8.ValidString(line) {
				t.Fatalf("fold split a UTF-8 sequence: %q", line)
			}
		}
		if unfolded := strings.ReplaceAll(folded, "\r\n ", ""); unfolded != "SUMMARY:"+value {
			t.Fatalf("unfold mismatch: %q", unfolded)
		}
	}
	if folded := icalFold("SUMMARY:short"); folded != "SUMMARY:short" {
		t.Fatalf("short line folded: %q", folded)
	}
}

func shiftCalendarFixture() participantAssignmentDTO {
	return participantAssignmentDTO{
		EventID:                 "event-1",
		EventTitle:              "Night Market; Ünïcødé, \\ edition",
		StaffingItemID:          "6f2c1d2e-0000-4000-8000-000000000001",
		Title:                   "Soundcheck",
		Kind:                    "shift",
		Status:                  "assigned",
		ParticipantRequirements: "Bring a DI, a 1/4\" cable; arrive early.\nAsk for Sam at the door — ☎️ is on the wall. " + strings.Repeat("Ünïcødé ", 12),
		locationDisplay:         "Warehouse, Back room",
		startsAt:                sql.NullTime{Time: time.Date(2026, 7, 1, 13, 0, 0, 0, time.FixedZone("CDT", -5*3600)), Valid: true},
		endsAt:                  sql.NullTime{Time: time.Date(2026, 7, 1, 19, 0, 0, 0, time.UTC), Valid: true},
		revisedAt:               time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

func TestParticipantShiftCalendarParsesWithEscapingAndFolding(t *testing.T) {
	body := participantShiftCalendar([]participantAssignmentDTO{shiftCalendarFixture()})
	if strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") {
		t.Fatalf("calendar contains bare LF: %q", body)
	}
	events := parseICalendar(t, body)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	event := events[0]
	fixture := shiftCalendarFixture()
	for name, want := range map[string]string{
		"UID":           "staffing-6f2c1d2e-0000-4000-8000-000000000001@os.subcult.tv",
		"DTSTART":       "20260701T180000Z",
		"DTEND":         "20260701T190000Z",
		"DTSTAMP":       "20260601T120000Z",
		"LAST-MODIFIED": "20260601T120000Z",
		"SUMMARY":       "Soundcheck · " + fixture.EventTitle,
		"LOCATION":      "Warehouse, Back room",
		"DESCRIPTION":   "Event: " + fixture.EventTitle + "\nAssignment: shift\n\nRequirements:\n" + strings.TrimSpace(fixture.ParticipantRequirements),
		"STATUS":        "CONFIRMED",
	} {
		if event[name] != want {
			t.Errorf("%s = %q, want %q", name, event[name], want)
		}
	}
}

func TestParticipantShiftCalendarRescheduleKeepsUIDAndAdvancesRevision(t *testing.T) {
	original := shiftCalendarFixture()
	rescheduled := shiftCalendarFixture()
	rescheduled.startsAt.Time = original.startsAt.Time.Add(2 * time.Hour)
	rescheduled.endsAt.Time = original.endsAt.Time.Add(2 * time.Hour)
	rescheduled.revisedAt = original.revisedAt.Add(90 * time.Minute)

	before := parseICalendar(t, participantShiftCalendar([]participantAssignmentDTO{original}))[0]
	after := parseICalendar(t, participantShiftCalendar([]participantAssignmentDTO{rescheduled}))[0]
	if before["UID"] != after["UID"] {
		t.Fatalf("UID changed on reschedule: %q -> %q", before["UID"], after["UID"])
	}
	if after["DTSTART"] != "20260701T200000Z" || after["DTEND"] != "20260701T210000Z" {
		t.Fatalf("rescheduled times = %q-%q", after["DTSTART"], after["DTEND"])
	}
	beforeSequence, errBefore := strconv.Atoi(before["SEQUENCE"])
	afterSequence, errAfter := strconv.Atoi(after["SEQUENCE"])
	if errBefore != nil || errAfter != nil || afterSequence <= beforeSequence {
		t.Fatalf("SEQUENCE did not increase: %q -> %q", before["SEQUENCE"], after["SEQUENCE"])
	}
	if before["LAST-MODIFIED"] >= after["LAST-MODIFIED"] {
		t.Fatalf("LAST-MODIFIED did not advance: %q -> %q", before["LAST-MODIFIED"], after["LAST-MODIFIED"])
	}
	if got := icalSequence(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)); got != 0 {
		t.Fatalf("pre-epoch sequence = %d, want 0", got)
	}
}

func TestParticipantShiftCalendarOmitsUntimedAndHandlesOpenEnd(t *testing.T) {
	untimed := shiftCalendarFixture()
	untimed.StaffingItemID = "untimed"
	untimed.startsAt = sql.NullTime{}
	openEnded := shiftCalendarFixture()
	openEnded.StaffingItemID = "open-ended"
	openEnded.endsAt = sql.NullTime{}
	openEnded.ParticipantRequirements = ""
	openEnded.locationDisplay = ""

	events := parseICalendar(t, participantShiftCalendar([]participantAssignmentDTO{untimed, openEnded}))
	if len(events) != 1 || events[0]["UID"] != "staffing-open-ended@os.subcult.tv" {
		t.Fatalf("events = %#v", events)
	}
	if _, ok := events[0]["DTEND"]; ok {
		t.Fatalf("open-ended shift has DTEND: %#v", events[0])
	}
	if _, ok := events[0]["LOCATION"]; ok {
		t.Fatalf("empty location emitted: %#v", events[0])
	}
	if events[0]["DESCRIPTION"] != "Event: "+openEnded.EventTitle+"\nAssignment: shift" {
		t.Fatalf("description = %q", events[0]["DESCRIPTION"])
	}

	empty := parseICalendar(t, participantShiftCalendar(nil))
	if len(empty) != 0 {
		t.Fatalf("empty calendar events = %#v", empty)
	}
}
