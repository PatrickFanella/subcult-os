import { afterEach, describe, expect, it, vi } from 'vitest';
import { doorLinkRequest } from './doorLink';

afterEach(() => vi.unstubAllGlobals());
describe('volunteer door transport', () => {
  it('keeps the token out of URLs and uses no session, cache, or referrer', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => [] });
    vi.stubGlobal('fetch', fetcher);
    await doorLinkRequest('event-1', 'private-token', 'tickets?query=Guest');
    const [url, options] = fetcher.mock.calls[0];
    expect(url).toBe('/api/door-links/event-1/tickets?query=Guest');
    expect(url).not.toContain('private-token');
    expect(options).toMatchObject({ credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer', headers: { 'X-Door-Link': 'private-token' } });
  });
  it('posts only the ticket code and reports duplicate admission clearly', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: false, json: async () => ({ error: 'already checked in' }) });
    vi.stubGlobal('fetch', fetcher);
    await expect(doorLinkRequest('event-1', 'private-token', 'check-ins', { code: 'ticket-code' })).rejects.toThrow('already checked in');
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher.mock.calls[0][1]).toMatchObject({ method: 'POST', body: '{"code":"ticket-code"}' });
  });
});
