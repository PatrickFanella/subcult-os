import { describe, expect, it } from 'vitest';

import {
	buildEventEditPayload,
	createEmptyEventEditForm,
	eventEditImageSelectionFromResult,
	eventEditReadinessWarnings,
	requireEventEditPayload,
} from './eventEditModel';

describe('eventEditModel', () => {
	it('builds a normalized payload', () => {
		const expectedStartsAt = new Date('2026-06-19 20:00').toISOString();
		const payload = buildEventEditPayload({
			...createEmptyEventEditForm(),
			title: ' Night Market ',
			startsAt: '2026-06-19 20:00',
			publicDescription: ' Doors at eight ',
			locationDisplay: ' Warehouse ',
			imageUrl: ' https://cdn.example.test/hero.jpg ',
			ticketAllocation: '25',
			pricingMode: 'fixed',
			ticketPriceDollars: '12.50',
			ticketCurrency: 'usd',
		});

		expect(payload).toMatchObject({
			title: 'Night Market',
			publicDescription: 'Doors at eight',
			locationDisplay: 'Warehouse',
			imageUrl: 'https://cdn.example.test/hero.jpg',
			ticketAllocation: 25,
			pricingMode: 'fixed',
			ticketPriceCents: 1250,
			ticketCurrency: 'USD',
		});
		expect(payload?.startsAt).toBe(expectedStartsAt);
	});

	it('reports readiness warnings for missing fields', () => {
		const warnings = eventEditReadinessWarnings({
			...createEmptyEventEditForm(),
			startsAt: 'not-a-date',
			ticketAllocation: '0',
			imageUrl: '',
			title: '',
			publicDescription: '',
			locationDisplay: '',
		}, null);
		expect(warnings).toEqual(
			expect.arrayContaining([
				'Add an event title.',
				'Use a valid start date/time.',
				'Add a public location.',
				'Add an attendee-facing description.',
				'Set ticket capacity greater than zero.',
				'Add a hero image for the public page.',
			]),
		);
	});

	it('maps an image picker result to selection state', () => {
		const selected = eventEditImageSelectionFromResult({
			canceled: false,
			assets: [{ uri: 'https://cdn.example.test/image.png' }],
		} as never);

		expect(selected).toEqual({
			selectedImage: { uri: 'https://cdn.example.test/image.png' },
			imageUrl: 'https://cdn.example.test/image.png',
			notice: 'Image selected. Save the event to upload it.',
		});
	});

	it('requires complete publishable payload', () => {
		expect(() => requireEventEditPayload({
			...createEmptyEventEditForm(),
			title: 'Ready',
			publicDescription: 'Desc',
			locationDisplay: 'Venue',
			startsAt: 'not-a-date',
			ticketAllocation: '0',
		})).toThrow('Check the date, capacity, and price fields.');
	});
});
