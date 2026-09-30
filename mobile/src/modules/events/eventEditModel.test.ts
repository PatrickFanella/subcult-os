import { describe, expect, it } from 'vitest';
import type { EventDTO } from '@/api/types';

import {
	buildEventEditPayload,
	createEmptyEventEditForm,
	createEventEditFormFromEvent,
	eventEditImageSelectionFromResult,
	eventEditReadinessWarnings,
	requireEventEditPayload,
} from './eventEditModel';

function event(startsAt: string): EventDTO {
	return { id: 'event-a', workspaceId: 'workspace-a', title: 'Synthetic event', startsAt,
		publicDescription: 'Description', locationDisplay: 'Room', imageUrl: null,
		ticketAllocation: 2, pricingMode: 'free', ticketPriceCents: 0, ticketCurrency: 'usd',
		reservedCount: 1, checkedInCount: 0, staffingOpenCount: 0, staffingAssignedCount: 0,
		staffingCompletedCount: 0, staffingCancelledCount: 0, status: 'published', publicSlug: 'event-a', publicUrl: '/e/event-a' };
}

describe('eventEditModel', () => {
	it.each(['2026-06-18T10:15:45.123456Z', '2026-11-01T07:30:45.123456Z', '2026-11-01T01:30:45.123456-05:00'])('preserves the exact hydrated start %s during an unrelated edit', startsAt => {
		const form = createEventEditFormFromEvent(event(startsAt));
		expect(buildEventEditPayload({ ...form, publicDescription: 'Updated description' })?.startsAt).toBe(startsAt);
	});

	it('parses a deliberately changed start instead of retaining the hydrated source', () => {
		const form = createEventEditFormFromEvent(event('2026-11-01T07:30:45.123456Z'));
		const changed = '2026-11-02 12:34';
		expect(buildEventEditPayload({ ...form, startsAt: changed })?.startsAt).toBe(new Date(changed.replace(' ', 'T')).toISOString());
		expect(buildEventEditPayload({ ...form, startsAt: changed })?.startsAt).not.toBe(form.startsAtSource);
	});
	it('does not retain an invalid source when the current input is valid', () => {
		const form = { ...createEmptyEventEditForm(), startsAt: '2026-11-02 12:34', startsAtSource: 'invalid' };
		expect(buildEventEditPayload(form)?.startsAt).toBe(new Date('2026-11-02T12:34').toISOString());
	});

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
