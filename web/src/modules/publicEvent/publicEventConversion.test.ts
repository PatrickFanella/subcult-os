import { describe, expect, it } from 'vitest';

import {
	publicEventConversionSummary,
	publicEventPrimaryCtaLabel,
	publicEventReservationSuccessCopy,
	publicEventRoleSectionIntro,
} from './publicEventConversion';

describe('publicEventConversion', () => {
	it('labels primary reservation states', () => {
		expect(publicEventPrimaryCtaLabel({ pricingMode: 'free', isFull: false })).toBe('Reserve free ticket');
		expect(publicEventPrimaryCtaLabel({ pricingMode: 'fixed', isFull: false })).toBe('Buy ticket');
		expect(publicEventPrimaryCtaLabel({ pricingMode: 'free', isFull: true })).toBe('Sold out');
		expect(publicEventPrimaryCtaLabel({ pricingMode: 'fixed', isFull: true }, true)).toBe('Sold out');
		expect(publicEventPrimaryCtaLabel({ pricingMode: 'fixed', isFull: false }, true)).toBe('Reserving…');
	});

	it('summarizes conversion state without provider-specific language', () => {
		expect(publicEventConversionSummary(null, 'Free')).toBe('Email required to send the ticket. Display name optional. No account needed.');
		expect(publicEventConversionSummary({ pricingMode: 'fixed', isFull: false, remainingTickets: 10, ticketPriceCents: 1800, ticketCurrency: 'usd' }, '$18.00')).toBe(
			'Secure checkout for $18.00. Email is required for the ticket link.',
		);
		expect(publicEventConversionSummary({ pricingMode: 'free', isFull: false, remainingTickets: 1, ticketPriceCents: 0, ticketCurrency: 'usd' }, 'Free')).toBe(
			'1 spot remains. Email is required for the ticket link.',
		);
		expect(publicEventConversionSummary({ pricingMode: 'free', isFull: true, remainingTickets: 0, ticketPriceCents: 0, ticketCurrency: 'usd' }, 'Free')).toBe(
			'Sold out. Ask the host about returns or future dates.',
		);
	});

	it('describes success and public role section state', () => {
		expect(publicEventReservationSuccessCopy({ code: 'ABCD1234' })).toBe('Ticket reserved. Save code ABCD1234 and show it at the door.');
		expect(publicEventRoleSectionIntro(null)).toBe('Checking public roles for this event.');
		expect(publicEventRoleSectionIntro(0)).toBe('No public roles are open right now.');
		expect(publicEventRoleSectionIntro(2)).toBe('Applying for a role does not affect your ticket.');
	});
});
