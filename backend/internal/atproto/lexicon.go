package atproto

import (
	"encoding/json"
	"fmt"

	indigolexicon "github.com/bluesky-social/indigo/atproto/lexicon"
)

// LexiconContractDir is the repository-relative path to the admitted Lexicon
// documents, resolved the same way the syntax fixtures are: relative to this
// package's directory at test/run time.
const LexiconContractDir = "../../../contracts/lexicons"

// LoadLexiconCatalog reads every admitted Lexicon JSON document from dirPath
// into an Indigo BaseCatalog. It does not accept Indigo types across this
// package's boundary; callers only ever see plain NSIDs, maps and errors.
func LoadLexiconCatalog(dirPath string) (*indigolexicon.BaseCatalog, error) {
	catalog := indigolexicon.NewBaseCatalog()
	if err := catalog.LoadDirectory(dirPath); err != nil {
		return nil, fmt.Errorf("load lexicon contracts: %w", err)
	}
	return catalog, nil
}

// ValidateAdmittedRecord checks recordJSON against the admitted Lexicon
// identified by nsid using two independent passes:
//
//  1. Indigo's structural Lexicon validation (required fields, string/array
//     bounds, known-value enums, datetime/at-uri/cid syntax, $type match).
//  2. A public-projection allowlist pass that this package owns. Lexicon
//     validation intentionally permits additive unknown fields for forward
//     compatibility; the public projection boundary described in
//     docs/development/atproto-kernel.md must instead reject any field the
//     admitted schema did not declare, so private or operational data can
//     never ride along inside an otherwise-valid record.
func ValidateAdmittedRecord(catalog *indigolexicon.BaseCatalog, nsid string, recordJSON []byte) error {
	var record map[string]any
	if err := json.Unmarshal(recordJSON, &record); err != nil {
		return fmt.Errorf("record is not a JSON object: %w", err)
	}

	if err := indigolexicon.ValidateRecord(catalog, record, nsid, 0); err != nil {
		return err
	}

	def, err := catalog.Resolve(nsid)
	if err != nil {
		return err
	}
	if err := assertAllowlisted(catalog, def.Def, record); err != nil {
		return fmt.Errorf("field not in public allowlist: %w", err)
	}
	return nil
}

// assertAllowlisted recursively walks record data alongside its resolved
// Lexicon definition and fails on any object key the schema did not declare.
// It mirrors Indigo's own validateData dispatch shape but only checks key
// membership; bounds, formats and required fields are Indigo's job above.
func assertAllowlisted(catalog indigolexicon.Catalog, def any, data any) error {
	switch v := def.(type) {
	case indigolexicon.SchemaRecord:
		return assertAllowlisted(catalog, v.Record, data)
	case indigolexicon.SchemaObject:
		obj, ok := data.(map[string]any)
		if !ok {
			return fmt.Errorf("expected an object")
		}
		for key, value := range obj {
			if key == "$type" {
				continue
			}
			fieldDef, declared := v.Properties[key]
			if !declared {
				return fmt.Errorf("field %q is not declared by the admitted schema", key)
			}
			if err := assertAllowlisted(catalog, fieldDef.Inner, value); err != nil {
				return err
			}
		}
		return nil
	case indigolexicon.SchemaArray:
		items, ok := data.([]any)
		if !ok {
			return fmt.Errorf("expected an array")
		}
		for _, item := range items {
			if err := assertAllowlisted(catalog, v.Items.Inner, item); err != nil {
				return err
			}
		}
		return nil
	case indigolexicon.SchemaRef:
		resolved, err := catalog.Resolve(v.Ref)
		if err != nil {
			return err
		}
		return assertAllowlisted(catalog, resolved.Def, data)
	case indigolexicon.SchemaUnion:
		obj, ok := data.(map[string]any)
		if !ok {
			return fmt.Errorf("expected an object for union member")
		}
		typeValue, _ := obj["$type"].(string)
		if typeValue == "" {
			return fmt.Errorf("union member missing $type")
		}
		resolved, err := catalog.Resolve(typeValue)
		if err != nil {
			return err
		}
		return assertAllowlisted(catalog, resolved.Def, data)
	default:
		// Scalar leaf types (string, integer, boolean, bytes, cid-link,
		// blob, token) have no nested keys to allowlist.
		return nil
	}
}
