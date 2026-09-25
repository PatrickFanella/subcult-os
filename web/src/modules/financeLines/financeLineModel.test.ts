import { describe, expect, it } from 'vitest';
import {
  type FinanceLineLike,
  centsFromDecimal,
  currentFinanceLines,
  decimalFromCents,
  financeTotals,
  nextFinanceLineRetry,
  preserveHydratedUTC,
  toLocalDateTime,
  toUTC,
} from './financeLineModel';

describe('finance decimal cents', () => {
  it('preserves exact cents and rejects imprecise input', () => {
    expect(centsFromDecimal('123.45')).toBe(12345);
    expect(centsFromDecimal('0.01')).toBe(1);
    expect(centsFromDecimal('0')).toBe(0);
    expect(centsFromDecimal('1.234')).toBeNull();
    expect(centsFromDecimal('1e2')).toBeNull();
    expect(centsFromDecimal('10000000.01')).toBeNull();
  });

  it('hydrates exact stored cents without float formatting', () => {
    expect(decimalFromCents(0)).toBe('0.00');
    expect(decimalFromCents(12345)).toBe('123.45');
  });
});

describe('finance current-line projection', () => {
  const original: FinanceLineLike = { id: 'payable-root', entryType: 'payable', direction: 'expense', currency: 'usd', amountCents: 100 };
  const correction: FinanceLineLike = { id: 'payable-current', correctsLineId: 'payable-root', entryType: 'payable', direction: 'expense', currency: 'usd', amountCents: 125 };

  it('uses a correction in place of its historical line for totals', () => {
    expect(financeTotals([original, correction])).toEqual([{
      key: 'usd|payable|expense',
      entryType: 'payable',
      direction: 'expense',
      currency: 'usd',
      amountCents: 125,
    }]);
  });

  it('keeps a stable original payable identity available while exposing only current leaves', () => {
    expect(currentFinanceLines([original, correction])).toEqual([correction]);
    expect([original, correction].filter((line) => line.entryType === 'payable' && !line.correctsLineId)).toEqual([original]);
  });
});

describe('finance dates', () => {
  it('sends local form values as UTC and hydrates stored UTC timestamps for a local control', () => {
    const utc = toUTC('2026-11-01T01:30');
    expect(utc).toMatch(/Z$/);
    if (typeof utc !== "string") throw new Error("Expected a valid UTC timestamp");
    expect(toLocalDateTime(utc)).toMatch(/^2026-11-01T\d{2}:\d{2}$/);
  });

  it('keeps empty optional dates absent and rejects invalid values', () => {
    expect(toUTC('')).toBeUndefined();
    expect(toUTC('not-a-date')).toBeNull();
    expect(toLocalDateTime('not-a-date')).toBe('');
  });

  it('keeps an unedited hydrated correction timestamp exact', () => {
    const stored = '2026-11-01T12:34:56.123456Z';
    const localInput = toLocalDateTime(stored);

    expect(preserveHydratedUTC(localInput, stored)).toBe(stored);
    expect(preserveHydratedUTC('2026-11-02T12:34', stored)).toMatch(/Z$/);
  });
});

describe('finance retry identity', () => {
  it('reuses an idempotency key for an unchanged failed payload', () => {
    const first = nextFinanceLineRetry(null, 'same-payload', () => 'retry-one');

    expect(nextFinanceLineRetry(first, 'same-payload', () => 'retry-two')).toBe(first);
  });

  it('creates a new key when the payload changes', () => {
    const first = nextFinanceLineRetry(null, 'first-payload', () => 'retry-one');

    expect(nextFinanceLineRetry(first, 'changed-payload', () => 'retry-two')).toEqual({
      fingerprint: 'changed-payload',
      key: 'retry-two',
    });
  });
});
