import { useEffect, useRef, useState } from 'react';
import { ApiError, postJSON } from '../api';
import type { LifecycleNoticeDTO } from '../domain';
import { nextLifecycleIntentRetry, type LifecycleIntentRetry } from '../modules/lifecycleIntents/lifecycleIntentModel';
import { noticeOutcomeGuidance, noticeOutcomeSummary } from '../modules/lifecycleIntents/noticeOutcomes';
import { Button } from '../ui/Button';

export function LifecycleNoticeOutcomes({ eventId, changeId, notice, onUpdate, onAuthorizationLoss }: {
  eventId: string; changeId: string; notice: LifecycleNoticeDTO;
  onUpdate: (notice: LifecycleNoticeDTO) => void; onAuthorizationLoss: () => void;
}) {
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [retry, setRetry] = useState<LifecycleIntentRetry | null>(null);
  const generation = useRef(0);
  useEffect(() => {
    generation.current += 1; setNote(''); setRetry(null); setError(null); setBusy(false);
    return () => { generation.current += 1; };
  }, [eventId, changeId, notice.id]);
  async function saveReview() {
    if (busy || !note.trim()) return;
    const current = ++generation.current;
    const identity = nextLifecycleIntentRetry(retry, JSON.stringify({ eventId, changeId, noticeId: notice.id, note }), () => crypto.randomUUID());
    setRetry(identity); setBusy(true); setError(null);
    try {
      const next = await postJSON<LifecycleNoticeDTO>(`/api/events/${eventId}/lifecycle-intents/${changeId}/notice/reviews`, { requestKey: identity.key, note });
      if (generation.current !== current) return;
      onUpdate(next); setNote(''); setRetry(null);
    } catch (cause) {
      if (generation.current !== current) return;
      if (cause instanceof ApiError && (cause.status === 401 || cause.status === 403)) { setNote(''); setRetry(null); onAuthorizationLoss(); }
      if (cause instanceof ApiError && cause.status === 409) setRetry(null);
      setError(cause instanceof ApiError ? cause.message : 'The review could not be saved. Retry with the same note to recover an uncertain response.');
    } finally { if (generation.current === current) setBusy(false); }
  }
  return <div className="space-y-4 bg-surface-inset p-4" aria-label="Notice delivery outcomes">
    <p className="font-medium">Queued: {notice.subject}</p><pre className="whitespace-pre-wrap font-sans text-sm">{notice.body}</pre>
    <p className="text-sm text-fg-secondary">Queue approval is recorded. Provider acceptance and delivery feedback are separate. Pending retries or quarantine may retain unknown earlier acceptance; suppression blocks further attempts.</p>
    <p className="text-sm font-medium" aria-label="Delivery summary">{noticeOutcomeSummary(notice.recipients)}</p>
    <ul className="max-h-80 space-y-3 overflow-auto text-sm">{notice.recipients.map((recipient) => <li key={recipient.email}>
      <p className="break-all">{recipient.email} · {recipient.status.replaceAll('_', ' ')} · attempts {recipient.attempts} · feedback {recipient.feedback}</p>
      <p className="text-fg-secondary">{noticeOutcomeGuidance(recipient.status, recipient.attempts)}</p>
    </li>)}</ul>
    <section className="space-y-3 border-t border-stroke-subtle pt-4" aria-label="Owner review log">
      <h4 className="font-medium">Owner review log</h4>
      <p className="text-sm text-fg-secondary">Record what you checked and any follow-up you plan. Each note saves the server’s delivery outcomes at that moment. Saving a review does not send, retry, or mark a message delivered.</p>
      {(notice.reviews ?? []).length === 0 ? <p className="text-sm text-fg-muted">No reviews recorded.</p> : <ol className="space-y-3">{notice.reviews.map((review) => <li key={review.id} className="border border-stroke-subtle p-3 text-sm">
        <time dateTime={review.recordedAt} className="text-fg-secondary">{new Date(review.recordedAt).toLocaleString()}</time>
        <p className="whitespace-pre-wrap break-words">{review.note}</p>
        <p className="text-fg-muted">Outcomes when saved: {noticeOutcomeSummary(review.recipients)}</p>
      </li>)}</ol>}
      <form className="space-y-3" onSubmit={(event) => { event.preventDefault(); void saveReview(); }}>
        <label className="block text-sm">Private review note<textarea value={note} maxLength={2000} required disabled={busy} onChange={(event) => setNote(event.target.value)} rows={3} className="field mt-2 block py-3" /></label>
        {error && <p role="alert" className="text-sm text-status-warning">{error}</p>}
        <Button type="submit" variant="secondary" busy={busy} disabled={!note.trim()}>{busy ? 'Saving review…' : 'Save review note'}</Button>
      </form>
    </section>
  </div>;
}
