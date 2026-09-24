package atproto

import (
	"encoding/json"
	"os"
	"testing"
)

type lexiconFixture struct {
	Source  string             `json:"source"`
	Records []lexiconFixtureNS `json:"records"`
}

type lexiconFixtureNS struct {
	NSID  string               `json:"nsid"`
	Cases []lexiconFixtureCase `json:"cases"`
}

type lexiconFixtureCase struct {
	Name   string         `json:"name"`
	Valid  bool           `json:"valid"`
	Reason string         `json:"reason"`
	Record map[string]any `json:"record"`
}

func TestSharedLexiconConformanceFixture(t *testing.T) {
	body, err := os.ReadFile("../../../contracts/atproto-lexicon.fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture lexiconFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Source == "" {
		t.Fatal("fixture provenance is required")
	}

	catalog, err := LoadLexiconCatalog(LexiconContractDir)
	if err != nil {
		t.Fatalf("load admitted lexicon contracts: %v", err)
	}

	if len(fixture.Records) == 0 {
		t.Fatal("fixture must cover at least one admitted NSID")
	}

	for _, group := range fixture.Records {
		t.Run(group.NSID, func(t *testing.T) {
			if len(group.Cases) == 0 {
				t.Fatalf("no cases for %s", group.NSID)
			}
			for _, testCase := range group.Cases {
				t.Run(testCase.Name, func(t *testing.T) {
					recordJSON, err := json.Marshal(testCase.Record)
					if err != nil {
						t.Fatalf("marshal fixture record: %v", err)
					}
					validationErr := ValidateAdmittedRecord(catalog, group.NSID, recordJSON)
					if (validationErr == nil) != testCase.Valid {
						t.Fatalf("valid = %v, want %v (reason: %q, error: %v)",
							validationErr == nil, testCase.Valid, testCase.Reason, validationErr)
					}
				})
			}
		})
	}
}
