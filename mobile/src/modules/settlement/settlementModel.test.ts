import { describe, expect, it } from 'vitest';

import { mobileCloseoutHandoffCopy, mobileCloseoutStatusLabel } from './settlementModel';

describe('settlementModel', () => {
	it('labels closed Events for operator handoff', () => {
		expect(mobileCloseoutStatusLabel('end_of_night')).toBe('Closed out');
		expect(mobileCloseoutStatusLabel('published')).toBe('Open');
		expect(mobileCloseoutHandoffCopy('end_of_night')).toBe('Review the full Event Report and Settlement on web.');
		expect(mobileCloseoutHandoffCopy('draft')).toBe('Close the Event from web when the room is done.');
	});
});
