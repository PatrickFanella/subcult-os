package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
)

// fixturePublicLinkFetcher is an injectable atprotocol.RecordFetcher for
// tests: it never makes a network call. Results are keyed by the record
// URI and can be mutated between calls (for refresh tests) or set to
// return an error.
type fixturePublicLinkFetcher struct {
	byURI map[string]atprotocol.ResolvedRecord
	err   map[string]error
}

func newFixturePublicLinkFetcher() *fixturePublicLinkFetcher {
	return &fixturePublicLinkFetcher{
		byURI: map[string]atprotocol.ResolvedRecord{},
		err:   map[string]error{},
	}
}

func (f *fixturePublicLinkFetcher) FetchRecord(_ context.Context, ref atprotocol.RecordRef) (atprotocol.ResolvedRecord, error) {
	if err, ok := f.err[ref.URI]; ok {
		return atprotocol.ResolvedRecord{}, err
	}
	if record, ok := f.byURI[ref.URI]; ok {
		return record, nil
	}
	return atprotocol.ResolvedRecord{}, &atprotocol.RecordNotFoundError{URI: ref.URI}
}

func (f *fixturePublicLinkFetcher) setRecord(uri, did, handle, cid string) {
	value, _ := json.Marshal(map[string]any{
		"$type": "tv.subcult.event.occurrence",
		"name":  "Signal Night",
		"profile": map[string]any{
			"uri": "at://" + did + "/tv.subcult.profile/self",
			"cid": "bafyreiprofilecidxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		"startsAt":  "2026-10-01T20:00:00Z",
		"createdAt": "2026-09-01T00:00:00Z",
	})
	delete(f.err, uri)
	f.byURI[uri] = atprotocol.ResolvedRecord{
		DID: did, Handle: handle, Collection: "tv.subcult.event.occurrence",
		URI: uri, CID: cid, Value: value,
	}
}

func (f *fixturePublicLinkFetcher) setInvalidRecord(uri, did, cid string) {
	// Missing the required "profile" and "createdAt" fields, so admitted
	// Lexicon validation must reject it.
	value, _ := json.Marshal(map[string]any{
		"$type":    "tv.subcult.event.occurrence",
		"name":     "Broken Record",
		"startsAt": "2026-10-01T20:00:00Z",
	})
	delete(f.err, uri)
	f.byURI[uri] = atprotocol.ResolvedRecord{DID: did, URI: uri, CID: cid, Value: value}
}

func (f *fixturePublicLinkFetcher) setError(uri string, err error) {
	delete(f.byURI, uri)
	f.err[uri] = err
}

func (f *fixturePublicLinkFetcher) setNotFound(uri string) {
	f.setError(uri, &atprotocol.RecordNotFoundError{URI: uri})
}

const testPublicLinkDID = "did:plc:eventlinktestauthority0000"
const testPublicLinkURI = "at://" + testPublicLinkDID + "/tv.subcult.event.occurrence/3l7aaaaaaaaaa"

func newPublicLinkFixture(t *testing.T) (lifecycleFixture, *fixturePublicLinkFetcher) {
	t.Helper()
	fx := newCulturalFixture(t)
	fetcher := newFixturePublicLinkFetcher()
	fx.app.recordFetcher = fetcher
	return fx, fetcher
}

func TestEventPublicLinkPreviewDoesNotPersist(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")

	resp := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/preview", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	body := mustObject(t, resp.JSON)
	sourceIdentity := mustObject(t, body["sourceIdentity"])
	if sourceIdentity["did"] != testPublicLinkDID {
		t.Fatalf("sourceIdentity.did = %v, want %v", sourceIdentity["did"], testPublicLinkDID)
	}
	if sourceIdentity["handle"] != "creator.example.test" {
		t.Fatalf("sourceIdentity.handle = %v", sourceIdentity["handle"])
	}
	if body["cid"] != "bafyreicidone" {
		t.Fatalf("cid = %v", body["cid"])
	}
	eventFields := mustObject(t, body["event"])
	if eventFields["name"] != "Signal Night" {
		t.Fatalf("event.name = %v", eventFields["name"])
	}

	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_public_links where event_id = $1`, eventID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("preview must not persist a link, found %d rows", count)
	}
}

func TestEventPublicLinkPreviewRejectsInvalidRecord(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setInvalidRecord(testPublicLinkURI, testPublicLinkDID, "bafyreiinvalid")

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/preview", map[string]any{"publicUri": testPublicLinkURI}, http.StatusUnprocessableEntity)
}

func TestEventPublicLinkAttachIsIdempotent(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")

	first := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	firstID := mustString(t, first.JSON, "id")

	second := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	secondID := mustString(t, second.JSON, "id")
	if firstID != secondID {
		t.Fatalf("repeated attach created a new row: %s != %s", firstID, secondID)
	}

	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_public_links where event_id = $1 and public_uri = $2`, eventID, testPublicLinkURI).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one row after repeated attach, got %d", count)
	}
}

func TestEventPublicLinkCrossWorkspaceAccessDenied(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")
	attach := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	linkID := mustString(t, attach.JSON, "id")

	other := newLifecycleFixture(t, fx.app)

	postJSON(t, other.app, other.ownerCookie, "/api/events/"+eventID+"/public-links/preview", map[string]any{"publicUri": testPublicLinkURI}, http.StatusForbidden)
	getJSON(t, other.app, other.ownerCookie, "/api/events/"+eventID+"/public-links", http.StatusForbidden)
	postJSON(t, other.app, other.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusForbidden)
	postJSON(t, other.app, other.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID+"/refresh", map[string]any{}, http.StatusForbidden)
	doJSON(t, http.MethodDelete, other.app, other.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID, nil, http.StatusForbidden)
}

func TestEventPublicLinkListAndDetach(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")

	attach := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	linkID := mustString(t, attach.JSON, "id")

	list := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/public-links", http.StatusOK)
	items, ok := list.JSON.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected exactly one listed link, got %#v", list.JSON)
	}

	doJSON(t, http.MethodDelete, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID, nil, http.StatusOK)
	doJSON(t, http.MethodDelete, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID, nil, http.StatusNotFound)

	listAfter := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", http.StatusOK)
	itemsAfter, ok := listAfter.JSON.([]any)
	if !ok || len(itemsAfter) != 0 {
		t.Fatalf("expected no links after detach, got %#v", listAfter.JSON)
	}
}

func TestEventPublicLinkRefreshDetectsChangedCID(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")
	attach := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	linkID := mustString(t, attach.JSON, "id")

	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidtwo")
	refreshed := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID+"/refresh", map[string]any{}, http.StatusOK)
	body := mustObject(t, refreshed.JSON)
	if body["status"] != "changed" {
		t.Fatalf("status = %v, want changed", body["status"])
	}
	if body["observedCid"] != "bafyreicidtwo" {
		t.Fatalf("observedCid = %v, want updated cid", body["observedCid"])
	}
}

func TestEventPublicLinkRefreshDetectsUnavailable(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")
	attach := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	linkID := mustString(t, attach.JSON, "id")

	fetcher.setError(testPublicLinkURI, errors.New("identity resolution failed: dial tcp timeout"))
	refreshed := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID+"/refresh", map[string]any{}, http.StatusOK)
	body := mustObject(t, refreshed.JSON)
	if body["status"] != "unavailable" {
		t.Fatalf("status = %v, want unavailable", body["status"])
	}
	if body["lastError"] == nil {
		t.Fatal("expected lastError to be recorded")
	}
	if body["observedCid"] != "bafyreicidone" {
		t.Fatalf("observedCid should stay unchanged while unavailable, got %v", body["observedCid"])
	}
}

func TestEventPublicLinkRefreshDetectsDeleted(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")
	fetcher.setRecord(testPublicLinkURI, testPublicLinkDID, "creator.example.test", "bafyreicidone")
	attach := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": testPublicLinkURI}, http.StatusOK)
	linkID := mustString(t, attach.JSON, "id")

	fetcher.setNotFound(testPublicLinkURI)
	refreshed := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID+"/refresh", map[string]any{}, http.StatusOK)
	body := mustObject(t, refreshed.JSON)
	if body["status"] != "deleted" {
		t.Fatalf("status = %v, want deleted", body["status"])
	}

	// A deleted or unavailable public record must not touch the local
	// event row itself; event editing stays usable.
	stillEditable := patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID, map[string]any{"title": "Signal Night (rescheduled)"}, http.StatusOK)
	if mustString(t, stillEditable.JSON, "title") != "Signal Night (rescheduled)" {
		t.Fatalf("event editing must remain usable after a deleted public link: %#v", stillEditable.JSON)
	}
}

func TestEventPublicLinkPreviewRequiresDIDAuthority(t *testing.T) {
	fx, _ := newPublicLinkFixture(t)
	event := createEvent(t, fx, "Signal Night", 100)
	eventID := mustString(t, event, "id")

	handleURI := fmt.Sprintf("at://creator.example.test/tv.subcult.event.occurrence/3l7aaaaaaaaaa")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/preview", map[string]any{"publicUri": handleURI}, http.StatusBadRequest)
}
