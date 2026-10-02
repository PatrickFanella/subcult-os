import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError, api, postJSON } from '../api';
import { Button } from '../ui/Button';
import type { PublicArchiveItemDTO } from '../domain';

type ArchiveDraft = {
  kind: PublicArchiveItemDTO['kind']; title: string; attributionName: string; attributionUrl: string; externalUrl: string;
  intendedUse: PublicArchiveItemDTO['intendedUse']; rightsAssertion: PublicArchiveItemDTO['rightsAssertion']; evidenceReference: string;
};
const emptyDraft = (): ArchiveDraft => ({ kind: 'link', title: '', attributionName: '', attributionUrl: '', externalUrl: '', intendedUse: 'link_only', rightsAssertion: 'permission_asserted', evidenceReference: '' });
const archivePayload = (draft: ArchiveDraft) => ({ ...draft, attributionUrl: draft.attributionUrl.trim() || null, externalUrl: draft.externalUrl.trim() || null });

export function PublicArchiveItemsPanel({ eventId }: { eventId: string }) {
  const [items, setItems] = useState<PublicArchiveItemDTO[]>([]);
  const [draft, setDraft] = useState<ArchiveDraft>(emptyDraft);
  const [correctionFor, setCorrectionFor] = useState<PublicArchiveItemDTO | null>(null);
  const [unavailableFor, setUnavailableFor] = useState<PublicArchiveItemDTO | null>(null);
  const [unavailableReason, setUnavailableReason] = useState('');
  const [loadState, setLoadState] = useState<'loading' | 'ready' | 'failed' | 'denied'>('loading');
  const [loadedEventId, setLoadedEventId] = useState<string | null>(null);
  const [reload, setReload] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [needsReload, setNeedsReload] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventGeneration = useRef(0);
  const pendingWrite = useRef(false);

  function clearDrafts() {
    setDraft(emptyDraft()); setCorrectionFor(null); setUnavailableFor(null); setUnavailableReason('');
  }

  useEffect(() => {
    const generation = ++eventGeneration.current;
    pendingWrite.current = false;
    setLoadState('loading'); setError(null); setItems([]); clearDrafts(); setSubmitting(false); setNeedsReload(false);
    api<PublicArchiveItemDTO[]>(`/api/events/${eventId}/public-archive-items`)
      .then((loaded) => {
        if (eventGeneration.current !== generation) return;
        setItems(loaded); setLoadedEventId(eventId); setLoadState('ready');
      })
      .catch((caught) => {
        if (eventGeneration.current !== generation) return;
        setItems([]);
        const denied = caught instanceof ApiError && (caught.status === 401 || caught.status === 403);
        setLoadState(denied ? 'denied' : 'failed');
        setError(denied ? 'Owner access is required to view archive approvals.' : caught instanceof Error ? caught.message : 'Unable to load archive approvals');
      });
    return () => { ++eventGeneration.current; pendingWrite.current = false; };
  }, [eventId, reload]);

  const ready = loadState === 'ready' && loadedEventId === eventId;
  const blocked = !ready || submitting || needsReload;
  const updateDraft = <K extends keyof ArchiveDraft>(key: K, value: ArchiveDraft[K]) => setDraft((current) => ({ ...current, [key]: value }));
  const startCorrection = (item: PublicArchiveItemDTO) => {
    if (blocked) return;
    setCorrectionFor(item); setUnavailableFor(null); setUnavailableReason('');
    setDraft({ kind: item.kind, title: item.title, attributionName: item.attributionName, attributionUrl: item.attributionUrl ?? '', externalUrl: item.externalUrl ?? '', intendedUse: item.intendedUse, rightsAssertion: item.rightsAssertion, evidenceReference: item.evidenceReference });
  };

  // A rejected or unconfirmed write must not be silently replayed: approvals
  // have no idempotency key, so reloading is the reconciliation step.
  async function writeArchive<T>(path: string, payload: unknown, apply: (result: T) => void) {
    if (blocked || pendingWrite.current) return;
    pendingWrite.current = true; setSubmitting(true); setError(null);
    const generation = eventGeneration.current;
    try {
      const result = await postJSON<T>(path, payload);
      if (eventGeneration.current === generation) apply(result);
    } catch (caught) {
      if (eventGeneration.current !== generation) return;
      if (caught instanceof ApiError && (caught.status === 401 || caught.status === 403)) {
        ++eventGeneration.current;
        setItems([]); clearDrafts(); setLoadState('denied'); setNeedsReload(false);
        pendingWrite.current = false; setSubmitting(false);
        setError('Owner access is required to view archive approvals.');
      } else if (caught instanceof ApiError && caught.status === 400) {
        setError(caught.message);
      } else {
        setNeedsReload(true);
        setError('The save was not confirmed. Reload approvals to check the ledger before making another change. Reloading clears this draft.');
      }
    } finally {
      if (eventGeneration.current === generation) {
        pendingWrite.current = false; setSubmitting(false);
      }
    }
  }
  async function submitApproval(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    await writeArchive<PublicArchiveItemDTO>(`/api/events/${eventId}/public-archive-items`, archivePayload(draft), (item) => {
      setItems((current) => [...current, item]); clearDrafts();
    });
  }
  async function submitCorrection(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault(); if (!correctionFor) return;
    const itemID = correctionFor.id;
    await writeArchive<PublicArchiveItemDTO>(`/api/events/${eventId}/public-archive-items/${itemID}/correct`, archivePayload(draft), (replacement) => {
      setItems((current) => current.map((item) => item.id === itemID ? { ...item, status: 'corrected' as const } : item).concat(replacement)); clearDrafts();
    });
  }
  async function markUnavailable() {
    if (!unavailableFor || !unavailableReason.trim()) return;
    const itemID = unavailableFor.id, reason = unavailableReason.trim();
    await writeArchive(`/api/events/${eventId}/public-archive-items/${itemID}/unavailable`, { reason }, () => {
      setItems((current) => current.map((item) => item.id === itemID ? { ...item, status: 'unavailable' as const, unavailableReason: reason } : item)); clearDrafts();
    });
  }
  const reloadApprovals = () => {
    if (pendingWrite.current) return;
    ++eventGeneration.current; setItems([]); clearDrafts(); setLoadState('loading'); setError(null); setReload((current) => current + 1);
  };
  const formTitle = correctionFor ? 'Correct approved item' : 'Approve an archive credit or link';
  const submitLabel = correctionFor ? 'Save correction' : 'Approve for future archive';
  return <main className="min-h-screen px-4 py-6 text-fg-primary sm:px-6 lg:px-8"><section className="mx-auto w-full max-w-3xl space-y-6">
    <header className="rounded-panel border border-stroke-subtle bg-surface-panel p-6"><p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Event archive</p><h1 className="mt-2 text-3xl font-bold text-fg-primary">Approved for a future public archive</h1><p className="mt-3 text-sm leading-6 text-fg-secondary">This is a private owner ledger. Every approval remains unpublished until a separate public archive is built. A rights assertion records what the owner was told; it does not prove rights or authorize copying, uploads, rehosting, or media delivery.</p><a className="btn-secondary mt-4 px-4 text-sm" href={`/events/${eventId}`}>Back to event archive and editor</a></header>
    {error ? <p role="alert" className="rounded-2xl border border-status-danger/20 bg-status-surface-danger p-4 text-sm text-status-danger">{error}</p> : null}
    {loadState === 'loading' ? <p role="status" className="text-sm text-fg-secondary">Loading approvals…</p> : null}
    {loadState === 'failed' || loadState === 'denied' || needsReload ? <Button variant="secondary" onClick={reloadApprovals}>Reload approvals</Button> : null}
    {ready ? <>
    <form aria-busy={submitting || undefined} className="space-y-4 rounded-panel border border-stroke-subtle bg-surface-panel p-6" onSubmit={correctionFor ? submitCorrection : submitApproval}><fieldset disabled={blocked} className="space-y-4"><div><p className="text-xs uppercase tracking-[0.2em] text-fg-muted">{correctionFor ? 'Replacement creates an immutable correction chain' : 'Private, unpublished approval'}</p><h2 className="mt-2 text-xl font-bold text-fg-primary">{formTitle}</h2></div><label className="block text-sm"><span>Item type</span><select className="field mt-2 py-3" value={draft.kind} onChange={(event) => updateDraft('kind', event.target.value as ArchiveDraft['kind'])}><option value="link">External link</option><option value="credit">Display credit</option></select></label><label className="block text-sm"><span>Title</span><input required maxLength={300} className="field mt-2 py-3" value={draft.title} onChange={(event) => updateDraft('title', event.target.value)} /></label><label className="block text-sm"><span>Attribution name</span><input required maxLength={300} className="field mt-2 py-3" value={draft.attributionName} onChange={(event) => updateDraft('attributionName', event.target.value)} /></label><div className="grid gap-4 sm:grid-cols-2"><label className="block text-sm"><span>Attribution URL (optional)</span><input type="url" maxLength={2000} placeholder="https://example.org/artist" className="field mt-2 py-3" value={draft.attributionUrl} onChange={(event) => updateDraft('attributionUrl', event.target.value)} /></label><label className="block text-sm"><span>External URL (optional)</span><input type="url" maxLength={2000} placeholder="https://example.org/work" className="field mt-2 py-3" value={draft.externalUrl} onChange={(event) => updateDraft('externalUrl', event.target.value)} /></label></div><div className="grid gap-4 sm:grid-cols-2"><label className="block text-sm"><span>Intended public use</span><select className="field mt-2 py-3" value={draft.intendedUse} onChange={(event) => updateDraft('intendedUse', event.target.value as ArchiveDraft['intendedUse'])}><option value="link_only">Link only</option><option value="display_credit">Display credit</option></select></label><label className="block text-sm"><span>Rights assertion</span><select className="field mt-2 py-3" value={draft.rightsAssertion} onChange={(event) => updateDraft('rightsAssertion', event.target.value as ArchiveDraft['rightsAssertion'])}><option value="permission_asserted">Permission asserted</option><option value="owned">Owned</option><option value="licensed">Licensed</option><option value="public_domain">Public domain</option></select></label></div><label className="block text-sm"><span>Evidence reference (optional)</span><input maxLength={500} className="field mt-2 py-3" value={draft.evidenceReference} onChange={(event) => updateDraft('evidenceReference', event.target.value)} /></label><div className="flex flex-wrap gap-3"><Button type="submit" busy={submitting} disabled={blocked}>{submitting ? 'Saving…' : submitLabel}</Button>{correctionFor ? <Button variant="secondary" disabled={blocked} onClick={clearDrafts}>Cancel correction</Button> : null}</div></fieldset></form>
    <section className="rounded-panel border border-stroke-subtle bg-surface-panel p-6"><h2 className="text-xl font-bold text-fg-primary">Approval ledger</h2>{items.length === 0 ? <p className="mt-4 text-sm text-fg-secondary">No archive items are approved yet.</p> : <ul className="mt-4 space-y-3">{items.map((item) => <li key={item.id} className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4"><div className="flex flex-wrap items-start justify-between gap-3"><div><p className="break-words font-medium text-fg-primary">{item.title}</p><p className="mt-1 break-words text-sm text-fg-secondary">{item.attributionName} · {item.intendedUse === 'link_only' ? 'Link only' : 'Display credit'} · {item.status}</p>{item.unavailableReason ? <p className="mt-2 break-words text-sm text-status-warning">Unavailable: {item.unavailableReason}</p> : null}</div>{item.status === 'approved' ? <div className="flex gap-2"><Button variant="secondary" disabled={blocked} onClick={() => startCorrection(item)}>Correct</Button><Button variant="secondary" disabled={blocked} onClick={() => { clearDrafts(); setUnavailableFor(item); }}>Unavailable</Button></div> : null}</div></li>)}</ul>}</section>
    {unavailableFor ? <section className="rounded-panel border border-status-warning/20 bg-status-surface-warning p-6"><h2 className="text-xl font-bold text-fg-primary">Mark unavailable</h2><p className="mt-2 text-sm text-fg-secondary">Keep the approval record and record a short public-safe reason. This does not remove or publish anything.</p><label className="mt-4 block text-sm"><span>Reason</span><textarea disabled={blocked} required maxLength={500} className="field mt-2 min-h-28 py-3" value={unavailableReason} onChange={(event) => setUnavailableReason(event.target.value)} /></label><div className="mt-4 flex gap-3"><Button busy={submitting} onClick={() => void markUnavailable()} disabled={blocked || !unavailableReason.trim()}>Mark unavailable</Button><Button variant="secondary" disabled={blocked} onClick={clearDrafts}>Cancel</Button></div></section> : null}
    </> : null}
  </section></main>;
}
