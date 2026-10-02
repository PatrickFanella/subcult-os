import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError } from '../api';
import type { CurrentWorkspaceDTO, MemberDTO } from '../domain';
import { MembershipAccess } from '../components/MembershipAccess';
import { roleHint, roleLabel } from '../modules/workspace/memberAuthority';
import { assignableMemberRoles, changeWorkspaceMemberRole, editableMemberRole, isAssignableMemberRole, loadMemberRoleWorkspace } from '../modules/workspace/memberRoleWrites';
import type { AssignableMemberRole } from '../modules/workspace/memberRoleWrites';
import { Button } from '../ui/Button';
import { Notice } from '../ui/Notice';

export function MemberRolesView({ workspaceId }: { workspaceId: string }) {
  const [workspace, setWorkspace] = useState<CurrentWorkspaceDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [drafts, setDrafts] = useState<Record<string, AssignableMemberRole>>({});
  const [actioning, setActioning] = useState<string | null>(null);
  const [reloadRequired, setReloadRequired] = useState(false);
  const [notice, setNotice] = useState('');
  const active = useRef(true);
  const pending = useRef(false);
  const blocked = useRef(false);
  const readVersion = useRef(0);
  const currentWorkspace = useRef(workspace);

  useLayoutEffect(() => {
    active.current = true;
    return () => { active.current = false; readVersion.current += 1; };
  }, []);

  async function refresh() {
    const version = ++readVersion.current;
    setLoading(true);
    setError(null);
    try {
      const fresh = await loadMemberRoleWorkspace(workspaceId);
      if (!active.current || version !== readVersion.current) return false;
      currentWorkspace.current = fresh;
      blocked.current = false;
      setWorkspace(fresh);
      setDrafts({});
      setReloadRequired(false);
      return true;
    } catch (caught) {
      if (!active.current || version !== readVersion.current) return false;
      currentWorkspace.current = null;
      blocked.current = true;
      setWorkspace(null);
      setDrafts({});
      setNotice('');
      setError(caught instanceof ApiError && (caught.status === 401 || caught.status === 403)
        ? 'Workspace role access is unavailable. Sign in or refresh to check access.'
        : 'Unable to read this workspace. Refresh before changing any roles.');
      setReloadRequired(true);
      return false;
    } finally {
      if (active.current && version === readVersion.current) setLoading(false);
    }
  }

  useEffect(() => { void refresh(); }, [workspaceId]);

  async function submit(event: FormEvent<HTMLFormElement>, membership: MemberDTO) {
    event.preventDefault();
    const current = currentWorkspace.current;
    const member = current?.members.find(row => row.id === membership.id);
    if (!active.current || pending.current || blocked.current || current?.id !== workspaceId || current.role !== 'owner' || member?.accessState !== 'active') return;
    const configuredRole = editableMemberRole(member.role);
    const selected = drafts[member.id] ?? configuredRole;
    if (configuredRole === null || !selected || !isAssignableMemberRole(selected) || selected === configuredRole) return;
    pending.current = true;
    setActioning(member.id);
    setError(null);
    setNotice('');
    try {
      const receipt = await changeWorkspaceMemberRole(workspaceId, member.id, selected);
      if (!active.current) return;
      setNotice(`${roleLabel(receipt.role)} role saved. Checking current workspace access…`);
      const reloaded = await refresh();
      if (!active.current) return;
      setNotice(reloaded ? 'Role saved. Current workspace access is shown below.' : 'Role saved, but current workspace access could not be refreshed. Refresh to inspect it before another change.');
    } catch (caught) {
      if (!active.current) return;
      if (caught instanceof ApiError && (caught.status === 400 || caught.status === 409)) {
        // A validation/last-owner rejection is known not to have saved the role.
        setError(caught.message);
      } else {
        blocked.current = true;
        setReloadRequired(true);
        if (caught instanceof ApiError && (caught.status === 401 || caught.status === 403)) {
          currentWorkspace.current = null;
          setWorkspace(null);
          setDrafts({});
          setError('Workspace role access is unavailable. Sign in or refresh to check access.');
        } else {
          setError('The role-save outcome is uncertain. Refresh and inspect this workspace before another change.');
        }
      }
    } finally {
      if (active.current) { pending.current = false; setActioning(null); }
    }
  }

  const canManage = workspace?.role === 'owner';
  return <main className="min-h-screen bg-surface-canvas px-4 py-10 text-fg-primary">
    <section className="mx-auto min-w-0 max-w-4xl space-y-6 [overflow-wrap:anywhere]">
      <header className="rounded-panel border border-stroke-subtle bg-surface-panel p-6">
        <h1 className="text-3xl font-bold">Member roles</h1>
        {workspace ? <p className="mt-2 text-fg-secondary">{workspace.name}</p> : null}
        <p className="mt-2 text-sm text-fg-secondary">Owners can assign workspace roles. Review the capabilities before saving. Role changes do not restore expired or revoked access.</p>
        <div className="mt-4 flex flex-wrap gap-3">
          <a className="btn-secondary inline-flex px-4 py-3 text-sm" href={`/workspace?workspaceId=${encodeURIComponent(workspaceId)}`}>Workspace</a>
          <Button variant="secondary" disabled={loading || Boolean(actioning)} onClick={() => { if (!pending.current) { setNotice(''); void refresh(); } }}>Refresh workspace</Button>
          <a className="inline-flex px-4 py-3 text-sm underline" href="/login">Sign in</a>
        </div>
      </header>
      {error ? <Notice tone="danger">{error}</Notice> : null}
      <p role="status" aria-atomic="true" className={notice ? 'rounded-control border border-stroke-subtle bg-surface-panel px-4 py-3 text-sm' : 'sr-only'}>{notice}</p>
      {loading ? <p>Loading current membership access…</p> : workspace ? <>
        {!canManage ? <Notice>Member roles are read-only for your current workspace access.</Notice> : null}
        <p className="text-sm text-fg-muted">Access status reflects the last workspace refresh. Keep at least one active owner.</p>
        {workspace.members.map(member => {
          const configuredRole = editableMemberRole(member.role);
          const selected = drafts[member.id] ?? configuredRole;
          const editable = canManage && member.accessState === 'active' && configuredRole !== null;
          return <article key={member.id} className="min-w-0 rounded-panel border border-stroke-subtle bg-surface-panel p-6">
            <h2 className="text-lg font-semibold">{member.displayName ?? member.email}</h2>
            <p className="text-sm text-fg-secondary">{member.email}</p>
            <p className="mt-2 text-sm">Configured role: {roleLabel(member.role)}</p>
            <MembershipAccess member={member} />
            {editable ? <form className="mt-4" onSubmit={event => { void submit(event, member); }}>
              <fieldset className="space-y-3" disabled={Boolean(actioning) || reloadRequired}>
                <label className="block text-sm">
                  <span>Role to assign</span>
                  <select className="field mt-2 block py-3" value={selected ?? ''} onChange={event => setDrafts(current => ({ ...current, [member.id]: event.target.value as AssignableMemberRole }))}>
                    {assignableMemberRoles.map(role => <option key={role} value={role}>{roleLabel(role)}</option>)}
                  </select>
                </label>
                <p className="text-sm text-fg-secondary">{selected ? roleHint(selected) : 'Choose a role.'}</p>
                <Button type="submit" busy={actioning === member.id} disabled={selected === configuredRole}>Assign {selected ? roleLabel(selected) : ''} role</Button>
              </fieldset>
            </form> : null}
          </article>;
        })}
      </> : null}
    </section>
  </main>;
}
