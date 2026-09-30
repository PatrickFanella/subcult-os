import { ApiError, api, postJSON } from '../../api';
import type { EventFinanceLineDTO } from '../../domain';
import { nextFinanceLineRetry, type FinanceLineRetry } from './financeLineModel';

export type FinanceLinePayload = Omit<EventFinanceLineDTO, 'id' | 'eventId' | 'createdByPersonId' | 'createdAt'>;

// One event instance owns one session. State changes precede the first await;
// React rendering is not the authority for whether another request may begin.
export function createFinanceLineSession(eventId: string, createKey: () => string = () => crypto.randomUUID()) {
  let active = true;
  let version = 0;
  let phase: 'unavailable' | 'loading' | 'ready' | 'saving' = 'unavailable';
  let retry: FinanceLineRetry | null = null;
  const path = `/api/events/${encodeURIComponent(eventId)}/finance-lines`;
  const current = (run: number) => active && run === version;
  return {
    canSave() { return active && phase === 'ready'; },
    activate() { active = true; },
    dispose() { active = false; version += 1; phase = 'unavailable'; retry = null; },
    async read(): Promise<EventFinanceLineDTO[] | null> {
      if (!active || phase === 'saving') return null;
      const run = ++version;
      phase = 'loading';
      try {
        const lines = await api<EventFinanceLineDTO[]>(path);
        if (!current(run)) return null;
        if (!Array.isArray(lines) || lines.some(line => !line.id || line.eventId !== eventId)) throw new Error('Ledger response did not match this event.');
        phase = 'ready';
        retry = null;
        return lines;
      } catch (error) {
        if (!current(run)) return null;
        phase = 'unavailable';
        throw error;
      }
    },
    async save(payload: FinanceLinePayload): Promise<EventFinanceLineDTO | null> {
      if (!active || phase !== 'ready') return null;
      phase = 'saving';
      const run = version;
      try {
        retry = nextFinanceLineRetry(retry, JSON.stringify(payload), createKey);
        const created = await postJSON<EventFinanceLineDTO>(path, { ...payload, requestKey: retry.key });
        if (!current(run)) return null;
        if (!created.id || created.eventId !== eventId || created.entryType !== payload.entryType
          || created.direction !== payload.direction || created.amountCents !== payload.amountCents
          || created.currency !== payload.currency || (created.correctsLineId ?? '') !== (payload.correctsLineId ?? '')
          || (created.payableLineId ?? '') !== (payload.payableLineId ?? '')) throw new Error('Ledger receipt did not match this record.');
        phase = 'ready';
        retry = null;
        return created;
      } catch (error) {
        if (!current(run)) return null;
        // Validation/conflict rejections are known not to have recorded this write.
        phase = error instanceof ApiError && (error.status === 400 || error.status === 409) ? 'ready' : 'unavailable';
        throw error;
      }
    },
  };
}
