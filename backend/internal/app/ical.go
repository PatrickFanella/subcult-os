package app

import (
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// icalEvent is one VEVENT in an RFC 5545 calendar file. Revised is used for
// DTSTAMP and LAST-MODIFIED and derives SEQUENCE, so a later revision of the
// same UID replaces an earlier import in calendar clients.
type icalEvent struct {
	UID         string
	Revised     time.Time
	Start       time.Time
	End         *time.Time
	Summary     string
	Description string
	Location    string
}

const (
	icalDateTimeLayout = "20060102T150405Z"
	icalMaxLineOctets  = 75
)

// icalSequenceEpoch anchors SEQUENCE values. Seconds since this instant fit
// the RFC 5545 32-bit integer range until 2094 and only increase while the
// revision timestamp increases.
var icalSequenceEpoch = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func icalSequence(revised time.Time) int64 {
	seconds := int64(revised.Sub(icalSequenceEpoch) / time.Second)
	if seconds < 0 {
		return 0
	}
	return seconds
}

// writeICalendar serializes events as a VCALENDAR with CRLF line endings,
// escaped TEXT values and lines folded at 75 octets without splitting UTF-8
// sequences. All times are written in UTC.
func writeICalendar(prodID string, events []icalEvent) string {
	var b strings.Builder
	line := func(name, value string) {
		b.WriteString(icalFold(name + ":" + value))
		b.WriteString("\r\n")
	}
	line("BEGIN", "VCALENDAR")
	line("VERSION", "2.0")
	line("PRODID", prodID)
	line("CALSCALE", "GREGORIAN")
	for _, event := range events {
		revised := event.Revised.UTC().Format(icalDateTimeLayout)
		line("BEGIN", "VEVENT")
		line("UID", icalEscapeText(event.UID))
		line("DTSTAMP", revised)
		line("LAST-MODIFIED", revised)
		line("SEQUENCE", strconv.FormatInt(icalSequence(event.Revised), 10))
		line("DTSTART", event.Start.UTC().Format(icalDateTimeLayout))
		if event.End != nil {
			line("DTEND", event.End.UTC().Format(icalDateTimeLayout))
		}
		line("SUMMARY", icalEscapeText(event.Summary))
		if event.Description != "" {
			line("DESCRIPTION", icalEscapeText(event.Description))
		}
		if event.Location != "" {
			line("LOCATION", icalEscapeText(event.Location))
		}
		line("STATUS", "CONFIRMED")
		line("END", "VEVENT")
	}
	line("END", "VCALENDAR")
	return b.String()
}

// icalEscapeText applies RFC 5545 section 3.3.11 TEXT escaping. Line breaks
// become \n; other control characters except HTAB are not allowed in TEXT
// and are dropped.
func icalEscapeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case ';':
			b.WriteString(`\;`)
		case ',':
			b.WriteString(`\,`)
		case '\n':
			b.WriteString(`\n`)
		default:
			if r == '\t' || !unicode.IsControl(r) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// icalFold splits a content line into physical lines of at most 75 octets,
// each continuation starting with one space (RFC 5545 section 3.1). It breaks
// only between runes. The result has no trailing CRLF.
func icalFold(contentLine string) string {
	if len(contentLine) <= icalMaxLineOctets {
		return contentLine
	}
	var b strings.Builder
	lineOctets := 0
	for _, r := range contentLine {
		size := utf8.RuneLen(r)
		if size < 0 {
			r, size = utf8.RuneError, utf8.RuneLen(utf8.RuneError)
		}
		if lineOctets+size > icalMaxLineOctets {
			b.WriteString("\r\n ")
			lineOctets = 1
		}
		b.WriteRune(r)
		lineOctets += size
	}
	return b.String()
}
