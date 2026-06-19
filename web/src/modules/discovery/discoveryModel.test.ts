import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	discoveryDescription,
	discoveryEmptyStateCopy,
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
		expect(discoverySearchLabel).toBe('Search published events');
		expect(discoverySearchPlaceholder).toBe('Title, description, or location');
		expect(discoveryLoadingCopy).toBe('Loading published events…');
		expect(discoveryViewEventLabel).toBe('View event');
		expect(discoveryDescription).toBe('Browse published events without opening private workspace pages.');
	});

	it('formats discovery result labels without drifting copy', () => {
		expect(formatDiscoveryDateTime('2026-06-14T23:00:00.000Z')).toContain('2026');
		expect(discoveryPricingLabel({ pricingMode: 'free', ticketPriceCents: 0, ticketCurrency: 'usd' } as never)).toBe('Free');
		expect(discoveryPricingLabel({ pricingMode: 'fixed', ticketPriceCents: 1800, ticketCurrency: 'usd' } as never)).toBe('$18.00');
		expect(discoveryRemainingLabel({ isFull: false, remainingTickets: 1 } as never)).toBe('1 ticket left');
		expect(discoveryRemainingLabel({ isFull: true, remainingTickets: 0 } as never)).toBe('Sold out');
		expect(discoveryEmptyStateCopy('')).toBe('No published events are discoverable yet.');
		expect(discoveryEmptyStateCopy('market')).toBe('No events matched your search.');
	});
});
