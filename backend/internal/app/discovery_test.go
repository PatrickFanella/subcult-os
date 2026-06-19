package app

import (
	"errors"
	"strings"
	"testing"
)

func TestDiscoveryPolicyEnabled(t *testing.T) {
	if !newDiscoveryPolicy().enabled() {
		t.Fatal("expected discovery to stay enabled")
	}
}

func TestDiscoveryPolicyNormalizesSearchQuery(t *testing.T) {
	query, err := newDiscoveryPolicy().normalizeSearchQuery("  MaRkEt  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query != "market" {
		t.Fatalf("unexpected query normalization: %q", query)
	}
}

func TestDiscoveryPolicyRejectsOverlongSearchQuery(t *testing.T) {
	_, err := newDiscoveryPolicy().normalizeSearchQuery(strings.Repeat("a", 121))
	if !errors.Is(err, errDiscoveryQueryTooLong) {
		t.Fatalf("expected overlong search query error, got %v", err)
	}
}
