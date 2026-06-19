import { describe, expect, it } from 'vitest';

import {
	eventLifecycleLabel,
	eventLifecycleSummary,
	eventLifecycleSurface,
	eventLifecycleTone,
	isClosedEvent,
	isDraftEvent,
	isPublishedEvent,
} from './eventLifecycle';

describe('eventLifecycle', () => {
	it('classifies event statuses', () => {
		expect(isDraftEvent('draft')).toBe(true);
		expect(isPublishedEvent('published')).toBe(true);
		expect(isClosedEvent('end_of_night')).toBe(true);
		expect(isDraftEvent('published')).toBe(false);
	});

	it('provides lifecycle copy', () => {
		expect(eventLifecycleLabel('draft')).toBe('Draft');
		expect(eventLifecycleLabel('published')).toBe('Published');
		expect(eventLifecycleLabel('end_of_night')).toBe('End of Night');
		expect(eventLifecycleTone('draft')).toBe('border-amber-400/30 bg-amber-400/10 text-amber-200');
		expect(eventLifecycleTone('published')).toBe('border-emerald-400/30 bg-emerald-400/10 text-emerald-200');
		expect(eventLifecycleTone('end_of_night')).toBe('border-fuchsia-400/30 bg-fuchsia-400/10 text-fuchsia-200');
		expect(eventLifecycleSurface('draft')).toBe('border-amber-400/20 bg-amber-400/[0.06]');
		expect(eventLifecycleSurface('published')).toBe('border-emerald-400/20 bg-emerald-400/[0.06]');
		expect(eventLifecycleSurface('end_of_night')).toBe('border-fuchsia-400/20 bg-fuchsia-400/[0.06]');
		expect(eventLifecycleSummary('draft')).toBe('Private until the checklist is complete and the public page goes live.');
		expect(eventLifecycleSummary('published')).toBe('Live now. Keep the public page handy and end the night when the door closes.');
		expect(eventLifecycleSummary('end_of_night')).toBe('Closed out. Review the report and jump back to the workspace when you are done.');
	});
});
