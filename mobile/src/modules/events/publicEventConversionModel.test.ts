import { describe, expect, it } from 'vitest';

import { publicEventPrimaryActionLabel, publicEventStickyCtaHint } from './publicEventConversionModel';

describe('publicEventConversionModel', () => {
	it('labels primary actions', () => {
		expect(publicEventPrimaryActionLabel({ pricingMode: 'free', isFull: false, remainingTickets: 2 })).toBe('Reserve free ticket');
		expect(publicEventPrimaryActionLabel({ pricingMode: 'fixed', isFull: false, remainingTickets: 2 })).toBe('Buy ticket');
		expect(publicEventPrimaryActionLabel({ pricingMode: 'free', isFull: true, remainingTickets: 0 })).toBe('Sold out');
		expect(publicEventPrimaryActionLabel({ pricingMode: 'fixed', isFull: true, remainingTickets: 0 }, true)).toBe('Sold out');
		expect(publicEventPrimaryActionLabel({ pricingMode: 'free', isFull: false, remainingTickets: 2 }, true)).toBe('Reserving…');
	});

	it('describes sticky CTA hints', () => {
		expect(publicEventStickyCtaHint({ pricingMode: 'free', isFull: false, remainingTickets: 0 }, 'Free')).toBe('0 spots left');
		expect(publicEventStickyCtaHint({ pricingMode: 'free', isFull: false, remainingTickets: 1 }, 'Free')).toBe('1 spot left');
		expect(publicEventStickyCtaHint({ pricingMode: 'fixed', isFull: false, remainingTickets: 0 }, '$18.00')).toBe('$18.00 · secure checkout');
		expect(publicEventStickyCtaHint({ pricingMode: 'fixed', isFull: false, remainingTickets: 5 }, '$18.00')).toBe('$18.00 · secure checkout');
		expect(publicEventStickyCtaHint({ pricingMode: 'free', isFull: true, remainingTickets: 0 }, 'Free')).toBe('No tickets remain for this Event.');
	});
});
