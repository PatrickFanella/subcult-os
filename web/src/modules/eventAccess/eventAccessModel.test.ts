import { describe, expect, it } from 'vitest';
import { accessExpiry, accessTimestamp, accessValues, nextAccessRetry, unknownAccessEntry } from './eventAccessModel';

describe('event access information', () => {
  it('keeps unknown distinct from negative assertions and text availability', () => {
    expect(unknownAccessEntry('entry').effectiveValue).toBe('unknown');
    expect(accessValues('entry')).toEqual(['unknown', 'yes', 'no']);
    expect(accessValues('seating')).toEqual(['unknown', 'available', 'limited', 'not_available']);
    expect(accessValues('sensory')).toEqual(['unknown', 'known']);
  });
  it('preserves exact hydrated instants and treats edits as UTC', () => {
    expect(accessTimestamp('2026-10-01T12:30', '2026-10-01T12:30:47.123Z')).toBe('2026-10-01T12:30:47.123Z');
    expect(accessTimestamp('2026-10-01T12:31', '2026-10-01T12:30:47.123Z')).toBe('2026-10-01T12:31:00.000Z');
    expect(accessTimestamp('')).toBeNull();
  });
  it('rejects malformed and calendar-invalid review dates', () => {
    expect(() => accessTimestamp('yesterday')).toThrow();
    expect(() => accessTimestamp('2026-02-31T12:30')).toThrow();
  });
  it('expires at the exact deadline without expiring an undated unknown or an assertion without expiry', () => {
    const entry = { ...unknownAccessEntry('entry'), value: 'yes' as const, expiresAt: '2026-10-01T12:30:00Z' };
    expect(accessExpiry(entry, Date.parse('2026-10-01T12:29:59Z'))).toBe(false);
    expect(accessExpiry(entry, Date.parse('2026-10-01T12:30:00Z'))).toBe(true);
    expect(accessExpiry(unknownAccessEntry('entry'), Date.now())).toBe(false);
  });
  it('keeps replay identity only while the event, topic and decision payload are unchanged', () => {
    const first = nextAccessRetry(null, 'event-a:entry:revision-1', () => 'first');
    expect(nextAccessRetry(first, 'event-a:entry:revision-1', () => 'unused')).toBe(first);
    expect(nextAccessRetry(first, 'event-b:entry:revision-1', () => 'changed').key).toBe('changed');
    expect(nextAccessRetry(first, 'event-a:bathrooms:revision-1', () => 'topic').key).toBe('topic');
  });
});
