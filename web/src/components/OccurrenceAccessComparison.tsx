import { useEffect, useRef, useState } from 'react';
import { ApiError, api } from '../api';
import type { AccessComparisonOccurrenceDTO, OccurrenceAccessComparisonDTO } from '../domain';
import { Button } from '../ui/Button';
import { accessTopics, unknownAccessEntry } from '../modules/eventAccess/eventAccessModel';
import { AccessEntrySummary } from './AccessEntrySummary';

export function AccessComparisonResult({ comparison }: { comparison: OccurrenceAccessComparisonDTO }) {
  const { occurrence, event, venue } = comparison;
  return <div className="space-y-4">
    <div className="rounded-control bg-surface-inset p-3"><p className="break-words font-bold">{occurrence.name}</p><p className="text-sm text-fg-secondary">{occurrence.startsAt} · {occurrence.status}</p><p className="break-words text-xs text-fg-muted">Occurrence revision: {occurrence.updatedAt}. Compared at {comparison.evaluatedAt}.</p></div>
    {venue ? <p className="break-words text-sm">Venue: <a className="underline" href={`/workspace/${comparison.workspaceId}/places/${venue.placeId}/access-info`}>{venue.placeName}</a></p> : <p className="rounded-control border border-stroke-subtle p-3 text-fg-secondary">This occurrence has no linked venue. Venue conditions are unknown.</p>}
    <div className="space-y-3">{accessTopics.map(({ topic, label }) => <article key={topic} className="rounded-panel border border-stroke-subtle bg-surface-panel p-4"><h3 className="mb-3 font-bold">{label}</h3><div className="grid gap-4 sm:grid-cols-2"><div><h4 className="mb-2 text-xs uppercase tracking-wide text-fg-muted">Venue information</h4><AccessEntrySummary entry={venue?.entries.find((entry) => entry.topic === topic) ?? unknownAccessEntry(topic, 'venue')} evaluatedAt={comparison.evaluatedAt} /></div><div><h4 className="mb-2 text-xs uppercase tracking-wide text-fg-muted">Event information</h4><AccessEntrySummary entry={event.entries.find((entry) => entry.topic === topic) ?? unknownAccessEntry(topic)} evaluatedAt={comparison.evaluatedAt} /></div></div></article>)}</div>
  </div>;
}

export function OccurrenceAccessComparison({ eventId, onAuthorizationLoss }: { eventId: string; onAuthorizationLoss: () => void }) {
  const generation = useRef(0);
  const [occurrences, setOccurrences] = useState<AccessComparisonOccurrenceDTO[]>([]);
  const [occurrenceId, setOccurrenceId] = useState('');
  const [comparison, setComparison] = useState<OccurrenceAccessComparisonDTO | null>(null);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [reload, setReload] = useState(0);
  function fail(error: unknown) {
    setComparison(null);
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { generation.current += 1; setOccurrences([]); setOccurrenceId(''); setBusy(false); onAuthorizationLoss(); }
    else setNotice(error instanceof Error ? error.message : 'Could not compare access information. Refresh to try again.');
  }
  useEffect(() => {
    const current = ++generation.current; setBusy(true); setNotice(null); setComparison(null);
    void api<AccessComparisonOccurrenceDTO[]>(`/api/events/${eventId}/access-info/occurrences`).then(async (items) => {
      if (current !== generation.current) return;
      setOccurrences(items);
      if (!occurrenceId) return;
      if (!items.some((item) => item.id === occurrenceId)) { setOccurrenceId(''); setNotice('The selected occurrence is no longer available.'); return; }
      const next = await api<OccurrenceAccessComparisonDTO>(`/api/events/${eventId}/access-info/occurrences/${occurrenceId}/comparison`);
      if (current === generation.current) setComparison(next);
    }).catch((error) => { if (current === generation.current) fail(error); }).finally(() => { if (current === generation.current) setBusy(false); });
    return () => { generation.current += 1; };
  }, [eventId, occurrenceId, reload]);
  function select(id: string) { generation.current += 1; setComparison(null); setNotice(null); setOccurrenceId(id); }
  return <section aria-label="Occurrence venue comparison" className="space-y-4 rounded-panel border border-stroke-subtle p-5">
    <div className="flex flex-wrap items-center justify-between gap-3"><h2 className="text-xl font-bold">Compare venue and event information</h2><Button variant="secondary" busy={busy} onClick={() => { generation.current += 1; setComparison(null); setReload((value) => value + 1); }}>Refresh comparison</Button></div>
    <p className="text-sm text-fg-secondary">Choose an occurrence to see its linked venue beside the event worksheet. These are separately recorded conditions. An event may use a different entrance or arrangement. Comparing them does not verify that the venue conditions apply at this occurrence.</p>
    <label className="block">Occurrence<select value={occurrenceId} onChange={(event) => select(event.target.value)} className="field mt-1 py-3"><option value="">Choose an occurrence</option>{occurrences.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.startsAt} ({item.status})</option>)}</select></label>
    {notice && <p role="status" className="rounded-control border border-stroke-subtle p-3">{notice}</p>}
    {busy ? <p role="status" className="text-sm text-fg-secondary">Loading current comparison…</p> : comparison ? <AccessComparisonResult comparison={comparison} /> : !notice && <p className="text-sm text-fg-secondary">{occurrences.length === 0 ? 'No occurrences recorded for this event.' : 'Choose an occurrence to compare all six topics.'}</p>}
  </section>;
}
