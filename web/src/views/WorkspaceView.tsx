import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type { CurrentUserDTO, CurrentWorkspaceDTO, EventDTO } from '../domain';

function signOut() {
  void postJSON('/api/auth/logout', {}).finally(() => {
    window.location.href = '/login';
  });
}

function roleLabel(role: string) {
  return role === 'owner' ? 'Owner' : 'Member';
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function WorkspaceView() {
  const [me, setMe] = useState<CurrentUserDTO | null>(null);
  const [workspace, setWorkspace] = useState<CurrentWorkspaceDTO | null>(null);
  const [events, setEvents] = useState<EventDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [workspaceName, setWorkspaceName] = useState('');
  const [creatingWorkspace, setCreatingWorkspace] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [sendingInvite, setSendingInvite] = useState(false);

  const workspaceSummaries = useMemo(() => me?.workspaces ?? [], [me]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);

      try {
        const user = await api<CurrentUserDTO>('/api/me');
        if (cancelled) return;
        setMe(user);

        if (user.workspaces.length === 0) {
          setWorkspace(null);
          setEvents([]);
          return;
        }

        const currentWorkspace = await api<CurrentWorkspaceDTO>('/api/workspaces/current').catch(() => null);
        if (cancelled) return;

        if (currentWorkspace) {
          setWorkspace({
            ...currentWorkspace,
            members: currentWorkspace.members ?? [],
            invitations: currentWorkspace.invitations ?? [],
          });
          const loadedEvents = await api<EventDTO[]>(`/api/workspaces/${currentWorkspace.id}/events`).catch(() => []);
          if (cancelled) return;
          setEvents(loadedEvents ?? []);
        } else {
          const fallback = user.workspaces[0];
          const fallbackWorkspace: CurrentWorkspaceDTO = {
            id: fallback.id,
            name: fallback.name,
            role: fallback.role,
            members: [],
            invitations: [],
          };
          setWorkspace(fallbackWorkspace);
          const loadedEvents = await api<EventDTO[]>(`/api/workspaces/${fallbackWorkspace.id}/events`).catch(() => []);
          if (cancelled) return;
          setEvents(loadedEvents);
        }
      } catch (caught) {
        if (!cancelled) {
          setError(caught instanceof Error ? caught.message : 'Unable to load workspace');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCreateWorkspace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreatingWorkspace(true);
    setError(null);

    try {
      await postJSON('/api/workspaces', { name: workspaceName.trim() });
      window.location.reload();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to create workspace');
    } finally {
      setCreatingWorkspace(false);
    }
  }

  async function handleInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace) return;

    setSendingInvite(true);
    setError(null);

    try {
      await postJSON(`/api/workspaces/${workspace.id}/invitations`, { email: inviteEmail.trim() });
      setInviteEmail('');
      window.location.reload();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to send invite');
    } finally {
      setSendingInvite(false);
    }
  }

  const workspaceId = workspace?.id ?? '';

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-6xl space-y-6">
        <header className="flex flex-col gap-4 rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">subcult-os</p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-white">Workspace</h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">
              Create a workspace, invite a member, and launch the first event slice.
            </p>
          </div>

          <div className="flex flex-wrap gap-2 text-sm">
            <a className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href="/login">
              Auth
            </a>
            <button className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" type="button" onClick={signOut}>
              Sign out
            </button>
          </div>
        </header>

        {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}

        {loading ? (
          <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Loading workspace…</div>
        ) : me && me.workspaces.length === 0 ? (
          <section className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
            <div className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
              <p className="text-xs uppercase tracking-[0.3em] text-amber-300">No workspace yet</p>
              <h2 className="mt-2 text-2xl font-semibold text-white">Create one to start</h2>
              <p className="mt-2 text-sm leading-6 text-zinc-400">You need a workspace before you can invite members or publish events.</p>
            </div>

            <form className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleCreateWorkspace}>
              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Workspace name</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  value={workspaceName}
                  onChange={(event) => setWorkspaceName(event.target.value)}
                  required
                />
              </label>
              <button className="mt-4 w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60" type="submit" disabled={creatingWorkspace}>
                {creatingWorkspace ? 'Creating…' : 'Create workspace'}
              </button>
            </form>
          </section>
        ) : workspace ? (
          <>
            <section className="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
              <div className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-xl shadow-black/30">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Current workspace</p>
                    <h2 className="mt-2 text-3xl font-semibold tracking-tight text-white">{workspace.name}</h2>
                    <p className="mt-2 text-sm text-zinc-400">You are signed in as {me?.email ?? 'a member'}.</p>
                  </div>
                  <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.25em] text-zinc-300">{roleLabel(workspace.role)}</span>
                </div>

                <div className="mt-6 grid gap-3 sm:grid-cols-3">
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Members</p>
                    <p className="mt-2 text-2xl font-semibold text-white">{workspace.members.length}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Invites</p>
                    <p className="mt-2 text-2xl font-semibold text-white">{workspace.invitations.length}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Events</p>
                    <p className="mt-2 text-2xl font-semibold text-white">{events.length}</p>
                  </div>
                </div>

                <div className="mt-6 space-y-3">
                  <p className="text-xs uppercase tracking-[0.3em] text-zinc-500">Members</p>
                  <div className="space-y-2">
                    {workspace.members.length === 0 ? <p className="text-sm text-zinc-400">No member rows returned yet.</p> : null}
                    {workspace.members.map((member) => (
                      <div key={member.id} className="flex items-center justify-between rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm">
                        <div>
                          <p className="font-medium text-white">{member.displayName ?? member.email}</p>
                          <p className="text-zinc-400">{member.email}</p>
                        </div>
                        <span className="text-xs uppercase tracking-[0.25em] text-zinc-400">{roleLabel(member.role)}</span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="mt-6 space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="text-xs uppercase tracking-[0.3em] text-zinc-500">Events</p>
                    <a className="rounded-full bg-amber-300 px-3 py-2 text-xs font-medium uppercase tracking-[0.25em] text-zinc-950 transition hover:bg-amber-200" href={`/events/new?workspaceId=${workspace.id}`}>
                      New event
                    </a>
                  </div>

                  <div className="space-y-3">
                    {events.length === 0 ? <p className="text-sm text-zinc-400">No events yet.</p> : null}
                    {events.map((event) => (
                      <article key={event.id} className="rounded-2xl border border-white/10 bg-white/5 p-4">
                        <div className="flex flex-wrap items-start justify-between gap-3">
                          <div>
                            <h3 className="text-lg font-medium text-white">{event.title}</h3>
                            <p className="mt-1 text-sm text-zinc-400">{formatDateTime(event.startsAt)}</p>
                            <p className="mt-2 text-sm leading-6 text-zinc-300 line-clamp-3">{event.publicDescription}</p>
                          </div>
                          <span className="rounded-full border border-white/10 bg-zinc-900 px-3 py-1 text-xs uppercase tracking-[0.25em] text-zinc-400">{event.status}</span>
                        </div>
                        <div className="mt-4 flex flex-wrap gap-2 text-sm">
                          <a className="rounded-full bg-white px-3 py-2 font-medium text-zinc-950 transition hover:bg-zinc-200" href={`/events/${event.id}`}>
                            Edit
                          </a>
                          <a className="rounded-full border border-white/10 bg-white/5 px-3 py-2 text-zinc-200 transition hover:bg-white/10" href={`/door/${event.id}`}>
                            Door
                          </a>
                          {event.publicUrl ? (
                            <a className="rounded-full border border-white/10 bg-white/5 px-3 py-2 text-zinc-200 transition hover:bg-white/10" href={event.publicUrl}>
                              Public page
                            </a>
                          ) : null}
                        </div>
                      </article>
                    ))}
                  </div>
                </div>
              </div>

              <aside className="space-y-6">
                <form className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleInvite}>
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Invite member</p>
                  <label className="mt-4 block space-y-2 text-sm">
                    <span className="text-zinc-300">Email</span>
                    <input
                      className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                      type="email"
                      autoComplete="email"
                      required
                      value={inviteEmail}
                      onChange={(event) => setInviteEmail(event.target.value)}
                    />
                  </label>
                  <button className="mt-4 w-full rounded-2xl bg-white px-4 py-3 font-medium text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70" type="submit" disabled={sendingInvite}>
                    {sendingInvite ? 'Sending…' : 'Send invite'}
                  </button>
                </form>

                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Workspace access</p>
                  <div className="mt-4 space-y-2 text-sm text-zinc-400">
                    {workspaceSummaries.map((summary) => (
                      <div key={summary.id} className="flex items-center justify-between rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
                        <span className="text-zinc-200">{summary.name}</span>
                        <span className="text-xs uppercase tracking-[0.25em] text-zinc-500">{roleLabel(summary.role)}</span>
                      </div>
                    ))}
                  </div>
                  <p className="mt-4 text-xs leading-6 text-zinc-500">
                    Create events from the workspace page or jump straight to the Door view once a ticketed event exists.
                  </p>
                </section>
              </aside>
            </section>

            <p className="px-1 text-xs uppercase tracking-[0.3em] text-zinc-500">Workspace ID {workspaceId}</p>
          </>
        ) : null}
      </section>
    </main>
  );
}
