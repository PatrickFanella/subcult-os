import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api';
import type { CommitmentDTO, EventDTO, EventReportDTO, EventTemplateDTO } from '../../domain';

import {
	loadEventEditorArchive,
	loadEventEditorCommitments,
	loadEventEditorEvent,
	loadEventEditorNotifications,
	loadEventEditorReport,
	loadEventEditorReminders,
	loadEventEditorSettlement,
	loadEventEditorTemplates,
	loadEventEditorWorkspace,
	type ApiClient,
} from './eventEditorLoaders';

function event(overrides: Partial<EventDTO> = {}): EventDTO {
	return {
		id: 'event-1',
		workspaceId: 'workspace-1',
		title: 'Night Market',
		startsAt: '2026-06-18T10:15:00.000Z',
		publicDescription: 'A late set.',
		locationDisplay: 'The Hall',
		imageUrl: null,
		ticketAllocation: 100,
		pricingMode: 'free',
		ticketPriceCents: 0,
		ticketCurrency: 'usd',
		reservedCount: 26,
		checkedInCount: 20,
		staffingOpenCount: 1,
		staffingAssignedCount: 0,
		staffingCompletedCount: 0,
		staffingCancelledCount: 0,
		status: 'draft',
		publicSlug: null,
		publicUrl: null,
		...overrides,
	};
}

function commitment(overrides: Partial<CommitmentDTO>): CommitmentDTO {
	return {
		id: 'commitment-1',
		workspaceId: 'workspace-1',
		eventId: 'event-1',
		contactId: null,
		title: 'Check cables',
		description: 'Bring spares',
		dueAt: null,
		status: 'open',
		ownerPersonId: null,
		createdByPersonId: 'person-1',
		completedAt: null,
		completedByPersonId: null,
		createdAt: '2026-06-18T09:00:00.000Z',
		updatedAt: '2026-06-18T09:00:00.000Z',
		...overrides,
	};
}

function template(overrides: Partial<EventTemplateDTO>): EventTemplateDTO {
	return {
		id: 'template-1',
		workspaceId: 'workspace-1',
		name: 'Default',
		title: 'Night Market',
		publicDescription: 'A late set.',
		locationDisplay: 'The Hall',
		ticketAllocation: 100,
		pricingMode: 'free',
		ticketPriceCents: 0,
		ticketCurrency: 'usd',
		privateNotes: '',
		createdAt: '2026-06-18T09:00:00.000Z',
		updatedAt: '2026-06-18T09:00:00.000Z',
		...overrides,
	};
}

describe('event editor loaders', () => {
	it('loads the base event without blocking on reports', async () => {
		const apiClient = vi.fn(async (path: string) => {
			if (path === '/api/events/event-1') return event({ status: 'draft' });
			throw new Error(`unexpected request ${path}`);
		}) as unknown as ApiClient;

		await expect(loadEventEditorEvent(apiClient, 'event-1')).resolves.toEqual(event({ status: 'draft' }));
		expect((apiClient as unknown as { mock: { calls: unknown[][] } }).mock.calls.map(([path]) => path)).toEqual(['/api/events/event-1']);
	});

	it('loads optional reports through a separate non-blocking loader', async () => {
		const closedApiClient = vi.fn(async (path: string) => {
			if (path === '/api/events/event-1/report') return { id: 'report-1', title: 'End of night', eventId: 'event-1', startsAt: '2026-06-18T10:15:00.000Z', generatedAt: '2026-06-18T11:00:00.000Z', generatedByMemberEmail: 'owner@example.com', ticketsReserved: 1, ticketsCheckedIn: 1, noShows: 0, ticketAllocation: 100, publicUrl: '/e/night-market' } satisfies EventReportDTO;
			throw new Error(`unexpected request ${path}`);
		}) as unknown as ApiClient;

		await expect(loadEventEditorReport(closedApiClient, 'event-1')).resolves.toEqual({ data: {
				id: 'report-1',
				title: 'End of night',
				eventId: 'event-1',
				startsAt: '2026-06-18T10:15:00.000Z',
				generatedAt: '2026-06-18T11:00:00.000Z',
				generatedByMemberEmail: 'owner@example.com',
				ticketsReserved: 1,
				ticketsCheckedIn: 1,
				noShows: 0,
				ticketAllocation: 100,
				publicUrl: '/e/night-market',
		}, denied: false });
		expect((closedApiClient as unknown as { mock: { calls: unknown[][] } }).mock.calls.map(([path]) => path)).toEqual(['/api/events/event-1/report']);

		await expect(loadEventEditorReport((async () => {
			throw new ApiError(404, 'report not found', {});
		}) as unknown as ApiClient, 'event-1')).resolves.toEqual({ data: null, denied: false });
	});

	it.each([401, 429, 500, 503])('preserves report and authority errors (%s)', async (status) => {
		const error = new ApiError(status, 'Unavailable', {});
		const client = vi.fn().mockRejectedValue(error) as ApiClient;
		await expect(loadEventEditorReport(client, 'event-1')).rejects.toBe(error);
		await expect(loadEventEditorWorkspace(client, 'workspace-1')).rejects.toBe(error);
	});

	it('clears private finance panels when the server denies finance access', async () => {
		const denied = new ApiError(403, 'forbidden', {});
		const client = vi.fn().mockRejectedValue(denied) as ApiClient;

		await expect(loadEventEditorReport(client, 'event-1')).resolves.toEqual({ data: null, denied: true });
		await expect(loadEventEditorSettlement(client, 'event-1')).resolves.toEqual({ data: null, denied: true });
	});

	it('does not hide a missing workspace or a transport failure', async () => {
		const missing = new ApiError(404, 'Workspace not found', {});
		await expect(loadEventEditorWorkspace(vi.fn().mockRejectedValue(missing) as ApiClient, 'workspace-1')).rejects.toBe(missing);
		const error = new TypeError('Network unavailable');
		const client = vi.fn().mockRejectedValue(error) as ApiClient;
		await expect(loadEventEditorReport(client, 'event-1')).rejects.toBe(error);
		await expect(loadEventEditorWorkspace(client, 'workspace-1')).rejects.toBe(error);
	});

	it('normalizes missing resources and orders returned collections', async () => {
		await expect(loadEventEditorArchive((async () => {
			throw new ApiError(404, 'not found', {});
		}) as unknown as ApiClient, 'event-1')).resolves.toBeNull();

		await expect(loadEventEditorCommitments((async (path: string) => {
			if (path === '/api/events/event-1/commitments') {
				return [
					commitment({ id: 'done', status: 'done', dueAt: '2026-06-18T10:00:00.000Z', createdAt: '2026-06-18T08:00:00.000Z' }),
					commitment({ id: 'open-late', status: 'open', dueAt: '2026-06-18T11:00:00.000Z', createdAt: '2026-06-18T10:00:00.000Z' }),
					commitment({ id: 'open-early', status: 'open', dueAt: '2026-06-18T09:00:00.000Z', createdAt: '2026-06-18T09:00:00.000Z' }),
				];
			}
			throw new Error(`unexpected request ${path}`);
		}) as unknown as ApiClient, 'event-1')).resolves.toEqual({
			commitments: [
				commitment({ id: 'open-early', status: 'open', dueAt: '2026-06-18T09:00:00.000Z', createdAt: '2026-06-18T09:00:00.000Z' }),
				commitment({ id: 'open-late', status: 'open', dueAt: '2026-06-18T11:00:00.000Z', createdAt: '2026-06-18T10:00:00.000Z' }),
				commitment({ id: 'done', status: 'done', dueAt: '2026-06-18T10:00:00.000Z', createdAt: '2026-06-18T08:00:00.000Z' }),
			],
			denied: false,
		});

		await expect(loadEventEditorCommitments((async () => {
			throw new ApiError(403, 'forbidden', {});
		}) as unknown as ApiClient, 'event-1')).resolves.toEqual({ commitments: null, denied: true });

		await expect(loadEventEditorTemplates((async (path: string) => {
			if (path === '/api/workspaces/workspace-1/event-templates') {
				return [
					template({ id: 'b', name: 'Beta', createdAt: '2026-06-18T09:00:00.000Z' }),
					template({ id: 'a-new', name: 'Alpha', createdAt: '2026-06-18T10:00:00.000Z' }),
					template({ id: 'a-old', name: 'Alpha', createdAt: '2026-06-18T08:00:00.000Z' }),
				];
			}
			throw new Error(`unexpected request ${path}`);
		}) as unknown as ApiClient, 'workspace-1')).resolves.toEqual([
			template({ id: 'a-old', name: 'Alpha', createdAt: '2026-06-18T08:00:00.000Z' }),
			template({ id: 'a-new', name: 'Alpha', createdAt: '2026-06-18T10:00:00.000Z' }),
			template({ id: 'b', name: 'Beta', createdAt: '2026-06-18T09:00:00.000Z' }),
		]);

		await expect(loadEventEditorNotifications((async () => {
			throw new ApiError(404, 'not found', {});
		}) as unknown as ApiClient, 'event-1')).resolves.toBeNull();
		await expect(loadEventEditorReminders((async () => {
			throw new ApiError(403, 'forbidden', {});
		}) as unknown as ApiClient, 'event-1')).resolves.toBeNull();
	});
});
