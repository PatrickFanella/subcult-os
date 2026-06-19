import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	discoveryDescription,
	discoveryEmptyBody,
	discoveryEmptyStateCopy,
	discoveryEmptyTitle,
	discoveryErrorCopy,
	discoveryLoadingCopy,
	discoveryPricingLabel,
	discoveryRemainingLabel,
	discoverySearchLabel,
	discoverySearchPlaceholder,
	discoveryViewEventLabel,
	formatDiscoveryDateTime,
	getRequestedDiscoveryQuery,
} from './discoveryModel';

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('discovery model helpers', () => {
	it('reads query params and preserves discovery copy', () => {
		vi.stubGlobal('window', { location: { search: '?q=Market' } });

		expect(getRequestedDiscoveryQuery()).toBe('Market');
		expect(discoverySearchLabel).toBe('Search published Events');
		expect(discoverySearchPlaceholder).toBe('Search published Events');
		expect(discoveryLoadingCopy).toBe('Loading published Events…');
		expect(discoveryViewEventLabel).toBe('View Event');
		expect(discoveryDescription).toBe('Browse published events without opening private workspace pages.');
	});

	it('formats discovery result labels without drifting copy', () => {
		expect(formatDiscoveryDateTime('2026-06-14T23:00:00.000Z')).toContain('2026');
		expect(discoveryPricingLabel({ pricingMode: 'free', ticketPriceCents: 0, ticketCurrency: 'usd' } as never)).toBe('Free');
		expect(discoveryPricingLabel({ pricingMode: 'fixed', ticketPriceCents: 1800, ticketCurrency: 'usd' } as never)).toBe(
			new Intl.NumberFormat([], { style: 'currency', currency: 'USD' }).format(18),
		);
		expect(discoveryRemainingLabel({ isFull: false, remainingTickets: 1 } as never)).toBe('1 ticket left');
		expect(discoveryRemainingLabel({ isFull: true, remainingTickets: 0 } as never)).toBe('Sold out');
		expect(discoveryEmptyTitle('')).toBe('No published Events yet');
		expect(discoveryEmptyTitle('market')).toBe('No Events match that search yet');
		expect(discoveryEmptyBody('')).toBe('Published Events will appear here when Hosts share them.');
		expect(discoveryEmptyBody('market')).toBe('No published Events matched “market”. Try another search.');
		expect(discoveryEmptyStateCopy('')).toBe('Published Events will appear here when Hosts share them.');
		expect(discoveryEmptyStateCopy('market')).toBe('No published Events matched “market”. Try another search.');
		expect(discoveryErrorCopy('offline')).toBe('Could not load published Events: offline');
	});
});
