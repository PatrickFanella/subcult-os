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
		expect(eventLifecycleTone('draft')).toBe('border-status-warning/30 bg-status-surface-warning text-status-warning');
		expect(eventLifecycleTone('published')).toBe('border-status-success/30 bg-status-surface-success text-status-success');
		expect(eventLifecycleTone('end_of_night')).toBe('border-status-info/30 bg-status-surface-info text-status-info');
		expect(eventLifecycleSurface('draft')).toBe('border-status-warning/20 bg-status-surface-warning');
		expect(eventLifecycleSurface('published')).toBe('border-status-success/20 bg-status-surface-success');
		expect(eventLifecycleSurface('end_of_night')).toBe('border-status-info/20 bg-status-surface-info');
		expect(eventLifecycleSummary('draft')).toBe('Private until the checklist is complete and the public page goes live.');
		expect(eventLifecycleSummary('published')).toBe('Live now. Keep the public page handy and end the night when the door closes.');
		expect(eventLifecycleSummary('end_of_night')).toBe('Closed out. Review the report and jump back to the workspace when you are done.');
	});
});
