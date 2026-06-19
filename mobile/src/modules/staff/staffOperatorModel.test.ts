import { describe, expect, it } from 'vitest';

import { nextSelectedEvent, selectedEventLabel, selectedWorkspaceLabel, staffSelectionEmptyCopy, staffSelectionReady } from './staffOperatorModel';

describe('staffOperatorModel', () => {
	const workspace = { id: 'workspace-1', name: 'Main Workspace', role: 'owner' } as const;
	const event = { id: 'event-1', title: 'Night Market' } as never;

	it('labels selected Workspace and Event', () => {
		expect(selectedWorkspaceLabel(workspace)).toBe('Main Workspace');
		expect(selectedWorkspaceLabel(null)).toBe('Select a Workspace');
		expect(selectedEventLabel(event)).toBe('Night Market');
		expect(selectedEventLabel(null)).toBe('Select an Event');
	});

	it('reports readiness only when Workspace and Event are selected', () => {
		expect(staffSelectionReady(workspace, event)).toBe(true);
		expect(staffSelectionReady(workspace, null)).toBe(false);
		expect(staffSelectionReady(null, event)).toBe(false);
	});

	it('chooses the persisted Event when still available', () => {
		const events = [{ id: 'event-1', title: 'One' }, { id: 'event-2', title: 'Two' }] as never[];
		expect(nextSelectedEvent('event-2', events)?.id).toBe('event-2');
		expect(nextSelectedEvent('missing', events)?.id).toBe('event-1');
		expect(nextSelectedEvent(null, [])).toBeNull();
	});

	it('keeps empty-state copy stable', () => {
		expect(staffSelectionEmptyCopy([], [])).toBe('No Workspaces available for this account.');
		expect(staffSelectionEmptyCopy([workspace], [])).toBe('No Events available in this Workspace yet.');
	});
});
