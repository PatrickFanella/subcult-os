import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { ApiError, api, postJSON } from '../api';
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
  const occurrence = useMemo(() => occurrences.find((item) => item.id === occurrenceId) ?? null, [occurrences, occurrenceId]);

  function clearPrivateState() { setEvents([]); setEventId(''); setOccurrences([]); setOccurrenceId(''); setIntents([]); setReason(''); setActionKinds(['operational_notice']); setRetry(null); setBusy(false); }
  function fail(error: unknown) { if (error instanceof ApiError && (error.status === 401 || error.status === 403)) clearPrivateState(); setNotice(message(error)); }
  useEffect(() => {
    let active = true; const current = ++generation.current; clearPrivateState(); setNotice(null);
    void api<EventDTO[]>(`/api/workspaces/${workspaceId}/events`).then((next) => { if (active && current === generation.current) setEvents(next); }).catch((error) => { if (active) fail(error); });
    return () => { active = false; generation.current += 1; clearPrivateState(); };
  }, [workspaceId]);
  useEffect(() => {
    let active = true; setOccurrences([]); setOccurrenceId(''); setIntents([]);
    if (eventId) {
      void Promise.all([api<EventOccurrenceDTO[]>(`/api/events/${eventId}/occurrences`), api<LifecycleIntentDTO[]>(`/api/events/${eventId}/lifecycle-intents`)])
        .then(([nextOccurrences, nextIntents]) => { if (active) { setOccurrences(nextOccurrences); setIntents(nextIntents); } }).catch((error) => { if (active) fail(error); });
    }
    return () => { active = false; };
  }, [eventId]);
  function toggle(kind: LifecycleActionKind) { setActionKinds((current) => current.includes(kind) ? current.filter((item) => item !== kind) : [...current, kind]); }
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!eventId || !occurrence || busy) return; setBusy(true); setNotice(null); const current = generation.current;
    const payload = JSON.stringify({ eventId, occurrenceId: occurrence.id, expectedUpdatedAt: occurrence.updatedAt, expectedPublicCid: occurrence.publicCid ?? '', kind, reason, actionKinds });
    const nextRetry = nextLifecycleIntentRetry(retry, payload, () => crypto.randomUUID()); setRetry(nextRetry);
    try {
      const created = await postJSON<LifecycleIntentDTO>(`/api/events/${eventId}/lifecycle-intents`, { decisionKey: nextRetry.key, occurrenceId: occurrence.id, expectedUpdatedAt: occurrence.updatedAt, expectedPublicCid: occurrence.publicCid ?? '', kind, reason, actionKinds });
      if (current === generation.current) { setIntents((items) => items.some((item) => item.id === created.id) ? items : [created, ...items]); setReason(''); setRetry(null); setNotice('Recorded an unexecuted lifecycle intent. Each action remains a draft.'); }
    } catch (error) { if (current === generation.current) fail(error); } finally { if (current === generation.current) setBusy(false); }
  }
  async function supersede(intent: LifecycleIntentDTO) {
    if (busy || !eventId) return; setBusy(true); setNotice(null); const current = generation.current;
    try { const next = await postJSON<LifecycleIntentDTO>(`/api/events/${eventId}/lifecycle-intents/${intent.id}/supersede`, {}); if (current === generation.current) setIntents((items) => items.map((item) => item.id === next.id ? next : item)); }
    catch (error) { if (current === generation.current) fail(error); } finally { if (current === generation.current) setBusy(false); }
  }
  return <main className="mx-auto max-w-4xl space-y-6 p-6 text-zinc-100">
    <a className="text-sm text-zinc-400 underline" href={`/workspace?workspaceId=${workspaceId}`}>Back to workspace</a>
    <header><p className="text-xs uppercase tracking-[0.2em] text-amber-300">Private operator worklist</p><h1 className="text-3xl font-semibold">Lifecycle intents</h1><p className="mt-2 max-w-2xl text-zinc-400">Record an approved cancellation or reschedule decision against one exact occurrence revision. Actions start as unexecuted drafts. A listing notice requires separate message and recipient review before queuing. Recording a decision does not change the event, contact a provider, publish a record, or issue a refund.</p></header>
    {notice && <p role="status" className="rounded border border-zinc-700 p-3">{notice}</p>}
    <form className="space-y-3 rounded border border-zinc-800 p-4" onSubmit={submit}>
      <h2 className="font-medium">Record a decision</h2><label>Event<select required value={eventId} disabled={busy} onChange={(e) => setEventId(e.target.value)} className="mt-1 w-full rounded bg-zinc-900 p-2"><option value="">Choose an event</option>{events.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>
      <label>Occurrence<select required value={occurrenceId} disabled={busy || !eventId} onChange={(e) => setOccurrenceId(e.target.value)} className="mt-1 w-full rounded bg-zinc-900 p-2"><option value="">Choose the exact occurrence</option>{occurrences.map((item) => <option key={item.id} value={item.id}>{item.name} — {item.startsAt} ({item.status})</option>)}</select></label>
      {occurrence && <p className="rounded bg-zinc-900 p-3 text-sm text-zinc-300">Revision: <code>{occurrence.updatedAt}</code><br/>Recorded public CID: <code>{occurrence.publicCid ?? 'none'}</code></p>}
      <label>Decision<select value={kind} disabled={busy} onChange={(e) => setKind(e.target.value as typeof kind)} className="mt-1 w-full rounded bg-zinc-900 p-2"><option value="cancellation">Cancellation</option><option value="reschedule">Reschedule</option></select></label>
      <label>Reason<textarea required value={reason} maxLength={1000} disabled={busy} onChange={(e) => setReason(e.target.value)} className="mt-1 min-h-24 w-full rounded bg-zinc-900 p-2" /></label>
      <fieldset><legend className="text-sm text-zinc-300">Draft action work</legend>{actionOptions.map((item) => <label key={item.value} className="mr-4 inline-block"><input type="checkbox" checked={actionKinds.includes(item.value)} disabled={busy} onChange={() => toggle(item.value)} /> {item.label}</label>)}</fieldset>
      <button disabled={busy || !occurrence || actionKinds.length === 0} className="rounded bg-amber-300 px-3 py-2 text-zinc-950">Record unexecuted intent</button>
    </form>
    {eventId && <section className="space-y-3"><h2 className="text-xl font-semibold">Recorded worklist</h2>{intents.length === 0 ? <p className="text-zinc-400">No lifecycle intents recorded for this event.</p> : intents.map((intent) => <article key={intent.id} className="rounded border border-zinc-800 p-4"><div className="flex flex-wrap justify-between gap-2"><p><span className="rounded bg-amber-400/15 px-2 py-1 text-sm text-amber-200">{intent.kind}</span> <span className="ml-2 text-sm text-zinc-400">{intent.status === 'approved' ? 'Decision recorded' : 'Superseded'}</span></p>{intent.status === 'approved' && <button disabled={busy} onClick={() => supersede(intent)} className="text-sm underline">Supersede unsent actions</button>}</div><p className="mt-3">{intent.reason}</p><p className="mt-2 text-xs text-zinc-500">Occurrence {intent.occurrenceId} · revision {intent.targetRevision}</p><ul className="mt-3 space-y-1 text-sm">{intent.actions.map((action) => <li key={action.id}>{action.actionKind.replace('_', ' ')}: <strong>{action.status}</strong> · attempts {action.attemptCount}{action.status === 'unknown' && ' · reconcile before retrying'}</li>)}</ul>{intent.actions.some((action) => action.actionKind === 'operational_notice') && <LifecycleNoticePreview eventId={eventId} changeId={intent.id} canApprove={intent.status === 'approved'} onAuthorizationLoss={() => { clearPrivateState(); setNotice('Your owner access is no longer available.'); }} />}</article>)}</section>}
  </main>;
}
