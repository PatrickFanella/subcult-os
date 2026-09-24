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

// ResolvedRecord is a fetched, but not yet Lexicon-validated, public AT
// record plus the source identity information observed while resolving it.
// Value is the raw record JSON exactly as returned by the repo; callers
// (application code) are responsible for running it through
// ValidateAdmittedRecord before treating any field as trustworthy.
type ResolvedRecord struct {
	DID        string
	Handle     string
	Collection string
	RecordKey  string
	URI        string
	CID        string
	Value      json.RawMessage
}

// RecordNotFoundError distinguishes "the repo no longer has this record"
// (deleted) from any other resolution/network failure (unavailable), so
// callers can tell those two outcomes apart.
type RecordNotFoundError struct {
	URI string
}

func (e *RecordNotFoundError) Error() string {
	return fmt.Sprintf("record not found: %s", e.URI)
}

// RecordFetcher resolves the identity behind a RecordRef and fetches the
// record it names, applying this package's public-only outbound policy.
// Tests supply a fixture-backed implementation instead of talking to a
// network; no test in this repository should depend on live network access.
type RecordFetcher interface {
	FetchRecord(ctx context.Context, ref RecordRef) (ResolvedRecord, error)
}

// IdentityRecordFetcher is the production RecordFetcher: it resolves the
// record's authority through a hardened identity.Directory, then issues one
// com.atproto.repo.getRecord request against the resolved PDS endpoint
// through the same public-only, no-proxy HTTP client used for AT OAuth.
type IdentityRecordFetcher struct {
	directory identity.Directory
	client    *http.Client
}

// NewIdentityRecordFetcher builds the default production fetcher. If the
// default identity directory does not have the expected shape to harden
// (see hardenIdentityDirectory), FetchRecord fails closed instead of
// falling back to an unhardened directory.
func NewIdentityRecordFetcher() *IdentityRecordFetcher {
	directory := identity.DefaultDirectory()
	if !hardenIdentityDirectory(directory) {
		directory = nil
	}
	return &IdentityRecordFetcher{directory: directory, client: publicOnlyHTTPClient(10 * time.Second)}
}

func (f *IdentityRecordFetcher) FetchRecord(ctx context.Context, ref RecordRef) (ResolvedRecord, error) {
	if f == nil || f.directory == nil {
		return ResolvedRecord{}, errors.New("AT identity resolver is unavailable")
	}
	atid, err := indigosyntax.ParseAtIdentifier(ref.Authority.Value)
	if err != nil {
		return ResolvedRecord{}, fmt.Errorf("invalid record authority: %w", err)
	}
	ident, err := f.directory.Lookup(ctx, atid)
	if err != nil {
		return ResolvedRecord{}, fmt.Errorf("resolve record authority: %w", err)
	}
	pdsEndpoint := ident.PDSEndpoint()
	if strings.TrimSpace(pdsEndpoint) == "" {
		return ResolvedRecord{}, errors.New("resolved identity has no PDS endpoint")
	}
	endpointURL, err := url.Parse(pdsEndpoint)
	if err != nil || endpointURL.Scheme != "https" || endpointURL.Hostname() == "" {
		return ResolvedRecord{}, errors.New("resolved PDS endpoint is not a safe HTTPS URL")
	}
	endpointURL.Path = strings.TrimRight(endpointURL.Path, "/") + "/xrpc/com.atproto.repo.getRecord"
	query := endpointURL.Query()
	query.Set("repo", ident.DID.String())
	query.Set("collection", ref.Collection)
	query.Set("rkey", ref.RecordKey)
	endpointURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil)
	if err != nil {
		return ResolvedRecord{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return ResolvedRecord{}, fmt.Errorf("fetch public record: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ResolvedRecord{}, fmt.Errorf("read public record response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Error == "RecordNotFound" || resp.StatusCode == http.StatusNotFound {
			return ResolvedRecord{}, &RecordNotFoundError{URI: ref.URI}
		}
		return ResolvedRecord{}, fmt.Errorf("public record fetch failed with status %d", resp.StatusCode)
	}

	var out struct {
		URI   string          `json:"uri"`
		CID   string          `json:"cid"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return ResolvedRecord{}, fmt.Errorf("decode public record response: %w", err)
	}
	if strings.TrimSpace(out.CID) == "" || len(out.Value) == 0 {
		return ResolvedRecord{}, errors.New("public record response is missing cid or value")
	}

	handle := ""
	if ident.Handle.String() != "" && ident.Handle.String() != indigosyntax.HandleInvalid.String() {
		handle = ident.Handle.String()
	}
	return ResolvedRecord{
		DID:        ident.DID.String(),
		Handle:     handle,
		Collection: ref.Collection,
		RecordKey:  ref.RecordKey,
		URI:        ref.URI,
		CID:        out.CID,
		Value:      out.Value,
	}, nil
}
