import type { EventStatus } from '../../domain';

export const EVENT_STATUS_DRAFT = 'draft' as const;
export const EVENT_STATUS_PUBLISHED = 'published' as const;
export const EVENT_STATUS_END_OF_NIGHT = 'end_of_night' as const;

export function isDraftEvent(status: EventStatus | null | undefined) {
	return status === EVENT_STATUS_DRAFT;
}

export function isPublishedEvent(status: EventStatus | null | undefined) {
	return status === EVENT_STATUS_PUBLISHED;
}

export function isClosedEvent(status: EventStatus | null | undefined) {
	return status === EVENT_STATUS_END_OF_NIGHT;
}

export function eventLifecycleLabel(status: EventStatus) {
	switch (status) {
		case EVENT_STATUS_DRAFT:
			return 'Draft';
		case EVENT_STATUS_PUBLISHED:
			return 'Published';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'End of Night';
	}
}

export function eventLifecycleTone(status: EventStatus) {
	switch (status) {
		case EVENT_STATUS_DRAFT:
			return 'border-status-warning/30 bg-status-surface-warning text-status-warning';
		case EVENT_STATUS_PUBLISHED:
			return 'border-status-success/30 bg-status-surface-success text-status-success';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'border-status-info/30 bg-status-surface-info text-status-info';
	}
}

export function eventLifecycleSurface(status: EventStatus) {
	switch (status) {
		case EVENT_STATUS_DRAFT:
			return 'border-status-warning/20 bg-status-surface-warning';
		case EVENT_STATUS_PUBLISHED:
			return 'border-status-success/20 bg-status-surface-success';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'border-status-info/20 bg-status-surface-info';
	}
}

export function eventLifecycleSummary(status: EventStatus) {
	switch (status) {
		case EVENT_STATUS_DRAFT:
			return 'Private until the checklist is complete and the public page goes live.';
		case EVENT_STATUS_PUBLISHED:
			return 'Live now. Keep the public page handy and end the night when the door closes.';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'Closed out. Review the report and jump back to the workspace when you are done.';
	}
}
