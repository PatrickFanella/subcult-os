import { describe, expect, it } from 'vitest';

import {
	discoveryDefaultErrorBody,
	discoveryEmptyBody,
	discoveryEmptyTitle,
	discoveryLoadingCopy,
	discoveryPricingLabel,
	discoverySubtitle,
	discoveryViewDetailsLabel,
} from './discoveryModel';

describe('discoveryModel', () => {
	it('exposes stable copy', () => {
		expect(discoveryLoadingCopy).toBe('Loading events…');
		expect(discoveryEmptyTitle).toBe('No published events yet');
		expect(discoveryEmptyBody).toBe('Published web events will appear here automatically.');
		expect(discoveryDefaultErrorBody).toBe('Unable to load events');
		expect(discoveryViewDetailsLabel).toBe('View Details');
	});

	it('formats pricing and subtitles', () => {
		expect(discoveryPricingLabel({ pricingMode: 'free', ticketPriceCents: 0, ticketCurrency: 'usd' })).toBe('Free');
		expect(discoveryPricingLabel({ pricingMode: 'fixed', ticketPriceCents: 2500, ticketCurrency: 'usd' })).toBe(new Intl.NumberFormat([], { style: 'currency', currency: 'USD' }).format(25));
		expect(discoverySubtitle({ publicDescription: '' })).toBe('Published event');
	});
});
