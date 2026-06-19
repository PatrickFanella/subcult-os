import { describe, expect, it } from 'vitest';

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
} from './ticketJourney';

describe('ticketJourney helpers', () => {
	it('formats ticket copy without changing the journey wording', () => {
		expect(ticketJourneyStatusLabel('reserved')).toBe('Reserved');
		expect(ticketJourneyStatusBadge('checked_in')).toBe('Access granted');
		expect(ticketJourneyDoorStatusBadge('reserved')).toBe('Needs check-in');
		expect(ticketJourneyCodeCopy('checked_in')).toContain('already been scanned');
		expect(ticketJourneyStatusCopy('checked_in', '2026-06-18T10:00:00Z')).toBe('Checked in at 2026-06-18T10:00:00Z');
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
