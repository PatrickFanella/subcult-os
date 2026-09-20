import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { AtUri, isValidAtIdentifier, isValidNsid } from '@atproto/syntax';
import { describe, expect, it } from 'vitest';

type FixtureCase = { value: string; valid: boolean };
type Fixture = {
  source: string;
  accountIdentifiers: FixtureCase[];
  collections: FixtureCase[];
  recordRefs: FixtureCase[];
};

const fixture = JSON.parse(
  readFileSync(resolve(process.cwd(), '../contracts/atproto-syntax.fixtures.json'), 'utf8'),
) as Fixture;

function isExactRecordRef(value: string) {
  try {
    const uri = new AtUri(value);
    return Boolean(uri.collection && uri.rkey);
  } catch {
    return false;
  }
}

describe('shared AT Protocol syntax conformance', () => {
  it('retains fixture provenance', () => {
    expect(fixture.source).toContain('no Subcults source copied');
  });

  it.each(fixture.accountIdentifiers)('validates account identifier $value', ({ value, valid }) => {
    expect(isValidAtIdentifier(value)).toBe(valid);
  });

  it.each(fixture.collections)('validates collection $value', ({ value, valid }) => {
    expect(isValidNsid(value)).toBe(valid);
  });

  it.each(fixture.recordRefs)('validates exact record reference $value', ({ value, valid }) => {
    expect(isExactRecordRef(value)).toBe(valid);
  });
});
