import { FormEvent, useEffect, useRef, useState } from 'react';
import { ApiError, api, postJSON } from '../api';
import { Button } from '../ui/Button';
import type { VenueAccessPlaceDTO, VenueAccessPlaceIndexDTO } from '../domain';
import { nextAccessRetry, type AccessRetry } from '../modules/eventAccess/eventAccessModel';

export function VenueAccessIndexView({ workspaceId }: { workspaceId: string }) {
  const generation = useRef(0);
  const [reload, setReload] = useState(0);
  const [places, setPlaces] = useState<VenueAccessPlaceDTO[]>([]);
  const [nextAfter, setNextAfter] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [retry, setRetry] = useState<AccessRetry | null>(null);
  const [busy, setBusy] = useState(false);
  const [allowed, setAllowed] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const endpoint = `/api/workspaces/${workspaceId}/venue-access`;
  function clearPrivateState() { setPlaces([]); setNextAfter(null); setName(''); setRetry(null); setAllowed(false); }
  function fail(error: unknown) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { generation.current += 1; clearPrivateState(); setBusy(false); }
    setNotice(error instanceof ApiError ? error.message : 'Could not update venue references. Retry with the same name to recover an uncertain save.');
  }
  useEffect(() => {
    const current = ++generation.current; clearPrivateState(); setBusy(true); setNotice(null);
    void api<VenueAccessPlaceIndexDTO>(endpoint).then((result) => {
      if (current === generation.current) { setPlaces(result.places); setNextAfter(result.nextAfter ?? null); setAllowed(true); }
    }).catch((error) => { if (current === generation.current) fail(error); }).finally(() => { if (current === generation.current) setBusy(false); });
    return () => { generation.current += 1; };
  }, [endpoint, reload]);
  async function loadMore() {
    if (!nextAfter || busy) return;
    const current = generation.current; setBusy(true); setNotice(null);
    try {
      const result = await api<VenueAccessPlaceIndexDTO>(`${endpoint}?after=${encodeURIComponent(nextAfter)}`);
      if (current === generation.current) { setPlaces((items) => [...items, ...result.places.filter((place) => !items.some((item) => item.id === place.id))]); setNextAfter(result.nextAfter ?? null); }
    } catch (error) { if (current === generation.current) fail(error); }
    finally { if (current === generation.current) setBusy(false); }
  }
  async function create(event: FormEvent) {
    event.preventDefault(); if (busy || !allowed || !name.trim()) return;
    const current = generation.current; const normalized = name.trim();
    const attempt = nextAccessRetry(retry, JSON.stringify({ endpoint, name: normalized }), () => crypto.randomUUID());
    setRetry(attempt); setBusy(true); setNotice(null);
    try {
      const place = await postJSON<VenueAccessPlaceDTO>(endpoint, { name: normalized, requestKey: attempt.key });
      if (current === generation.current) { setPlaces((items) => items.some((item) => item.id === place.id) ? items : [place, ...items]); setName(''); setRetry(null); setNotice('Venue reference saved. Its access topics start as unknown.'); }
    } catch (error) { if (current === generation.current) fail(error); }
    finally { if (current === generation.current) setBusy(false); }
  }
  return <main className="mx-auto max-w-4xl space-y-6 p-6 text-fg-primary">
    <a className="text-sm text-fg-secondary underline" href={`/workspace?workspaceId=${workspaceId}`}>Back to workspace</a>
    <header><p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Private owner worksheet</p><h1 className="text-3xl font-extrabold">Venue access information</h1><p className="mt-2 max-w-2xl text-fg-secondary">Record venue observations with a source, review date and correction history. An event needs its own verification; venue observations do not automatically populate event information.</p></header>
    {notice && <p role="status" className="rounded border border-stroke-subtle p-3">{notice}</p>}
    {!allowed && !busy && notice && <Button variant="secondary" onClick={() => setReload((value) => value + 1)}>Try loading again</Button>}
    {allowed && <><form onSubmit={create} className="space-y-3 rounded border border-stroke-subtle p-4"><h2 className="text-xl font-bold">Add a venue reference</h2><label className="block" htmlFor="venue-name">Venue name</label><input id="venue-name" required maxLength={600} value={name} disabled={busy} onChange={(event) => setName(event.target.value)} className="w-full rounded border border-stroke-subtle bg-surface-inset p-3"/><p className="text-sm text-fg-secondary">A name identifies the worksheet. Adding it does not assert any access facilities or publish a listing.</p><Button type="submit" busy={busy} disabled={!name.trim()}>Add venue reference</Button></form>
    <section className="space-y-3"><h2 className="text-xl font-bold">Venue worksheets</h2>{places.length === 0 ? <p className="text-fg-secondary">No venue references yet.</p> : <ul className="space-y-2">{places.map((place) => <li key={place.id}><a className="block break-words rounded border border-stroke-subtle bg-surface-raised p-4 underline" href={`/workspace/${workspaceId}/places/${place.id}/access-info`}>{place.name}</a></li>)}</ul>}{nextAfter && <Button variant="secondary" busy={busy} onClick={() => void loadMore()}>Load more venues</Button>}</section></>}
  </main>;
}
