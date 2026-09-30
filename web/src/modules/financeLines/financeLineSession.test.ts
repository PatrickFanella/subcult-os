import { afterEach, describe, expect, it, vi } from 'vitest';
import { createFinanceLineSession, type FinanceLinePayload } from './financeLineSession';

const payload: FinanceLinePayload = { entryType: 'budget', direction: 'expense', amountCents: 1250, currency: 'usd', label: 'Budget', reason: 'Plan' };
const receipt = { ...payload, id: 'line-a', eventId: 'event/a', createdByPersonId: 'owner-a', createdAt: '2026-09-30T00:00:00Z' };
const response = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status });
afterEach(() => vi.unstubAllGlobals());

describe('private finance session', () => {
  it('does not write before a successful private read, including while the read is pending', async () => {
    let resolve!: (value: Response) => void;
    const fetch = vi.fn().mockImplementation(() => new Promise<Response>(done => { resolve = done; }));
    vi.stubGlobal('fetch', fetch);
    const session = createFinanceLineSession('event/a');
    expect(await session.save(payload)).toBeNull();
    const read = session.read();
    expect(await session.save(payload)).toBeNull();
    resolve(response([])); await read;
    expect(session.canSave()).toBe(true);
    expect(fetch).toHaveBeenCalledTimes(1);
  });
  it('admits one held write synchronously and sends an encoded path with one idempotency key', async () => {
    let resolve!: (value: Response) => void;
    const fetch = vi.fn().mockResolvedValueOnce(response([])).mockImplementationOnce(() => new Promise<Response>(done => { resolve = done; }));
    vi.stubGlobal('fetch', fetch);
    const session = createFinanceLineSession('event/a', () => 'key-a');
    await session.read();
    const first = session.save(payload);
    expect(session.canSave()).toBe(false);
    expect(await session.save(payload)).toBeNull();
    expect(await session.read()).toBeNull();
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(fetch.mock.calls[1][0]).toBe('/api/events/event%2Fa/finance-lines');
    expect(JSON.parse(fetch.mock.calls[1][1].body)).toEqual({ ...payload, requestKey: 'key-a' });
    resolve(response(receipt)); expect(await first).toEqual(receipt);
    expect(session.canSave()).toBe(true);
  });
  it.each([401, 403, 500])('fences failed read %s until a fresh successful read', async status => {
    const fetch = vi.fn().mockResolvedValueOnce(response({ error: 'Unavailable' }, status));
    if (status === 401) fetch.mockResolvedValueOnce(response({ error: 'Session unavailable' }, 401));
    fetch.mockResolvedValueOnce(response([]));
    vi.stubGlobal('fetch', fetch);
    const session = createFinanceLineSession('event/a');
    await expect(session.read()).rejects.toMatchObject({ status });
    expect(await session.save(payload)).toBeNull();
    expect(fetch).toHaveBeenCalledTimes(status === 401 ? 2 : 1);
    await session.read(); expect(session.canSave()).toBe(true);
  });
  it.each([{}, [{ ...receipt, eventId: 'event-b' }]])('rejects invalid or cross-event ledger reads', async value => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(value)));
    const session = createFinanceLineSession('event/a');
    await expect(session.read()).rejects.toThrow('did not match');
    expect(session.canSave()).toBe(false);
  });
  it.each([400, 409])('keeps known write rejection %s editable with a stable unchanged request key', async status => {
    const fetch = vi.fn().mockResolvedValueOnce(response([])).mockResolvedValueOnce(response({ error: 'Rejected' }, status)).mockResolvedValueOnce(response(receipt));
    vi.stubGlobal('fetch', fetch);
    const createKey = vi.fn(() => 'key-a'); const session = createFinanceLineSession('event/a', createKey);
    await session.read(); await expect(session.save(payload)).rejects.toMatchObject({ status });
    expect(session.canSave()).toBe(true); await session.save(payload);
    expect(createKey).toHaveBeenCalledTimes(1);
    expect(JSON.parse(fetch.mock.calls[2][1].body).requestKey).toBe('key-a');
  });
  it.each([401, 403, 500])('fences denied or unknown write outcome %s without automatic replay', async status => {
    const fetch = vi.fn().mockResolvedValueOnce(response([])).mockResolvedValueOnce(response({ error: 'Unavailable' }, status));
    if (status === 401) fetch.mockResolvedValueOnce(response({ error: 'Session unavailable' }, 401));
    fetch.mockResolvedValueOnce(response([]));
    vi.stubGlobal('fetch', fetch);const session = createFinanceLineSession('event/a');
    await session.read();await expect(session.save(payload)).rejects.toMatchObject({ status });
    expect(await session.save(payload)).toBeNull();expect(fetch).toHaveBeenCalledTimes(status === 401 ? 3 : 2);
    await session.read();expect(session.canSave()).toBe(true);
  });
  it('fences a lost response after the transport was called', async () => {
    const fetch = vi.fn().mockResolvedValueOnce(response([])).mockRejectedValueOnce(new TypeError('Disconnected'));
    vi.stubGlobal('fetch', fetch);const session = createFinanceLineSession('event/a');
    await session.read();await expect(session.save(payload)).rejects.toThrow('Disconnected');
    expect(await session.save(payload)).toBeNull();expect(fetch).toHaveBeenCalledTimes(2);
  });
  it.each([{ ...receipt, eventId: 'event-b' }, { ...receipt, amountCents: 999 }, { ...receipt, correctsLineId: 'another-line' }, { ...receipt, id: '' }])('fences mismatched receipts before they enter the ledger', async value => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(response([])).mockResolvedValueOnce(response(value)));
    const session = createFinanceLineSession('event/a');await session.read();
    await expect(session.save(payload)).rejects.toThrow('did not match');expect(session.canSave()).toBe(false);
  });
  it('ignores a departed write even after the same session is reactivated and freshly read', async () => {
    let resolve!: (value: Response) => void;
    const fetch = vi.fn().mockResolvedValueOnce(response([])).mockImplementationOnce(() => new Promise<Response>(done => { resolve = done; })).mockResolvedValueOnce(response([]));
    vi.stubGlobal('fetch', fetch);const session = createFinanceLineSession('event/a');await session.read();
    const write = session.save(payload);session.dispose();session.activate();await session.read();
    resolve(response(receipt));expect(await write).toBeNull();expect(session.canSave()).toBe(true);
  });
  it('ignores a departed read and never permits writes after disposal', async () => {
    let resolve!: (value: Response) => void;
    const fetch = vi.fn().mockImplementation(() => new Promise<Response>(done => { resolve = done; }));
    vi.stubGlobal('fetch', fetch);const session = createFinanceLineSession('event/a');const read = session.read();session.dispose();
    resolve(response([]));expect(await read).toBeNull();expect(await session.save(payload)).toBeNull();expect(fetch).toHaveBeenCalledTimes(1);
  });
});
