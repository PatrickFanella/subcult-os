import * as React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { EventFinanceLinesPanel } from './EventFinanceLinesPanel';
import type { EventFinanceLineDTO } from '../domain';

vi.mock('react', async () => {
  const actual = await vi.importActual<typeof React>('react');
  return { ...actual, useState: vi.fn((initial: unknown) => [typeof initial === 'function' ? (initial as () => unknown)() : initial, vi.fn()]) };
});
afterEach(() => vi.mocked(React.useState).mockReset());
const line: EventFinanceLineDTO = { id: 'line-a', eventId: 'event-a', entryType: 'payable', direction: 'expense', amountCents: 8000, currency: 'usd', label: 'Private venue obligation', reason: 'Private agreement', createdByPersonId: 'owner-a', createdAt: '2026-09-30T00:00:00Z' };
function render(access: string, lines: EventFinanceLineDTO[] = [], notice: string | null = null) {
  const values = [access, lines, { entryType: 'budget', direction: 'expense', amount: '', currency: 'usd', label: '', reason: '', dueAt: '', dueAtSource: '', occurredAt: '', occurredAtSource: '', payableLineId: '', correctsLineId: '' }, false, notice];
  vi.mocked(React.useState).mockImplementation((() => [values.shift(), vi.fn()]) as unknown as typeof React.useState);
  return renderToStaticMarkup(<EventFinanceLinesPanel eventId="event-a" allowed />);
}
describe('finance access rendering', () => {
  it('renders nothing without the parent finance permission', () => {
    expect(renderToStaticMarkup(<EventFinanceLinesPanel eventId="event-a" allowed={false} />)).toBe('');
  });
  it('withholds the form and empty-ledger claims while loading', () => {
    const html = render('loading');
    expect(html).toContain('Loading private ledger'); expect(html).not.toContain('<form');
    expect(html).not.toContain('No current finance lines'); expect(html).not.toContain('No finance history');
  });
  it('offers refresh instead of a form after unavailable access', () => {
    const html = render('unavailable', [], 'Finance access is unavailable.');
    expect(html).toContain('Refresh ledger');expect(html).not.toContain('<form');
    expect(html).toContain('role="status" aria-atomic="true"');
  });
  it('shows the editor and truthful empty claims after a successful empty read', () => {
    const html = render('ready');expect(html).toContain('<form');expect(html).toContain('Record line');
    expect(html).toContain('No current finance lines');expect(html).toContain('No finance history');
    expect(html).toContain('do not send, execute, or confirm a provider payment');
  });
  it('keeps payable totals separate and preserves retained history after a successful read', () => {
    const html = render('ready', [line]);expect(html).toContain('USD 80.00');expect(html).toContain('Private venue obligation');
    expect(html).toContain('Private agreement');expect(html).toContain('Retained history');
    expect(html).not.toContain('No finance history');
  });
});
