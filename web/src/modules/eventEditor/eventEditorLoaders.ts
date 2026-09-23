import { ApiError, api } from '../../api';
import type {
	CommitmentDTO,
	CurrentWorkspaceDTO,
	EventArchiveDTO,
	EventDTO,
	EventParticipantDTO,
	EventReportDTO,
	EventRoleApplicationDTO,
	EventRoleDTO,
	EventSettlementDTO,
	EventStaffingItemDTO,
	EventTemplateDTO,
	NotificationEventDTO,
	ReminderEventDTO,
} from '../../domain';
import { sortCommitments, sortTemplates } from './eventEditorModel';
import { sortRunOfShowItems } from '../runOfShow/runOfShowModel';

export type ApiClient = typeof api;

export type EventEditorCommitmentsLoadResult = {
	commitments: CommitmentDTO[] | null;
	denied: boolean;
};

export async function loadEventEditorEvent(apiClient: ApiClient, eventId: string) {
	return apiClient<EventDTO>(`/api/events/${eventId}`);
}

export async function loadEventEditorReport(apiClient: ApiClient, eventId: string) {
	try {
		return await apiClient<EventReportDTO>(`/api/events/${eventId}/report`);
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 404) return null;
		throw caught;
	}
}

export async function loadEventEditorArchive(apiClient: ApiClient, eventId: string) {
	try {
		return await apiClient<EventArchiveDTO>(`/api/events/${eventId}/archive`);
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 404) {
			return null;
		}

		throw caught;
	}
}

export async function loadEventEditorWorkspace(apiClient: ApiClient, workspaceId: string) {
	return apiClient<CurrentWorkspaceDTO>(`/api/workspaces/${workspaceId}`);
}

export async function loadEventEditorRoleApplications(apiClient: ApiClient, eventId: string) {
	const [roles, applications] = await Promise.all([
		apiClient<EventRoleDTO[]>(`/api/events/${eventId}/roles`),
		apiClient<EventRoleApplicationDTO[]>(`/api/events/${eventId}/role-applications`),
	]);

	return { roles, applications };
}

export async function loadEventEditorNotifications(apiClient: ApiClient, eventId: string) {
	try {
		return await apiClient<NotificationEventDTO[]>(`/api/events/${eventId}/notifications`);
	} catch (caught) {
		if (caught instanceof ApiError && (caught.status === 403 || caught.status === 404)) {
			return null;
		}

		throw caught;
	}
}

export async function loadEventEditorReminders(apiClient: ApiClient, eventId: string) {
	try {
		return await apiClient<ReminderEventDTO[]>(`/api/events/${eventId}/reminders`);
	} catch (caught) {
		if (caught instanceof ApiError && (caught.status === 403 || caught.status === 404)) {
			return null;
		}

		throw caught;
	}
}

export async function loadEventEditorParticipants(apiClient: ApiClient, eventId: string) {
	return apiClient<EventParticipantDTO[]>(`/api/events/${eventId}/participants`);
}

export async function loadEventEditorStaffing(apiClient: ApiClient, eventId: string) {
	return sortRunOfShowItems(await apiClient<EventStaffingItemDTO[]>(`/api/events/${eventId}/staffing`));
}

export async function loadEventEditorSettlement(apiClient: ApiClient, eventId: string) {
	try {
		return await apiClient<EventSettlementDTO>(`/api/events/${eventId}/settlement`);
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 404) {
			return null;
		}

		throw caught;
	}
}

export async function loadEventEditorCommitments(apiClient: ApiClient, eventId: string): Promise<EventEditorCommitmentsLoadResult> {
	try {
		return {
			commitments: sortCommitments(await apiClient<CommitmentDTO[]>(`/api/events/${eventId}/commitments`)),
			denied: false,
		};
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 403) {
			return { commitments: null, denied: true };
		}

		throw caught;
	}
}

export async function loadEventEditorTemplates(apiClient: ApiClient, workspaceId: string) {
	try {
		return sortTemplates(await apiClient<EventTemplateDTO[]>(`/api/workspaces/${workspaceId}/event-templates`));
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 403) {
			return null;
		}

		throw caught;
	}
}
