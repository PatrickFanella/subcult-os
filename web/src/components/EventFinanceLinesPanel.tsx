import { FormEvent, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { ApiError } from '../api';
import type { EventFinanceLineDTO } from '../domain';
import {
  centsFromDecimal,
  currentFinanceLines,
  decimalFromCents,
  financeTotals,
  preserveHydratedUTC,
  toLocalDateTime,
} from '../modules/financeLines/financeLineModel';

import { createFinanceLineSession } from '../modules/financeLines/financeLineSession';
import { Button } from '../ui/Button';

type EntryType = EventFinanceLineDTO['entryType'];
type Direction = EventFinanceLineDTO['direction'];
type Draft = {
  entryType: EntryType;
  direction: Direction;
  amount: string;
  currency: string;
  label: string;
  reason: string;
  dueAt: string;
  dueAtSource: string;
  occurredAt: string;
  occurredAtSource: string;
  payableLineId: string;
  correctsLineId: string;
};

const empty = (): Draft => ({ entryType: 'budget', direction: 'expense', amount: '', currency: 'usd', label: '', reason: '', dueAt: '', dueAtSource: '', occurredAt: '', occurredAtSource: '', payableLineId: '', correctsLineId: '' });
function money(cents: number, currency: string) {
  return `${currency.toUpperCase()} ${decimalFromCents(cents)}`;
}
function errorText(error: unknown) {
  return error instanceof ApiError ? error.message : 'Finance lines could not be updated.';
}
function labelForType(value: EntryType) {
  return value === 'actual_payment' ? 'Manual actual payment' : value[0].toUpperCase() + value.slice(1);
}

export function EventFinanceLinesPanel({ eventId, allowed }: { eventId: string; allowed: boolean }) {
  return allowed ? <FinanceLedger key={eventId} eventId={eventId} /> : null;
}

function FinanceLedger({ eventId }: { eventId: string }) {
  const session = useMemo(() => createFinanceLineSession(eventId), [eventId]);
  const active = useRef(true);
  const lifetime = useRef(0);
  const [access, setAccess] = useState<'loading' | 'ready' | 'unavailable'>('loading');
  const [lines, setLines] = useState<EventFinanceLineDTO[]>([]);
  const [draft, setDraft] = useState<Draft>(empty);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const leaves = useMemo(() => currentFinanceLines(lines), [lines]);
  const totals = useMemo(() => financeTotals(lines), [lines]);
  // Payable roots are deliberately stable obligation identities. A correction
  // changes the current payable detail, not the identity that manual actuals cite.
  const payableRoots = useMemo(
    () => lines.filter((line) => line.entryType === 'payable' && !line.correctsLineId),
    [lines],
  );

  function clearPrivateState() {
    setLines([]);
    setDraft(empty());
    setBusy(false);
    setNotice(null);
  }

  function changeEntryType(entryType: EntryType) {
    setDraft((current) => ({
      ...current,
      entryType,
      direction: entryType === 'payable' ? 'expense' : current.direction,
      dueAt: '',
      dueAtSource: '',
      occurredAt: '',
      occurredAtSource: '',
      payableLineId: '',
      correctsLineId: '',
    }));
  }

  function selectCorrection(id: string) {
    const line = leaves.find((item) => item.id === id);
    if (!line) {
      setDraft(empty());
      return;
    }
    setDraft({
      entryType: line.entryType,
      direction: line.direction,
      amount: decimalFromCents(line.amountCents),
      currency: line.currency,
      label: line.label,
      reason: line.reason,
      dueAt: toLocalDateTime(line.dueAt),
      dueAtSource: line.dueAt ?? '',
      occurredAt: toLocalDateTime(line.occurredAt),
      occurredAtSource: line.occurredAt ?? '',
      payableLineId: line.payableLineId ?? '',
      correctsLineId: line.id,
    });
  }

  useLayoutEffect(() => {
    active.current = true;
    lifetime.current += 1;
    session.activate();
    return () => { active.current = false; lifetime.current += 1; session.dispose(); };
  }, [session]);

  async function refresh() {
    const run = lifetime.current;
    clearPrivateState();
    setAccess('loading');
    try {
      const value = await session.read();
      if (!active.current || run !== lifetime.current || value === null) return;
      setLines(value);
      setAccess('ready');
    } catch (error) {
      if (!active.current || run !== lifetime.current) return;
      clearPrivateState();
      setAccess('unavailable');
      setNotice(errorText(error));
    }
  }

  useEffect(() => { void refresh(); }, [session]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!active.current || busy || access !== 'ready' || !session.canSave()) return;

    const amountCents = centsFromDecimal(draft.amount);
    const dueAt = preserveHydratedUTC(draft.dueAt, draft.dueAtSource);
    const occurredAt = preserveHydratedUTC(draft.occurredAt, draft.occurredAtSource);
    const isCorrection = draft.correctsLineId !== '';
    if (
      amountCents === null
      || (!isCorrection && amountCents <= 0)
      || !/^[a-z]{3}$/.test(draft.currency)
      || dueAt === null
      || occurredAt === null
    ) {
      setNotice('Enter a valid amount, three-letter currency, and valid dates. New lines must be greater than zero.');
      return;
    }

    const body = {
      entryType: draft.entryType,
      direction: draft.direction,
      amountCents,
      currency: draft.currency,
      label: draft.label,
      reason: draft.reason,
      dueAt,
      occurredAt,
      payableLineId: draft.payableLineId || undefined,
      correctsLineId: draft.correctsLineId || undefined,
    };
    const run = lifetime.current;
    setBusy(true);
    setNotice(null);
    try {
      const created = await session.save(body);
      if (!active.current || run !== lifetime.current || created === null) return;
      setLines(value => value.some(line => line.id === created.id) ? value : [created, ...value]);
      setDraft(empty());
      setNotice('Recorded manually. This does not execute or confirm a provider payment.');
    } catch (error) {
      if (!active.current || run !== lifetime.current) return;
      if (error instanceof ApiError && (error.status === 400 || error.status === 409)) {
        setNotice(errorText(error));
      } else {
        clearPrivateState();
        setAccess('unavailable');
        setNotice(error instanceof ApiError && (error.status === 401 || error.status === 403)
          ? 'Finance access is unavailable. Refresh to check current access.'
          : 'The record-save outcome is uncertain. Refresh and inspect the ledger before another record.');
      }
    } finally {
      if (active.current && run === lifetime.current) setBusy(false);
    }
  }

  return (
    <section className="mt-6 rounded-2xl border border-stroke-subtle bg-surface-inset p-5">
      <p className="text-xs uppercase tracking-[0.25em] text-fg-muted">Finance ledger</p>
      <h3 className="mt-2 text-xl font-semibold text-fg-primary">Budgets, payables, and manual actuals</h3>
      <p className="mt-2 text-sm text-fg-secondary">
        Manual actual payments are records only. They do not send, execute, or confirm a provider payment, and they do not alter ticket settlement totals.
      </p>

      <p role="status" aria-atomic="true" className={notice ? 'mt-3 border border-stroke-subtle p-3 text-sm' : 'sr-only'}>{notice ?? ''}</p>
      {access === 'loading' ? <p className="mt-3 text-sm">Loading private ledger…</p> : null}
      {access === 'unavailable' ? <Button variant="secondary" className="mt-3" onClick={() => { void refresh(); }}>Refresh ledger</Button> : null}

      {access === 'ready' ? <form className="mt-4 space-y-4" onSubmit={submit}>
        <fieldset disabled={busy} className="grid gap-3 md:grid-cols-2">
          <label>
            Type
            <select
              value={draft.entryType}
              disabled={draft.correctsLineId !== ''}
              onChange={(event) => changeEntryType(event.target.value as EntryType)}
              className="field mt-1 py-3"
            >
              <option value="budget">Budget</option>
              <option value="payable">Payable</option>
              <option value="actual_payment">Manual actual payment</option>
            </select>
          </label>
          <label>
            Direction
            <select
              value={draft.direction}
              disabled={draft.entryType === 'payable' || draft.correctsLineId !== ''}
              onChange={(event) => setDraft((current) => ({ ...current, direction: event.target.value as Direction, payableLineId: '' }))}
              className="field mt-1 py-3"
            >
              <option value="expense">Expense</option>
              <option value="income">Income</option>
            </select>
          </label>
          <label>
            Amount
            <input required inputMode="decimal" value={draft.amount} onChange={(event) => setDraft((current) => ({ ...current, amount: event.target.value }))} className="field mt-1 py-3" />
          </label>
          <label>
            Currency
            <input required maxLength={3} disabled={draft.correctsLineId !== ''} value={draft.currency} onChange={(event) => setDraft((current) => ({ ...current, currency: event.target.value.toLowerCase(), payableLineId: '' }))} className="field mt-1 py-3" />
          </label>
          <label>
            Label
            <input required maxLength={200} value={draft.label} onChange={(event) => setDraft((current) => ({ ...current, label: event.target.value }))} className="field mt-1 py-3" />
          </label>
          <label>
            Reason
            <input required maxLength={1000} value={draft.reason} onChange={(event) => setDraft((current) => ({ ...current, reason: event.target.value }))} className="field mt-1 py-3" />
          </label>
          {draft.entryType === 'payable' && (
            <label>
              Due at (optional, your local time)
              <input type="datetime-local" value={draft.dueAt} onChange={(event) => setDraft((current) => ({ ...current, dueAt: event.target.value }))} className="field mt-1 py-3" />
            </label>
          )}
          {draft.entryType === 'actual_payment' && (
            <label>
              Occurred at (your local time)
              <input required type="datetime-local" value={draft.occurredAt} onChange={(event) => setDraft((current) => ({ ...current, occurredAt: event.target.value }))} className="field mt-1 py-3" />
            </label>
          )}
          {draft.entryType === 'actual_payment' && draft.direction === 'expense' && (
            <label>
              Stable payable obligation
              <select disabled={draft.correctsLineId !== ''} value={draft.payableLineId} onChange={(event) => setDraft((current) => ({ ...current, payableLineId: event.target.value }))} className="field mt-1 py-3">
                <option value="">Not linked to a payable</option>
                {payableRoots.filter((line) => line.currency === draft.currency).map((line) => (
                  <option key={line.id} value={line.id}>{line.label} · {money(line.amountCents, line.currency)}</option>
                ))}
              </select>
            </label>
          )}
          <label>
            Correct current line
            <select value={draft.correctsLineId} onChange={(event) => selectCorrection(event.target.value)} className="field mt-1 py-3">
              <option value="">New line</option>
              {leaves.map((line) => <option key={line.id} value={line.id}>{labelForType(line.entryType)} · {line.direction} · {line.label} · {money(line.amountCents, line.currency)}</option>)}
            </select>
          </label>
        </fieldset>
        <Button type="submit" busy={busy} disabled={busy}>
          {busy ? 'Saving…' : draft.correctsLineId ? 'Record correction' : 'Record line'}
        </Button>
      </form> : null}

      <section className="mt-6" aria-labelledby="finance-current-totals">
        <h4 id="finance-current-totals" className="text-sm font-semibold uppercase tracking-[0.18em] text-status-info">Current ledger totals</h4>
        <p className="mt-1 text-sm text-fg-secondary">Each category remains separate from ticket settlement receipts.</p>
        <dl className="mt-3 grid gap-2 sm:grid-cols-2">
          {totals.map((total) => (
            <div key={total.key} className="border border-stroke-subtle bg-surface-inset p-3 text-sm">
              <dt className="text-fg-secondary">{labelForType(total.entryType as EntryType)} · {total.direction}</dt>
              <dd className="mt-1 font-semibold text-fg-primary">{money(total.amountCents, total.currency)}</dd>
            </div>
          ))}
          {access === 'ready' && totals.length === 0 && <p className="text-sm text-fg-secondary">No current finance lines recorded.</p>}
        </dl>
      </section>

      <section className="mt-6" aria-labelledby="finance-history">
        <h4 id="finance-history" className="text-sm font-semibold uppercase tracking-[0.18em] text-status-info">Retained history</h4>
        <ol className="mt-3 space-y-2">
          {lines.map((line) => (
            <li key={line.id} className="border border-stroke-subtle bg-surface-inset p-3 text-sm">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span>{labelForType(line.entryType)} · {line.direction} · {line.label}</span>
                <strong>{money(line.amountCents, line.currency)}</strong>
              </div>
              <p className="mt-2 text-fg-secondary">{line.reason}</p>
              <p className="mt-2 break-all text-xs text-fg-muted">Recorded by {line.createdByPersonId} at {line.createdAt}</p>
              {line.correctsLineId && <p className="break-all text-xs text-fg-muted">Corrects {line.correctsLineId}</p>}
              {line.payableLineId && <p className="break-all text-xs text-fg-muted">Stable payable obligation {line.payableLineId}</p>}
            </li>
          ))}
          {access === 'ready' && lines.length === 0 && <li className="text-sm text-fg-secondary">No finance history recorded.</li>}
        </ol>
      </section>
    </section>
  );
}
