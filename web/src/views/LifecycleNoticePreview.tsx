import { useEffect, useRef, useState } from 'react';
import { ApiError, api, postJSON } from '../api';
import type { LifecycleNoticeDTO, LifecycleNoticePreviewDTO } from '../domain';
import { nextLifecycleIntentRetry, type LifecycleIntentRetry } from '../modules/lifecycleIntents/lifecycleIntentModel';

export function LifecycleNoticePreview({ eventId, changeId, canApprove = true, onAuthorizationLoss }: { eventId: string; changeId: string; canApprove?: boolean; onAuthorizationLoss: () => void }) {
  const generation = useRef(0);
  const [ticketHolders, setTicketHolders] = useState(true);
  const [assignedCrew, setAssignedCrew] = useState(false);
  const [preview, setPreview] = useState<LifecycleNoticePreviewDTO | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [queued, setQueued] = useState<LifecycleNoticeDTO | null>(null);
  const [retry, setRetry] = useState<LifecycleIntentRetry | null>(null);
  useEffect(() => { setQueued(null); setRetry(null); }, [eventId, changeId]);
  useEffect(() => {
    generation.current += 1; setPreview(null); setBusy(false); setError(null);
    return () => { generation.current += 1; };
  }, [eventId, changeId, canApprove, ticketHolders, assignedCrew]);
  function fail(cause: unknown) {
    if (cause instanceof ApiError && (cause.status === 401 || cause.status === 403)) { setPreview(null); setQueued(null); setRetry(null); onAuthorizationLoss(); }
    if (cause instanceof ApiError && cause.status === 409) { setPreview(null); setRetry(null); }
    setError(cause instanceof ApiError ? cause.message : 'The listing notice could not be updated.');
  }
  async function approve() {
    if (busy || !preview || queued || !canApprove) return;
    const current = ++generation.current; setBusy(true); setError(null);
    const identity = nextLifecycleIntentRetry(retry, JSON.stringify({ eventId, changeId, hash: preview.previewHash, audiences: preview.audiences }), () => crypto.randomUUID()); setRetry(identity);
    try {
      const notice = await postJSON<LifecycleNoticeDTO>(`/api/events/${eventId}/lifecycle-intents/${changeId}/notice`, { requestKey: identity.key, previewHash: preview.previewHash, audiences: preview.audiences });
      if (generation.current === current) { setQueued(notice); setPreview(null); setRetry(null); }
    } catch (cause) { if (generation.current === current) fail(cause); }
    finally { if (generation.current === current) setBusy(false); }
  }
  async function refresh() {
    if (busy) return;
    const current = ++generation.current; setBusy(true); setError(null); setPreview(null);
    try { const notice = await api<LifecycleNoticeDTO>(`/api/events/${eventId}/lifecycle-intents/${changeId}/notice`); if (generation.current === current) setQueued(notice); }
    catch (cause) { if (generation.current === current) fail(cause); }
    finally { if (generation.current === current) setBusy(false); }
  }
  async function review() {
    if (busy || !canApprove || (!ticketHolders && !assignedCrew)) return;
    const current = ++generation.current; setBusy(true); setPreview(null); setError(null);
    try {
      const next = await postJSON<LifecycleNoticePreviewDTO>(`/api/events/${eventId}/lifecycle-intents/${changeId}/notice-preview`, {
        audiences: [...(ticketHolders ? ['ticket_holders'] : []), ...(assignedCrew ? ['assigned_crew'] : [])],
      });
      if (generation.current === current) setPreview(next);
    } catch (cause) {
      if (generation.current !== current) return;
      fail(cause);
    } finally { if (generation.current === current) setBusy(false); }
  }
  return <section className="mt-4 space-y-3 border-t border-zinc-800 pt-4" aria-label="Listing notice preview">
    <h3 className="font-medium">Review a listing-change notice</h3>
    <p className="text-sm text-zinc-400">Save the listing cancellation or reschedule first, then record its exact revision. Review the message and recipients before approving its queue. Messages remain held while sending is disabled.</p>
    {!canApprove && <p className="text-sm text-amber-200">This decision was superseded. Inspect any existing notice outcomes before recording a correction.</p>}
    {!queued && canApprove && <fieldset disabled={busy} className="flex flex-wrap gap-4 text-sm"><legend className="mb-2">Recipients</legend>
      <label><input type="checkbox" checked={ticketHolders} onChange={(e) => setTicketHolders(e.target.checked)} /> Ticket holders</label>
      <label><input type="checkbox" checked={assignedCrew} onChange={(e) => setAssignedCrew(e.target.checked)} /> Assigned crew</label>
    </fieldset>}
    {!queued && canApprove && <button disabled={busy || (!ticketHolders && !assignedCrew)} onClick={() => void review()} className="rounded border border-zinc-600 px-3 py-2 text-sm">Preview listing notice</button>}
    <button disabled={busy} onClick={() => void refresh()} className="ml-3 text-sm underline">{queued ? 'Refresh delivery outcomes' : 'Check queued notice'}</button>
    {busy && <p role="status" className="text-sm text-zinc-400">Updating notice…</p>}
    {error && <p role="alert" className="text-sm text-amber-200">{error}</p>}
    {preview && canApprove && <div className="space-y-3 rounded bg-zinc-900 p-3">
      <p className="font-medium">{preview.subject}</p><pre className="whitespace-pre-wrap font-sans text-sm">{preview.body}</pre>
      <p className="text-sm">{preview.recipients.filter((recipient) => !recipient.suppressed).length} eligible · {preview.recipients.filter((recipient) => recipient.suppressed).length} suppressed</p>
      {preview.recipients.length === 0 ? <p className="text-sm text-zinc-400">No recipients match the selected operational relationships.</p> : <ul className="max-h-64 space-y-1 overflow-auto text-sm">{preview.recipients.map((recipient) => <li key={recipient.email} className="break-all">{recipient.email} · {recipient.sourceType === 'ticket' ? 'ticket holder' : 'assigned crew'}{recipient.suppressed ? ' · suppressed; would be withheld' : ''}</li>)}</ul>}
      <p className="text-xs text-zinc-500">Recipients reflect current relationships. Any later approval must recheck the listing, recipients and suppression.</p>
      <button disabled={busy || preview.recipients.length === 0} onClick={() => void approve()} className="rounded bg-amber-300 px-3 py-2 text-sm text-zinc-950">Approve and queue this notice</button>
    </div>}
    {queued && <div className="space-y-3 rounded bg-zinc-900 p-3" aria-label="Notice delivery outcomes">
      <p className="font-medium">Queued: {queued.subject}</p><pre className="whitespace-pre-wrap font-sans text-sm">{queued.body}</pre>
      <p className="text-sm text-zinc-400">Queue approval is recorded. Provider acceptance and delivery feedback are separate. Pending retries or quarantine may retain unknown earlier acceptance; suppression blocks further attempts.</p>
      <ul className="max-h-64 space-y-1 overflow-auto text-sm">{queued.recipients.map((recipient) => <li key={recipient.email} className="break-all">{recipient.email} · {recipient.status.replaceAll('_', ' ')} · attempts {recipient.attempts} · feedback {recipient.feedback}</li>)}</ul>
    </div>}
  </section>;
}
