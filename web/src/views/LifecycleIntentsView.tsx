import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { ApiError, api, postJSON } from '../api';
import { Button } from '../ui/Button';
import { LifecycleActionRow } from './LifecycleActionRow';
import { LifecycleNoticePreview } from './LifecycleNoticePreview';
import type { EventDTO, EventOccurrenceDTO, LifecycleActionKind, LifecycleIntentDTO } from '../domain';
import { nextLifecycleIntentRetry, type LifecycleIntentRetry } from '../modules/lifecycleIntents/lifecycleIntentModel';

const actionOptions: Array<{ value: LifecycleActionKind; label: string }> = [
  { value: 'public_record', label: 'Public record review' },
  { value: 'provider_ticket', label: 'Ticket provider review' },
  { value: 'operational_notice', label: 'Operational notice draft' },
  { value: 'refund', label: 'Refund review' },
];

function message(error: unknown) { return error instanceof ApiError ? error.message : 'The private lifecycle worklist could not be updated.'; }

export function LifecycleIntentsView({ workspaceId }: { workspaceId: string }) {
  const generation = useRef(0);
  const eventGeneration = useRef(0);
  const refreshGeneration = useRef(0);
  const [events, setEvents] = useState<EventDTO[]>([]);
  const [eventId, setEventId] = useState('');
  const [occurrences, setOccurrences] = useState<EventOccurrenceDTO[]>([]);
  const [occurrenceId, setOccurrenceId] = useState('');
  const [intents, setIntents] = useState<LifecycleIntentDTO[]>([]);
  const [kind, setKind] = useState<'cancellation' | 'reschedule'>('cancellation');
  const [reason, setReason] = useState('');
  const [actionKinds, setActionKinds] = useState<LifecycleActionKind[]>(['operational_notice']);
  const [retry, setRetry] = useState<LifecycleIntentRetry | null>(null);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const occurrence = useMemo(() => occurrences.find((item) => item.id === occurrenceId) ?? null, [occurrences, occurrenceId]);

  function clearPrivateState() { eventGeneration.current += 1; refreshGeneration.current += 1; setRefreshing(false); setEvents([]); setEventId(''); setOccurrences([]); setOccurrenceId(''); setIntents([]); setReason(''); setActionKinds(['operational_notice']); setRetry(null); setBusy(false); }
  function fail(error: unknown) { if (error instanceof ApiError && (error.status === 401 || error.status === 403)) clearPrivateState(); setNotice(message(error)); }
  useEffect(() => {
    let active = true; const current = ++generation.current; clearPrivateState(); setNotice(null);
    void api<EventDTO[]>(`/api/workspaces/${workspaceId}/events`).then((next) => { if (active && current === generation.current) setEvents(next); }).catch((error) => { if (active) fail(error); });
    return () => { active = false; generation.current += 1; clearPrivateState(); };
  }, [workspaceId]);
  useEffect(() => {
    let active = true; const current = ++eventGeneration.current; const workspaceCurrent = generation.current; refreshGeneration.current += 1; setRefreshing(false); setNotice(null); setBusy(false); setRetry(null); setReason(''); setOccurrences([]); setOccurrenceId(''); setIntents([]);
    if (eventId) {
      void Promise.all([api<EventOccurrenceDTO[]>(`/api/events/${eventId}/occurrences`), api<LifecycleIntentDTO[]>(`/api/events/${eventId}/lifecycle-intents`)])
        .then(([nextOccurrences, nextIntents]) => { if (active && current === eventGeneration.current && workspaceCurrent === generation.current) { setOccurrences(nextOccurrences); setIntents(nextIntents); } }).catch((error) => { if (active && current === eventGeneration.current && workspaceCurrent === generation.current) fail(error); });
    }
    return () => { active = false; };
  }, [eventId]);
  async function refreshWorklist() {
    if (!eventId) return;
    const current = generation.current; const eventCurrent = eventGeneration.current; const request = ++refreshGeneration.current;
    const isCurrent = () => current === generation.current && eventCurrent === eventGeneration.current && request === refreshGeneration.current;
    setRefreshing(true);
    try {
      const next = await api<LifecycleIntentDTO[]>(`/api/events/${eventId}/lifecycle-intents`);
      if (isCurrent()) setIntents(next);
    } catch (error) { if (isCurrent()) fail(error); }
    finally { if (isCurrent()) setRefreshing(false); }
  }
  function toggle(kind: LifecycleActionKind) { setActionKinds((current) => current.includes(kind) ? current.filter((item) => item !== kind) : [...current, kind]); }
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!eventId || !occurrence || busy) return; refreshGeneration.current += 1; setRefreshing(false); setBusy(true); setNotice(null); const current = generation.current; const eventCurrent = eventGeneration.current;
    const payload = JSON.stringify({ eventId, occurrenceId: occurrence.id, expectedUpdatedAt: occurrence.updatedAt, expectedPublicCid: occurrence.publicCid ?? '', kind, reason, actionKinds });
    const nextRetry = nextLifecycleIntentRetry(retry, payload, () => crypto.randomUUID()); setRetry(nextRetry);
    try {
      const created = await postJSON<LifecycleIntentDTO>(`/api/events/${eventId}/lifecycle-intents`, { decisionKey: nextRetry.key, occurrenceId: occurrence.id, expectedUpdatedAt: occurrence.updatedAt, expectedPublicCid: occurrence.publicCid ?? '', kind, reason, actionKinds });
      if (current === generation.current && eventCurrent === eventGeneration.current) { setIntents((items) => items.some((item) => item.id === created.id) ? items : [created, ...items]); setReason(''); setRetry(null); setNotice('Recorded an unexecuted lifecycle intent. Each action remains a draft.'); }
    } catch (error) { if (current === generation.current && eventCurrent === eventGeneration.current) fail(error); } finally { if (current === generation.current && eventCurrent === eventGeneration.current) setBusy(false); }
  }
  async function supersede(intent: LifecycleIntentDTO) {
    if (busy || !eventId) return; refreshGeneration.current += 1; setRefreshing(false); setBusy(true); setNotice(null); const current = generation.current; const eventCurrent = eventGeneration.current;
    try { const next = await postJSON<LifecycleIntentDTO>(`/api/events/${eventId}/lifecycle-intents/${intent.id}/supersede`, {}); if (current === generation.current && eventCurrent === eventGeneration.current) setIntents((items) => items.map((item) => item.id === next.id ? next : item)); }
    catch (error) { if (current === generation.current && eventCurrent === eventGeneration.current) fail(error); } finally { if (current === generation.current && eventCurrent === eventGeneration.current) setBusy(false); }
  }
  return <main className="mx-auto max-w-4xl space-y-6 p-6 text-fg-primary">
    <a className="text-sm text-fg-secondary underline" href={`/workspace?workspaceId=${workspaceId}`}>Back to workspace</a>
    <header><p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Private operator worklist</p><h1 className="text-3xl font-extrabold">Lifecycle intents</h1><p className="mt-2 max-w-2xl text-fg-secondary">Record an approved cancellation or reschedule decision against one exact occurrence revision. Actions start as unexecuted drafts. A listing notice requires separate message and recipient review before queuing. Recording a decision does not change the event, contact a provider, publish a record, or issue a refund.</p></header>
    {notice && <p role="status" className="rounded border border-stroke-subtle p-3">{notice}</p>}
    <form className="space-y-3 rounded border border-stroke-subtle p-4" onSubmit={submit}>
      <h2 className="font-medium">Record a decision</h2><label>Event<select required value={eventId} disabled={busy} onChange={(e) => { eventGeneration.current += 1; setEventId(e.target.value); }} className="mt-1 w-full rounded bg-surface-inset p-2"><option value="">Choose an event</option>{events.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>
      <label>Occurrence<select required value={occurrenceId} disabled={busy || !eventId} onChange={(e) => setOccurrenceId(e.target.value)} className="mt-1 w-full rounded bg-surface-inset p-2"><option value="">Choose the exact occurrence</option>{occurrences.map((item) => <option key={item.id} value={item.id}>{item.name} — {item.startsAt} ({item.status})</option>)}</select></label>
      {occurrence && <p className="rounded bg-surface-inset p-3 text-sm text-fg-secondary">Revision: <code>{occurrence.updatedAt}</code><br/>Recorded public CID: <code>{occurrence.publicCid ?? 'none'}</code></p>}
      <label>Decision<select value={kind} disabled={busy} onChange={(e) => setKind(e.target.value as typeof kind)} className="mt-1 w-full rounded bg-surface-inset p-2"><option value="cancellation">Cancellation</option><option value="reschedule">Reschedule</option></select></label>
      <label>Reason<textarea required value={reason} maxLength={1000} disabled={busy} onChange={(e) => setReason(e.target.value)} className="mt-1 min-h-24 w-full rounded bg-surface-inset p-2" /></label>
      <fieldset><legend className="text-sm text-fg-secondary">Draft action work</legend>{actionOptions.map((item) => <label key={item.value} className="mr-4 inline-block"><input type="checkbox" checked={actionKinds.includes(item.value)} disabled={busy} onChange={() => toggle(item.value)} /> {item.label}</label>)}</fieldset>
      <button disabled={busy || !occurrence || actionKinds.length === 0} className="rounded bg-action-primary px-3 py-2 text-fg-inverse">Record unexecuted intent</button>
    </form>
    {eventId && <section className="space-y-3"><div className="flex flex-wrap items-center justify-between gap-3"><h2 className="text-xl font-extrabold">Recorded worklist</h2><Button variant="secondary" busy={refreshing} disabled={busy} onClick={() => void refreshWorklist()}>Refresh worklist</Button></div>{intents.length === 0 ? <p className="text-fg-secondary">No lifecycle intents recorded for this event.</p> : intents.map((intent) => <article key={intent.id} className="rounded border border-stroke-subtle p-4"><div className="flex flex-wrap justify-between gap-2"><p><span className="rounded bg-status-surface-warning px-2 py-1 text-sm text-status-warning">{intent.kind}</span> <span className="ml-2 text-sm text-fg-secondary">{intent.status === 'approved' ? 'Decision recorded' : 'Superseded'}</span></p>{intent.status === 'approved' && <button disabled={busy} onClick={() => supersede(intent)} className="text-sm underline">Supersede unsent actions</button>}</div><p className="mt-3">{intent.reason}</p><p className="mt-2 text-xs text-fg-muted">Occurrence {intent.occurrenceId} · revision {intent.targetRevision}</p><ul className="mt-3 space-y-1 text-sm">{intent.actions.map((action) => <LifecycleActionRow key={action.id} action={action} />)}</ul>{intent.actions.some((action) => action.actionKind === 'operational_notice') && <LifecycleNoticePreview eventId={eventId} changeId={intent.id} canApprove={intent.status === 'approved'} onQueued={() => void refreshWorklist()} onAuthorizationLoss={() => { clearPrivateState(); setNotice('Your owner access is no longer available.'); }} />}</article>)}</section>}
  </main>;
}
