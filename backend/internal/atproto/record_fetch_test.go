package atproto

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	indigosyntax "github.com/bluesky-social/indigo/atproto/syntax"
)

const testFetchDID = "did:plc:vwzwgnygau7ed7b7wt5ux7y2"
const testFetchHandle = "creator.example.test"

// newTestIdentityFetcher builds an IdentityRecordFetcher whose directory
// resolves testFetchDID to server.URL and whose HTTP client is the test
// server's own client (not the hardened public-only one, which correctly
// refuses to dial loopback addresses; SSRF hardening itself is exercised by
// oauth_flow_test.go). No test in this file makes a real network call.
func newTestIdentityFetcher(t *testing.T, server *httptest.Server) *IdentityRecordFetcher {
	t.Helper()
	dir := identity.NewMockDirectory()
	did := indigosyntax.DID(testFetchDID)
	dir.Insert(identity.Identity{
		DID:    did,
		Handle: indigosyntax.Handle(testFetchHandle),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: server.URL},
		},
	})
	return &IdentityRecordFetcher{directory: dir, client: server.Client()}
}

func mustRecordRef(t *testing.T, uri string) RecordRef {
	t.Helper()
	ref, err := ParseRecordRef(uri)
	if err != nil {
		t.Fatalf("parse record ref: %v", err)
	}
	return ref
}

func TestIdentityRecordFetcherFetchesRecord(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.getRecord" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("repo"); got != testFetchDID {
			t.Fatalf("repo query = %q, want %q", got, testFetchDID)
		}
		if got := r.URL.Query().Get("collection"); got != "tv.subcult.event.occurrence" {
			t.Fatalf("collection query = %q", got)
		}
		if got := r.URL.Query().Get("rkey"); got != "3l6z6b6b6b6b6b" {
			t.Fatalf("rkey query = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uri": "at://" + testFetchDID + "/tv.subcult.event.occurrence/3l6z6b6b6b6b6b",
			"cid": "bafyreigfetchedcid",
			"value": map[string]any{
				"$type":     "tv.subcult.event.occurrence",
				"name":      "Fetched Occurrence",
				"startsAt":  "2026-10-01T20:00:00Z",
				"createdAt": "2026-09-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	fetcher := newTestIdentityFetcher(t, server)
	ref := mustRecordRef(t, "at://"+testFetchDID+"/tv.subcult.event.occurrence/3l6z6b6b6b6b6b")
	record, err := fetcher.FetchRecord(t.Context(), ref)
	if err != nil {
		t.Fatalf("FetchRecord: %v", err)
	}
	if record.DID != testFetchDID {
		t.Fatalf("DID = %q, want %q", record.DID, testFetchDID)
	}
	if record.Handle != testFetchHandle {
		t.Fatalf("Handle = %q, want %q", record.Handle, testFetchHandle)
	}
	if record.CID != "bafyreigfetchedcid" {
		t.Fatalf("CID = %q", record.CID)
	}
	var decoded map[string]any
	if err := json.Unmarshal(record.Value, &decoded); err != nil {
		t.Fatalf("decode value: %v", err)
	}
	if decoded["name"] != "Fetched Occurrence" {
		t.Fatalf("value.name = %v", decoded["name"])
	}
}

func TestIdentityRecordFetcherRecordNotFound(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "RecordNotFound", "message": "could not locate record"})
	}))
	defer server.Close()

	fetcher := newTestIdentityFetcher(t, server)
	ref := mustRecordRef(t, "at://"+testFetchDID+"/tv.subcult.event.occurrence/missing")
	_, err := fetcher.FetchRecord(t.Context(), ref)
	if err == nil {
		t.Fatal("expected an error")
	}
	var notFound *RecordNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected RecordNotFoundError, got %T: %v", err, err)
	}
}

func TestIdentityRecordFetcherUpstreamFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	fetcher := newTestIdentityFetcher(t, server)
	ref := mustRecordRef(t, "at://"+testFetchDID+"/tv.subcult.event.occurrence/broken")
	_, err := fetcher.FetchRecord(t.Context(), ref)
	if err == nil {
		t.Fatal("expected an error")
	}
	var notFound *RecordNotFoundError
	if errors.As(err, &notFound) {
		t.Fatal("a 500 upstream failure must not be classified as RecordNotFoundError")
	}
}
