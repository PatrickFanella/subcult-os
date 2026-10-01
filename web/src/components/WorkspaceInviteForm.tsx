import type { FormEvent } from 'react';
import type { WorkspaceRole } from '../domain';
import { Button } from '../ui/Button';

export function WorkspaceInviteForm({ role, email, busy, notice, onEmailChange, onSubmit }: {
  role: WorkspaceRole | undefined;
  email: string;
  busy: boolean;
  notice: string | null;
  onEmailChange: (email: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (role !== 'owner') return null;

  return (
    <form id="invite-member" className="rounded-panel border border-stroke-subtle bg-surface-panel p-6" onSubmit={onSubmit}>
      <p className="text-xs uppercase tracking-[0.3em] text-fg-muted">Invite member</p>
      <p className="mt-2 text-sm leading-6 text-fg-secondary">Create a member invitation and queue its email. The invitation appears below; email delivery is a separate step.</p>
      <label className="mt-4 block space-y-2 text-sm">
        <span className="text-fg-secondary">Email</span>
        <input
          className="field py-3"
          type="email"
          autoComplete="email"
          required
          disabled={busy}
          value={email}
          onChange={(event) => onEmailChange(event.target.value)}
        />
      </label>
      <Button className="mt-4 w-full" type="submit" busy={busy}>
        {busy ? 'Creating…' : 'Create invite'}
      </Button>
      <p role="status" aria-atomic="true" className={notice ? 'mt-3 rounded-control border border-status-success/20 bg-status-surface-success px-4 py-3 text-sm text-status-success' : 'sr-only'}>{notice ?? ''}</p>
    </form>
  );
}
