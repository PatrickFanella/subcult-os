import { describe, expect, it } from 'vitest';
import type { PublicDiscoveryOccurrenceDTO } from '../../domain';
import {
	discoveryDetailPathFor,
	discoveryOccurrencesErrorCopy,
	formatOccurrenceDateTime,
	handoffButtonLabel,
	handoffLocalPath,
	handoffUnavailableReasonCopy,
	isOccurrenceUnavailable,
	occurrenceHasPublicCoordinates,
	occurrenceLocationSummary,
	occurrenceStatusLabel,
	projectOccurrencesToPlot,
} from './discoveryOccurrenceModel';

function occurrence(overrides: Partial<PublicDiscoveryOccurrenceDTO> = {}): PublicDiscoveryOccurrenceDTO {
	return {
		uri: 'at://did:plc:abc/tv.subcult.event.occurrence/1',
		source: { did: 'did:plc:abc', uri: 'at://did:plc:abc/tv.subcult.event.occurrence/1' },
		name: 'Test Occurrence',
		startsAt: '2026-10-01T20:00:00.000Z',
		status: 'scheduled',
		projectionStatus: 'active',
		handoff: { kind: 'none', reason: 'no_mapping' },
		...overrides,
	};
}

describe('discoveryOccurrenceModel', () => {
	it('flags non-active projection status as unavailable', () => {
		expect(isOccurrenceUnavailable(occurrence({ projectionStatus: 'active' }))).toBe(false);
		expect(isOccurrenceUnavailable(occurrence({ projectionStatus: 'deleted' }))).toBe(true);
		expect(isOccurrenceUnavailable(occurrence({ projectionStatus: 'unavailable' }))).toBe(true);
	});

	it('labels status from projection status first, then the record status', () => {
		expect(occurrenceStatusLabel(occurrence({ projectionStatus: 'deleted' }))).toBe('No longer published');
		expect(occurrenceStatusLabel(occurrence({ projectionStatus: 'unavailable' }))).toBe('Source unavailable');
		expect(occurrenceStatusLabel(occurrence({ status: 'cancelled' }))).toBe('Cancelled');
		expect(occurrenceStatusLabel(occurrence({ status: 'scheduled' }))).toBe('Scheduled');
	});

	it('converts the same instant into different event zones, including the calendar date', () => {
		const instant = '2026-10-01T20:00:00.000Z';
		const chicago = formatOccurrenceDateTime(instant, 'America/Chicago');
		const tokyo = formatOccurrenceDateTime(instant, 'Asia/Tokyo');
		const wallTime = (zone: string) => new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short', timeZone: zone }).format(new Date(instant));
		expect(chicago).toContain(wallTime('America/Chicago'));
		expect(tokyo).toContain(wallTime('Asia/Tokyo'));
		expect(chicago.split(' (')[0]).not.toBe(tokyo.split(' (')[0]);
		expect(chicago).toContain('America/Chicago');
		expect(tokyo).toContain('Asia/Tokyo');
	});

	it('distinguishes repeated fall-back wall times using their actual offsets', () => {
		const daylight = formatOccurrenceDateTime('2026-11-01T06:30:00Z', 'America/Chicago');
		const standard = formatOccurrenceDateTime('2026-11-01T07:30:00Z', 'America/Chicago');
		expect(daylight.split(' (')[0]).toBe(standard.split(' (')[0]);
		expect(daylight).not.toBe(standard);
	});

	it('skips the nonexistent spring-forward hour in the event zone', () => {
		const before = formatOccurrenceDateTime('2026-03-08T07:30:00Z', 'America/Chicago');
		const after = formatOccurrenceDateTime('2026-03-08T08:30:00Z', 'America/Chicago');
		const wall = (instant: string) => new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short', timeZone: 'America/Chicago' }).format(new Date(instant));
		expect(before).toContain(wall('2026-03-08T07:30:00Z'));
		expect(after).toContain(wall('2026-03-08T08:30:00Z'));
		expect(before.slice(before.lastIndexOf('('))).not.toBe(after.slice(after.lastIndexOf('(')));
	});

	it('uses explicit UTC for missing or invalid zones without endorsing the bad label', () => {
		const instant = '2026-10-01T20:00:00Z';
		const utc = new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short', timeZone: 'UTC' }).format(new Date(instant));
		for (const zone of [undefined, '', 'Invalid/Zone', ' America/Chicago ']) {
			const result = formatOccurrenceDateTime(instant, zone);
			expect(result).toBe(`${utc} (UTC; event time zone unavailable)`);
		}
		expect(formatOccurrenceDateTime('not-a-date', 'America/Chicago')).toBe('not-a-date');
	});

	it('summarizes a safe public location without ever needing a street address field', () => {
		expect(occurrenceLocationSummary(undefined)).toBe('Location to be announced');
		expect(occurrenceLocationSummary({ name: 'The Venue', locality: 'Chicago', region: 'IL', country: 'US' })).toBe('The Venue, Chicago, IL, US');
	});

	it('only treats numeric lat/lon pairs as public coordinates', () => {
		expect(occurrenceHasPublicCoordinates(undefined)).toBe(false);
		expect(occurrenceHasPublicCoordinates({ name: 'x' })).toBe(false);
		expect(occurrenceHasPublicCoordinates({ name: 'x', latitude: '41.8', longitude: '-87.6' })).toBe(true);
		expect(occurrenceHasPublicCoordinates({ name: 'x', latitude: 'nope', longitude: '-87.6' })).toBe(false);
	});

	it('projects only occurrences with public coordinates onto the plot', () => {
		const withLocation = occurrence({ uri: 'uri-a', location: { name: 'Venue', latitude: '0', longitude: '0' } });
		const withoutLocation = occurrence({ uri: 'uri-b' });
		const points = projectOccurrencesToPlot([withLocation, withoutLocation], 200, 100);
		expect(points).toHaveLength(1);
		expect(points[0]).toEqual({ uri: 'uri-a', x: 100, y: 50 });
	});

	it('labels the handoff button and never invents a fallback destination', () => {
		expect(handoffButtonLabel({ kind: 'local', eventSlug: 'night-market' })).toBe('Reserve');
		expect(handoffButtonLabel({ kind: 'none', reason: 'no_mapping' })).toBe('Reservation unavailable');
		expect(handoffLocalPath({ kind: 'local', eventSlug: 'night-market' })).toBe('/e/night-market');
		expect(handoffLocalPath({ kind: 'none', reason: 'no_mapping' })).toBeNull();
	});

	it('gives a specific unavailable-reason message per handoff reason', () => {
		expect(handoffUnavailableReasonCopy({ kind: 'none', reason: 'no_mapping' })).toMatch(/linked/);
		expect(handoffUnavailableReasonCopy({ kind: 'none', reason: 'event_not_published' })).toMatch(/published/);
		expect(handoffUnavailableReasonCopy({ kind: 'none' })).toMatch(/no reservation destination/i);
	});

	it('strips the at:// scheme for the detail path and leaves other input alone', () => {
		expect(discoveryDetailPathFor('at://did:plc:abc/tv.subcult.event.occurrence/1')).toBe('did:plc:abc/tv.subcult.event.occurrence/1');
		expect(discoveryDetailPathFor('did:plc:abc/tv.subcult.event.occurrence/1')).toBe('did:plc:abc/tv.subcult.event.occurrence/1');
	});

	it('produces a stable error copy fallback', () => {
		expect(discoveryOccurrencesErrorCopy(null)).toBe('Could not load discovery occurrences.');
		expect(discoveryOccurrencesErrorCopy('boom')).toBe('Could not load discovery occurrences: boom');
	});
});
