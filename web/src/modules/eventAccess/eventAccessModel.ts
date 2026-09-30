import type { EventAccessRevisionDTO, EventAccessTopic } from '../../domain';

export const accessTopics: Array<{ topic: EventAccessTopic; label: string }> = [
  { topic: 'entry', label: 'Step-free entry' }, { topic: 'bathrooms', label: 'Step-free bathrooms' },
  { topic: 'seating', label: 'Seating' }, { topic: 'sensory', label: 'Sensory conditions' },
  { topic: 'transit', label: 'Transit' }, { topic: 'contact', label: 'Access contact' },
];
export const accessValueLabels = { unknown: 'Unknown', yes: 'Yes', no: 'No', available: 'Available', limited: 'Limited', not_available: 'Not available', known: 'Information recorded' };
export const accessSourceLabels = { unknown: 'No source recorded', organizer_assertion: 'Organizer assertion', event_observation: 'Event-specific observation', external_reference: 'External reference' };
export function accessValues(topic: EventAccessTopic): Array<EventAccessRevisionDTO['value']> {
  return topic === 'entry' || topic === 'bathrooms' ? ['unknown', 'yes', 'no'] : topic === 'seating' ? ['unknown', 'available', 'limited', 'not_available'] : ['unknown', 'known'];
}
export function accessTimestamp(value: string, original?: string): string | null {
  if (!value) return null;
  if (original?.slice(0, 16) === value) return original;
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) throw new Error('Enter valid review and expiry times in UTC.');
  const date = new Date(`${value}:00Z`);
  if (Number.isNaN(date.getTime()) || date.toISOString().slice(0, 16) !== value) throw new Error('Enter valid review and expiry times in UTC.');
  return date.toISOString();
}
export function accessExpiry(entry: EventAccessRevisionDTO, now: number) {
  return Boolean(entry.expiresAt && Date.parse(entry.expiresAt) <= now);
}
export function unknownAccessEntry(topic: EventAccessTopic): EventAccessRevisionDTO {
  return { evaluatedAt: '', topic, scope: 'event', revision: 0, value: 'unknown', effectiveValue: 'unknown', needsReview: true, details: '', sourceKind: 'unknown', sourceReference: '', correctionReason: '' };
}
export type AccessRetry = { payload: string; key: string };
export function nextAccessRetry(current: AccessRetry | null, payload: string, createKey: () => string): AccessRetry {
  return current?.payload === payload ? current : { payload, key: createKey() };
}
