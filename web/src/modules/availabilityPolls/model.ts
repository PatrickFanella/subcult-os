export type PollAnswer = 'yes' | 'if_need_be' | 'no';
export interface PollOption { id: string; startsAt: string; endsAt: string | null; timezone: string }
export interface PollResponse { displayName: string; privateNote?: string; answers: Record<string, PollAnswer> }
export interface AvailabilityPoll {
  id: string; prompt: string; status: 'open' | 'closed'; showNames: boolean;
  closesAt: string | null; chosenOptionId: string | null; options: PollOption[];
  responses?: PollResponse[]; ownResponse?: PollResponse;
  totals: Record<string, Record<PollAnswer, number>>;
}

export function formatPollTime(instant: string, timeZone?: string) {
  return new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short', ...(timeZone ? { timeZone } : {}) }).format(new Date(instant));
}
export function pollOptionLabel(option: PollOption, viewerZone?: string) {
  const local = formatPollTime(option.startsAt, viewerZone) + (option.endsAt ? ` – ${formatPollTime(option.endsAt, viewerZone)}` : '');
  const event = formatPollTime(option.startsAt, option.timezone) + (option.endsAt ? ` – ${formatPollTime(option.endsAt, option.timezone)}` : '');
  return { local, event: `${option.timezone}: ${event}` };
}

export function pollEditCapability(pollId: string) {
  const key = `subcult-poll-${pollId}-edit`;
  try { const existing = localStorage.getItem(key); if (existing && existing.length >= 32) return { token: existing, retained: true }; } catch { /* Keep this tab usable without storage. */ }
  const random = crypto.getRandomValues(new Uint8Array(32));
  const token = [...random].map((byte) => byte.toString(16).padStart(2, '0')).join('');
  try { localStorage.setItem(key, token); return { token, retained: true }; }
  catch { return { token, retained: false }; }
}

export async function pollRequest<T>(pollId: string, token: string, editToken: string, body?: PollResponse): Promise<T> {
  const response = await fetch(`/api/availability-polls/${encodeURIComponent(pollId)}${body ? '/responses' : ''}`, {
    method: body ? 'POST' : 'GET', headers: { 'Content-Type': 'application/json', 'X-Poll-Link': token, 'X-Poll-Edit': editToken },
    credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer', ...(body ? { body: JSON.stringify(body) } : {}),
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(typeof data.error === 'string' ? data.error : 'Poll request failed');
  return data as T;
}

// Reject spring-forward gaps rather than silently moving an option an hour.
export function pollInstantFromInput(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) throw new Error('Choose a valid local date and time.');
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
  if (local !== value) throw new Error('This local time does not exist because the clock changes. Choose another time.');
  return date.toISOString();
}
