import type { EventStatus } from '@/api/types';

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
			return 'Live';
		case EVENT_STATUS_END_OF_NIGHT:
			return 'Closed';
	}
}
