import type { WorkspaceDTO, WorkspaceSummaryDTO } from '../domain';
import { roleLabel } from '../modules/workspace/memberAuthority';

// Shown instead of the operator home when the signed-in person is a
// participant in the selected workspace. Participants have no operator
// access, so this page loads nothing beyond the workspace name and role.
export function ParticipantWorkspaceHome({ workspace, workspaces, email }: {
  workspace: WorkspaceDTO;
  workspaces: WorkspaceSummaryDTO[];
  email: string | undefined;
}) {
  const others = workspaces.filter((summary) => summary.id !== workspace.id);
  return (
    <section className="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
      <div className="rounded-panel border border-stroke-subtle bg-surface-panel p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Current workspace</p>
            <h2 className="mt-2 text-3xl font-bold text-fg-primary">{workspace.name}</h2>
            <p className="mt-2 text-sm text-fg-secondary">You are signed in as {email ?? 'a participant'}.</p>
          </div>
          <span className="border border-stroke-subtle bg-surface-inset px-3 py-1 text-xs uppercase tracking-[0.2em] text-fg-secondary">{roleLabel(workspace.role)}</span>
        </div>
        <p className="mt-4 max-w-2xl text-sm leading-6 text-fg-secondary">
          Your shifts, the open places you can claim and your calendar download are in your participant portal.
          Event planning, crew lists and notes stay with the organizers.
        </p>
        <a className="btn-primary mt-4 inline-flex px-4" href="/participant">Open my shifts</a>
      </div>
      {others.length > 0 ? (
        <aside className="rounded-panel border border-stroke-subtle bg-surface-panel p-6">
          <p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Other workspaces</p>
          <div className="mt-4 space-y-2">
            {others.map((summary) => (
              <a key={summary.id} className="flex items-center justify-between gap-3 rounded-2xl border border-stroke-subtle bg-surface-inset px-4 py-3 hover:bg-surface-panel" href={`/workspace?workspaceId=${summary.id}`}>
                <span className="min-w-0 truncate text-fg-primary">{summary.name}</span>
                <span className="shrink-0 text-xs uppercase tracking-[0.2em] text-fg-muted">{roleLabel(summary.role)}</span>
              </a>
            ))}
          </div>
        </aside>
      ) : null}
    </section>
  );
}
