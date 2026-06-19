import type { PublicEventSummaryDTO } from '../../domain';

export const discoveryBrowseLabel = 'Public browse';
export const discoveryBadgeLabel = 'Discover events';
export const discoveryScopeLabel = 'Published only';
export const discoveryTitle = 'Discover events';
export const discoveryDescription = 'Browse published events without opening private workspace pages.';
export const discoverySearchLabel = 'Search published Events';
export const discoverySearchPlaceholder = 'Search published Events';
export const discoveryLoadingCopy = 'Loading published Events…';
export const discoveryViewEventLabel = 'View Event';

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
	return discoveryEmptyBody(searchQuery);
}

export function discoveryEmptyTitle(searchQuery: string) {
	return searchQuery.trim() ? 'No Events match that search yet' : 'No published Events yet';
}

export function discoveryEmptyBody(searchQuery: string) {
	const query = searchQuery.trim();
	return query ? `No published Events matched “${query}”. Try another search.` : 'Published Events will appear here when Hosts share them.';
}

export function discoveryErrorCopy(error?: string | null) {
	return error ? `Could not load published Events: ${error}` : 'Could not load published Events.';
}
