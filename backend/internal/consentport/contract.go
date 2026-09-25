// Package consentport defines an offline, synthetic-only research format for
// moving consent evidence between two applications. It is not an application
// API, database model, delivery authorization mechanism, or wire protocol.
package consentport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	// FormatV1 identifies this deliberately narrow research fixture shape.
	FormatV1 = "subcult.consent-port/v1"
	// TransferEvidenceOnly makes the non-authorizing transfer mode explicit.
	TransferEvidenceOnly = "evidence_only"
	// MaxBundleBytes bounds input before JSON parsing so a fixture importer
	// cannot be used as an unbounded memory sink.
	MaxBundleBytes = 256 * 1024
	maxRecords     = 64
)

// Bundle is an offline research artifact. Exporter identifies the system
// producing evidence and the authority reference under which it did so.
// It must never contain raw confirm/withdraw tokens, credentials, message
// bodies, ticket/contact/membership records, or public URLs.
type Bundle struct {
	Format       string   `json:"format"`
	TransferMode string   `json:"transferMode"`
	Exporter     Exporter `json:"exporter"`
	CreatedAt    string   `json:"createdAt"`
	Records      []Record `json:"records"`
}

type Exporter struct {
	ID           string `json:"id"`
	AuthorityRef string `json:"authorityRef"`
}

// Record preserves evidence about one source grant. It cannot assert that a
// receiving application may contact a recipient.
type Record struct {
	SourceGrantRef string       `json:"sourceGrantRef"`
	Sender         Sender       `json:"sender"`
	Recipient      Recipient    `json:"recipient"`
	Channel        string       `json:"channel"`
	Purpose        string       `json:"purpose"`
	Scope          string       `json:"scope"`
	Profile        string       `json:"profile"`
	Disclosure     Disclosure   `json:"disclosure"`
	Capture        Capture      `json:"capture"`
	Verification   Verification `json:"verification"`
	Withdrawal     *Negative    `json:"withdrawal,omitempty"`
	Suppression    *Negative    `json:"suppression,omitempty"`
	Retention      Retention    `json:"retention"`
}

type Sender struct {
	ID           string `json:"id"`
	ControllerID string `json:"controllerId"`
}

type Recipient struct {
	Kind              string `json:"kind"`
	NormalizedAddress string `json:"normalizedAddress"`
}

type Disclosure struct {
	Version string `json:"version"`
	Text    string `json:"text"`
	SHA256  string `json:"sha256"`
}

type Capture struct {
	Source    string `json:"source"`
	GrantedAt string `json:"grantedAt"`
}

type Verification struct {
	Method     string `json:"method"`
	VerifiedAt string `json:"verifiedAt"`
}

type Negative struct {
	At         string `json:"at"`
	Reason     string `json:"reason"`
	Provenance string `json:"provenance"`
}

type Retention struct {
	Restriction string `json:"restriction"`
}

// Decode reads one bounded, complete JSON document and rejects fields not in
// this research contract. A caller must still apply its own local trust and
// policy rules; Validate is structural only.
func Decode(r io.Reader) (Bundle, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBundleBytes+1))
	if err != nil {
		return Bundle{}, fmt.Errorf("read consent-port bundle: %w", err)
	}
	if len(data) > MaxBundleBytes {
		return Bundle{}, fmt.Errorf("consent-port bundle exceeds %d bytes", MaxBundleBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var bundle Bundle
	if err := decoder.Decode(&bundle); err != nil {
		return Bundle{}, fmt.Errorf("decode consent-port bundle: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Bundle{}, errors.New("consent-port bundle contains multiple JSON values")
		}
		return Bundle{}, fmt.Errorf("read trailing consent-port JSON: %w", err)
	}
	if err := bundle.Validate(); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

// Encode validates before producing a bounded research fixture.
func Encode(bundle Bundle) ([]byte, error) {
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("encode consent-port bundle: %w", err)
	}
	if len(data) > MaxBundleBytes {
		return nil, fmt.Errorf("consent-port bundle exceeds %d bytes", MaxBundleBytes)
	}
	return data, nil
}

// Validate checks format integrity only. It intentionally does not decide
// whether a sender, profile, scope, disclosure, or grant is locally trusted.
func (b Bundle) Validate() error {
	if b.Format != FormatV1 {
		return errors.New("unsupported consent-port format")
	}
	if b.TransferMode != TransferEvidenceOnly {
		return errors.New("consent-port transfer mode must be evidence_only")
	}
	if err := required("exporter.id", b.Exporter.ID); err != nil {
		return err
	}
	if err := required("exporter.authorityRef", b.Exporter.AuthorityRef); err != nil {
		return err
	}
	if err := timestamp("createdAt", b.CreatedAt); err != nil {
		return err
	}
	if len(b.Records) == 0 || len(b.Records) > maxRecords {
		return fmt.Errorf("consent-port bundle must contain 1 through %d records", maxRecords)
	}
	seen := make(map[string]struct{}, len(b.Records))
	for index, record := range b.Records {
		if err := record.validate(); err != nil {
			return fmt.Errorf("record %d: %w", index, err)
		}
		if _, exists := seen[record.SourceGrantRef]; exists {
			return fmt.Errorf("record %d: duplicate sourceGrantRef", index)
		}
		seen[record.SourceGrantRef] = struct{}{}
	}
	return nil
}

func (r Record) validate() error {
	for _, field := range []struct{ name, value string }{
		{"sourceGrantRef", r.SourceGrantRef}, {"sender.id", r.Sender.ID},
		{"sender.controllerId", r.Sender.ControllerID}, {"scope", r.Scope},
		{"profile", r.Profile}, {"disclosure.version", r.Disclosure.Version}, {"disclosure.text", r.Disclosure.Text},
		{"capture.source", r.Capture.Source}, {"retention.restriction", r.Retention.Restriction},
	} {
		if err := required(field.name, field.value); err != nil {
			return err
		}
	}
	if r.Recipient.Kind != "email" || !isNormalizedEmail(r.Recipient.NormalizedAddress) {
		return errors.New("recipient must be a normalized email address")
	}
	if r.Channel != "email" || r.Purpose != "announcement" {
		return errors.New("record must be an email announcement")
	}
	if len(r.Disclosure.SHA256) != 64 || r.Disclosure.SHA256 != strings.ToLower(r.Disclosure.SHA256) {
		return errors.New("disclosure.sha256 must be a SHA-256 hex digest")
	}
	if _, err := hex.DecodeString(r.Disclosure.SHA256); err != nil {
		return errors.New("disclosure.sha256 must be a SHA-256 hex digest")
	}
	disclosureDigest := sha256.Sum256([]byte(r.Disclosure.Text))
	if r.Disclosure.SHA256 != hex.EncodeToString(disclosureDigest[:]) {
		return errors.New("disclosure.sha256 does not match disclosure.text")
	}
	if err := timestamp("capture.grantedAt", r.Capture.GrantedAt); err != nil {
		return err
	}
	if r.Verification.Method != "email_link" {
		return errors.New("verification.method must be email_link")
	}
	if err := timestamp("verification.verifiedAt", r.Verification.VerifiedAt); err != nil {
		return err
	}
	if r.Withdrawal != nil {
		if err := r.Withdrawal.validate("withdrawal"); err != nil {
			return err
		}
	}
	if r.Suppression != nil {
		if err := r.Suppression.validate("suppression"); err != nil {
			return err
		}
	}
	return nil
}

func (n Negative) validate(prefix string) error {
	if err := timestamp(prefix+".at", n.At); err != nil {
		return err
	}
	if err := required(prefix+".reason", n.Reason); err != nil {
		return err
	}
	return required(prefix+".provenance", n.Provenance)
}

func required(name, value string) error {
	if strings.TrimSpace(value) == "" || len(value) > 256 {
		return fmt.Errorf("%s is required and must be at most 256 bytes", name)
	}
	return nil
}

func timestamp(name, value string) error {
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s must be RFC3339: %w", name, err)
	}
	return nil
}

func isNormalizedEmail(value string) bool {
	if value == "" || value != strings.ToLower(strings.TrimSpace(value)) || strings.Count(value, "@") != 1 {
		return false
	}
	local, domain, _ := strings.Cut(value, "@")
	return local != "" && domain != "" && !strings.ContainsAny(value, " \t\r\n")
}
