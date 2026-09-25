package app

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventSettlementCSVExportFinanceScopeAndContents(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "=Night, \"Market\"\n✓", 8)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, "buyer-private@example.test", "Buyer Private", "paid", 1500, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("cancelled"), "Cancelled", "cancelled", 1500, "usd")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{
		"amountCents": 275, "label": "\t=SUM(1,1), \"quoted\"\n✓", "reason": "\x01@danger, \"reason\"\nΔ",
	}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{
		"amountCents": -75, "label": "Refund correction", "reason": "Synthetic correction",
	}, http.StatusOK)

	path := "/api/events/" + eventID + "/exports/settlement.csv"
	owner := getSettlementCSV(t, fx.app, fx.ownerCookie, path, http.StatusOK)
	if got := owner.Header().Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if got := owner.Header().Get("Content-Disposition"); got != `attachment; filename="event-settlement.csv"` {
		t.Fatalf("content disposition = %q", got)
	}
	if got := owner.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache control = %q", got)
	}
	if got := owner.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff = %q", got)
	}
	if strings.Contains(owner.Body.String(), "buyer-private@example.test") || strings.Contains(owner.Body.String(), "Buyer Private") {
		t.Fatal("finance export leaked ticket participant data")
	}

	rows, err := csv.NewReader(strings.NewReader(owner.Body.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows=%d, want header, settlement, two adjustments: %#v", len(rows), rows)
	}
	header := rows[0]
	if len(header) != 40 || header[0] != "record_type" || header[11] != "gross_paid_revenue_cents" || header[12] != "adjustment_total_cents" || header[13] != "net_total_cents" || header[21] != "adjustment_label" || header[39] != "finance_current_total_cents" {
		t.Fatalf("unexpected header: %#v", header)
	}
	settlement := rows[1]
	if settlement[0] != "settlement" || settlement[1] != eventID || settlement[7] != "open" || settlement[11] != "1500" || settlement[12] != "200" || settlement[13] != "1700" || settlement[14] != "1" || settlement[16] != "1" || settlement[20] != "" {
		t.Fatalf("unexpected settlement row: %#v", settlement)
	}
	if settlement[4] != "'=Night, \"Market\"\n✓" {
		t.Fatalf("event title formula guard missing: %q", settlement[4])
	}
	for _, index := range []int{5, 6} {
		if _, err := time.Parse(time.RFC3339Nano, settlement[index]); err != nil || !strings.HasSuffix(settlement[index], "Z") {
			t.Fatalf("settlement date %d = %q, err=%v", index, settlement[index], err)
		}
	}
	adjustment := rows[2]
	// The adjustment request trims outer whitespace before storage. Its retained
	// leading formula marker must still be rendered as a literal string.
	if adjustment[0] != "adjustment" || adjustment[11] != "" || adjustment[12] != "" || adjustment[13] != "" || adjustment[20] != "275" || adjustment[21] != "'=SUM(1,1), \"quoted\"\n✓" || adjustment[22] != "'\x01@danger, \"reason\"\nΔ" || adjustment[19] == "" || adjustment[23] == "" {
		t.Fatalf("unexpected adjustment row: %#v", adjustment)
	}
	if _, err := time.Parse(time.RFC3339Nano, adjustment[24]); err != nil || !strings.HasSuffix(adjustment[24], "Z") {
		t.Fatalf("adjustment date = %q, err=%v", adjustment[24], err)
	}
	negativeAdjustment := rows[3]
	if negativeAdjustment[0] != "adjustment" || negativeAdjustment[11] != "" || negativeAdjustment[12] != "" || negativeAdjustment[13] != "" || negativeAdjustment[20] != "-75" || negativeAdjustment[21] != "Refund correction" || negativeAdjustment[22] != "Synthetic correction" {
		t.Fatalf("unexpected negative adjustment row: %#v", negativeAdjustment)
	}

	memberPersonID, memberRowID := memberIdentity(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'finance' where id = $1`, memberRowID); err != nil {
		t.Fatal(err)
	}
	finance := getSettlementCSV(t, fx.app, fx.memberCookie, path, http.StatusOK)
	if finance.Body.String() != owner.Body.String() {
		t.Fatal("finance export differed from owner export")
	}
	if memberPersonID == "" {
		t.Fatal("expected finance fixture person")
	}
}

func TestEventSettlementCSVExportFormulaSafety(t *testing.T) {
	cases := map[string]struct {
		input string
		want  string
	}{
		"plain text":                {input: "ordinary text", want: "ordinary text"},
		"formula":                   {input: "=SUM(1,1)", want: "'=SUM(1,1)"},
		"leading whitespace":        {input: "\t +SUM(1,1)", want: "'\t +SUM(1,1)"},
		"leading control character": {input: "\x01@danger", want: "'\x01@danger"},
		"blank whitespace":          {input: "\t\n", want: "\t\n"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if got := csvFormulaSafe(testCase.input); got != testCase.want {
				t.Fatalf("csvFormulaSafe(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestSettlementCSVRetainsFinanceHistoryAndSeparatesCurrentTotals(t *testing.T) {
	var body strings.Builder
	writer := csv.NewWriter(&body)
	payableID, correctedID := "payable-root", "payable-correction"
	snapshot := eventSettlementExportSnapshot{
		EventID: "event", SettlementID: "settlement", ReportID: "report", EventTitle: "Event", Currency: "usd",
		StartsAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinanceLines: []eventFinanceExportRow{
			{financeLineDTO: financeLineDTO{ID: payableID, EntryType: "payable", Direction: "expense", AmountCents: 500, Currency: "usd", Label: "=original", Reason: "planned", CreatedByPersonID: "actor", CreatedAt: "2026-01-01T00:00:00Z"}},
			{financeLineDTO: financeLineDTO{ID: correctedID, EntryType: "payable", Direction: "expense", AmountCents: 300, Currency: "usd", Label: "replacement", Reason: "corrected", CorrectsLineID: &payableID, CreatedByPersonID: "actor", CreatedAt: "2026-01-01T01:00:00Z"}, IsCurrent: true},
			{financeLineDTO: financeLineDTO{ID: "payment", EntryType: "actual_payment", Direction: "expense", AmountCents: 300, Currency: "usd", Label: "cash", Reason: "recorded", PayableLineID: &payableID, CreatedByPersonID: "actor", CreatedAt: "2026-01-01T02:00:00Z"}, IsCurrent: true},
		},
	}
	if err := writeSettlementCSV(writer, snapshot); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	rows, err := csv.NewReader(strings.NewReader(body.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	column := map[string]int{}
	for index, name := range rows[0] {
		column[name] = index
	}
	for _, name := range []string{"finance_line_id", "finance_amount_cents", "finance_corrects_line_id", "finance_is_current", "finance_current_total_cents"} {
		if _, ok := column[name]; !ok {
			t.Fatalf("missing %s from %#v", name, rows[0])
		}
	}
	var history, totals [][]string
	for _, row := range rows[1:] {
		switch row[column["record_type"]] {
		case "finance_line_history":
			history = append(history, row)
		case "finance_current_total":
			totals = append(totals, row)
		}
	}
	if len(history) != 3 || len(totals) != 2 {
		t.Fatalf("history=%d totals=%d rows=%#v", len(history), len(totals), rows)
	}
	if history[0][column["finance_amount_cents"]] != "500" || history[0][column["finance_is_current"]] != "" || history[0][column["finance_label"]] != "'=original" {
		t.Fatalf("original provenance lost: %#v", history[0])
	}
	if history[1][column["finance_corrects_line_id"]] != payableID || history[1][column["finance_is_current"]] != "true" {
		t.Fatalf("correction provenance lost: %#v", history[1])
	}
	if history[2][column["finance_payable_line_id"]] != payableID {
		t.Fatalf("stable payable root link lost: %#v", history[2])
	}
	for _, total := range totals {
		if total[column["finance_amount_cents"]] != "" || total[column["finance_current_total_cents"]] != "300" || total[column["finance_currency"]] != "usd" {
			t.Fatalf("incorrect separately counted total: %#v", total)
		}
	}
}

func TestEventSettlementCSVExportDeniesNonFinanceAndRevokedSessions(t *testing.T) {
	fx := settledCSVExportFixture(t)
	eventID := fx.eventID
	path := "/api/events/" + eventID + "/exports/settlement.csv"

	getSettlementCSV(t, fx.fixture.app, nil, path, http.StatusForbidden)
	memberPersonID, memberRowID := memberIdentity(t, fx.fixture)
	for _, role := range []string{roleOrganizer, roleCrew, roleDoor} {
		if _, err := fx.fixture.app.db.Exec(t.Context(), `update workspace_members set role = $2, revoked_at = null where id = $1`, memberRowID, role); err != nil {
			t.Fatal(err)
		}
		getSettlementCSV(t, fx.fixture.app, fx.fixture.memberCookie, path, http.StatusForbidden)
	}
	if _, err := fx.fixture.app.db.Exec(t.Context(), `update workspace_members set role = 'finance', revoked_at = null where id = $1`, memberRowID); err != nil {
		t.Fatal(err)
	}
	getSettlementCSV(t, fx.fixture.app, fx.fixture.memberCookie, path, http.StatusOK)
	postJSON(t, fx.fixture.app, fx.fixture.ownerCookie, "/api/workspaces/"+fx.fixture.workspaceID+"/members/"+memberRowID+"/revoke", map[string]any{}, http.StatusOK)
	getSettlementCSV(t, fx.fixture.app, fx.fixture.memberCookie, path, http.StatusForbidden)
	if memberPersonID == "" {
		t.Fatal("expected member identity")
	}

	getSettlementCSV(t, fx.fixture.app, nil, "/api/public/events/not-a-real-event/exports/settlement.csv", http.StatusNotFound)
	beforeClose := createEvent(t, fx.fixture, "Unclosed export", 1)
	getSettlementCSV(t, fx.fixture.app, fx.fixture.ownerCookie, "/api/events/"+mustString(t, beforeClose, "id")+"/exports/settlement.csv", http.StatusNotFound)
}

func TestEventSettlementCSVExportFinalizedSnapshotIsStable(t *testing.T) {
	fx := settledCSVExportFixture(t)
	path := "/api/events/" + fx.eventID + "/exports/settlement.csv"
	before := getSettlementCSV(t, fx.fixture.app, fx.fixture.ownerCookie, path, http.StatusOK).Body.String()
	postJSON(t, fx.fixture.app, fx.fixture.ownerCookie, "/api/events/"+fx.eventID+"/settlement/finalize", map[string]any{}, http.StatusOK)
	after := getSettlementCSV(t, fx.fixture.app, fx.fixture.ownerCookie, path, http.StatusOK).Body.String()
	if before == after {
		t.Fatal("finalization did not change the export")
	}
	finalizedRows, err := csv.NewReader(strings.NewReader(after)).ReadAll()
	if err != nil || len(finalizedRows) < 2 || finalizedRows[1][7] != "finalized" || finalizedRows[1][8] == "" || finalizedRows[1][9] == "" {
		t.Fatalf("expected finalization provenance in stable export: rows=%#v err=%v", finalizedRows, err)
	}
	again := getSettlementCSV(t, fx.fixture.app, fx.fixture.ownerCookie, path, http.StatusOK).Body.String()
	if after != again {
		t.Fatal("finalized settlement export changed without a correction")
	}
}

type settledCSVFixture struct {
	fixture lifecycleFixture
	eventID string
}

func settledCSVExportFixture(t *testing.T) settledCSVFixture {
	t.Helper()
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "CSV Closeout", 4)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("paid"), "Paid", "paid", 1500, "usd")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	return settledCSVFixture{fixture: fx, eventID: eventID}
}

func getSettlementCSV(t *testing.T, app *App, cookie *http.Cookie, path string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	app.mux.ServeHTTP(response, req)
	if response.Code != wantStatus {
		t.Fatalf("GET %s status=%d body=%s, want %d", path, response.Code, response.Body.String(), wantStatus)
	}
	return response
}
