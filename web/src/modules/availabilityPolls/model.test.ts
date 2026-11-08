import { afterEach, describe, expect, it, vi } from 'vitest';
import { pollEditCapability, pollInstantFromInput, pollOptionLabel, pollRequest } from './model';

afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
describe('availability polls', () => {
  it('rejects a nonexistent spring-forward local time instead of shifting it', () => {
    vi.stubEnv('TZ', 'America/Chicago');
    expect(() => pollInstantFromInput('2026-03-08T02:30')).toThrow('does not exist');
    expect(pollInstantFromInput('2026-03-08T03:30')).toBe('2026-03-08T08:30:00.000Z');
  });
  it('distinguishes the two 1:30 AM instants across the Chicago DST fall-back', () => {
    const first = pollOptionLabel({ id: 'a', startsAt: '2026-11-01T06:30:00Z', endsAt: null, timezone: 'America/Chicago' }, 'UTC');
    const second = pollOptionLabel({ id: 'b', startsAt: '2026-11-01T07:30:00Z', endsAt: null, timezone: 'America/Chicago' }, 'UTC');
    expect(first.event).toContain('1:30'); expect(first.event).toContain('CDT');
    expect(second.event).toContain('1:30'); expect(second.event).toContain('CST');
    expect(first.local).not.toBe(second.local);
  });
  it('reuses one edit capability and survives denied storage without retaining names or notes', () => {
    const storage = new Map<string, string>();
    vi.stubGlobal('localStorage', { getItem: (key: string) => storage.get(key), setItem: (key: string, value: string) => storage.set(key, value) });
    const first = pollEditCapability('poll'); expect(first.retained).toBe(true); expect(first.token.length).toBe(64);
    expect(pollEditCapability('poll')).toEqual(first);
    expect([...storage.keys()]).toEqual(['subcult-poll-poll-edit']);
    vi.stubGlobal('localStorage', { getItem: () => { throw new Error('denied'); }, setItem: () => { throw new Error('denied'); } });
    expect(pollEditCapability('other')).toMatchObject({ retained: false });
  });
  it('sends poll and edit capabilities only in headers, without cookies, caching or session refresh', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ saved: true }) }); vi.stubGlobal('fetch', fetcher);
    await pollRequest('poll', 'private-poll-token', 'private-edit-token', { displayName: 'Name', answers: { option: 'yes' } });
    expect(fetcher.mock.calls[0][0]).toBe('/api/availability-polls/poll/responses');
    expect(fetcher.mock.calls[0][1]).toMatchObject({ credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer', headers: { 'X-Poll-Link': 'private-poll-token', 'X-Poll-Edit': 'private-edit-token' } });
  });
});
