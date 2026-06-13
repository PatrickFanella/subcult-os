import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { api, patchJSON, postJSON } from '../api';
import type { EventDTO, EventReportDTO, EventStatus } from '../domain';

type FormState = {
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  ticketAllocation: string;
  pricingMode: 'free' | 'fixed';
  ticketPriceDollars: string;
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
    pricingMode: 'free',
    ticketPriceDollars: '0.00',
  };
}

function formFromEvent(event: EventDTO): FormState {
  return {
    title: event.title,
    startsAt: toInputValue(event.startsAt),
    publicDescription: event.publicDescription,
    locationDisplay: event.locationDisplay,
    ticketAllocation: String(event.ticketAllocation),
    pricingMode: event.pricingMode,
    ticketPriceDollars: (event.ticketPriceCents / 100).toFixed(2),
  };
}

function formsMatch(left: FormState, right: FormState) {
  return (
    left.title === right.title &&
    left.startsAt === right.startsAt &&
    left.publicDescription === right.publicDescription &&
    left.locationDisplay === right.locationDisplay &&
    left.ticketAllocation === right.ticketAllocation &&
    left.pricingMode === right.pricingMode &&
    left.ticketPriceDollars === right.ticketPriceDollars
  );
}

function formatCurrencyValue(value: number) {
  return new Intl.NumberFormat([], { style: 'currency', currency: 'USD' }).format(value);
}

function priceInCents(value: string) {
  const parsed = Number(value);
  if (Number.isNaN(parsed)) {
    return 0;
  }

  return Math.round(parsed * 100);
}

function pricingSummary(event: EventDTO | null) {
  if (!event || event.pricingMode === 'free') {
    return 'Free reservation';
  }

  return `${formatCurrencyValue(event.ticketPriceCents / 100)} USD`;
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function statusLabel(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'Draft';
    case 'published':
      return 'Published';
    case 'end_of_night':
      return 'End of Night';
  }
}

function statusTone(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'border-amber-400/30 bg-amber-400/10 text-amber-200';
    case 'published':
      return 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200';
    case 'end_of_night':
      return 'border-fuchsia-400/30 bg-fuchsia-400/10 text-fuchsia-200';
  }
}

function statusSummary(status: EventStatus) {
  switch (status) {
    case 'draft':
      return 'Private until the checklist is complete and the public page goes live.';
    case 'published':
      return 'Live now. Keep the public page handy and end the night when the door closes.';
    case 'end_of_night':
      return 'Closed out. Review the report and jump back to the workspace when you are done.';
  }
}

export function EventEditorView({ eventId }: { eventId: string }) {
  const creating = isNewEvent(eventId);
  const workspaceId = useMemo(getWorkspaceId, []);
  const [event, setEvent] = useState<EventDTO | null>(null);
  const [report, setReport] = useState<EventReportDTO | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [initialForm, setInitialForm] = useState<FormState>(emptyForm);
  const [loading, setLoading] = useState(!creating);
  const [saving, setSaving] = useState(false);
  const [actioning, setActioning] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const hasWorkspace = workspaceId !== '';
  const closed = event?.status === 'end_of_night';
  const pricingLocked = (event?.reservedCount ?? 0) > 0 || closed;
  const dirty = useMemo(() => !formsMatch(form, initialForm), [form, initialForm]);

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

        const loadedForm = formFromEvent(loaded);
        setForm(loadedForm);
        setInitialForm(loadedForm);

        if (loaded.status === 'end_of_night') {
          const loadedReport = await api<EventReportDTO>(`/api/events/${eventId}/report`).catch(() => null);
          if (!cancelled && loadedReport) {
            setReport(loadedReport);
          }
        } else {
          setReport(null);
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

  useEffect(() => {
    if (creating) {
      const blank = emptyForm();
      setEvent(null);
      setReport(null);
      setForm(blank);
      setInitialForm(blank);
    }
  }, [creating]);

  async function persist() {
    const ticketPriceCents = form.pricingMode === 'fixed' ? priceInCents(form.ticketPriceDollars) : 0;

    const payload = {
      title: form.title.trim(),
      startsAt: fromInputValue(form.startsAt),
      publicDescription: form.publicDescription.trim(),
      locationDisplay: form.locationDisplay.trim(),
      ticketAllocation: Number(form.ticketAllocation),
      pricingMode: form.pricingMode,
      ticketPriceCents,
      ticketCurrency: 'usd',
    };

    if (creating) {
      if (!hasWorkspace) {
        throw new Error('workspaceId is required to create an event');
      }

      const created = await postJSON<EventDTO>(`/api/workspaces/${workspaceId}/events`, payload);
      window.location.href = `/events/${created.id}`;
      return;
    }

    if (!dirty) {
      setMessage('Nothing to save yet');
      return;
    }

    const updated = await patchJSON<EventDTO>(`/api/events/${eventId}`, payload);
    const updatedForm = formFromEvent(updated);
    setEvent(updated);
    setForm(updatedForm);
    setInitialForm(updatedForm);
    setMessage('Saved');
  }

  async function handleSubmit(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    setSaving(true);
    setMessage(null);
    setError(null);

    try {
      await persist();
    } catch (caught) {
      if (caught instanceof Error && caught.message === 'no changes provided') {
        setMessage('Nothing changed');
      } else {
        setError(caught instanceof Error ? caught.message : 'Unable to save event');
      }
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
      const publishedForm = formFromEvent(published);
      setEvent(published);
      setForm(publishedForm);
      setInitialForm(publishedForm);
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
      const closedReport = await postJSON<EventReportDTO>(`/api/events/${event.id}/end-of-night`, {});
      setReport(closedReport);
      setEvent((current) => (current ? { ...current, status: 'end_of_night' } : current));
      setMessage('End of night complete');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to end event');
    } finally {
      setActioning(false);
    }
  }

  const effective = event ?? null;
  const lifecycleLabel = effective ? statusLabel(effective.status) : creating ? 'Draft' : 'Loading';
  const lifecycleTone = effective ? statusTone(effective.status) : 'border-white/10 bg-white/5 text-zinc-300';
  const lifecycleSummary = effective
    ? statusSummary(effective.status)
    : creating
      ? hasWorkspace
        ? 'Complete the form below to draft the event before publishing.'
        : 'Events are created from a workspace. Open one to start a new event.'
      : 'Loading event details.';

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-4xl space-y-6">
        <header className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Event editor</p>
              <h1 className="mt-2 text-3xl font-semibold tracking-tight text-white">{creating ? 'New event' : effective?.title ?? 'Loading event'}</h1>
              <p className="mt-2 text-sm leading-6 text-zinc-400">Set the public page, ticket pricing, and door flow from one mobile-friendly editor.</p>
            </div>

            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span className={`rounded-full border px-4 py-2 text-xs uppercase tracking-[0.25em] ${lifecycleTone}`}>{lifecycleLabel}</span>
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
              <a className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href="/workspace">
                Workspace
              </a>
            </div>
          </div>

          <p className="mt-4 max-w-2xl text-sm leading-6 text-zinc-400">{lifecycleSummary}</p>

          {effective ? (
            <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Lifecycle</p>
                <p className="mt-2 text-sm font-medium text-white">{lifecycleLabel}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Public URL</p>
                {effective.publicUrl ? (
                  <a className="mt-2 block break-all text-sm font-medium text-white transition hover:text-amber-200" href={effective.publicUrl}>
                    {effective.publicUrl}
                  </a>
                ) : (
                  <p className="mt-2 text-sm font-medium text-white">Not published yet</p>
                )}
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Door URL</p>
                <a className="mt-2 block break-all text-sm font-medium text-white transition hover:text-amber-200" href={`/door/${effective.id}`}>
                  /door/{effective.id}
                </a>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Reserved / checked in</p>
                <p className="mt-2 text-sm font-medium text-white">
                  {effective.reservedCount} / {effective.checkedInCount}
                </p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Pricing</p>
                <p className="mt-2 text-sm font-medium text-white">{pricingSummary(effective)}</p>
              </div>
            </div>
          ) : null}
        </header>

        {loading ? <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Loading event…</div> : null}
        {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
        {message ? <p className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">{message}</p> : null}

        {!loading ? (
          <div className="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
            {creating && !hasWorkspace ? (
              <section className="space-y-4 rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Create from workspace</p>
                <h2 className="text-2xl font-semibold text-white">Events start inside a workspace</h2>
                <p className="max-w-xl text-sm leading-6 text-zinc-400">
                  Open the workspace first, then use its New event button so this event can inherit the right workspace context.
                </p>
                <div className="flex flex-wrap gap-3 text-sm">
                  <a className="rounded-2xl bg-white px-4 py-3 font-medium text-zinc-950 transition hover:bg-zinc-200" href="/workspace">
                    Go to workspace
                  </a>
                  <a className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 font-medium text-zinc-100 transition hover:bg-white/10" href="/">
                    Home
                  </a>
                </div>
              </section>
            ) : (
              <form className="space-y-4 rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleSubmit}>
                <label className="block space-y-2 text-sm">
                  <span className="text-zinc-300">Title</span>
                  <input
                    className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8 disabled:cursor-not-allowed disabled:opacity-60"
                    value={form.title}
                    onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))}
                    required
                    disabled={closed}
                  />
                </label>

                <label className="block space-y-2 text-sm">
                  <span className="text-zinc-300">Starts at</span>
                  <input
                    className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8 disabled:cursor-not-allowed disabled:opacity-60"
                    type="datetime-local"
                    value={form.startsAt}
                    onChange={(event) => setForm((current) => ({ ...current, startsAt: event.target.value }))}
                    required
                    disabled={closed}
                  />
                </label>

                <label className="block space-y-2 text-sm">
                  <span className="text-zinc-300">Location</span>
                  <input
                    className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8 disabled:cursor-not-allowed disabled:opacity-60"
                    value={form.locationDisplay}
                    onChange={(event) => setForm((current) => ({ ...current, locationDisplay: event.target.value }))}
                    required
                    disabled={closed}
                  />
                </label>

                <label className="block space-y-2 text-sm">
                  <span className="text-zinc-300">Public description</span>
                  <textarea
                    className="min-h-40 w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8 disabled:cursor-not-allowed disabled:opacity-60"
                    value={form.publicDescription}
                    onChange={(event) => setForm((current) => ({ ...current, publicDescription: event.target.value }))}
                    required
                    disabled={closed}
                  />
                </label>

                <label className="block space-y-2 text-sm">
                  <span className="text-zinc-300">Ticket allocation</span>
                  <input
                    className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8 disabled:cursor-not-allowed disabled:opacity-60"
                    type="number"
                    min="1"
                    step="1"
                    value={form.ticketAllocation}
                    onChange={(event) => setForm((current) => ({ ...current, ticketAllocation: event.target.value }))}
                    required
                    disabled={closed}
                  />
                </label>

                <fieldset className={`rounded-[1.5rem] border p-4 ${pricingLocked ? 'border-white/10 bg-white/5 opacity-70' : 'border-white/10 bg-white/5'}`} disabled={pricingLocked}>
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Pricing</p>
                      <h2 className="mt-2 text-lg font-semibold text-white">Free or fixed paid tickets</h2>
                    </div>
                    <span className="rounded-full border border-white/10 bg-black/20 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-zinc-300">
                      USD only
                    </span>
                  </div>

                  <div className="mt-4 grid gap-3 sm:grid-cols-2">
                    <label className={`cursor-pointer rounded-2xl border p-4 transition ${form.pricingMode === 'free' ? 'border-amber-300/40 bg-amber-300/10 text-white' : 'border-white/10 bg-white/5 text-zinc-300 hover:bg-white/8'}`}>
                      <input
                        className="sr-only"
                        type="radio"
                        name="pricingMode"
                        value="free"
                        checked={form.pricingMode === 'free'}
                        onChange={() => setForm((current) => ({ ...current, pricingMode: 'free', ticketPriceDollars: '0.00' }))}
                        disabled={closed || pricingLocked}
                      />
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold">Free reservation</p>
                          <p className="mt-1 text-sm leading-6 text-current/70">Guests reserve without paying. Keep the old no-cost flow.</p>
                        </div>
                        <span className="rounded-full border border-current/15 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em]">Free</span>
                      </div>
                    </label>

                    <label className={`cursor-pointer rounded-2xl border p-4 transition ${form.pricingMode === 'fixed' ? 'border-amber-300/40 bg-amber-300/10 text-white' : 'border-white/10 bg-white/5 text-zinc-300 hover:bg-white/8'}`}>
                      <input
                        className="sr-only"
                        type="radio"
                        name="pricingMode"
                        value="fixed"
                        checked={form.pricingMode === 'fixed'}
                        onChange={() => setForm((current) => ({ ...current, pricingMode: 'fixed' }))}
                        disabled={closed || pricingLocked}
                      />
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold">Fixed paid ticket</p>
                          <p className="mt-1 text-sm leading-6 text-current/70">Guests pay through Stripe Checkout in USD.</p>
                        </div>
                        <span className="rounded-full border border-current/15 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em]">Paid</span>
                      </div>
                    </label>
                  </div>

                  {form.pricingMode === 'fixed' ? (
                    <label className="mt-4 block space-y-2 text-sm">
                      <span className="text-zinc-300">Price in USD</span>
                      <input
                        className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8 disabled:cursor-not-allowed disabled:opacity-60"
                        type="number"
                        min="0.5"
                        step="0.01"
                        inputMode="decimal"
                        value={form.ticketPriceDollars}
                        onChange={(event) => setForm((current) => ({ ...current, ticketPriceDollars: event.target.value }))}
                        required
                        disabled={closed || pricingLocked}
                      />
                      <p className="text-xs leading-5 text-zinc-500">Enter dollars; we convert to cents for checkout. Minimum recommended price is $0.50.</p>
                    </label>
                  ) : (
                    <p className="mt-4 text-sm leading-6 text-zinc-400">Free events keep the existing reservation flow and do not send guests to Stripe.</p>
                  )}

                  {pricingLocked ? <p className="mt-4 text-sm leading-6 text-zinc-400">Pricing is locked once tickets exist or after the event closes.</p> : null}
                </fieldset>

                {!closed ? (
                  <button className="w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60" type="submit" disabled={saving || !dirty}>
                    {saving ? 'Saving…' : dirty ? (creating ? 'Create event' : 'Save event') : creating ? 'Fill in details' : 'No changes'}
                  </button>
                ) : (
                  <p className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-zinc-400">This event is closed. Editing is disabled.</p>
                )}
              </form>
            )}

            <aside className="space-y-6">
              {creating && hasWorkspace ? (
                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Publish checklist</p>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Ready to go live?</h2>
                  <ul className="mt-4 space-y-3 text-sm leading-6 text-zinc-400">
                    <li>• Title, start time, location, and public description are filled out.</li>
                    <li>• Ticket allocation matches the number of tickets you want to reserve.</li>
                    <li>• Save before publishing so the public page and Door links stay in sync.</li>
                  </ul>
                </section>
              ) : null}

              {!creating && effective?.status === 'draft' ? (
                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Publish checklist</p>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Before you publish</h2>
                  <ul className="mt-4 space-y-3 text-sm leading-6 text-zinc-400">
                    <li>• Confirm the public title and description read well on mobile.</li>
                    <li>• Check the start time, location, and ticket allocation.</li>
                    <li>• Make sure the event is saved before you open the public page.</li>
                  </ul>
                </section>
              ) : null}

              {!creating && effective?.status === 'published' ? (
                <section className="rounded-[1.75rem] border border-emerald-400/20 bg-emerald-400/10 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-emerald-200">Live event</p>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Next step: end of night</h2>
                  <div className="mt-4 space-y-3 text-sm">
                    {effective.publicUrl ? (
                      <a className="block rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-white transition hover:bg-black/30" href={effective.publicUrl}>
                        Public page: {effective.publicUrl}
                      </a>
                    ) : null}
                    <a className="block rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-white transition hover:bg-black/30" href={`/door/${effective.id}`}>
                      Door URL: /door/{effective.id}
                    </a>
                    <div className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-white">
                      Reserved {effective.reservedCount} · Checked in {effective.checkedInCount}
                    </div>
                    <button className="door-action w-full rounded-2xl bg-white px-4 py-3 text-left font-medium text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70" type="button" onClick={handleEndOfNight} disabled={actioning}>
                      End of night
                    </button>
                  </div>
                </section>
              ) : null}

              {report ? (
                <section className="rounded-[1.75rem] border border-fuchsia-400/20 bg-zinc-950/95 p-6 shadow-2xl shadow-black/30">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Report summary</p>
                  <h2 className="mt-2 text-2xl font-semibold text-white">{report.title}</h2>
                  <p className="mt-2 text-sm text-zinc-400">Generated {formatDateTime(report.generatedAt)} by {report.generatedByMemberEmail}</p>
                  <p className="mt-3 text-sm leading-6 text-zinc-300">This is the end-of-night snapshot for the event.</p>
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
                  <div className="mt-4 flex flex-wrap gap-3 text-sm">
                    <a className="rounded-2xl bg-white px-4 py-3 font-medium text-zinc-950 transition hover:bg-zinc-200" href={report.publicUrl}>
                      Public page
                    </a>
                    <a className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 font-medium text-zinc-100 transition hover:bg-white/10" href="/workspace">
                      Workspace
                    </a>
                  </div>
                </section>
              ) : null}

              {!creating && effective && effective.status !== 'end_of_night' ? (
                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Actions</p>
                  <div className="mt-4 flex flex-col gap-3">
                    {effective.status === 'draft' ? (
                      <button className="door-action rounded-2xl bg-white px-4 py-3 text-left font-medium text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70" type="button" onClick={handlePublish} disabled={actioning}>
                        Publish public page
                      </button>
                    ) : null}

                    {effective.publicUrl ? (
                      <a className="door-action rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-left font-medium text-zinc-100 transition hover:bg-white/10" href={effective.publicUrl}>
                        Open public URL
                      </a>
                    ) : null}

                    <a className="door-action rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-left font-medium text-zinc-100 transition hover:bg-white/10" href={`/door/${effective.id}`}>
                      Open Door
                    </a>
                  </div>
                </section>
              ) : null}

              {creating && hasWorkspace ? (
                <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
                  <p className="text-xs uppercase tracking-[0.3em] text-zinc-500">Workspace link</p>
                  <p className="mt-2 text-sm leading-6 text-zinc-400">Finish the draft here, then return to the workspace to publish or share it.</p>
                  <a className="mt-4 inline-flex rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-medium text-zinc-100 transition hover:bg-white/10" href="/workspace">
                    Back to workspace
                  </a>
                </section>
              ) : null}
            </aside>
          </div>
        ) : null}
      </section>
    </main>
  );
}
