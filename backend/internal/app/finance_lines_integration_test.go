package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEventFinanceLinesAppendOnlyCorrectionsAndStablePayableRoot(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Finance ledger", 1), "id")
	base := func(key string) map[string]any {
		return map[string]any{"entryType": "payable", "direction": "expense", "amountCents": 500, "currency": "usd", "label": "Venue", "reason": "Agreement", "requestKey": key}
	}
	first := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", base("00000000-0000-4000-8000-000000000001"), http.StatusOK)
	firstID := mustString(t, first.JSON, "id")

	zeroOriginal := base("00000000-0000-4000-8000-000000000002")
	zeroOriginal["amountCents"] = 0
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", zeroOriginal, http.StatusBadRequest)

	correction := base("00000000-0000-4000-8000-000000000003")
	correction["amountCents"] = 0
	correction["correctsLineId"] = firstID
	corrected := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", correction, http.StatusOK)
	correctedID := mustString(t, corrected.JSON, "id")

	stale := base("00000000-0000-4000-8000-000000000004")
	stale["correctsLineId"] = firstID
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", stale, http.StatusConflict)

	payment := map[string]any{"entryType": "actual_payment", "direction": "expense", "amountCents": 500, "currency": "usd", "label": "Cash", "reason": "Recorded by operator", "occurredAt": "2026-01-02T03:04:05-05:00", "payableLineId": firstID, "requestKey": "00000000-0000-4000-8000-000000000005"}
	paid := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", payment, http.StatusOK)
	paidObject := mustObject(t, paid.JSON)
	if paidObject["payableLineId"] != firstID || paidObject["occurredAt"] != "2026-01-02T08:04:05Z" {
		t.Fatalf("payment root link or UTC timestamp = %#v", paidObject)
	}

	paymentCorrection := map[string]any{"entryType": "actual_payment", "direction": "expense", "amountCents": 450, "currency": "usd", "label": "Cash corrected", "reason": "Count corrected", "occurredAt": "2026-01-02T03:04:05-05:00", "payableLineId": firstID, "correctsLineId": mustString(t, paid.JSON, "id"), "requestKey": "00000000-0000-4000-8000-000000000006"}
	paymentCorrectionResponse := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", paymentCorrection, http.StatusOK)
	correctionReplay := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", paymentCorrection, http.StatusOK)
	if mustString(t, correctionReplay.JSON, "id") != mustString(t, paymentCorrectionResponse.JSON, "id") {
		t.Fatal("same request key did not replay original correction")
	}
	replayed := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", payment, http.StatusOK)
	if mustString(t, replayed.JSON, "id") != mustString(t, paid.JSON, "id") {
		t.Fatal("same request key did not replay original payment")
	}
	changed := map[string]any{"entryType": "actual_payment", "direction": "expense", "amountCents": 400, "currency": "usd", "label": "Cash", "reason": "Recorded by operator", "occurredAt": "2026-01-02T03:04:05-05:00", "payableLineId": firstID, "requestKey": "00000000-0000-4000-8000-000000000005"}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", changed, http.StatusConflict)

	lines := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/finance-lines", http.StatusOK)
	items, ok := lines.JSON.([]any)
	if !ok || len(items) != 4 {
		t.Fatalf("history = %#v", lines.JSON)
	}
	if correctedID == firstID {
		t.Fatal("correction overwrote original")
	}
}

func financeLinePayload(key string) map[string]any {
	return map[string]any{"entryType": "budget", "direction": "income", "amountCents": 100, "currency": "usd", "label": "Donation", "reason": "Operator record", "requestKey": key}
}

func TestEventFinanceLinesAuthorityMatrix(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Finance authority", 1), "id")
	path := "/api/events/" + eventID + "/finance-lines"
	getJSON(t, fx.app, fx.ownerCookie, path, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, path, financeLinePayload("10000000-0000-4000-8000-000000000001"), http.StatusOK)

	_, memberID := memberIdentity(t, fx)
	for _, role := range []string{roleOrganizer, roleCrew, roleDoor} {
		if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role=$2, revoked_at=null, removed_at=null, expires_at=null where id=$1`, memberID, role); err != nil {
			t.Fatal(err)
		}
		getJSON(t, fx.app, fx.memberCookie, path, http.StatusForbidden)
		postJSON(t, fx.app, fx.memberCookie, path, financeLinePayload("10000000-0000-4000-8000-00000000000"+string(role[len(role)-1])), http.StatusForbidden)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role='finance', revoked_at=null, expires_at=null where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, path, http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, path, financeLinePayload("10000000-0000-4000-8000-000000000010"), http.StatusOK)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=clock_timestamp() where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, path, http.StatusForbidden)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=null, expires_at=clock_timestamp()-interval '1 second' where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.memberCookie, path, financeLinePayload("10000000-0000-4000-8000-000000000011"), http.StatusForbidden)
	getJSON(t, fx.app, nil, path, http.StatusForbidden)
}

func TestEventFinanceLinesRejectInvalidAndCrossScopedPayableLinks(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Finance links", 1), "id")
	path := "/api/events/" + eventID + "/finance-lines"
	payable := map[string]any{"entryType": "payable", "direction": "expense", "amountCents": 500, "currency": "usd", "label": "Venue", "reason": "Agreement", "requestKey": "20000000-0000-4000-8000-000000000001"}
	root := postJSON(t, fx.app, fx.ownerCookie, path, payable, http.StatusOK)
	rootID := mustString(t, root.JSON, "id")
	correction := map[string]any{"entryType": "payable", "direction": "expense", "amountCents": 400, "currency": "usd", "label": "Venue", "reason": "Correction", "correctsLineId": rootID, "requestKey": "20000000-0000-4000-8000-000000000002"}
	corrected := postJSON(t, fx.app, fx.ownerCookie, path, correction, http.StatusOK)
	correctedID := mustString(t, corrected.JSON, "id")
	payment := func(key, currency, link string) map[string]any {
		return map[string]any{"entryType": "actual_payment", "direction": "expense", "amountCents": 400, "currency": currency, "label": "Cash", "reason": "Record", "occurredAt": "2026-01-02T03:04:05Z", "payableLineId": link, "requestKey": key}
	}
	postJSON(t, fx.app, fx.ownerCookie, path, payment("20000000-0000-4000-8000-000000000003", "usd", correctedID), http.StatusBadRequest)
	postJSON(t, fx.app, fx.ownerCookie, path, payment("20000000-0000-4000-8000-000000000004", "eur", rootID), http.StatusBadRequest)
	otherID := mustString(t, createEvent(t, fx, "Other finance", 1), "id")
	otherPayable := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+otherID+"/finance-lines", map[string]any{"entryType": "payable", "direction": "expense", "amountCents": 500, "currency": "usd", "label": "Other", "reason": "Other", "requestKey": "20000000-0000-4000-8000-000000000005"}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, path, payment("20000000-0000-4000-8000-000000000006", "usd", mustString(t, otherPayable.JSON, "id")), http.StatusBadRequest)
	otherWorkspace := newLifecycleFixture(t, fx.app)
	otherWorkspaceEvent := mustString(t, createEvent(t, otherWorkspace, "Cross workspace finance", 1), "id")
	otherWorkspacePayable := postJSON(t, fx.app, otherWorkspace.ownerCookie, "/api/events/"+otherWorkspaceEvent+"/finance-lines", map[string]any{"entryType": "payable", "direction": "expense", "amountCents": 500, "currency": "usd", "label": "Elsewhere", "reason": "Elsewhere", "requestKey": "20000000-0000-4000-8000-000000000011"}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, path, payment("20000000-0000-4000-8000-000000000012", "usd", mustString(t, otherWorkspacePayable.JSON, "id")), http.StatusBadRequest)

	for _, bad := range []map[string]any{
		{"entryType": "receipt", "direction": "income", "amountCents": 1, "currency": "usd", "label": "x", "reason": "x", "requestKey": "20000000-0000-4000-8000-000000000013"},
		{"entryType": "payable", "direction": "income", "amountCents": 1, "currency": "usd", "label": "x", "reason": "x", "requestKey": "not-a-uuid"},
		{"entryType": "budget", "direction": "income", "amountCents": -1, "currency": "usd", "label": "x", "reason": "x", "requestKey": "20000000-0000-4000-8000-000000000007"},
		{"entryType": "budget", "direction": "income", "amountCents": 1, "currency": "us", "label": "x", "reason": "x", "requestKey": "20000000-0000-4000-8000-000000000008"},
		{"entryType": "budget", "direction": "income", "amountCents": 1, "currency": "usd", "label": "x", "reason": "x", "dueAt": "2026-01-01T00:00:00Z", "requestKey": "20000000-0000-4000-8000-000000000009"},
		{"entryType": "actual_payment", "direction": "income", "amountCents": 1, "currency": "usd", "label": "x", "reason": "x", "requestKey": "20000000-0000-4000-8000-000000000010"},
		{"entryType": "budget", "direction": "income", "amountCents": 1, "currency": "usd", "label": "x", "reason": "x", "requestKey": "not-a-uuid"},
		{"entryType": "payable", "direction": "expense", "amountCents": 1, "currency": "usd", "label": "x", "reason": "x", "dueAt": "not-a-time", "requestKey": "20000000-0000-4000-8000-000000000014"},
	} {
		postJSON(t, fx.app, fx.ownerCookie, path, bad, http.StatusBadRequest)
	}
}

func TestEventFinanceLineCorrectionRaceHasOneWinner(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Finance correction race", 1), "id")
	path := "/api/events/" + eventID + "/finance-lines"
	original := postJSON(t, fx.app, fx.ownerCookie, path, financeLinePayload("30000000-0000-4000-8000-000000000001"), http.StatusOK)
	originalID := mustString(t, original.JSON, "id")
	start := make(chan struct{})
	results := make(chan int, 2)
	for _, key := range []string{"30000000-0000-4000-8000-000000000002", "30000000-0000-4000-8000-000000000003"} {
		payload := financeLinePayload(key)
		payload["amountCents"] = 50
		payload["correctsLineId"] = originalID
		go func(payload map[string]any) {
			<-start
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(fx.ownerCookie)
			rec := httptest.NewRecorder()
			fx.app.Handler().ServeHTTP(rec, req)
			results <- rec.Code
		}(payload)
	}
	close(start)
	first, second := <-results, <-results
	if !((first == http.StatusOK && second == http.StatusConflict) || (first == http.StatusConflict && second == http.StatusOK)) {
		t.Fatalf("concurrent corrections = %d, %d; want one 200 and one 409", first, second)
	}
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_finance_lines where event_id=$1 and corrects_line_id=$2`, eventID, originalID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("successors=%d err=%v", count, err)
	}
}

func TestEventFinanceLineRechecksRevocationAfterEventLock(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Finance revoke race", 1), "id")
	path := "/api/events/" + eventID + "/finance-lines"
	_, memberID := memberIdentity(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role='finance' where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(t.Context(), `select id from events where id=$1 for update`, eventID); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		body, _ := json.Marshal(financeLinePayload("40000000-0000-4000-8000-000000000001"))
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(fx.memberCookie)
		rec := httptest.NewRecorder()
		fx.app.Handler().ServeHTTP(rec, req)
		result <- rec.Code
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting bool
		if err := fx.app.db.QueryRow(t.Context(), `select exists (select 1 from pg_stat_activity where datname=current_database() and wait_event_type='Lock' and query like '%select id from events where id=$1 for update%')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("finance request did not block on event lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=clock_timestamp() where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != http.StatusForbidden {
			t.Fatalf("revoked blocked finance request=%d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for blocked finance request")
	}
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_finance_lines where event_id=$1`, eventID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("revoked request wrote rows=%d err=%v", count, err)
	}
}
