// The fragment never travels in the request URL. Do not persist the token or
// guest results in browser storage or use the session-refreshing operator API.
export async function doorLinkRequest<T>(eventId: string, token: string, action: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/door-links/${encodeURIComponent(eventId)}/${action}`, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Door-Link': token },
    credentials: 'omit', cache: 'no-store', referrerPolicy: 'no-referrer',
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(typeof data.error === 'string' ? data.error : 'Door request failed');
  return data as T;
}
