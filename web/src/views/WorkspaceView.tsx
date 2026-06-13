import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type {
  CurrentUserDTO,
  CurrentWorkspaceDTO,
  DevEmailOutboxMessageDTO,
  EventDTO,
  EventStatus,
  InvitationCreatedDTO,
  InvitationDTO,
  MemberDTO,
} from '../domain';

type CurrentWorkspaceResponse = Omit<CurrentWorkspaceDTO, 'members' | 'invitations'> & {
  members?: MemberDTO[] | null;
  invitations?: InvitationDTO[] | null;
};

export function normalizeCurrentWorkspace(workspace: CurrentWorkspaceResponse): CurrentWorkspaceDTO {
  return {
    ...workspace,
    members: workspace.members ?? [],
    invitations: workspace.invitations ?? [],
  };
}

function signOut() {
  void postJSON('/api/auth/logout', {}).finally(() => {
    window.location.href = '/login';
  });
}

function roleLabel(role: string) {
  return role === 'owner' ? 'Owner' : 'Member';
}

function roleHint(role: string) {
  return role === 'owner' ? 'Can invite members and publish events' : 'Can help run the room';
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function formatShortDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date);
}

function eventStatusLabel(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'Draft';
    case 'published':
      return 'Live';
    case 'end_of_night':
      return 'Closed';
  }
}

function eventStatusTone(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'border-amber-400/25 bg-amber-400/10 text-amber-200';
    case 'published':
      return 'border-emerald-400/25 bg-emerald-400/10 text-emerald-200';
    case 'end_of_night':
      return 'border-fuchsia-400/25 bg-fuchsia-400/10 text-fuchsia-200';
  }
}

function eventStatusSurface(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'border-amber-400/20 bg-amber-400/[0.06]';
    case 'published':
      return 'border-emerald-400/20 bg-emerald-400/[0.06]';
    case 'end_of_night':
      return 'border-fuchsia-400/20 bg-fuchsia-400/[0.06]';
  }
}

function eventStatusSummary(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'Keep shaping the page, then publish when it is ready.';
    case 'published':
      return 'Live now. Keep the Door open and wrap when the room closes.';
    case 'end_of_night':
      return 'Closed out. Review the report and prep the next one.';
  }
}

function eventCountLabel(event: EventDTO) {
  return `Reserved ${event.reservedCount} / Checked in ${event.checkedInCount}`;
}

type OperatorAction = {
  label: string;
  href: string;
  variant: 'primary' | 'secondary' | 'ghost';
};

type OperatorGuidance = {
  eyebrow: string;
  title: string;
  body: string;
  tone: 'amber' | 'emerald' | 'fuchsia' | 'zinc';
  actions: OperatorAction[];
};

function buildOperatorGuidance(events: EventDTO[], workspaceId: string): OperatorGuidance {
  const newestDraft = events.find((event) => event.status === 'draft') ?? null;
  const newestPublished = events.find((event) => event.status === 'published') ?? null;
  const newestClosed = events.find((event) => event.status === 'end_of_night') ?? null;

  if (events.length === 0) {
    return {
      eyebrow: 'Start here',
      title: 'Create the first event',
      body: 'Shape the room, publish the page, and get the door ready for the first wave of guests.',
      tone: 'amber',
      actions: [
        { label: 'Create event', href: `/events/new?workspaceId=${workspaceId}`, variant: 'primary' },
        { label: 'Invite member', href: '#invite-member', variant: 'secondary' },
      ],
    };
  }

  if (newestDraft) {
    return {
      eyebrow: 'Draft ready',
      title: `Finish ${newestDraft.title}`,
      body: 'Polish the checklist, then publish when the page feels right.',
      tone: 'amber',
      actions: [
        { label: 'Continue editing', href: `/events/${newestDraft.id}`, variant: 'primary' },
        { label: 'Create next event', href: `/events/new?workspaceId=${workspaceId}`, variant: 'secondary' },
      ],
    };
  }

  if (newestPublished) {
    return {
      eyebrow: 'Live now',
      title: `${newestPublished.title} is on the floor`,
      body: 'Open the Door, share the public page, and end the night when the room quiets down.',
      tone: 'emerald',
      actions: [
        { label: 'Open Door', href: `/door/${newestPublished.id}`, variant: 'primary' },
        { label: 'Share public page', href: newestPublished.publicUrl ?? `/e/${newestPublished.id}`, variant: 'secondary' },
        { label: 'End night', href: `/events/${newestPublished.id}`, variant: 'ghost' },
      ],
    };
  }

  if (newestClosed) {
    return {
      eyebrow: 'Wrapped',
      title: `Review ${newestClosed.title}`,
      body: 'Read the report, reset the room, and set up the next event slice.',
      tone: 'fuchsia',
      actions: [
        { label: 'View report', href: `/events/${newestClosed.id}`, variant: 'primary' },
        { label: 'Create next event', href: `/events/new?workspaceId=${workspaceId}`, variant: 'secondary' },
      ],
    };
  }

  return {
    eyebrow: 'Ready',
    title: 'Run the room from here',
    body: 'Use the workspace to keep the door moving: publish the next event, invite help, and close out cleanly.',
    tone: 'zinc',
    actions: [{ label: 'Create event', href: `/events/new?workspaceId=${workspaceId}`, variant: 'primary' }],
  };
}

function toneSurface(tone: OperatorGuidance['tone']) {
  switch (tone) {
    case 'amber':
      return 'border-amber-400/20 bg-amber-400/[0.07]';
    case 'emerald':
      return 'border-emerald-400/20 bg-emerald-400/[0.07]';
    case 'fuchsia':
      return 'border-fuchsia-400/20 bg-fuchsia-400/[0.07]';
    case 'zinc':
      return 'border-white/10 bg-white/[0.04]';
  }
}

function toneLabel(tone: OperatorGuidance['tone']) {
  switch (tone) {
    case 'amber':
      return 'text-amber-200';
    case 'emerald':
      return 'text-emerald-200';
    case 'fuchsia':
      return 'text-fuchsia-200';
    case 'zinc':
      return 'text-zinc-200';
  }
}

function extractInviteToken(body: string) {
  return body.match(/\/invite\/([A-Za-z0-9_-]+)/)?.[1] ?? null;
}

function extractTicketCode(body: string) {
  return body.match(/\/tickets\/([A-Za-z0-9_-]+)/)?.[1] ?? null;
}

async function loadDevEmailOutbox() {
  try {
    const response = await fetch('/api/dev/email-outbox', { credentials: 'include' });

    if (response.status === 401 || response.status === 404 || !response.ok) {
      return null;
    }

    const data = await response.json().catch(() => null);
    if (!Array.isArray(data)) {
      return null;
    }

    return data as DevEmailOutboxMessageDTO[];
  } catch {
    return null;
  }
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
  const [inviteNotice, setInviteNotice] = useState<string | null>(null);
  const [emailOutbox, setEmailOutbox] = useState<DevEmailOutboxMessageDTO[] | null>(null);

  const workspaceSummaries = useMemo(() => me?.workspaces ?? [], [me]);
  const orderedEvents = useMemo(() => [...events].sort((left, right) => new Date(right.startsAt).getTime() - new Date(left.startsAt).getTime()), [events]);
  const statusCounts = useMemo(
    () =>
      orderedEvents.reduce(
        (accumulator, event) => {
          accumulator[event.status] += 1;
          return accumulator;
        },
        { draft: 0, published: 0, end_of_night: 0 } satisfies Record<EventStatus, number>,
      ),
    [orderedEvents],
  );
  const guidance = useMemo(() => buildOperatorGuidance(orderedEvents, workspace?.id ?? ''), [orderedEvents, workspace?.id]);

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

        const currentWorkspace = await api<CurrentWorkspaceResponse>('/api/workspaces/current').catch(() => null);
        if (cancelled) return;

        if (currentWorkspace) {
          setWorkspace(normalizeCurrentWorkspace(currentWorkspace));
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

  useEffect(() => {
    let cancelled = false;

    async function loadOutbox() {
      if (!workspace) {
        setEmailOutbox(null);
        return;
      }

      const loaded = await loadDevEmailOutbox();
      if (!cancelled) {
        setEmailOutbox(loaded ? [...loaded].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()).slice(0, 5) : null);
      }
    }

    void loadOutbox();

    return () => {
      cancelled = true;
    };
  }, [workspace?.id]);

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

    const email = inviteEmail.trim();
    if (!email) {
      setError('Enter an email address before sending an invite.');
      return;
    }

    setSendingInvite(true);
    setError(null);
    setInviteNotice(null);

    try {
      const created = await postJSON<InvitationCreatedDTO>(`/api/workspaces/${workspace.id}/invitations`, { email });
      setInviteEmail('');
      setWorkspace((current) =>
        current
          ? {
              ...current,
              invitations: [
                ...current.invitations,
                {
                  id: created.id,
                  email: created.email,
                  role: created.role,
                  token: created.token,
                  acceptedAt: null,
                },
              ],
            }
          : current,
      );
      setInviteNotice(`Invite sent to ${created.email}.`);
      const loaded = await loadDevEmailOutbox();
      setEmailOutbox(loaded ? [...loaded].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()).slice(0, 5) : null);
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
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-white">Operator home</h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">
              Run the room from one place: create the next event, invite help, and keep the Door moving.
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
              <div className="space-y-6">
                <div className="grid gap-4 xl:grid-cols-[1.08fr_0.92fr]">
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

                    <div className="mt-6 grid gap-3 sm:grid-cols-3">
                      <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-4">
                        <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Draft</p>
                        <p className="mt-2 text-2xl font-semibold text-white">{statusCounts.draft}</p>
                      </div>
                      <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-4">
                        <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Live</p>
                        <p className="mt-2 text-2xl font-semibold text-white">{statusCounts.published}</p>
                      </div>
                      <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-4">
                        <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Closed</p>
                        <p className="mt-2 text-2xl font-semibold text-white">{statusCounts.end_of_night}</p>
                      </div>
                    </div>
                  </div>

                  <div className={`rounded-[1.75rem] border p-6 shadow-xl shadow-black/20 ${toneSurface(guidance.tone)}`}>
                    <p className={`text-xs uppercase tracking-[0.3em] ${toneLabel(guidance.tone)}`}>{guidance.eyebrow}</p>
                    <h3 className="mt-2 text-2xl font-semibold tracking-tight text-white">{guidance.title}</h3>
                    <p className="mt-3 max-w-xl text-sm leading-6 text-zinc-300">{guidance.body}</p>
                    <div className="mt-6 flex flex-wrap gap-2 text-sm">
                      {guidance.actions.map((action) =>
                        action.variant === 'primary' ? (
                          <a key={action.label} className="rounded-full bg-amber-300 px-4 py-2 font-medium text-zinc-950 transition hover:bg-amber-200" href={action.href}>
                            {action.label}
                          </a>
                        ) : action.variant === 'secondary' ? (
                          <a key={action.label} className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href={action.href}>
                            {action.label}
                          </a>
                        ) : (
                          <a key={action.label} className="rounded-full border border-white/10 bg-black/15 px-4 py-2 text-zinc-200 transition hover:bg-black/30" href={action.href}>
                            {action.label}
                          </a>
                        ),
                      )}
                    </div>
                  </div>
                </div>

                <div className="mt-6 space-y-3">
                  <p className="text-xs uppercase tracking-[0.3em] text-zinc-500">Members</p>
                  <div className="space-y-2">
                    {workspace.members.length === 0 ? (
                      <div className="rounded-2xl border border-dashed border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
                        <p className="font-medium text-white">No members yet</p>
                        <p className="mt-1 leading-6">Invite the first operator and this roster will populate automatically.</p>
                      </div>
                    ) : null}
                    {workspace.members.map((member) => (
                      <div key={member.id} className="flex items-center justify-between gap-3 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm">
                        <div>
                          <p className="font-medium text-white">{member.displayName ?? member.email}</p>
                          <p className="text-zinc-400">{member.email}</p>
                        </div>
                        <div className="text-right">
                          <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[11px] uppercase tracking-[0.25em] text-zinc-300">{roleLabel(member.role)}</span>
                          <p className="mt-2 text-xs leading-5 text-zinc-500">{roleHint(member.role)}</p>
                        </div>
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
                    {events.length === 0 ? (
                      <div className="rounded-2xl border border-dashed border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
                        <p className="font-medium text-white">No events yet</p>
                        <p className="mt-1 leading-6">Create the first event to turn this workspace into a live operator home.</p>
                      </div>
                    ) : null}
                    {orderedEvents.map((event) => (
                      <article key={event.id} className={`rounded-[1.5rem] border p-4 ${eventStatusSurface(event.status)}`}>
                        <div className="flex flex-wrap items-start justify-between gap-3">
                          <div>
                            <h3 className="text-lg font-medium text-white">{event.title}</h3>
                            <p className="mt-1 text-sm text-zinc-400">{formatDateTime(event.startsAt)}</p>
                            <p className="mt-2 text-sm leading-6 text-zinc-300 line-clamp-3">{event.publicDescription}</p>
                          </div>
                          <span className={`rounded-full border px-3 py-1 text-xs uppercase tracking-[0.25em] ${eventStatusTone(event.status)}`}>{eventStatusLabel(event.status)}</span>
                        </div>
                        <p className="mt-3 text-sm font-medium text-zinc-200">{eventStatusSummary(event.status)}</p>
                        <div className="mt-4 rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-zinc-300">{eventCountLabel(event)}</div>
                        <div className="mt-4 flex flex-wrap gap-2 text-sm">
                          <a className="rounded-full bg-white px-3 py-2 font-medium text-zinc-950 transition hover:bg-zinc-200" href={`/events/${event.id}`}>
                            {event.status === 'draft' ? 'Finish draft' : event.status === 'published' ? 'View editor' : 'View report'}
                          </a>
                          {event.status === 'draft' ? (
                            <a className="rounded-full border border-white/10 bg-white/5 px-3 py-2 text-zinc-200 transition hover:bg-white/10" href={`/events/${event.id}`}>
                              Publish checklist
                            </a>
                          ) : null}
                          {event.status === 'published' ? (
                            <>
                              <a className="rounded-full border border-white/10 bg-white/5 px-3 py-2 text-zinc-200 transition hover:bg-white/10" href={`/door/${event.id}`}>
                                Open Door
                              </a>
                              {event.publicUrl ? (
                                <a className="rounded-full border border-white/10 bg-white/5 px-3 py-2 text-zinc-200 transition hover:bg-white/10" href={event.publicUrl}>
                                  Share public page
                                </a>
                              ) : null}
                              <a className="rounded-full border border-white/10 bg-black/15 px-3 py-2 text-zinc-200 transition hover:bg-black/30" href={`/events/${event.id}`}>
                                End night
                              </a>
                            </>
                          ) : null}
                          {event.status === 'end_of_night' ? (
                            <a className="rounded-full border border-white/10 bg-white/5 px-3 py-2 text-zinc-200 transition hover:bg-white/10" href={`/events/new?workspaceId=${workspace.id}`}>
                              Create next event
                            </a>
                          ) : null}
                        </div>
                      </article>
                    ))}
                  </div>
                </div>
              </div>

              <aside className="space-y-6">
                <form id="invite-member" className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleInvite}>
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Invite member</p>
                  <p className="mt-2 text-sm leading-6 text-zinc-400">Send an invite without reloading the page; the new row appears below as soon as it lands.</p>
                  <label className="mt-4 block space-y-2 text-sm">
                    <span className="text-zinc-300">Email</span>
                    <input
                      className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                      type="email"
                      autoComplete="email"
                      required
                      value={inviteEmail}
                      onChange={(event) => {
                        setInviteEmail(event.target.value);
                        setInviteNotice(null);
                      }}
                    />
                  </label>
                  <button className="mt-4 w-full rounded-2xl bg-white px-4 py-3 font-medium text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70" type="submit" disabled={sendingInvite}>
                    {sendingInvite ? 'Sending…' : 'Send invite'}
                  </button>
                  {inviteNotice ? <p className="mt-3 rounded-2xl border border-emerald-400/20 bg-emerald-400/10 px-4 py-3 text-sm text-emerald-200">{inviteNotice}</p> : null}
                </form>

                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Invitations</p>
                  <p className="mt-2 text-sm leading-6 text-zinc-400">Pending vs accepted, with open links when the token is available.</p>

                  <div className="mt-4 space-y-3 text-sm">
                    {workspace.invitations.length === 0 ? (
                      <div className="rounded-2xl border border-dashed border-white/10 bg-white/[0.03] p-4 text-zinc-400">
                        No invitations yet. Send one above to start building the crew list.
                      </div>
                    ) : null}

                    {workspace.invitations.map((invitation) => {
                      const accepted = invitation.acceptedAt !== null;

                      return (
                        <article key={invitation.id} className="rounded-2xl border border-white/10 bg-white/5 p-4">
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <p className="font-medium text-white">{invitation.email}</p>
                              <p className="mt-1 text-zinc-400">{roleLabel(invitation.role)}</p>
                            </div>
                            <span className={`rounded-full border px-3 py-1 text-[11px] uppercase tracking-[0.25em] ${accepted ? 'border-emerald-400/20 bg-emerald-400/10 text-emerald-200' : 'border-amber-400/20 bg-amber-400/10 text-amber-200'}`}>
                              {accepted ? 'Accepted' : 'Pending'}
                            </span>
                          </div>

                          {invitation.acceptedAt ? <p className="mt-3 text-zinc-400">Accepted {formatShortDateTime(invitation.acceptedAt)}</p> : <p className="mt-3 text-zinc-500">Waiting for the invite to be accepted.</p>}

                          {invitation.token ? (
                            <a className="mt-3 inline-flex rounded-full bg-amber-300 px-3 py-2 text-xs font-medium text-zinc-950 transition hover:bg-amber-200" href={`/invite/${invitation.token}`}>
                              Open invite
                            </a>
                          ) : null}
                        </article>
                      );
                    })}
                  </div>
                </section>

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

                {emailOutbox && emailOutbox.length > 0 ? (
                  <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                    <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Dev email outbox</p>
                    <p className="mt-2 text-sm leading-6 text-zinc-400">Development-only mailbox. It surfaces recent invite and ticket emails with quick links when available.</p>

                    <div className="mt-4 space-y-3">
                      {emailOutbox.map((email) => {
                        const inviteToken = extractInviteToken(email.body);
                        const ticketCode = extractTicketCode(email.body);

                        return (
                          <article key={email.id} className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm">
                            <div className="flex items-start justify-between gap-3">
                              <div>
                                <p className="font-medium text-white">{email.subject}</p>
                                <p className="mt-1 text-zinc-400">To {email.recipientEmail}</p>
                              </div>
                              <p className="text-xs uppercase tracking-[0.25em] text-zinc-500">{formatDateTime(email.createdAt)}</p>
                            </div>

                            <p className="mt-3 line-clamp-3 whitespace-pre-wrap text-zinc-300">{email.body}</p>

                            <div className="mt-4 flex flex-wrap gap-2 text-xs uppercase tracking-[0.25em] text-zinc-500">
                              <span>{email.relatedType}</span>
                              <span>{email.relatedId}</span>
                            </div>

                            {inviteToken ? (
                              <a className="mt-4 inline-flex rounded-full bg-amber-300 px-3 py-2 text-xs font-medium text-zinc-950 transition hover:bg-amber-200" href={`/invite/${inviteToken}`}>
                                Open invite
                              </a>
                            ) : null}
                            {ticketCode ? (
                              <a className="mt-4 ml-2 inline-flex rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium text-zinc-200 transition hover:bg-white/10" href={`/tickets/${ticketCode}`}>
                                Open ticket
                              </a>
                            ) : null}
                          </article>
                        );
                      })}
                    </div>
                  </section>
                ) : null}
              </aside>
            </section>

            <p className="px-1 text-xs uppercase tracking-[0.3em] text-zinc-500">Workspace ID {workspaceId}</p>
          </>
        ) : null}
      </section>
    </main>
  );
}
