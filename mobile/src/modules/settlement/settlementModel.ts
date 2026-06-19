import type { EventStatus } from '@/api/types';

export function mobileCloseoutStatusLabel(status: EventStatus) {
	return status === 'end_of_night' ? 'Closed out' : 'Open';
}

export function mobileCloseoutHandoffCopy(status: EventStatus) {
	return status === 'end_of_night'
		? 'Review the full Event Report and Settlement on web.'
		: 'Close the Event from web when the room is done.';
}
