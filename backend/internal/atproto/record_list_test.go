package atproto

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	indigosyntax "github.com/bluesky-social/indigo/atproto/syntax"
)

const testListDID = "did:plc:vwzwgnygau7ed7b7wt5ux7y2"

func newTestIdentityLister(t *testing.T, server *httptest.Server) *IdentityRecordLister {
	t.Helper()
	dir := identity.NewMockDirectory()
	did := indigosyntax.DID(testListDID)
	dir.Insert(identity.Identity{
		DID:    did,
		Handle: indigosyntax.Handle("creator.example.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: server.URL},
		},
	})
	return &IdentityRecordLister{directory: dir, client: server.Client()}
}

func TestIdentityRecordListerListsRecordsWithCursorPaging(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.listRecords" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("collection"); got != "tv.subcult.profile" {
			t.Fatalf("collection = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want bounded page size", got)
		}
		calls++
		cursor := r.URL.Query().Get("cursor")
		switch cursor {
		case "":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"records": []map[string]any{
					{"uri": "at://" + testListDID + "/tv.subcult.profile/self", "cid": "bafy1", "value": map[string]any{"$type": "tv.subcult.profile", "displayName": "Alice", "createdAt": "2026-09-01T00:00:00Z"}},
				},
				"cursor": "page2",
			})
		case "page2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"records": []map[string]any{},
				"cursor":  "",
			})
		default:
			t.Fatalf("unexpected cursor %q", cursor)
		}
	}))
	defer server.Close()

	lister := newTestIdentityLister(t, server)
	ctx := t.Context()

	first, err := lister.ListRecords(ctx, testListDID, "tv.subcult.profile", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Records) != 1 || first.Cursor != "page2" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	second, err := lister.ListRecords(ctx, testListDID, "tv.subcult.profile", first.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Records) != 0 || second.Cursor != "" {
		t.Fatalf("unexpected second page: %+v", second)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestIdentityRecordListerRejectsOversizeRecord(t *testing.T) {
	oversize := strings.Repeat("a", RecordListMaxRecordBytes+1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{"uri": "at://" + testListDID + "/tv.subcult.profile/self", "cid": "bafy1", "value": map[string]any{"$type": "tv.subcult.profile", "displayName": oversize}},
			},
		})
	}))
	defer server.Close()

	lister := newTestIdentityLister(t, server)
	if _, err := lister.ListRecords(t.Context(), testListDID, "tv.subcult.profile", ""); err == nil {
		t.Fatal("expected oversize record to be rejected")
	}
}

// TestIdentityRecordListerRefusesPrivatePDSEndpoint proves that a PDS
// endpoint resolving to a private address is refused even when the identity
// directory itself is not hardened against it: the public-only outbound
// transport (ssrf.PublicOnlyTransport, wired in by publicOnlyHTTPClient)
// rejects the connection at dial time, independent of URL parsing.
func TestIdentityRecordListerRefusesPrivatePDSEndpoint(t *testing.T) {
	dir := identity.NewMockDirectory()
	did := indigosyntax.DID(testListDID)
	dir.Insert(identity.Identity{
		DID:    did,
		Handle: indigosyntax.Handle("creator.example.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://127.0.0.1:65535"},
		},
	})
	lister := &IdentityRecordLister{directory: dir, client: publicOnlyHTTPClient(2 * time.Second)}

	_, err := lister.ListRecords(t.Context(), testListDID, "tv.subcult.profile", "")
	if err == nil {
		t.Fatal("expected private PDS endpoint to be refused")
	}
}

func TestIdentityRecordListerFailsClosedWithoutDirectory(t *testing.T) {
	lister := &IdentityRecordLister{}
	if _, err := lister.ListRecords(t.Context(), testListDID, "tv.subcult.profile", ""); err == nil {
		t.Fatal("expected failure without a resolver")
	}
}
