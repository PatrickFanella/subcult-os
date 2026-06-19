import { describe, expect, it } from 'vitest';

import {
	discoveryEmptyBody,
	discoveryEmptyTitle,
	discoveryErrorCopy,
	discoveryLoadingCopy,
	discoveryPricingLabel,
	discoverySubtitle,
	discoverySearchLabel,
	discoverySearchPlaceholder,
	discoveryViewEventLabel,
} from './discoveryModel';

describe('discoveryModel', () => {
	it('exposes stable copy', () => {
		expect(discoveryLoadingCopy).toBe('Loading published Events…');
		expect(discoverySearchLabel).toBe('Search published Events');
		expect(discoverySearchPlaceholder).toBe('Search published Events');
		expect(discoveryViewEventLabel).toBe('View Event');
		expect(discoveryEmptyTitle('')).toBe('No published Events yet');
		expect(discoveryEmptyTitle('noise')).toBe('No Events match that search yet');
		expect(discoveryEmptyBody('')).toBe('Published Events will appear here when Hosts share them.');
		expect(discoveryEmptyBody('noise')).toBe('No published Events matched “noise”. Try another search.');
		expect(discoveryErrorCopy('offline')).toBe('Could not load published Events: offline');
	});

	it('formats pricing and subtitles', () => {
		expect(discoveryPricingLabel({ pricingMode: 'free', ticketPriceCents: 0, ticketCurrency: 'usd' })).toBe('Free');
		expect(discoveryPricingLabel({ pricingMode: 'fixed', ticketPriceCents: 2500, ticketCurrency: 'usd' })).toBe(new Intl.NumberFormat([], { style: 'currency', currency: 'USD' }).format(25));
		expect(discoverySubtitle({ publicDescription: '' })).toBe('Published event');
	});
});
