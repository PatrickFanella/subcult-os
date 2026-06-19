import type { PublicEventSummaryDTO } from '../../domain';

export const discoveryBrowseLabel = 'Public browse';
export const discoveryBadgeLabel = 'Discover events';
export const discoveryScopeLabel = 'Published only';
export const discoveryTitle = 'Discover events';
export const discoveryDescription = 'Browse published events without opening private workspace pages.';
export const discoverySearchLabel = 'Search published events';
export const discoverySearchPlaceholder = 'Title, description, or location';
export const discoveryLoadingCopy = 'Loading published events…';
export const discoveryViewEventLabel = 'View event';

export function getRequestedDiscoveryQuery() {
	if (typeof window === 'undefined') {
		return '';
	}

	return new URLSearchParams(window.location.search).get('q') ?? '';
}

export function formatDiscoveryDateTime(value: string) {
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function formatDiscoveryCurrency(cents: number, currency: string) {
	return new Intl.NumberFormat([], { style: 'currency', currency: currency.toUpperCase() }).format(cents / 100);
}

export function discoveryPricingLabel(event: Pick<PublicEventSummaryDTO, 'pricingMode' | 'ticketPriceCents' | 'ticketCurrency'>) {
	return event.pricingMode === 'free' ? 'Free' : formatDiscoveryCurrency(event.ticketPriceCents, event.ticketCurrency);
}

export function discoveryRemainingLabel(event: Pick<PublicEventSummaryDTO, 'isFull' | 'remainingTickets'>) {
	if (event.isFull) {
		return 'Sold out';
	}

	return event.remainingTickets === 1 ? '1 ticket left' : `${event.remainingTickets} tickets left`;
}

export function discoveryEmptyStateCopy(searchQuery: string) {
	return searchQuery ? 'No events matched your search.' : 'No published events are discoverable yet.';
}

export function discoveryErrorCopy(error: string) {
	return `Could not load published events. ${error}`;
}
