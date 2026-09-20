// Package atproto is the narrow protocol boundary owned by Subcult OS.
// It deliberately exposes plain strings instead of leaking unstable Indigo
// types into application modules.
package atproto

import (
	"errors"
	"fmt"

	indigosyntax "github.com/bluesky-social/indigo/atproto/syntax"
)

type IdentifierKind string

const (
	IdentifierDID    IdentifierKind = "did"
	IdentifierHandle IdentifierKind = "handle"
)

type AccountIdentifier struct {
	Value string
	Kind  IdentifierKind
}

type RecordRef struct {
	URI        string
	Authority  AccountIdentifier
	Collection string
	RecordKey  string
}

type StrongRef struct {
	Record RecordRef
	CID    string
}

// ParseAccountIdentifier validates protocol syntax only. A handle is not
// trusted until a resolver verifies its DID relationship bidirectionally.
func ParseAccountIdentifier(raw string) (AccountIdentifier, error) {
	identifier, err := indigosyntax.ParseAtIdentifier(raw)
	if err != nil {
		return AccountIdentifier{}, fmt.Errorf("invalid AT identifier: %w", err)
	}
	identifier = identifier.Normalize()
	kind := IdentifierHandle
	if identifier.IsDID() {
		kind = IdentifierDID
	}
	return AccountIdentifier{Value: identifier.String(), Kind: kind}, nil
}

// ParseRecordRef accepts only the restricted Lexicon AT URI shape naming an
// exact repository record: authority, collection, and record key.
func ParseRecordRef(raw string) (RecordRef, error) {
	uri, err := indigosyntax.ParseATURI(raw)
	if err != nil {
		return RecordRef{}, fmt.Errorf("invalid AT URI: %w", err)
	}
	uri = uri.Normalize()
	if uri.Collection().String() == "" || uri.RecordKey().String() == "" {
		return RecordRef{}, errors.New("AT URI must identify a collection record")
	}
	authority, err := ParseAccountIdentifier(uri.Authority().String())
	if err != nil {
		return RecordRef{}, err
	}
	return RecordRef{
		URI: uri.String(), Authority: authority,
		Collection: uri.Collection().String(), RecordKey: uri.RecordKey().String(),
	}, nil
}

// ParseStrongRef requires a DID-authority record URI plus a syntactically
// valid CID. Handle-authority URIs can be reassigned and are not durable.
func ParseStrongRef(rawURI, rawCID string) (StrongRef, error) {
	record, err := ParseRecordRef(rawURI)
	if err != nil {
		return StrongRef{}, err
	}
	if record.Authority.Kind != IdentifierDID {
		return StrongRef{}, errors.New("strong reference authority must be a DID")
	}
	cid, err := indigosyntax.ParseCID(rawCID)
	if err != nil {
		return StrongRef{}, fmt.Errorf("invalid CID: %w", err)
	}
	return StrongRef{Record: record, CID: cid.String()}, nil
}

func ParseCollection(raw string) (string, error) {
	nsid, err := indigosyntax.ParseNSID(raw)
	if err != nil {
		return "", fmt.Errorf("invalid collection NSID: %w", err)
	}
	return nsid.Normalize().String(), nil
}
