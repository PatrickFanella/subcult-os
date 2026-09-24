package atproto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	indigosyntax "github.com/bluesky-social/indigo/atproto/syntax"
)

// ListedRecord is one row returned by com.atproto.repo.listRecords, resolved
// through the same hardened identity path as RecordFetcher. Value is raw,
// unvalidated record JSON; callers must still run it through
// ValidateAdmittedRecord before treating any field as trustworthy.
type ListedRecord struct {
	URI   string
	CID   string
	Value json.RawMessage
}

// RecordListPage is one bounded page of a listRecords call: the records
// admitted so far this page and the opaque cursor to resume from, empty
// when the authority's repo has no further records in this collection.
type RecordListPage struct {
	Records []ListedRecord
	Cursor  string
}

// recordListMaxPageSize bounds how many records a single listRecords call
// requests, matching the stream path's bias toward small bounded units of
// work instead of one unbounded fetch.
const recordListMaxPageSize = 100

// RecordListMaxRecordBytes bounds an individual listed record's JSON value,
// mirroring the stream path's projectionMaxRecordBytes bound. A record
// larger than this is reported via ErrRecordTooLarge instead of being
// returned, so a backfill caller can quarantine it the same way the stream
// path quarantines an oversize commit event.
const RecordListMaxRecordBytes = 64 * 1024

// ErrRecordTooLarge marks a listed record that exceeded RecordListMaxRecordBytes.
var ErrRecordTooLarge = errors.New("listed record exceeds size bound")

// ErrProjectionCursorGap is returned by a RecordLister implementation when
// it cannot resume a listing from the cursor a caller supplied (for
// example, the caller's stored resume point predates what the source can
// still serve). A caller should treat this as a signal to restart the
// listing from scratch rather than silently skipping the gap.
var ErrProjectionCursorGap = errors.New("AT record source cannot resume from the given cursor")

// RecordLister lists records of one collection from one authority's PDS,
// paging with a bounded page size. Tests supply a fixture implementation;
// no test in this repository dials a live PDS.
type RecordLister interface {
	// ListRecords returns up to one bounded page of records for did's
	// repository in collection, starting after cursor (empty for the first
	// page).
	ListRecords(ctx context.Context, did, collection, cursor string) (RecordListPage, error)
}

// IdentityRecordLister is the production RecordLister: it resolves the
// authority through a hardened identity.Directory (safe public-IP
// resolution only; see hardenIdentityDirectory) and issues
// com.atproto.repo.listRecords requests against the resolved PDS through
// the same public-only, no-proxy HTTP client used elsewhere in this package.
type IdentityRecordLister struct {
	directory identity.Directory
	client    *http.Client
}

// NewIdentityRecordLister builds the default production lister. Like
// NewIdentityRecordFetcher, it fails closed (ListRecords always errors) if
// the default identity directory does not have the expected shape to
// harden, rather than falling back to an unhardened directory.
func NewIdentityRecordLister() *IdentityRecordLister {
	directory := identity.DefaultDirectory()
	if !hardenIdentityDirectory(directory) {
		directory = nil
	}
	return &IdentityRecordLister{directory: directory, client: publicOnlyHTTPClient(10 * time.Second)}
}

func (l *IdentityRecordLister) resolvePDSEndpoint(ctx context.Context, did string) (*url.URL, string, error) {
	if l == nil || l.directory == nil {
		return nil, "", errors.New("AT identity resolver is unavailable")
	}
	atid, err := indigosyntax.ParseAtIdentifier(did)
	if err != nil {
		return nil, "", fmt.Errorf("invalid authority DID: %w", err)
	}
	ident, err := l.directory.Lookup(ctx, atid)
	if err != nil {
		return nil, "", fmt.Errorf("resolve authority: %w", err)
	}
	pdsEndpoint := ident.PDSEndpoint()
	if strings.TrimSpace(pdsEndpoint) == "" {
		return nil, "", errors.New("resolved identity has no PDS endpoint")
	}
	endpointURL, err := url.Parse(pdsEndpoint)
	if err != nil || endpointURL.Scheme != "https" || endpointURL.Hostname() == "" {
		return nil, "", errors.New("resolved PDS endpoint is not a safe HTTPS URL")
	}
	return endpointURL, ident.DID.String(), nil
}

// ListRecords issues one bounded com.atproto.repo.listRecords request. The
// public-only HTTP transport (ssrf.PublicOnlyTransport, applied by
// publicOnlyHTTPClient) rejects a connection to a private, loopback, or
// link-local address at dial time regardless of what the identity directory
// or DNS resolution returned, so a PDS endpoint that resolves to a private
// address is refused even though URL parsing above only checks scheme.
func (l *IdentityRecordLister) ListRecords(ctx context.Context, did, collection, cursor string) (RecordListPage, error) {
	endpointURL, resolvedDID, err := l.resolvePDSEndpoint(ctx, did)
	if err != nil {
		return RecordListPage{}, err
	}
	endpointURL.Path = strings.TrimRight(endpointURL.Path, "/") + "/xrpc/com.atproto.repo.listRecords"
	query := endpointURL.Query()
	query.Set("repo", resolvedDID)
	query.Set("collection", collection)
	query.Set("limit", fmt.Sprintf("%d", recordListMaxPageSize))
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	endpointURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil)
	if err != nil {
		return RecordListPage{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := l.client.Do(req)
	if err != nil {
		return RecordListPage{}, fmt.Errorf("list public records: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return RecordListPage{}, fmt.Errorf("read list records response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return RecordListPage{}, fmt.Errorf("list records failed with status %d", resp.StatusCode)
	}

	var out struct {
		Records []struct {
			URI   string          `json:"uri"`
			CID   string          `json:"cid"`
			Value json.RawMessage `json:"value"`
		} `json:"records"`
		Cursor string `json:"cursor"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return RecordListPage{}, fmt.Errorf("decode list records response: %w", err)
	}

	page := RecordListPage{Cursor: out.Cursor}
	for _, record := range out.Records {
		if len(record.Value) > RecordListMaxRecordBytes {
			return RecordListPage{}, fmt.Errorf("%w: %s", ErrRecordTooLarge, record.URI)
		}
		page.Records = append(page.Records, ListedRecord{URI: record.URI, CID: record.CID, Value: record.Value})
	}
	return page, nil
}
