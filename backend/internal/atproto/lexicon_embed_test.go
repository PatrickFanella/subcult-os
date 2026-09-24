package atproto

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestEmbeddedLexiconsMatchContracts fails when backend/internal/atproto/lexicons
// drifts from contracts/lexicons, so the binary can never validate against a
// schema the shared contract does not declare.
func TestEmbeddedLexiconsMatchContracts(t *testing.T) {
	var embeddedNames []string
	if err := fs.WalkDir(embeddedLexicons, "lexicons", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			embeddedNames = append(embeddedNames, filepath.Base(p))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	contractEntries, err := os.ReadDir(LexiconContractDir)
	if err != nil {
		t.Fatal(err)
	}
	var contractNames []string
	for _, entry := range contractEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			contractNames = append(contractNames, entry.Name())
		}
	}
	sort.Strings(embeddedNames)
	sort.Strings(contractNames)
	if len(embeddedNames) == 0 || len(embeddedNames) != len(contractNames) {
		t.Fatalf("embedded lexicons %v do not match contracts %v", embeddedNames, contractNames)
	}
	for i := range embeddedNames {
		if embeddedNames[i] != contractNames[i] {
			t.Fatalf("embedded lexicons %v do not match contracts %v", embeddedNames, contractNames)
		}
		embedded, err := embeddedLexicons.ReadFile("lexicons/" + embeddedNames[i])
		if err != nil {
			t.Fatal(err)
		}
		contract, err := os.ReadFile(filepath.Join(LexiconContractDir, contractNames[i]))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(embedded, contract) {
			t.Fatalf("%s differs between backend/internal/atproto/lexicons and contracts/lexicons; copy the contract file over the embedded one", embeddedNames[i])
		}
	}
	if _, err := LoadEmbeddedLexiconCatalog(); err != nil {
		t.Fatal(err)
	}
}
