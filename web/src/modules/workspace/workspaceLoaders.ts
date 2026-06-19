import { ApiError, api } from '../../api';
import type {
	CommitmentDTO,
	ContactDTO,
	CurrentUserDTO,
	CurrentWorkspaceDTO,
	DevEmailOutboxMessageDTO,
	EventTemplateDTO,
	ReminderEventDTO,
	WorkspaceArchiveSummaryDTO,
} from '../../domain';
import { normalizeCurrentWorkspace } from './workspaceModel';

type CurrentWorkspaceResponse = Omit<CurrentWorkspaceDTO, 'members' | 'invitations'> & {
	members?: CurrentWorkspaceDTO['members'] | null;
	invitations?: CurrentWorkspaceDTO['invitations'] | null;
};

export async function loadCurrentWorkspace() {
	return api<CurrentWorkspaceResponse>('/api/workspaces/current').catch(() => null);
}

export async function loadWorkspaceById(workspaceID: string) {
	return normalizeCurrentWorkspace(await api<CurrentWorkspaceResponse>(`/api/workspaces/${workspaceID}`));
}

export async function loadWorkspaceFallback(user: CurrentUserDTO) {
	const currentWorkspace = await loadCurrentWorkspace();
	if (currentWorkspace) {
		return { workspace: normalizeCurrentWorkspace(currentWorkspace), source: 'current' as const };
	}

	const fallback = user.workspaces[0];
	if (!fallback) {
		return null;
	}

	return {
		workspace: {
			id: fallback.id,
			name: fallback.name,
			role: fallback.role,
			members: [],
			invitations: [],
		},
		source: 'default' as const,
	};
}

export async function loadWorkspaceArchives(workspaceID: string, query: string, fallbackToEmpty = false) {
	const path = query ? `/api/workspaces/${workspaceID}/archives?q=${encodeURIComponent(query)}` : `/api/workspaces/${workspaceID}/archives`;
	try {
		return await api<WorkspaceArchiveSummaryDTO[]>(path);
	} catch (caught) {
		if (!fallbackToEmpty || query) {
			throw caught;
		}
		return [];
	}
}

export async function loadWorkspaceContacts(workspaceID: string) {
	try {
		return { data: await api<ContactDTO[]>(`/api/workspaces/${workspaceID}/contacts`), denied: false };
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 403) {
			return { data: null, denied: true };
		}
		throw caught;
	}
}

export async function loadWorkspaceCommitments(workspaceID: string) {
	try {
		return { data: await api<CommitmentDTO[]>(`/api/workspaces/${workspaceID}/commitments`), denied: false };
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 403) {
			return { data: null, denied: true };
		}
		throw caught;
	}
}

export async function loadWorkspaceTemplates(workspaceID: string) {
	try {
		return await api<EventTemplateDTO[]>(`/api/workspaces/${workspaceID}/event-templates`);
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 403) {
			return null;
		}
		throw caught;
	}
}

export async function loadWorkspaceReminders(workspaceID: string) {
	return api<ReminderEventDTO[]>(`/api/workspaces/${workspaceID}/reminders`);
}

export async function loadDevEmailOutbox() {
	try {
		const response = await fetch('/api/dev/email-outbox', { credentials: 'include' });

		if (response.status === 401 || response.status === 404 || !response.ok) {
			return null;
		}

		const data = await response.json().catch(() => null);
		if (!Array.isArray(data)) {
			return null;
		}

		return data as DevEmailOutboxMessageDTO[];
	} catch {
		return null;
	}
}
