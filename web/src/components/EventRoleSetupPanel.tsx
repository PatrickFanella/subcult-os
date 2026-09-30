import { useLayoutEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError, postJSON } from '../api';
import type { EventRoleDTO } from '../domain';
import { emptyRoleDraft, rolePayload } from '../modules/eventRoles/roleDraft';
import { Button } from '../ui/Button';
import { Notice } from '../ui/Notice';

const inputClass = 'w-full rounded-control border border-stroke-subtle bg-surface-inset px-4 py-3 text-fg-primary disabled:cursor-not-allowed disabled:opacity-60';

export function EventRoleSetupPanel({ eventId, roles, allowed, onCreated }: {
  eventId: string;
  roles: EventRoleDTO[];
  allowed: boolean;
  onCreated: (role: EventRoleDTO) => void;
}) {
  const [draft, setDraft] = useState(emptyRoleDraft);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState('');
  const [reloadRequired, setReloadRequired] = useState(false);
  const [denied, setDenied] = useState(false);
  const submitting = useRef(false);
  const active = useRef(true);
  const eventRoles = roles.filter(role => role.eventId === eventId);
  useLayoutEffect(() => {
    active.current = true;
    return () => { active.current = false; };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!active.current || !allowed || denied || reloadRequired || submitting.current) return;
    let payload;
    try { payload = rolePayload(draft); }
    catch (caught) { setError(caught instanceof Error ? caught.message : 'Check the role details.'); return; }
    submitting.current = true;
    setBusy(true);
    setError(null);
    setNotice('');
    try {
      const created = await postJSON<EventRoleDTO>(`/api/events/${eventId}/roles`, payload);
      if (!active.current) return;
      if (created.eventId !== eventId || !created.id) throw new Error('Role response did not match this event.');
      onCreated(created);
      setDraft(emptyRoleDraft());
      setNotice(`${created.name} added as a ${created.public ? 'public application' : 'private'} role.`);
    } catch (caught) {
      if (!active.current) return;
      if (caught instanceof ApiError && (caught.status === 401 || caught.status === 403)) {
        setDenied(true);
        setDraft(emptyRoleDraft());
        setError('Role setup access is unavailable. Reload the event to check your access.');
      } else if (caught instanceof ApiError && caught.status === 400) {
        setError(caught.message);
      } else {
        // Creation is not idempotent. Inspect the saved list before another attempt.
        setReloadRequired(true);
        setError('The save outcome is uncertain. Reload the event and inspect its roles before creating another.');
      }
    } finally {
      if (active.current) { submitting.current = false; setBusy(false); }
    }
  }

  return (
    <section className="min-w-0 rounded-panel border border-stroke-subtle bg-surface-panel p-6 [overflow-wrap:anywhere]">
      <p className="text-xs uppercase tracking-[0.3em] text-fg-muted">Participation roles</p>
      <h2 className="mt-2 text-2xl font-extrabold text-fg-primary">Set up applications</h2>
      <p className="mt-2 text-sm leading-6 text-fg-secondary">Public roles accept interest through the published event page. Reviewing an application and assigning a task or shift are separate steps.</p>
      {!denied ? (
        <ul className="mt-4 space-y-3" aria-label="Event participation roles">
          {eventRoles.map(role => (
            <li key={role.id} className="rounded-card border border-stroke-subtle bg-surface-inset p-4">
              <p className="font-bold text-fg-primary">{role.name}</p>
              <p className="mt-1 whitespace-pre-wrap text-sm text-fg-secondary">{role.description || 'No description provided.'}</p>
              <p className="mt-2 text-sm text-fg-muted">{role.public ? 'Public applications' : 'Private role'} · {role.capacity === 0 ? 'No capacity limit' : `${role.capacity} places`} · {role.active ? 'Active' : 'Inactive'}</p>
            </li>
          ))}
        </ul>
      ) : null}
      {!denied && eventRoles.length === 0 ? <p className="mt-4 text-sm text-fg-secondary">No participation roles yet.</p> : null}
      {allowed && !denied ? (
        <form className="mt-5 space-y-4" onSubmit={submit}>
          <fieldset className="space-y-4" disabled={busy || reloadRequired}>
            <label className="block space-y-2 text-sm font-semibold text-fg-primary">Role name<input className={inputClass} value={draft.name} required onChange={e => setDraft(current => ({ ...current, name: e.target.value }))} /></label>
            <label className="block space-y-2 text-sm font-semibold text-fg-primary">Role description<textarea className={`${inputClass} min-h-28`} value={draft.description} onChange={e => setDraft(current => ({ ...current, description: e.target.value }))} /></label>
            <p className="text-sm text-fg-muted">Use participant-facing details. Keep operator notes in the staffing board. Descriptions can contain up to 2000 characters.</p>
            <label className="block space-y-2 text-sm font-semibold text-fg-primary">Role capacity<input className={inputClass} type="number" min="0" max="2147483647" step="1" value={draft.capacity} onChange={e => setDraft(current => ({ ...current, capacity: e.target.value }))} /></label>
            <p className="text-sm text-fg-muted">Use 0 for no capacity limit.</p>
            <label className="flex items-start gap-3 text-sm text-fg-primary"><input className="mt-1" type="checkbox" checked={draft.public} onChange={e => setDraft(current => ({ ...current, public: e.target.checked }))} /><span>Accept public applications<span className="mt-1 block text-fg-secondary">The role and description appear on the public page when this event is published.</span></span></label>
            <Button type="submit" busy={busy}>{busy ? 'Adding role…' : draft.public ? 'Add public application role' : 'Add private role'}</Button>
          </fieldset>
        </form>
      ) : !denied ? <p className="mt-4 text-sm text-fg-secondary">Only an owner can add roles while the event is open.</p> : null}
      <p role="status" aria-atomic="true" className="mt-3 text-sm text-status-success">{notice}</p>
      {error ? <Notice tone="danger">{error}</Notice> : null}
      {denied || reloadRequired ? <Button className="mt-3" variant="secondary" onClick={() => window.location.reload()}>Reload event</Button> : null}
    </section>
  );
}
