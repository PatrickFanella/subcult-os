import { afterEach, describe, expect, it, vi } from 'vitest';

import { api } from './api';

afterEach(() => vi.unstubAllGlobals());

describe('api session refresh', () => {
  it('rotates an expired browser session once and retries the request', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ id: 'person-1' }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api<{ id: string }>('/api/me')).resolves.toEqual({ id: 'person-1' });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/auth/refresh');
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/me');
  });

  it('does not recurse when refresh is unauthorized', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api('/api/me')).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('does not refresh in response to rejected credentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: 'invalid credentials' }), { status: 401 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await expect(api('/api/auth/login', { method: 'POST', body: '{}' })).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
