import { describe, expect, it } from 'vitest';
import type { TicketDTO } from '../../domain';

import {
	doorCheckInButtonLabel,
	formatTicketCode,
	ticketJourneyCodeCopy,
	ticketJourneyDoorStatusBadge,
	ticketJourneyPaymentBadge,
	ticketJourneyPaymentLabel,
	ticketJourneyPaymentSummary,
	ticketJourneyStatusBadge,
	ticketJourneyStatusCopy,
	ticketJourneyStatusLabel,
	ticketPaymentAllowsAdmission,
} from './ticketJourney';

describe('ticketJourney helpers', () => {
	it.each(['free', 'paid', 'pending', 'cancelled', 'unknown'])('qualifies admission copy for %s payment', (rawStatus) => {
		const paymentStatus = rawStatus as TicketDTO['paymentStatus'];
		const allowed = rawStatus === 'free' || rawStatus === 'paid';
		expect(ticketPaymentAllowsAdmission(paymentStatus)).toBe(allowed);
		for (const status of ['reserved', 'checked_in'] as const) {
			const ticket = { status, paymentStatus };
			if (allowed) {
				expect(ticketJourneyStatusBadge(ticket)).toBe(status === 'reserved' ? 'Bring to door' : 'Already checked in');
			} else {
				expect(ticketJourneyStatusBadge(ticket)).toBe('Not ready for entry');
				expect(ticketJourneyStatusCopy(ticket, 'Earlier')).toContain('not ready for entry');
				expect(ticketJourneyCodeCopy(ticket)).toContain('does not bypass payment');
			}
		}
	});

	it('does not invent payment confirmation for an unknown response status', () => {
		const ticket = { paymentStatus: 'unknown' as TicketDTO['paymentStatus'], amountCents: 0, currency: 'usd' };
		expect(ticketJourneyPaymentLabel(ticket.paymentStatus)).toBe('Payment unavailable');
		expect(ticketJourneyPaymentBadge(ticket)).toBe('Unverified');
		expect(ticketJourneyPaymentSummary(ticket)).toContain('unavailable');
	});
	it('formats ticket copy without changing the journey wording', () => {
		expect(ticketJourneyStatusLabel('reserved')).toBe('Reserved');
		expect(ticketJourneyStatusBadge({ status: 'checked_in', paymentStatus: 'free' })).toBe('Already checked in');
		expect(ticketJourneyDoorStatusBadge('reserved')).toBe('Needs check-in');
		expect(ticketJourneyCodeCopy({ status: 'checked_in', paymentStatus: 'free' })).toContain('already been scanned');
		expect(ticketJourneyStatusCopy({ status: 'checked_in', paymentStatus: 'free' }, '2026-06-18T10:00:00Z')).toBe('Checked in at 2026-06-18T10:00:00Z');
	});

	it('formats payment labels and summaries', () => {
		const ticket = { paymentStatus: 'paid' as const, amountCents: 1500, currency: 'usd' };
		expect(ticketJourneyPaymentLabel(ticket.paymentStatus)).toBe('Paid ticket');
		expect(ticketJourneyPaymentBadge(ticket)).toContain('15');
		expect(ticketJourneyPaymentSummary(ticket)).toContain('Paid');
		expect(ticketJourneyPaymentSummary({ paymentStatus: 'pending', amountCents: 1500, currency: 'usd' })).toContain('processing');
	});

	it('chunks codes consistently', () => {
		expect(formatTicketCode('abcdefghijkl')).toBe('abcd efgh ijkl');
	});

	it('mirrors mobile door check-in button labels', () => {
		expect(doorCheckInButtonLabel(true, false)).toBe('Checking in…');
		expect(doorCheckInButtonLabel(false, true)).toBe('Checked in');
		expect(doorCheckInButtonLabel(false, false)).toBe('Check in');
	});
});
