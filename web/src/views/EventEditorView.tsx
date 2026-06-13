import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { api, patchJSON, postJSON } from '../api';
import type { EventDTO, EventReportDTO } from '../domain';

type FormState = {
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  ticketAllocation: string;
};

function isNewEvent(eventId: string) {
  return eventId === '' || eventId === 'new';
}

function getWorkspaceId() {
  if (typeof window === 'undefined') {
    return '';
  }

  return new URLSearchParams(window.location.search).get('workspaceId') ?? '';
}

function toInputValue(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }

  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}

function fromInputValue(value: string) {
  return new Date(value).toISOString();
}

function emptyForm(): FormState {
  return {
    title: '',
    startsAt: '',
    publicDescription: '',
    locationDisplay: '',
    ticketAllocation: '1',
  };
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function EventEditorView({ eventId }: { eventId: string }) {
  const creating = isNewEvent(eventId);
  const workspaceId = useMemo(getWorkspaceId, []);
  const [event, setEvent] = useState<EventDTO | null>(null);
  const [report, setReport] = useState<EventReportDTO | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [loading, setLoading] = useState(!creating);
  const [saving, setSaving] = useState(false);
  const [actioning, setActioning] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (creating) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const loaded = await api<EventDTO>(`/api/events/${eventId}`);
        if (cancelled) return;
        setEvent(loaded);
        setForm({
          title: loaded.title,
          startsAt: toInputValue(loaded.startsAt),
          publicDescription: loaded.publicDescription,
          locationDisplay: loaded.locationDisplay,
          ticketAllocation: String(loaded.ticketAllocation),
        });

        if (loaded.status === 'end_of_night') {
          const loadedReport = await api<EventReportDTO>(`/api/events/${eventId}/report`).catch(() => null);
          if (!cancelled && loadedReport) {
            setReport(loadedReport);
          }
        }
      } catch (caught) {
        if (!cancelled) {
          setError(caught instanceof Error ? caught.message : 'Unable to load event');
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
  }, [creating, eventId]);

  async function persist() {
    const payload = {
      title: form.title.trim(),
      startsAt: fromInputValue(form.startsAt),
      publicDescription: form.publicDescription.trim(),
      locationDisplay: form.locationDisplay.trim(),
      ticketAllocation: Number(form.ticketAllocation),
    };

    if (creating) {
      if (!workspaceId) {
        throw new Error('workspaceId is required to create an event');
      }

      const created = await postJSON<EventDTO>(`/api/workspaces/${workspaceId}/events`, payload);
      window.location.href = `/events/${created.id}`;
      return;
    }

    const updated = await patchJSON<EventDTO>(`/api/events/${eventId}`, payload);
    setEvent(updated);
    setMessage('Saved');
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setMessage(null);
    setError(null);

    try {
      await persist();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to save event');
    } finally {
      setSaving(false);
    }
  }

  async function handlePublish() {
    if (!event) return;
    setActioning(true);
    setMessage(null);
    setError(null);

    try {
      const published = await postJSON<EventDTO>(`/api/events/${event.id}/publish`, {});
      setEvent(published);
      setMessage('Published');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to publish event');
    } finally {
      setActioning(false);
    }
  }

  async function handleEndOfNight() {
    if (!event) return;
    setActioning(true);
    setMessage(null);
    setError(null);

    try {
      const closed = await postJSON<EventDTO>(`/api/events/${event.id}/end-of-night`, {});
      setEvent(closed);
      const loadedReport = await api<EventReportDTO>(`/api/events/${event.id}/report`).catch(() => null);
      setReport(loadedReport);
      setMessage('End of night complete');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to end event');
    } finally {
      setActioning(false);
    }
  }

  const effective = event ?? null;

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-4xl space-y-6">
        <header className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Event editor</p>
              <h1 className="mt-2 text-3xl font-semibold tracking-tight text-white">{creating ? 'New event' : effective?.title ?? 'Loading event'}</h1>
              <p className="mt-2 text-sm leading-6 text-zinc-400">
                Free ticket event, direct-link public page, and mobile Door check-in.
              </p>
            </div>

            <div className="flex flex-wrap gap-2 text-sm">
              {effective?.publicUrl ? (
                <a className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href={effective.publicUrl}>
                  Public page
                </a>
              ) : null}
              {effective ? (
                <a className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href={`/door/${effective.id}`}>
                  Door
                </a>
              ) : null}
              <a className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href="/">
                Back
              </a>
            </div>
          </div>

          {effective ? (
            <div className="mt-6 grid gap-3 sm:grid-cols-4">
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Status</p>
                <p className="mt-2 text-sm font-medium text-white">{effective.status}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Reserved</p>
                <p className="mt-2 text-sm font-medium text-white">{effective.reservedCount}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Checked in</p>
                <p className="mt-2 text-sm font-medium text-white">{effective.checkedInCount}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Allocation</p>
                <p className="mt-2 text-sm font-medium text-white">{effective.ticketAllocation}</p>
              </div>
            </div>
          ) : null}
        </header>

        {loading ? <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Loading event…</div> : null}
        {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
        {message ? <p className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">{message}</p> : null}

        {!loading ? (
          <div className="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
            <form className="space-y-4 rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleSubmit}>
              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Title</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  value={form.title}
                  onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))}
                  required
                />
              </label>

              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Starts at</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  type="datetime-local"
                  value={form.startsAt}
                  onChange={(event) => setForm((current) => ({ ...current, startsAt: event.target.value }))}
                  required
                />
              </label>

              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Location</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  value={form.locationDisplay}
                  onChange={(event) => setForm((current) => ({ ...current, locationDisplay: event.target.value }))}
                  required
                />
              </label>

              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Public description</span>
                <textarea
                  className="min-h-40 w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  value={form.publicDescription}
                  onChange={(event) => setForm((current) => ({ ...current, publicDescription: event.target.value }))}
                  required
                />
              </label>

              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Ticket allocation</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  type="number"
                  min="1"
                  step="1"
                  value={form.ticketAllocation}
                  onChange={(event) => setForm((current) => ({ ...current, ticketAllocation: event.target.value }))}
                  required
                />
              </label>

              <button className="w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60" type="submit" disabled={saving}>
                {saving ? 'Saving…' : creating ? 'Create event' : 'Save event'}
              </button>
            </form>

            <aside className="space-y-6">
              <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Actions</p>
                <div className="mt-4 flex flex-col gap-3">
                  {!creating && effective?.status === 'draft' ? (
                    <button className="door-action rounded-2xl bg-white px-4 py-3 text-left font-medium text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70" type="button" onClick={handlePublish} disabled={actioning}>
                      Publish public page
                    </button>
                  ) : null}

                  {!creating && effective?.status === 'published' ? (
                    <button className="door-action rounded-2xl bg-white px-4 py-3 text-left font-medium text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70" type="button" onClick={handleEndOfNight} disabled={actioning}>
                      End of night
                    </button>
                  ) : null}

                  {effective?.publicUrl ? (
                    <a className="door-action rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-left font-medium text-zinc-100 transition hover:bg-white/10" href={effective.publicUrl}>
                      Open public URL
                    </a>
                  ) : null}

                  {effective ? (
                    <a className="door-action rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-left font-medium text-zinc-100 transition hover:bg-white/10" href={`/door/${effective.id}`}>
                      Open Door
                    </a>
                  ) : null}
                </div>
              </section>

              {report ? (
                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Report summary</p>
                  <h2 className="mt-2 text-2xl font-semibold text-white">{report.title}</h2>
                  <p className="mt-2 text-sm text-zinc-400">Generated {formatDateTime(report.generatedAt)} by {report.generatedByMemberEmail}</p>
                  <div className="mt-4 grid gap-3 sm:grid-cols-2">
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Reserved</p>
                      <p className="mt-2 text-lg font-semibold text-white">{report.ticketsReserved}</p>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Checked in</p>
                      <p className="mt-2 text-lg font-semibold text-white">{report.ticketsCheckedIn}</p>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">No-shows</p>
                      <p className="mt-2 text-lg font-semibold text-white">{report.noShows}</p>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Allocation</p>
                      <p className="mt-2 text-lg font-semibold text-white">{report.ticketAllocation}</p>
                    </div>
                  </div>
                </section>
              ) : null}
            </aside>
          </div>
        ) : null}
      </section>
    </main>
  );
}
