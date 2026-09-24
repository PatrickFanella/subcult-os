import { readFileSync, readdirSync } from 'node:fs';
import { resolve } from 'node:path';

import { Lexicons, type LexiconDoc } from '@atproto/lexicon';
import { describe, expect, it } from 'vitest';

type FixtureCase = {
  name: string;
  valid: boolean;
  reason?: string;
  record: Record<string, unknown>;
};
type FixtureGroup = { nsid: string; cases: FixtureCase[] };
type Fixture = { source: string; records: FixtureGroup[] };

// Same repository-relative contracts directory the Go conformance test and
// the syntax conformance test both consume; see docs/development/lexicon-contract.md.
const contractsDir = resolve(process.cwd(), '../contracts');
const lexiconDir = resolve(contractsDir, 'lexicons');

const lexiconFileNames = readdirSync(lexiconDir).filter((name) => name.endsWith('.json'));

// @atproto/lexicon's `Lexicons` constructor normalizes ref strings on the
// document objects it is given (prefixing them with "lex:") as a side
// effect. Parse a second, independent copy of each file for our own
// allowlist walk below so that in-place mutation cannot leak between the two
// concerns.
function loadLexiconDocs(): LexiconDoc[] {
  return lexiconFileNames.map((name) => JSON.parse(readFileSync(resolve(lexiconDir, name), 'utf8')) as LexiconDoc);
}

function loadRawDocs(): Record<string, unknown>[] {
  return lexiconFileNames.map(
    (name) => JSON.parse(readFileSync(resolve(lexiconDir, name), 'utf8')) as Record<string, unknown>,
  );
}

const lexicons = new Lexicons(loadLexiconDocs());

// Indexes every def (NSID#fragment) from the raw admitted documents so the
// allowlist walk below can resolve refs/unions without reaching into the
// @atproto/lexicon package's internal compiled representation.
const defsByRef = new Map<string, Record<string, unknown>>();
for (const doc of loadRawDocs()) {
  const id = doc.id as string;
  const defs = doc.defs as Record<string, Record<string, unknown>>;
  for (const [fragment, def] of Object.entries(defs)) {
    defsByRef.set(`${id}#${fragment}`, def);
  }
}

function resolveRef(ref: string): Record<string, unknown> {
  const stripped = ref.startsWith('lex:') ? ref.slice('lex:'.length) : ref;
  const withFragment = stripped.includes('#') ? stripped : `${stripped}#main`;
  const def = defsByRef.get(withFragment);
  if (!def) {
    throw new Error(`unresolved ref: ${ref}`);
  }
  return def;
}

// AT Protocol's official "datetime" format is deliberately lenient (it
// accepts ISO-8601 strings without an explicit UTC offset), but Subcult OS's
// own public time semantics require an explicit offset (see
// docs/development/lexicon-contract.md). @atproto/lexicon validates
// leniently by default, so this boundary enforces the stricter policy the
// same way the pinned Go Indigo validator already does by default. The
// regex mirrors Indigo's atproto/syntax datetimeRegex exactly.
const strictDatetime =
  /^[0-9]{4}-[01][0-9]-[0-3][0-9]T[0-2][0-9]:[0-6][0-9]:[0-6][0-9](\.[0-9]{1,20})?(Z|([+-][0-2][0-9]:[0-5][0-9]))$/;

/**
 * Recursively verifies that record data contains only fields declared by its
 * resolved Lexicon definition. Official Lexicon validation intentionally
 * allows additive unknown fields for forward compatibility; the public
 * projection boundary described in docs/development/atproto-kernel.md must
 * instead reject undeclared fields, so private or operational data can never
 * ride along inside an otherwise-valid record. This mirrors
 * backend/internal/atproto/lexicon.go's assertAllowlisted.
 */
function assertAllowlisted(def: Record<string, unknown>, data: unknown): void {
  switch (def.type) {
    case 'record':
      assertAllowlisted(def.record as Record<string, unknown>, data);
      return;
    case 'object': {
      if (typeof data !== 'object' || data === null || Array.isArray(data)) {
        throw new Error('expected an object');
      }
      const obj = data as Record<string, unknown>;
      const properties = (def.properties ?? {}) as Record<string, Record<string, unknown>>;
      for (const key of Object.keys(obj)) {
        if (key === '$type') continue;
        const fieldDef = properties[key];
        if (!fieldDef) {
          throw new Error(`field "${key}" is not declared by the admitted schema`);
        }
        assertAllowlisted(fieldDef, obj[key]);
      }
      return;
    }
    case 'array': {
      if (!Array.isArray(data)) {
        throw new Error('expected an array');
      }
      for (const item of data) {
        assertAllowlisted(def.items as Record<string, unknown>, item);
      }
      return;
    }
    case 'ref':
      assertAllowlisted(resolveRef(def.ref as string), data);
      return;
    case 'union': {
      if (typeof data !== 'object' || data === null) {
        throw new Error('expected an object for union member');
      }
      const typeValue = (data as Record<string, unknown>).$type;
      if (typeof typeValue !== 'string' || typeValue === '') {
        throw new Error('union member missing $type');
      }
      assertAllowlisted(resolveRef(typeValue), data);
      return;
    }
    case 'string':
      if (def.format === 'datetime') {
        if (typeof data !== 'string' || !strictDatetime.test(data)) {
          throw new Error('datetime must include an explicit UTC offset');
        }
      }
      return;
    default:
      // Other scalar leaf types have no nested keys to allowlist.
      return;
  }
}

function validateAdmittedRecord(nsid: string, record: Record<string, unknown>): void {
  lexicons.assertValidRecord(nsid, record);
  assertAllowlisted(resolveRef(nsid), record);
}

const fixture = JSON.parse(
  readFileSync(resolve(contractsDir, 'atproto-lexicon.fixtures.json'), 'utf8'),
) as Fixture;

describe('shared AT Protocol lexicon conformance', () => {
  it('retains fixture provenance', () => {
    expect(fixture.source).toContain('no Subcults source or fixtures copied');
  });

  it('covers at least one admitted NSID', () => {
    expect(fixture.records.length).toBeGreaterThan(0);
  });

  for (const group of fixture.records) {
    describe(group.nsid, () => {
      it.each(group.cases)('$name', ({ valid, record }) => {
        if (valid) {
          expect(() => validateAdmittedRecord(group.nsid, record)).not.toThrow();
        } else {
          expect(() => validateAdmittedRecord(group.nsid, record)).toThrow();
        }
      });
    });
  }
});
