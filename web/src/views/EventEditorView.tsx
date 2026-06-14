import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError, api, patchJSON, postJSON } from '../api';
import type { EventDTO, EventReportDTO, EventSettlementDTO, EventStatus } from '../domain';

type FormState = {
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  ticketAllocation: string;
  pricingMode: 'free' | 'fixed';
  ticketPriceDollars: string;
};

type SettlementAdjustmentFormState = {
  amountDollars: string;
  label: string;
  reason: string;
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

function formatMoney(cents: number, currency: string) {
  const normalizedCurrency = currency.trim().toUpperCase() || 'USD';
  return `${new Intl.NumberFormat([], { style: 'currency', currency: normalizedCurrency }).format(cents / 100)} ${normalizedCurrency}`;
}

function formatSignedMoney(cents: number, currency: string) {
  const sign = cents < 0 ? '-' : '+';
  return `${sign}${formatMoney(Math.abs(cents), currency)}`;
}

function priceInCents(value: string) {
  const parsed = Number(value);
  if (Number.isNaN(parsed)) {
    return 0;
  }

  return Math.round(parsed * 100);
}

function emptySettlementAdjustmentForm(): SettlementAdjustmentFormState {
  return {
    amountDollars: '',
    label: '',
    reason: '',
  };
}

function pricingSummary(event: EventDTO | null) {
  if (!event || event.pricingMode === 'free') {
    return 'Free reservation';
  }

  return formatMoney(event.ticketPriceCents, event.ticketCurrency);
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
  const [settlement, setSettlement] = useState<EventSettlementDTO | null>(null);
  const [settlementForm, setSettlementForm] = useState<SettlementAdjustmentFormState>(emptySettlementAdjustmentForm);
  const [settlementSubmitting, setSettlementSubmitting] = useState(false);
  const [settlementFinalizing, setSettlementFinalizing] = useState(false);

  const hasWorkspace = workspaceId !== '';
  const closed = event?.status === 'end_of_night';
  const pricingLocked = (event?.reservedCount ?? 0) > 0 || closed;
  const settlementFinalized = settlement?.status === 'finalized';
  const settlementOpen = settlement?.status === 'open';
  const dirty = useMemo(() => !formsMatch(form, initialForm), [form, initialForm]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (creating) {
        setLoading(false);
        setSettlement(null);
        setSettlementForm(emptySettlementAdjustmentForm());
        return;
      }

      setLoading(true);
      setError(null);
      setSettlement(null);

      try {
        const loaded = await api<EventDTO>(`/api/events/${eventId}`);
        if (cancelled) return;

        setEvent(loaded);

        const loadedForm = formFromEvent(loaded);
        setForm(loadedForm);
        setInitialForm(loadedForm);

        if (loaded.status === 'end_of_night') {
          const loadedReport = await api<EventReportDTO>(`/api/events/${eventId}/report`).catch(() => null);
          if (!cancelled) {
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
      setSettlement(null);
      setForm(blank);
      setInitialForm(blank);
      setSettlementForm(emptySettlementAdjustmentForm());
    }
  }, [creating]);

  useEffect(() => {
    let cancelled = false;

    async function loadSettlement() {
      if (creating || event?.status !== 'end_of_night' || !event) {
        setSettlement(null);
        return;
      }

      try {
        const loadedSettlement = await api<EventSettlementDTO>(`/api/events/${event.id}/settlement`);
        if (!cancelled) {
          setSettlement(loadedSettlement);
        }
      } catch (caught) {
        if (cancelled) return;

        if (caught instanceof ApiError && caught.status === 404) {
          setSettlement(null);
          return;
        }

        setError(caught instanceof Error ? caught.message : 'Unable to load settlement');
      }
    }

    void loadSettlement();

    return () => {
      cancelled = true;
    };
  }, [creating, event?.id, event?.status]);

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

  async function handleSettlementAdjustmentSubmit(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    if (!event || !settlement || settlement.status !== 'open') {
      setError('Settlement is locked');
      return;
    }

    setSettlementSubmitting(true);
    setMessage(null);
    setError(null);

    try {
      const updatedSettlement = await postJSON<EventSettlementDTO>(`/api/events/${event.id}/settlement/adjustments`, {
        amountCents: priceInCents(settlementForm.amountDollars),
        label: settlementForm.label.trim(),
        reason: settlementForm.reason.trim(),
      });

      setSettlement(updatedSettlement);
      setSettlementForm(emptySettlementAdjustmentForm());
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to add adjustment');
    } finally {
      setSettlementSubmitting(false);
    }
  }

  async function handleFinalizeSettlement() {
    if (!event || !settlement || settlement.status !== 'open') return;

    setSettlementFinalizing(true);
    setMessage(null);
    setError(null);

    try {
      const finalizedSettlement = await postJSON<EventSettlementDTO>(`/api/events/${event.id}/settlement/finalize`, {});
      setSettlement(finalizedSettlement);
      setMessage('Settlement finalized');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to finalize settlement');
    } finally {
      setSettlementFinalizing(false);
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
                  {report.settlementSummary?.currency ? (
                    <div className="mt-4 rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Settlement summary</p>
                      <div className="mt-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                        <div>
                          <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Gross paid revenue</p>
                          <p className="mt-2 text-lg font-semibold text-white">
                            {formatMoney(report.settlementSummary.grossPaidRevenueCents, report.settlementSummary.currency)}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Paid tickets</p>
                          <p className="mt-2 text-lg font-semibold text-white">{report.settlementSummary.paidTicketCount}</p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Pending tickets</p>
                          <p className="mt-2 text-lg font-semibold text-white">{report.settlementSummary.pendingTicketCount}</p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Cancelled tickets</p>
                          <p className="mt-2 text-lg font-semibold text-white">{report.settlementSummary.cancelledTicketCount}</p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Free tickets</p>
                          <p className="mt-2 text-lg font-semibold text-white">{report.settlementSummary.freeTicketCount}</p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Reserved total</p>
                          <p className="mt-2 text-lg font-semibold text-white">{report.settlementSummary.reservedCount}</p>
                        </div>
                      </div>
                    </div>
                  ) : null}
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

              {settlement ? (
                <section className="rounded-[1.75rem] border border-cyan-400/20 bg-zinc-950/95 p-6 shadow-2xl shadow-black/30">
                  <p className="text-xs uppercase tracking-[0.3em] text-cyan-300">Settlement closeout</p>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Review adjustments</h2>
                  <p className="mt-2 text-sm text-zinc-400">Status: {settlementFinalized ? 'finalized (locked)' : 'open'}</p>

                  {settlementFinalized ? (
                    <div className="mt-4 rounded-2xl border border-emerald-400/20 bg-emerald-400/10 p-4 text-sm text-emerald-100">
                      <p className="font-medium">Settlement locked</p>
                      <p className="mt-2 leading-6">
                        Finalized{settlement.finalizedAt ? ` on ${formatDateTime(settlement.finalizedAt)}` : ''}
                        {settlement.finalizedByPersonId ? ` by ${settlement.finalizedByPersonId}` : ''}.
                      </p>
                    </div>
                  ) : null}

                  <div className="mt-4 grid gap-3 sm:grid-cols-3">
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Gross revenue</p>
                      <p className="mt-2 text-lg font-semibold text-white">{formatMoney(settlement.grossPaidRevenueCents, settlement.currency)}</p>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Adjustment total</p>
                      <p className="mt-2 text-lg font-semibold text-white">{formatSignedMoney(settlement.adjustmentTotalCents, settlement.currency)}</p>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Net total</p>
                      <p className="mt-2 text-lg font-semibold text-white">{formatMoney(settlement.netTotalCents, settlement.currency)}</p>
                    </div>
                  </div>

                  <div className="mt-4 rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Adjustments</p>
                    {settlement.adjustments.length > 0 ? (
                      <div className="mt-3 space-y-3">
                        {settlement.adjustments.map((adjustment) => (
                          <div key={adjustment.id} className="rounded-2xl border border-white/10 bg-black/20 p-4">
                            <div className="flex flex-wrap items-start justify-between gap-3">
                              <div>
                                <p className="text-sm font-semibold text-white">{adjustment.label}</p>
                                <p className="mt-1 text-sm leading-6 text-zinc-400">{adjustment.reason}</p>
                              </div>
                              <p className="text-sm font-semibold text-white">{formatSignedMoney(adjustment.amountCents, settlement.currency)}</p>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="mt-3 text-sm leading-6 text-zinc-400">No adjustments yet.</p>
                    )}
                  </div>

                  <div className="mt-4 flex flex-wrap gap-3 text-sm">
                    {settlementOpen ? (
                      <button
                        className="rounded-2xl border border-emerald-400/20 bg-emerald-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-emerald-200 disabled:cursor-not-allowed disabled:bg-emerald-300/60"
                        type="button"
                        onClick={handleFinalizeSettlement}
                        disabled={settlementFinalizing}
                      >
                        {settlementFinalizing ? 'Finalizing…' : 'Finalize settlement'}
                      </button>
                    ) : null}
                  </div>

                  {settlementOpen ? (
                    <form className="mt-4 space-y-4 rounded-2xl border border-white/10 bg-white/5 p-4" onSubmit={handleSettlementAdjustmentSubmit}>
                      <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Add adjustment</p>
                      <label className="block space-y-2 text-sm">
                        <span className="text-zinc-300">Amount in USD</span>
                        <input
                          className="w-full rounded-2xl border border-white/10 bg-zinc-950/60 px-4 py-3 text-white outline-none transition focus:border-cyan-300/60 focus:bg-zinc-950/80 disabled:cursor-not-allowed disabled:opacity-60"
                          type="number"
                          step="0.01"
                          inputMode="decimal"
                          value={settlementForm.amountDollars}
                          onChange={(event) => setSettlementForm((current) => ({ ...current, amountDollars: event.target.value }))}
                          placeholder="-2.00"
                          required
                          disabled={settlementSubmitting}
                        />
                      </label>
                      <label className="block space-y-2 text-sm">
                        <span className="text-zinc-300">Label</span>
                        <input
                          className="w-full rounded-2xl border border-white/10 bg-zinc-950/60 px-4 py-3 text-white outline-none transition focus:border-cyan-300/60 focus:bg-zinc-950/80 disabled:cursor-not-allowed disabled:opacity-60"
                          value={settlementForm.label}
                          onChange={(event) => setSettlementForm((current) => ({ ...current, label: event.target.value }))}
                          required
                          disabled={settlementSubmitting}
                        />
                      </label>
                      <label className="block space-y-2 text-sm">
                        <span className="text-zinc-300">Reason</span>
                        <textarea
                          className="min-h-28 w-full rounded-2xl border border-white/10 bg-zinc-950/60 px-4 py-3 text-white outline-none transition focus:border-cyan-300/60 focus:bg-zinc-950/80 disabled:cursor-not-allowed disabled:opacity-60"
                          value={settlementForm.reason}
                          onChange={(event) => setSettlementForm((current) => ({ ...current, reason: event.target.value }))}
                          required
                          disabled={settlementSubmitting}
                        />
                      </label>
                      <button className="rounded-2xl bg-cyan-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-cyan-200 disabled:cursor-not-allowed disabled:bg-cyan-300/60" type="submit" disabled={settlementSubmitting}>
                        {settlementSubmitting ? 'Saving…' : 'Add adjustment'}
                      </button>
                    </form>
                  ) : (
                    <p className="mt-4 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm leading-6 text-zinc-400">Adjustments are locked after settlement finalization.</p>
                  )}
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
