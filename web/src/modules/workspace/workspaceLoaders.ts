import { ApiError, api } from '../../api';
import type {
	CommitmentDTO,
	ContactDTO,
	CurrentUserDTO,
	CurrentWorkspaceDTO,
	DevEmailOutboxMessageDTO,
	EventTemplateDTO,
	EventDTO,
	ReminderEventDTO,
	WorkspaceArchiveSummaryDTO,
} from '../../domain';
import { normalizeCurrentWorkspace } from './workspaceModel';

type CurrentWorkspaceResponse = Omit<CurrentWorkspaceDTO, 'members' | 'invitations'> & {
	members?: CurrentWorkspaceDTO['members'] | null;
	invitations?: CurrentWorkspaceDTO['invitations'] | null;
};

export async function loadCurrentWorkspace() {
	try {
		return await api<CurrentWorkspaceResponse>('/api/workspaces/current');
	} catch (caught) {
		if (caught instanceof ApiError && caught.status === 404) return null;
		throw caught;
	}
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
		workspace: await loadWorkspaceById(fallback.id),
		source: 'default' as const,
	};
}

export async function selectWorkspace(user: CurrentUserDTO, requestedID: string | null) {
	if (requestedID) {
		try {
			return { workspace: await loadWorkspaceById(requestedID), notice: null };
		} catch (caught) {
			if (!(caught instanceof ApiError) || ![403, 404].includes(caught.status)) throw caught;
		}
	}
	const fallback = await loadWorkspaceFallback(user);
	if (!fallback) return null;
	return {
		workspace: fallback.workspace,
		notice: requestedID ? 'That Workspace is not available. Showing an accessible Workspace instead.' : null,
	};
}

export async function loadWorkspaceOverview(workspaceID: string, query: string) {
	const [events, archives, contacts, commitments] = await Promise.all([
		api<EventDTO[]>(`/api/workspaces/${workspaceID}/events`),
		loadWorkspaceArchives(workspaceID, query),
		loadWorkspaceContacts(workspaceID),
		loadWorkspaceCommitments(workspaceID),
	]);
	return { events, archives, contacts, commitments };
}

export async function loadWorkspaceArchives(workspaceID: string, query: string) {
	const path = query ? `/api/workspaces/${workspaceID}/archives?q=${encodeURIComponent(query)}` : `/api/workspaces/${workspaceID}/archives`;
	return api<WorkspaceArchiveSummaryDTO[]>(path);
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
