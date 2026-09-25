package consentport_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/PatrickFanella/subcult-os/internal/consentport"
)

// receiverBundle and its related types are intentionally separate from the
// source contract. This receiver is a second, independently-written fixture
// application: it decodes JSON directly, applies a local trust policy, and
// does not call consentport.Decode or consentport.Bundle.Validate.
type receiverBundle struct {
	Format       string           `json:"format"`
	TransferMode string           `json:"transferMode"`
	Exporter     receiverExporter `json:"exporter"`
	CreatedAt    string           `json:"createdAt"`
	Records      []receiverRecord `json:"records"`
}

type receiverExporter struct {
	ID           string `json:"id"`
	AuthorityRef string `json:"authorityRef"`
}

type receiverRecord struct {
	SourceGrantRef string `json:"sourceGrantRef"`
	Sender         struct {
		ID           string `json:"id"`
		ControllerID string `json:"controllerId"`
	} `json:"sender"`
	Recipient struct {
		Kind              string `json:"kind"`
		NormalizedAddress string `json:"normalizedAddress"`
	} `json:"recipient"`
	Channel    string `json:"channel"`
	Purpose    string `json:"purpose"`
	Scope      string `json:"scope"`
	Profile    string `json:"profile"`
	Disclosure struct {
		Version string `json:"version"`
		Text    string `json:"text"`
		SHA256  string `json:"sha256"`
	} `json:"disclosure"`
	Capture struct {
		Source    string `json:"source"`
		GrantedAt string `json:"grantedAt"`
	} `json:"capture"`
	Verification struct {
		Method     string `json:"method"`
		VerifiedAt string `json:"verifiedAt"`
	} `json:"verification"`
	Withdrawal  *receiverNegative `json:"withdrawal,omitempty"`
	Suppression *receiverNegative `json:"suppression,omitempty"`
	Retention   struct {
		Restriction string `json:"restriction"`
	} `json:"retention"`
}

type receiverNegative struct {
	At         string `json:"at"`
	Reason     string `json:"reason"`
	Provenance string `json:"provenance"`
}

type receiverStatus string

const syntheticDisclosureSHA256 = "31b2f8c04c9cd824605bec3d578ee9a4da959d5a30fbf1e5b05e1a2908b1463b"

const (
	receiverEvidence   receiverStatus = "evidence"
	receiverWithdrawn  receiverStatus = "withdrawn"
	receiverSuppressed receiverStatus = "suppressed"
)

type receiverEntry struct {
	Status receiverStatus
	// Transfer is evidence-only: no accepted imported row can authorize a
	// message in this receiving fixture.
	CanContact bool
}

// knownSyntheticGrants is the receiver's local provenance registry. It is
// intentionally independent from the source JSON contract: a receiving app
// only accepts evidence for a source grant it already recognizes under a
// known continuing sender. This is synthetic test policy, not production
// identity matching or an application database.
var knownSyntheticGrants = map[string]receiverBinding{
	"grant-active-001":     {senderID: "workspace-synthetic-001", controllerID: "controller-synthetic-001", recipient: "active@example.test", channel: "email", purpose: "announcement", scope: "monthly-newsletter", profile: "newsletter"},
	"grant-withdrawn-002":  {senderID: "workspace-synthetic-001", controllerID: "controller-synthetic-001", recipient: "withdrawn@example.test", channel: "email", purpose: "announcement", scope: "monthly-newsletter", profile: "newsletter"},
	"grant-suppressed-003": {senderID: "workspace-synthetic-001", controllerID: "controller-synthetic-001", recipient: "suppressed@example.test", channel: "email", purpose: "announcement", scope: "monthly-newsletter", profile: "newsletter"},
}

type receiverBinding struct {
	senderID, controllerID, recipient, channel, purpose, scope, profile string
}

type independentReceiver struct {
	entries map[string]receiverEntry
}

// importBundle validates into a temporary map and swaps it only when every
// record passes. A malformed or untrusted bundle therefore has no partial
// effect. The policy is deliberately local to this fixture application.
func (r *independentReceiver) importBundle(data []byte) error {
	if len(data) > consentport.MaxBundleBytes {
		return errors.New("receiver bundle exceeds limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var bundle receiverBundle
	if err := decoder.Decode(&bundle); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("receiver requires one JSON value")
	}
	if bundle.Format != consentport.FormatV1 || bundle.TransferMode != consentport.TransferEvidenceOnly {
		return errors.New("receiver rejects an unknown format or permission transfer")
	}
	if err := receiverTimestamp("bundle createdAt", bundle.CreatedAt); err != nil {
		return err
	}
	if bundle.Exporter.ID != "synthetic-source.example.test" || bundle.Exporter.AuthorityRef != "synthetic-authority-2026-09" {
		return errors.New("receiver does not trust exporter authority")
	}
	if len(bundle.Records) == 0 || len(bundle.Records) > 64 {
		return errors.New("receiver rejects record count")
	}

	next := make(map[string]receiverEntry, len(r.entries)+len(bundle.Records))
	for key, value := range r.entries {
		next[key] = value
	}
	seen := make(map[string]struct{}, len(bundle.Records))
	for _, record := range bundle.Records {
		if err := validateReceiverRecord(record); err != nil {
			return err
		}
		key := bundle.Exporter.ID + ":" + record.SourceGrantRef
		if _, duplicate := seen[key]; duplicate {
			return errors.New("receiver rejects duplicate grant reference")
		}
		seen[key] = struct{}{}
		next[key] = combineReceiverEntry(next[key], record)
	}
	r.entries = next
	return nil
}

func validateReceiverRecord(record receiverRecord) error {
	binding, known := knownSyntheticGrants[record.SourceGrantRef]
	if !known {
		return errors.New("receiver rejects unknown grant provenance")
	}
	if record.Sender.ID != binding.senderID || record.Sender.ControllerID != binding.controllerID ||
		record.Recipient.Kind != "email" || record.Recipient.NormalizedAddress != binding.recipient ||
		record.Channel != binding.channel || record.Purpose != binding.purpose ||
		record.Scope != binding.scope || record.Profile != binding.profile {
		return errors.New("receiver rejects a grant reference rebound to different identity or scope")
	}
	if record.Disclosure.Version != "synthetic-v1" || record.Disclosure.Text != "Synthetic newsletter disclosure v1." || record.Disclosure.SHA256 != syntheticDisclosureSHA256 {
		return errors.New("receiver rejects unknown disclosure")
	}
	if record.Capture.Source != "explicit_form" || record.Retention.Restriction != "synthetic-research-only" {
		return errors.New("receiver rejects incomplete capture or retention evidence")
	}
	if err := receiverTimestamp("capture grantedAt", record.Capture.GrantedAt); err != nil {
		return err
	}
	if record.Verification.Method != "email_link" {
		return errors.New("receiver rejects incomplete verification")
	}
	if err := receiverTimestamp("verification verifiedAt", record.Verification.VerifiedAt); err != nil {
		return err
	}
	for _, negative := range []*receiverNegative{record.Withdrawal, record.Suppression} {
		if negative != nil {
			if negative.Reason == "" || negative.Provenance == "" {
				return errors.New("receiver rejects incomplete negative evidence")
			}
			if err := receiverTimestamp("negative evidence at", negative.At); err != nil {
				return err
			}
		}
	}
	return nil
}

// receiverTimestamp deliberately duplicates the source contract's parsing
// instead of reusing it, so the second application independently rejects
// malformed temporal evidence.
func receiverTimestamp(name, value string) error {
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return errors.New("receiver rejects malformed " + name)
	}
	return nil
}

func combineReceiverEntry(previous receiverEntry, record receiverRecord) receiverEntry {
	// Suppression and withdrawal are terminal for this fixture. Replaying old
	// active evidence cannot resurrect a negative state. Suppression wins if
	// both negative states are present, matching the source application's
	// suppression-first policy.
	if previous.Status == receiverSuppressed || record.Suppression != nil {
		return receiverEntry{Status: receiverSuppressed}
	}
	if previous.Status == receiverWithdrawn || record.Withdrawal != nil {
		return receiverEntry{Status: receiverWithdrawn}
	}
	return receiverEntry{Status: receiverEvidence, CanContact: false}
}

func TestSyntheticBundleStructuralContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "synthetic-audience-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := consentport.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Records) != 3 {
		t.Fatalf("records=%d, want 3", len(bundle.Records))
	}
	encoded, err := consentport.Encode(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consentport.Decode(bytes.NewReader(encoded)); err != nil {
		t.Fatalf("round trip: %v", err)
	}

	for name, mutate := range map[string]func(*consentport.Bundle){
		"altered disclosure text":   func(bundle *consentport.Bundle) { bundle.Records[0].Disclosure.Text = "Altered disclosure" },
		"altered disclosure digest": func(bundle *consentport.Bundle) { bundle.Records[0].Disclosure.SHA256 = strings.Repeat("b", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			altered := bundle
			altered.Records = append([]consentport.Record(nil), bundle.Records...)
			mutate(&altered)
			if _, err := consentport.Encode(altered); err == nil {
				t.Fatal("expected source contract rejection")
			}
		})
	}
}

func TestIndependentReceiverPreservesNegativeEvidenceAndNeverAuthorizesContact(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "synthetic-audience-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	receiver := &independentReceiver{}
	if err := receiver.importBundle(data); err != nil {
		t.Fatal(err)
	}

	assertReceiverEntry(t, receiver, "grant-active-001", receiverEvidence)
	assertReceiverEntry(t, receiver, "grant-withdrawn-002", receiverWithdrawn)
	assertReceiverEntry(t, receiver, "grant-suppressed-003", receiverSuppressed)

	// A later withdrawal wins over active evidence; a replay of the original
	// active source bundle cannot turn that grant back into usable evidence.
	delta := receiverFixtureBundle(t, "grant-active-001")
	delta.Records[0].Withdrawal = &receiverNegative{At: "2026-09-10T12:00:00Z", Reason: "recipient_requested", Provenance: "synthetic-unsubscribe"}
	withdrawalData, err := json.Marshal(delta)
	if err != nil {
		t.Fatal(err)
	}
	if err := receiver.importBundle(withdrawalData); err != nil {
		t.Fatal(err)
	}
	assertReceiverEntry(t, receiver, "grant-active-001", receiverWithdrawn)
	if err := receiver.importBundle(data); err != nil {
		t.Fatal(err)
	}
	assertReceiverEntry(t, receiver, "grant-active-001", receiverWithdrawn)
}

func TestIndependentReceiverRejectsUntrustedOrIncompleteEvidenceAtomically(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "synthetic-audience-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	receiver := &independentReceiver{}
	if err := receiver.importBundle(data); err != nil {
		t.Fatal(err)
	}
	before := cloneEntries(receiver.entries)

	for name, mutate := range map[string]func(*receiverBundle){
		"unknown exporter":        func(bundle *receiverBundle) { bundle.Exporter.ID = "unknown.example.test" },
		"unknown grant reference": func(bundle *receiverBundle) { bundle.Records[0].SourceGrantRef = "unknown-grant" },
		"sender rebinding":        func(bundle *receiverBundle) { bundle.Records[0].Sender.ID = "unknown-workspace" },
		"recipient rebinding":     func(bundle *receiverBundle) { bundle.Records[0].Recipient.NormalizedAddress = "other@example.test" },
		"purpose rebinding":       func(bundle *receiverBundle) { bundle.Records[0].Purpose = "transactional" },
		"profile rebinding":       func(bundle *receiverBundle) { bundle.Records[0].Profile = "different-profile" },
		"scope rebinding":         func(bundle *receiverBundle) { bundle.Records[0].Scope = "different-scope" },
		"unknown disclosure":      func(bundle *receiverBundle) { bundle.Records[0].Disclosure.SHA256 = strings.Repeat("b", 64) },
		"missing verification":    func(bundle *receiverBundle) { bundle.Records[0].Verification.VerifiedAt = "" },
		"malformed bundle time":   func(bundle *receiverBundle) { bundle.CreatedAt = "not-a-time" },
		"malformed capture time":  func(bundle *receiverBundle) { bundle.Records[0].Capture.GrantedAt = "not-a-time" },
		"malformed verify time":   func(bundle *receiverBundle) { bundle.Records[0].Verification.VerifiedAt = "not-a-time" },
		"malformed negative time": func(bundle *receiverBundle) {
			bundle.Records[0].Withdrawal = &receiverNegative{At: "not-a-time", Reason: "recipient_requested", Provenance: "synthetic-unsubscribe"}
		},
		"permission assertion": func(bundle *receiverBundle) { bundle.TransferMode = "permission" },
	} {
		t.Run(name, func(t *testing.T) {
			bundle := receiverFixtureBundle(t, "grant-active-001")
			mutate(&bundle)
			payload, err := json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if err := receiver.importBundle(payload); err == nil {
				t.Fatal("expected receiver refusal")
			}
			if !entriesEqual(before, receiver.entries) {
				t.Fatal("rejected import changed receiver state")
			}
		})
	}

	// Source-format validation independently rejects unbounded payloads and
	// forbidden fields before either fixture application sees a record.
	if _, err := consentport.Decode(strings.NewReader(strings.Repeat("x", consentport.MaxBundleBytes+1))); err == nil {
		t.Fatal("expected oversized bundle rejection")
	}
	if _, err := consentport.Decode(strings.NewReader(`{"format":"subcult.consent-port/v1","transferMode":"evidence_only","exporter":{"id":"x","authorityRef":"x"},"createdAt":"2026-09-24T00:00:00Z","records":[],"rawToken":"forbidden"}`)); err == nil {
		t.Fatal("expected forbidden field rejection")
	}
}

func receiverFixtureBundle(t *testing.T, grantRef string) receiverBundle {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "synthetic-audience-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bundle receiverBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	bundle.Records = bundle.Records[:1]
	bundle.Records[0].SourceGrantRef = grantRef
	bundle.Records[0].Withdrawal = nil
	bundle.Records[0].Suppression = nil
	return bundle
}

func assertReceiverEntry(t *testing.T, receiver *independentReceiver, grantRef string, want receiverStatus) {
	t.Helper()
	entry, ok := receiver.entries["synthetic-source.example.test:"+grantRef]
	if !ok || entry.Status != want {
		t.Fatalf("grant %s: entry=%+v present=%t want status=%s", grantRef, entry, ok, want)
	}
	if entry.CanContact {
		t.Fatalf("grant %s unexpectedly authorized contact", grantRef)
	}
}

func cloneEntries(entries map[string]receiverEntry) map[string]receiverEntry {
	clone := make(map[string]receiverEntry, len(entries))
	for key, value := range entries {
		clone[key] = value
	}
	return clone
}

func entriesEqual(left, right map[string]receiverEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
