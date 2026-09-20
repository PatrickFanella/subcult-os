package atproto

import (
	"encoding/json"
	"os"
	"testing"
)

type syntaxFixture struct {
	Source             string              `json:"source"`
	AccountIdentifiers []syntaxFixtureCase `json:"accountIdentifiers"`
	Collections        []syntaxFixtureCase `json:"collections"`
	RecordRefs         []syntaxFixtureCase `json:"recordRefs"`
}

type syntaxFixtureCase struct {
	Value string `json:"value"`
	Valid bool   `json:"valid"`
}

func TestSharedSyntaxConformanceFixture(t *testing.T) {
	body, err := os.ReadFile("../../../contracts/atproto-syntax.fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture syntaxFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Source == "" {
		t.Fatal("fixture provenance is required")
	}
	assertCases := func(t *testing.T, cases []syntaxFixtureCase, parse func(string) error) {
		t.Helper()
		for _, test := range cases {
			t.Run(test.Value, func(t *testing.T) {
				err := parse(test.Value)
				if (err == nil) != test.Valid {
					t.Fatalf("valid = %v, want %v (error: %v)", err == nil, test.Valid, err)
				}
			})
		}
	}
	assertCases(t, fixture.AccountIdentifiers, func(value string) error {
		_, err := ParseAccountIdentifier(value)
		return err
	})
	assertCases(t, fixture.Collections, func(value string) error {
		_, err := ParseCollection(value)
		return err
	})
	assertCases(t, fixture.RecordRefs, func(value string) error {
		_, err := ParseRecordRef(value)
		return err
	})
}
