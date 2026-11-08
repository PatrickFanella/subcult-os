import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { api, deleteJSON, postJSON } from '../api';
import { DoorView } from './DoorView';

interface DoorLink {
  id: string; label: string; validFrom: string; validUntil: string;
  revokedAt: string | null; checkedInCount: number;
}

export function DoorLinkView({ eventId }: { eventId: string }) {
  const token = typeof window === 'undefined' ? '' : window.location.hash?.slice(1) ?? '';
  useEffect(() => {
    const robots = document.createElement('meta'); robots.name = 'robots'; robots.content = 'noindex,nofollow';
    const referrer = document.createElement('meta'); referrer.name = 'referrer'; referrer.content = 'no-referrer';
    document.head.append(robots, referrer);
    return () => { robots.remove(); referrer.remove(); };
  }, []);
  if (!token) return <main className="mx-auto max-w-3xl p-6"><h1 className="text-2xl font-bold">Door link unavailable</h1><p>Open the complete link shared by the event owner.</p></main>;
  return <DoorView eventId={eventId} doorToken={token} />;
}

export function DoorLinksView({ eventId }: { eventId: string }) {
  const [links, setLinks] = useState<DoorLink[]>([]);
  const [label, setLabel] = useState('');
  const [from, setFrom] = useState('');
  const [until, setUntil] = useState('');
  const [sharedUrl, setSharedUrl] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [ready, setReady] = useState(false);
  const path = `/api/events/${eventId}/door-links`;
  useEffect(() => {
    let active = true;
    api<DoorLink[]>(path).then((rows) => { if (active) { setLinks(rows); setReady(true); } }).catch((err) => { if (active) setError(err instanceof Error ? err.message : 'Unable to load door links'); });
    return () => { active = false; };
  }, [path]);
  async function create(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(null); setSharedUrl('');
    try {
      const created = await postJSON<{ url: string }>(path, { label, validFrom: new Date(from).toISOString(), validUntil: new Date(until).toISOString() });
      setSharedUrl(created.url); setLabel('');
      setLinks(await api<DoorLink[]>(path));
    } catch (err) { setError(err instanceof Error ? err.message : 'Unable to create door link'); }
    finally { setBusy(false); }
  }
  async function revoke(id: string) {
    setBusy(true); setError(null); setSharedUrl('');
    try { await deleteJSON(`${path}/${id}`); setLinks(await api<DoorLink[]>(path)); }
    catch (err) { setError(err instanceof Error ? err.message : 'Unable to revoke door link'); }
    finally { setBusy(false); }
  }
  return <main className="mx-auto max-w-3xl space-y-6 p-6">
    <header><a className="underline" href={`/events/${eventId}`}>Back to event</a><h1 className="mt-4 text-3xl font-bold">Volunteer door links</h1>
      <p className="mt-2 text-fg-secondary">Owners can share admission access for this event without inviting a workspace member. Anyone with a link can search names and codes and check guests in during its time window. Share it privately and revoke it when the shift ends.</p></header>
    {error ? <p role="alert" className="text-status-danger">{error}</p> : null}
    {ready ? <form onSubmit={create} className="space-y-4 border border-stroke-subtle bg-surface-panel p-4">
      <label className="block">Link label<input className="field mt-1" required maxLength={80} value={label} onChange={(e) => setLabel(e.target.value)} placeholder="Front door — Sam" /></label>
      <p className="text-sm text-fg-secondary">Times use this device’s local timezone. Maximum window: seven days. The event must be published.</p>
      <label className="block">Valid from<input className="field mt-1" required type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} /></label>
      <label className="block">Valid until<input className="field mt-1" required type="datetime-local" value={until} onChange={(e) => setUntil(e.target.value)} /></label>
      <button className="btn-primary px-4 py-3" disabled={busy}>Create door link</button>
    </form> : null}
    {sharedUrl ? <section className="space-y-3 border border-stroke-subtle p-4" aria-live="polite"><h2 className="font-bold">Save this link now</h2><p>It is shown only once and cannot be recovered after leaving this page.</p><textarea aria-label="New door link" className="field min-h-28 break-all" readOnly value={sharedUrl} /><a href={sharedUrl} className="btn-secondary px-4 py-2" target="_blank" rel="noreferrer">Open volunteer door</a></section> : null}
    <section><h2 className="text-xl font-bold">Door record</h2><p className="text-sm text-fg-secondary">Check-ins remain attributed after revocation. Reload to update the counts.</p>
      {ready && !links.length ? <p className="mt-4">No door links yet.</p> : null}
      <ul className="mt-4 space-y-4">{links.map((link) => <li className="space-y-2 border border-stroke-subtle bg-surface-panel p-4" key={link.id}>
        <h3 className="font-bold break-words">{link.label}</h3><p>{new Date(link.validFrom).toLocaleString()} – {new Date(link.validUntil).toLocaleString()}</p>
        <p>{link.checkedInCount} checked in · {link.revokedAt ? 'Revoked' : new Date(link.validUntil).getTime() <= Date.now() ? 'Expired' : 'Not revoked'}</p>
        {!link.revokedAt ? <button className="btn-danger px-4 py-2" disabled={busy} onClick={() => revoke(link.id)}>Revoke {link.label}</button> : null}
      </li>)}</ul>
    </section>
  </main>;
}
