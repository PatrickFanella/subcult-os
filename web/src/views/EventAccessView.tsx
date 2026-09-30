import { useEffect, useRef, useState, type FormEvent } from 'react';
import { ApiError, api, postJSON } from '../api';
import type { AccessInformationRevisionDTO, EventAccessTopic, EventAccessWorksheetDTO, VenueAccessWorksheetDTO, EventDTO } from '../domain';
import { AccessEntrySummary } from '../components/AccessEntrySummary';
import { OccurrenceAccessComparison } from '../components/OccurrenceAccessComparison';
import { Button } from '../ui/Button';
import { accessSourceLabels, accessTimestamp, accessTopics, accessValueLabels, accessValues, nextAccessRetry, unknownAccessEntry, type AccessRetry } from '../modules/eventAccess/eventAccessModel';

type Worksheet = { evaluatedAt: string; entries: AccessInformationRevisionDTO[] };
type History = { topic: EventAccessTopic; revisions: AccessInformationRevisionDTO[]; nextBefore?: number };
type Draft = { entry: AccessInformationRevisionDTO; reviewed: string; expires: string; reason: string };
const draftFrom = (entry: AccessInformationRevisionDTO): Draft => ({ entry, reviewed: entry.reviewedAt?.slice(0, 16) ?? '', expires: entry.expiresAt?.slice(0, 16) ?? '', reason: '' });
const control = 'mt-1 w-full rounded-control border border-stroke-subtle bg-surface-inset p-3 text-fg-primary';

export { AccessEntrySummary } from '../components/AccessEntrySummary';

export function EventAccessView({ eventId }: { eventId: string }) { return <AccessInformationView key={eventId} eventId={eventId} />; }
export function VenueAccessView({ workspaceId, placeId }: { workspaceId: string; placeId: string }) { return <AccessInformationView key={`${workspaceId}:${placeId}`} workspaceId={workspaceId} placeId={placeId} />; }
function AccessInformationView({ eventId, workspaceId, placeId }: { eventId?: string; workspaceId?: string; placeId?: string }) {
  const scope = placeId ? 'venue' : 'event';
  const endpoint = placeId ? `/api/workspaces/${workspaceId}/places/${placeId}/access-info` : `/api/events/${eventId}/access-info`;
  const sources = scope === 'venue' ? ['organizer_assertion', 'venue_observation', 'external_reference'] as const : ['organizer_assertion', 'event_observation', 'external_reference'] as const;
  async function loadWorksheet() {
    if (placeId) { const next = await api<VenueAccessWorksheetDTO>(endpoint); return { name: next.placeName, worksheet: next as Worksheet }; }
    const [event, next] = await Promise.all([api<EventDTO>(`/api/events/${eventId}`), api<EventAccessWorksheetDTO>(endpoint)]);
    return { name: event.title, worksheet: next as Worksheet };
  }
  const generation = useRef(0); const topicGeneration = useRef(0); const readGeneration = useRef(0);
  const [name, setName] = useState<string | null>(null);
  const [worksheet, setWorksheet] = useState<Worksheet | null>(null);
  const [draft, setDraft] = useState<Draft>(() => draftFrom(unknownAccessEntry('entry', scope)));
  const [history, setHistory] = useState<History | null>(null);
  const [retry, setRetry] = useState<AccessRetry | null>(null);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [denied, setDenied] = useState(false);
  function clearPrivate() { generation.current += 1; topicGeneration.current += 1; readGeneration.current += 1; setName(null); setWorksheet(null); setHistory(null); setDraft(draftFrom(unknownAccessEntry('entry', scope))); setRetry(null); setBusy(false); }
  function fail(error: unknown) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { clearPrivate(); setDenied(true); }
    setNotice(error instanceof Error ? error.message : 'The private access worksheet could not be updated.');
  }
  useEffect(() => {
    clearPrivate(); setDenied(false); setNotice(null); const current = generation.current;
    void loadWorksheet()
      .then(({ name, worksheet: next }) => { if (current === generation.current) { setName(name); setWorksheet(next); setDraft(draftFrom(next.entries[0])); } }).catch((error) => { if (current === generation.current) fail(error); });
    return () => { clearPrivate(); };
  }, [endpoint]);
  async function refresh() {
    if (busy || denied) return;
    const current = generation.current; const request = ++readGeneration.current; setBusy(true); setNotice(null);
    try { const { name, worksheet: next } = await loadWorksheet(); if (current === generation.current && request === readGeneration.current) { setName(name); setWorksheet(next); if (!worksheet) setDraft(draftFrom(next.entries[0])); } }
    catch (error) { if (current === generation.current && request === readGeneration.current) fail(error); }
    finally { if (current === generation.current && request === readGeneration.current) setBusy(false); }
  }
  function selectTopic(topic: EventAccessTopic) { topicGeneration.current += 1; setHistory(null); setRetry(null); setNotice(null); setDraft(draftFrom(worksheet?.entries.find((entry) => entry.topic === topic) ?? unknownAccessEntry(topic, scope))); }
  function changeValue(value: AccessInformationRevisionDTO['value']) {
    setDraft((current) => ({ ...current, entry: value === 'unknown' ? { ...unknownAccessEntry(current.entry.topic, scope), revision: current.entry.revision } : { ...current.entry, value, sourceKind: current.entry.sourceKind === 'unknown' ? 'organizer_assertion' : current.entry.sourceKind }, reviewed: value === 'unknown' ? '' : current.reviewed, expires: value === 'unknown' ? '' : current.expires }));
  }
  async function loadHistory(older = false) {
    if (busy || denied) return;
    const current = generation.current; const topicCurrent = topicGeneration.current; setBusy(true); setNotice(null);
    try {
      const next = await api<History>(`${endpoint}/${draft.entry.topic}/history${older && history?.nextBefore ? `?before=${history.nextBefore}` : ''}`);
      if (current === generation.current && topicCurrent === topicGeneration.current) setHistory((previous) => older && previous ? { ...next, revisions: [...previous.revisions, ...next.revisions] } : next);
    } catch (error) { if (current === generation.current && topicCurrent === topicGeneration.current) fail(error); }
    finally { if (current === generation.current && topicCurrent === topicGeneration.current) setBusy(false); }
  }
  async function submit(e: FormEvent) {
    e.preventDefault(); if (busy || denied || !worksheet) return;
    const current = generation.current; const topicCurrent = topicGeneration.current; readGeneration.current += 1; setBusy(true); setNotice(null);
    try {
      const body = { expectedRevision: draft.entry.revision, value: draft.entry.value, details: draft.entry.details, sourceKind: draft.entry.sourceKind, sourceReference: draft.entry.sourceReference, reviewedAt: accessTimestamp(draft.reviewed, draft.entry.reviewedAt), expiresAt: accessTimestamp(draft.expires, draft.entry.expiresAt), correctionReason: draft.reason };
      const request = nextAccessRetry(retry, JSON.stringify({ endpoint, topic: draft.entry.topic, ...body }), () => crypto.randomUUID()); setRetry(request);
      const created = await postJSON<AccessInformationRevisionDTO>(`${endpoint}/${draft.entry.topic}`, { ...body, requestKey: request.key });
      if (current !== generation.current || topicCurrent !== topicGeneration.current) return;
      setWorksheet((previous) => previous && { ...previous, entries: previous.entries.map((entry) => entry.topic === created.topic && created.revision >= entry.revision ? created : entry) }); setDraft(draftFrom(created)); setHistory(null); setRetry(null); setNotice('Revision recorded privately. The previous record remains in correction history.');
    } catch (error) { if (current === generation.current && topicCurrent === topicGeneration.current) fail(error); }
    finally { if (current === generation.current && topicCurrent === topicGeneration.current) setBusy(false); }
  }
  return <main className="mx-auto max-w-5xl space-y-6 px-4 py-6 text-fg-primary sm:px-6">
    <a className="text-sm underline" href={placeId ? `/workspace/${workspaceId}/venue-access` : `/events/${eventId}`}>{placeId ? 'Back to venue worksheets' : 'Back to event'}</a>
    <header className="space-y-3"><p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Private {scope} worksheet</p><h1 className="text-3xl font-extrabold">Access information</h1>{name && <p className="text-lg font-bold">{name}</p>}
      <p className="max-w-3xl text-fg-secondary">{scope === 'venue' ? 'Record assertions about this venue, their source and when they were checked. An event may use a different entrance or arrangement, so venue assertions never become event observations automatically.' : 'Record conditions for this event, their source and when they were checked. Organizer assertions and recorded observations remain distinct. Venue information is not inherited.'} This worksheet remains private.</p>
      <p className="text-sm text-fg-secondary">Keep personal accommodation requests, diagnoses and attendee details out of this worksheet. Keep individual requests in your private communication with the attendee.</p>
    </header>
    {notice && <p role={denied ? 'alert' : 'status'} className="rounded-control border border-stroke-subtle p-3">{notice}</p>}
    {denied ? <p>Owner access is required. Private worksheet data has been cleared.</p> : !worksheet ? <div className="space-y-3"><p role="status">{notice ? 'No worksheet loaded. Try again to check your access and load the current information.' : 'Loading access information…'}</p>{notice && <Button variant="secondary" busy={busy} onClick={() => void refresh()}>Try loading again</Button>}</div> : <>
      <section aria-label="Current access information" className="space-y-4"><div className="flex flex-wrap items-center justify-between gap-3"><h2 className="text-xl font-bold">Current {scope} information</h2><Button variant="secondary" busy={busy} onClick={() => void refresh()}>Refresh information</Button></div>
        <p className="text-xs text-fg-muted">Worksheet checked at {worksheet.evaluatedAt}. Refresh to check for newer corrections and elapsed review dates.</p>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">{worksheet.entries.map((entry) => <article key={entry.topic} className="space-y-3 rounded-panel border border-stroke-subtle bg-surface-panel p-4"><h3 className="font-bold">{accessTopics.find((item) => item.topic === entry.topic)?.label}</h3><AccessEntrySummary entry={entry} evaluatedAt={worksheet.evaluatedAt} /><Button variant="ghost" disabled={busy} onClick={() => selectTopic(entry.topic)}>Edit {accessTopics.find((item) => item.topic === entry.topic)?.label.toLowerCase()}</Button></article>)}</div>
      </section>
      {eventId && <OccurrenceAccessComparison eventId={eventId} onAuthorizationLoss={() => { clearPrivate(); setDenied(true); setNotice('Owner access is no longer available. Private information has been cleared.'); }} />}
      <form onSubmit={submit} className="space-y-4 rounded-panel border border-stroke-subtle bg-surface-panel p-5"><h2 className="text-xl font-bold">Record an update</h2>
        <fieldset disabled={busy} className="grid min-w-0 gap-4 sm:grid-cols-2"><legend className="sr-only">Access information revision</legend>
          <label>Topic<select className={control} value={draft.entry.topic} onChange={(e) => selectTopic(e.target.value as EventAccessTopic)}>{accessTopics.map((item) => <option key={item.topic} value={item.topic}>{item.label}</option>)}</select></label>
          <label>Recorded value<select className={control} value={draft.entry.value} onChange={(e) => changeValue(e.target.value as AccessInformationRevisionDTO['value'])}>{accessValues(draft.entry.topic).map((value) => <option key={value} value={value}>{accessValueLabels[value]}</option>)}</select></label>
          {draft.entry.value !== 'unknown' && <>
            <label className="sm:col-span-2">Conditions and limits<textarea className={control} value={draft.entry.details} required={draft.entry.value === 'known'} maxLength={1000} onChange={(e) => setDraft((d) => ({ ...d, entry: { ...d.entry, details: e.target.value } }))} /></label>
            <label>Source type<select className={control} value={draft.entry.sourceKind} onChange={(e) => setDraft((d) => ({ ...d, entry: { ...d.entry, sourceKind: e.target.value as AccessInformationRevisionDTO['sourceKind'] } }))}>{sources.map((source) => <option key={source} value={source}>{accessSourceLabels[source]}</option>)}</select></label>
            <label>Source reference<input className={control} required maxLength={500} value={draft.entry.sourceReference} onChange={(e) => setDraft((d) => ({ ...d, entry: { ...d.entry, sourceReference: e.target.value } }))} /></label>
            <label>Review time (UTC)<input className={control} type="datetime-local" required value={draft.reviewed} onChange={(e) => setDraft((d) => ({ ...d, reviewed: e.target.value }))} /></label>
            <label>Review by (UTC, optional)<input className={control} type="datetime-local" value={draft.expires} onChange={(e) => setDraft((d) => ({ ...d, expires: e.target.value }))} /></label>
          </>}
          <label className="sm:col-span-2">Reason for this update<textarea className={control} required maxLength={1000} value={draft.reason} onChange={(e) => setDraft((d) => ({ ...d, reason: e.target.value }))} /></label>
        </fieldset>
        <p className="text-xs text-fg-muted">Updating revision {draft.entry.revision}. Corrections retain the previous record. Selecting Unknown withdraws the current assertion.</p>
        {worksheet.entries.find((entry) => entry.topic === draft.entry.topic)?.revision !== draft.entry.revision && <p role="alert" className="text-sm text-status-warning">A newer revision is available. Select this topic again to review it before correcting.</p>}
        <Button type="submit" busy={busy} disabled={!draft.reason.trim()}>Save private revision</Button>
      </form>
      <section className="space-y-3" aria-label="Correction history"><h2 className="text-xl font-bold">Correction history · {accessTopics.find((item) => item.topic === draft.entry.topic)?.label}</h2><Button variant="secondary" busy={busy} onClick={() => void loadHistory()}>Load correction history</Button>
        {history && <><ol className="space-y-3">{history.revisions.map((entry) => <li key={entry.id} className="space-y-2 rounded-control border border-stroke-subtle p-4"><p className="font-bold">Revision {entry.revision} · {accessValueLabels[entry.value]}</p><p className="whitespace-pre-wrap break-words">{entry.correctionReason}</p><p className="break-words text-xs text-fg-muted">Recorded: {entry.recordedAt}</p><AccessEntrySummary entry={entry} evaluatedAt={worksheet.evaluatedAt} /></li>)}</ol>{history.revisions.length === 0 && <p>No revisions recorded for this topic.</p>}{history.nextBefore && <Button variant="secondary" busy={busy} onClick={() => void loadHistory(true)}>Load older revisions</Button>}</>}
      </section>
    </>}
  </main>;
}
