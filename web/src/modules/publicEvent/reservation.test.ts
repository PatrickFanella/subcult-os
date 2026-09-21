import { afterEach, describe, expect, it, vi } from 'vitest';
import { api, postJSON } from '../../api';
import { reserveFreeTicket } from './reservation';

vi.mock('../../api', () => ({ api: vi.fn(), postJSON: vi.fn() }));
afterEach(() => vi.resetAllMocks());

describe('free reservation inventory refresh', () => {
  it('uses authoritative availability after the ticket is issued', async () => {
    const ticket = { code: 'test-ticket' };
    const event = { remainingTickets: 0, isFull: true };
    vi.mocked(postJSON).mockResolvedValue(ticket);
    vi.mocked(api).mockResolvedValue(event);
    await expect(reserveFreeTicket('night', { email: 'guest@example.test' })).resolves.toEqual({ ticket, event });
    expect(postJSON).toHaveBeenCalledExactlyOnceWith('/api/public/events/night/reservations', { email: 'guest@example.test' });
    expect(api).toHaveBeenCalledExactlyOnceWith('/api/public/events/night');
    expect(vi.mocked(postJSON).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(api).mock.invocationCallOrder[0]);
  });

  it('retains a confirmed ticket when availability cannot be refreshed', async () => {
    const ticket = { code: 'already-issued' };
    vi.mocked(postJSON).mockResolvedValue(ticket);
    vi.mocked(api).mockRejectedValue(new Error('event unavailable'));
    await expect(reserveFreeTicket('night', { email: 'guest@example.test' })).resolves.toEqual({ ticket, event: null });
    expect(postJSON).toHaveBeenCalledTimes(1);
  });

  it('does not report confirmation or refresh after a rejected reservation', async () => {
    vi.mocked(postJSON).mockRejectedValue(new Error('sold out'));
    await expect(reserveFreeTicket('night', { email: 'guest@example.test' })).rejects.toThrow('sold out');
    expect(api).not.toHaveBeenCalled();
  });
});
