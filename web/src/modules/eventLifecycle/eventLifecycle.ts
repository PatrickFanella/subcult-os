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
			return 'border-amber-400/30 bg-amber-400/10 text-amber-200';
		case EVENT_STATUS_PUBLISHED:
			return 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'border-fuchsia-400/30 bg-fuchsia-400/10 text-fuchsia-200';
	}
}

export function eventLifecycleSurface(status: EventStatus) {
	switch (status) {
		case EVENT_STATUS_DRAFT:
			return 'border-amber-400/20 bg-amber-400/[0.06]';
		case EVENT_STATUS_PUBLISHED:
			return 'border-emerald-400/20 bg-emerald-400/[0.06]';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'border-fuchsia-400/20 bg-fuchsia-400/[0.06]';
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
