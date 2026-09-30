import { useEffect, useRef, useState } from 'react';
import { ApiError, postJSON } from '../api';
import type { LifecycleNoticePreviewDTO } from '../domain';

export function LifecycleNoticePreview({ eventId, changeId, onAuthorizationLoss }: { eventId: string; changeId: string; onAuthorizationLoss: () => void }) {
  const generation = useRef(0);
  const [ticketHolders, setTicketHolders] = useState(true);
  const [assignedCrew, setAssignedCrew] = useState(false);
  const [preview, setPreview] = useState<LifecycleNoticePreviewDTO | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    generation.current += 1; setPreview(null); setBusy(false); setError(null);
    return () => { generation.current += 1; };
  }, [eventId, changeId, ticketHolders, assignedCrew]);
  async function review() {
    if (busy || (!ticketHolders && !assignedCrew)) return;
    const current = ++generation.current; setBusy(true); setPreview(null); setError(null);
    try {
      const next = await postJSON<LifecycleNoticePreviewDTO>(`/api/events/${eventId}/lifecycle-intents/${changeId}/notice-preview`, {
        audiences: [...(ticketHolders ? ['ticket_holders'] : []), ...(assignedCrew ? ['assigned_crew'] : [])],
      });
      if (generation.current === current) setPreview(next);
    } catch (cause) {
      if (generation.current !== current) return;
      if (cause instanceof ApiError && (cause.status === 401 || cause.status === 403)) { setPreview(null); onAuthorizationLoss(); }
      setError(cause instanceof ApiError ? cause.message : 'The notice preview could not be loaded.');
    } finally { if (generation.current === current) setBusy(false); }
  }
  return <section className="mt-4 space-y-3 border-t border-zinc-800 pt-4" aria-label="Listing notice preview">
    <h3 className="font-medium">Review a listing-change notice</h3>
    <p className="text-sm text-zinc-400">Save the listing cancellation or reschedule first, then record its exact revision. Preview the message and operational recipients here. Previewing sends nothing; approval and queuing are not yet available.</p>
    <fieldset disabled={busy} className="flex flex-wrap gap-4 text-sm"><legend className="mb-2">Recipients</legend>
      <label><input type="checkbox" checked={ticketHolders} onChange={(e) => setTicketHolders(e.target.checked)} /> Ticket holders</label>
      <label><input type="checkbox" checked={assignedCrew} onChange={(e) => setAssignedCrew(e.target.checked)} /> Assigned crew</label>
    </fieldset>
    <button disabled={busy || (!ticketHolders && !assignedCrew)} onClick={() => void review()} className="rounded border border-zinc-600 px-3 py-2 text-sm">{busy ? 'Loading preview…' : 'Preview listing notice'}</button>
    {error && <p role="alert" className="text-sm text-amber-200">{error}</p>}
    {preview && <div className="space-y-3 rounded bg-zinc-900 p-3">
      <p className="font-medium">{preview.subject}</p><pre className="whitespace-pre-wrap font-sans text-sm">{preview.body}</pre>
      <p className="text-sm">{preview.recipients.filter((recipient) => !recipient.suppressed).length} eligible · {preview.recipients.filter((recipient) => recipient.suppressed).length} suppressed</p>
      {preview.recipients.length === 0 ? <p className="text-sm text-zinc-400">No recipients match the selected operational relationships.</p> : <ul className="max-h-64 space-y-1 overflow-auto text-sm">{preview.recipients.map((recipient) => <li key={recipient.email} className="break-all">{recipient.email} · {recipient.sourceType === 'ticket' ? 'ticket holder' : 'assigned crew'}{recipient.suppressed ? ' · suppressed; would be withheld' : ''}</li>)}</ul>}
      <p className="text-xs text-zinc-500">Recipients reflect current relationships. Any later approval must recheck the listing, recipients and suppression.</p>
    </div>}
  </section>;
}
