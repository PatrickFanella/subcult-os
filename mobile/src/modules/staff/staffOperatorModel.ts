import type { EventDTO, WorkspaceSummaryDTO } from '@/api/types';

export function selectedWorkspaceLabel(workspace: WorkspaceSummaryDTO | null) {
	return workspace?.name ?? 'Select a Workspace';
}

export function selectedEventLabel(event: EventDTO | null) {
	return event?.title ?? 'Select an Event';
}

export function staffSelectionReady(workspace: WorkspaceSummaryDTO | null, event: EventDTO | null) {
	return Boolean(workspace && event);
}

export function staffSelectionEmptyCopy(workspaces: WorkspaceSummaryDTO[], events: EventDTO[]) {
	if (workspaces.length === 0) return 'No Workspaces available for this account.';
	if (events.length === 0) return 'No Events available in this Workspace yet.';
	return 'Choose a Workspace and Event to begin.';
}

export function nextSelectedEvent(currentEventId: string | null, events: EventDTO[]) {
	if (currentEventId) {
		const match = events.find((event) => event.id === currentEventId);
		if (match) return match;
	}
	return events[0] ?? null;
}
