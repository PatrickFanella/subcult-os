import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	archiveLearningLoopCopy,
	commitmentStatusLabel,
	commitmentStatusTone,
	deleteTemplateState,
	eventCountLabel,
	eventStatusLabel,
	eventStatusSummary,
	eventStatusSurface,
	eventStatusTone,
	getRequestedArchiveQuery,
	getRequestedWorkspaceId,
	sortCommitments,
	sortContacts,
	sortTemplates,
	staffingStatusCopy,
} from './workspaceModel';

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('workspace model helpers', () => {
	it('sorts templates, contacts, and commitments consistently', () => {
		expect(
			sortTemplates([
				{ id: 'template-b', name: 'Bravo', createdAt: '2026-06-13T20:05:00.000Z' } as never,
				{ id: 'template-a', name: 'Alpha', createdAt: '2026-06-13T20:10:00.000Z' } as never,
				{ id: 'template-a-old', name: 'Alpha', createdAt: '2026-06-13T20:00:00.000Z' } as never,
			]).map((template) => template.id),
		).toEqual(['template-a-old', 'template-a', 'template-b']);

		expect(
			sortContacts([
				{ id: 'contact-b', displayName: 'Bravo', createdAt: '2026-06-13T20:05:00.000Z' } as never,
				{ id: 'contact-a', displayName: 'Alpha', createdAt: '2026-06-13T20:10:00.000Z' } as never,
				{ id: 'contact-a-old', displayName: 'Alpha', createdAt: '2026-06-13T20:00:00.000Z' } as never,
			]).map((contact) => contact.id),
		).toEqual(['contact-a-old', 'contact-a', 'contact-b']);

		expect(
			sortCommitments([
				{ id: 'done', status: 'done', dueAt: '2026-06-14T00:00:00.000Z', createdAt: '2026-06-13T20:05:00.000Z' } as never,
				{ id: 'open-late', status: 'open', dueAt: '2026-06-14T03:00:00.000Z', createdAt: '2026-06-13T20:10:00.000Z' } as never,
				{ id: 'open-early', status: 'open', dueAt: '2026-06-13T23:00:00.000Z', createdAt: '2026-06-13T20:00:00.000Z' } as never,
			]).map((commitment) => commitment.id),
		).toEqual(['open-early', 'open-late', 'done']);
	});

	it('derives counts, labels, and staffing copy', () => {
		expect(eventCountLabel({ reservedCount: 7, checkedInCount: 3 } as never)).toBe('Reserved 7 / Checked in 3');
		expect(eventStatusLabel('published')).toBe('Live');
		expect(eventStatusLabel('end_of_night')).toBe('Closed');
		expect(eventStatusSummary('draft')).toBe('Keep shaping the page, then publish when it is ready.');
		expect(eventStatusSummary('published')).toBe('Live now. Keep the Door open and wrap when the room closes.');
		expect(eventStatusSummary('end_of_night')).toBe('Closed out. Review the report and prep the next one.');
		expect(eventStatusTone('published')).toBe('border-emerald-400/25 bg-emerald-400/10 text-emerald-200');
		expect(eventStatusSurface('end_of_night')).toBe('border-fuchsia-400/20 bg-fuchsia-400/[0.06]');
		expect(staffingStatusCopy({ staffingOpenCount: 0, staffingAssignedCount: 0, staffingCompletedCount: 0, staffingCancelledCount: 0 } as never)).toBe('No staffing items yet.');
		expect(staffingStatusCopy({ staffingOpenCount: 1, staffingAssignedCount: 0, staffingCompletedCount: 0, staffingCancelledCount: 0 } as never)).toBe('Unresolved staffing remains before closeout.');
		expect(staffingStatusCopy({ staffingOpenCount: 0, staffingAssignedCount: 0, staffingCompletedCount: 2, staffingCancelledCount: 1 } as never)).toBe('All staffing complete.');
		expect(commitmentStatusLabel('open')).toBe('Open');
		expect(commitmentStatusTone('done')).toContain('emerald');
		expect(archiveLearningLoopCopy([])).toBe('Closed events will become private workspace memory here.');
	});

	it('reads workspace and archive query params', () => {
		vi.stubGlobal('window', { location: { search: '?workspaceId=workspace-1&q=doors' } });

		expect(getRequestedWorkspaceId()).toBe('workspace-1');
		expect(getRequestedArchiveQuery()).toBe('doors');
	});

	it('clears the active editor when deleting the active template', () => {
		const next = deleteTemplateState(
			{
				templates: [
					{ id: 'template-1', name: 'One', createdAt: '2026-06-13T20:00:00.000Z' } as never,
					{ id: 'template-2', name: 'Two', createdAt: '2026-06-13T21:00:00.000Z' } as never,
				],
				editingTemplateId: 'template-1',
				templateDeletingId: 'template-1',
			},
			'template-1',
		);

		expect(next.templates?.map((template) => template.id)).toEqual(['template-2']);
		expect(next.editingTemplateId).toBeNull();
		expect(next.templateDeletingId).toBeNull();
		expect(next.templateNotice).toBe('Template deleted.');
		expect(next.resetTemplateEditor).toBe(true);
	});

	it('keeps the editor open when deleting a different template', () => {
		const next = deleteTemplateState(
			{
				templates: [
					{ id: 'template-1', name: 'One', createdAt: '2026-06-13T20:00:00.000Z' } as never,
					{ id: 'template-2', name: 'Two', createdAt: '2026-06-13T21:00:00.000Z' } as never,
				],
				editingTemplateId: 'template-2',
				templateDeletingId: 'template-1',
			},
			'template-1',
		);

		expect(next.templates?.map((template) => template.id)).toEqual(['template-2']);
		expect(next.editingTemplateId).toBe('template-2');
		expect(next.templateDeletingId).toBeNull();
		expect(next.resetTemplateEditor).toBe(false);
	});
});
