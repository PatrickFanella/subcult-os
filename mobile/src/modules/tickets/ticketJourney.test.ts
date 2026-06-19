import { describe, expect, it } from 'vitest';

import {
	buildTicketWalletSnapshot,
	formatTicketCode,
	ticketJourneyArrivalNotes,
	ticketJourneyDisplayName,
	ticketJourneyDoorBadge,
	ticketJourneySavedTicketStatus,
} from './ticketJourney';

describe('ticketJourney', () => {
	it('formats ticket codes and display names', () => {
		expect(formatTicketCode('ABCDEFGH')).toBe('ABCD EFGH');
		expect(ticketJourneyDisplayName({ displayName: null, email: 'guest@example.test' })).toBe('guest@example.test');
	});

	it('builds wallet snapshots with savedAt', () => {
		expect(buildTicketWalletSnapshot({ code: 'ABC', email: 'guest@example.test', displayName: 'Guest', status: 'reserved', paymentStatus: 'paid' }, '2026-06-18T00:00:00.000Z')).toEqual({
			code: 'ABC',
			email: 'guest@example.test',
			displayName: 'Guest',
			status: 'reserved',
			paymentStatus: 'paid',
			savedAt: '2026-06-18T00:00:00.000Z',
		});
	});

	it('describes pending and ready arrival states', () => {
		expect(ticketJourneyArrivalNotes('pending')).toContain('Payment is still pending');
		expect(ticketJourneyDoorBadge('checked_in')).toBe('Checked in');
		expect(ticketJourneySavedTicketStatus({ status: 'reserved', paymentStatus: 'pending' })).toBe('payment pending');
	});
});
