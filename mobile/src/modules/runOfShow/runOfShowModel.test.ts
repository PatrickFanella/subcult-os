import { describe, expect, it } from 'vitest';

import {
	buildCreateRunOfShowPayload,
	buildUpdateRunOfShowPayload,
	isRunOfShowTimeRangeValid,
	parseOptionalDateTime,
	sortRunOfShowItems,
	staffingKindLabel,
	staffingStatusLabel,
} from './runOfShowModel';

describe('runOfShowModel', () => {
	it('parses optional datetimes', () => {
		expect(parseOptionalDateTime('')).toBeNull();
		expect(parseOptionalDateTime('not a date')).toBe(false);
		expect(parseOptionalDateTime('2026-06-18 09:30')).toBe(new Date('2026-06-18 09:30').toISOString());
	});

	it('sorts staffing items by status then time then id', () => {
		const items = sortRunOfShowItems([
			{ id: 'b', eventId: 'event-1', title: 'B', kind: 'task', notes: '', startsAt: '2026-06-18T10:00:00.000Z', endsAt: null, assignedPersonId: null, assignedApplicationId: null, assigneeName: null, status: 'assigned', createdAt: '2026-06-18T09:00:00.000Z', updatedAt: '2026-06-18T09:00:00.000Z', completedAt: null, completedByPersonId: null },
			{ id: 'a', eventId: 'event-1', title: 'A', kind: 'task', notes: '', startsAt: null, endsAt: null, assignedPersonId: null, assignedApplicationId: null, assigneeName: null, status: 'open', createdAt: '2026-06-18T09:00:00.000Z', updatedAt: '2026-06-18T09:00:00.000Z', completedAt: null, completedByPersonId: null },
		]);

		expect(items.map((item) => item.id)).toEqual(['a', 'b']);
	});

	it('builds create and update payloads', () => {
		expect(buildCreateRunOfShowPayload({ title: ' Setup ', kind: 'task', notes: ' Notes ', startsAt: '', endsAt: '' }, null, null)).toEqual({
			title: 'Setup',
			kind: 'task',
			notes: 'Notes',
			startsAt: null,
			endsAt: null,
		});
		expect(buildUpdateRunOfShowPayload({ title: ' Wrap ', notes: ' Done ' }, '2026-06-18T10:00:00.000Z', null)).toEqual({
			title: 'Wrap',
			notes: 'Done',
			startsAt: '2026-06-18T10:00:00.000Z',
			clearEndsAt: true,
		});
	});

	it('labels statuses and time ranges', () => {
		expect(staffingStatusLabel('open')).toBe('Open');
		expect(staffingKindLabel('shift')).toBe('Shift');
		expect(isRunOfShowTimeRangeValid('2026-06-18T09:00:00.000Z', '2026-06-18T10:00:00.000Z')).toBe(true);
		expect(isRunOfShowTimeRangeValid('2026-06-18T10:00:00.000Z', '2026-06-18T09:00:00.000Z')).toBe(false);
	});
});
