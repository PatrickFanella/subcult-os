import type { CurrentWorkspaceDTO, CommitmentDTO, ContactDTO, EventDTO, EventStatus, EventTemplateDTO, InvitationDTO, MemberDTO, WorkspaceArchiveSummaryDTO } from '../../domain';

type CurrentWorkspaceResponse = Omit<CurrentWorkspaceDTO, 'members' | 'invitations'> & {
	members?: MemberDTO[] | null;
	invitations?: InvitationDTO[] | null;
};

export type TemplateDeletionState = {
	templates: EventTemplateDTO[] | null;
	editingTemplateId: string | null;
	templateDeletingId: string | null;
};

export type TemplateDeletionResult = {
	templates: EventTemplateDTO[] | null;
	editingTemplateId: string | null;
	templateDeletingId: string | null;
	templateNotice: string;
	resetTemplateEditor: boolean;
};

export function normalizeCurrentWorkspace(workspace: CurrentWorkspaceResponse): CurrentWorkspaceDTO {
	return {
		...workspace,
		members: workspace.members ?? [],
		invitations: workspace.invitations ?? [],
	};
}

export function getRequestedWorkspaceId() {
	if (typeof window === 'undefined') {
		return null;
	}

	return new URLSearchParams(window.location.search).get('workspaceId');
}

export function getRequestedArchiveQuery() {
	if (typeof window === 'undefined') {
		return '';
	}

	return new URLSearchParams(window.location.search).get('q') ?? '';
}

export function emptyContactForm() {
	return { displayName: '', email: '', phone: '', tags: '', notes: '' };
}

export function contactFormFrom(contact: ContactDTO) {
	return {
		displayName: contact.displayName,
		email: contact.email ?? '',
		phone: contact.phone ?? '',
		tags: contact.tags.join(', '),
		notes: contact.notes,
	};
}

export function emptyCommitmentForm() {
	return { title: '', description: '', dueAt: '', eventId: '' };
}

export function emptyTemplateForm() {
	return {
		name: '',
		title: '',
		publicDescription: '',
		locationDisplay: '',
		ticketAllocation: '1',
		pricingMode: 'free' as const,
		ticketPriceDollars: '0.00',
		privateNotes: '',
	};
}

export function templateFormFrom(template: EventTemplateDTO) {
	return {
		name: template.name,
		title: template.title,
		publicDescription: template.publicDescription,
		locationDisplay: template.locationDisplay,
		ticketAllocation: String(template.ticketAllocation),
		pricingMode: template.pricingMode,
		ticketPriceDollars: template.pricingMode === 'fixed' ? (template.ticketPriceCents / 100).toFixed(2) : '0.00',
		privateNotes: template.privateNotes,
	};
}

export function sortTemplates(templates: EventTemplateDTO[]) {
	return [...templates].sort((left, right) => left.name.localeCompare(right.name) || new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime());
}

export function templatePricingLabel(template: EventTemplateDTO) {
	if (template.pricingMode === 'free') {
		return 'Free reservation';
	}

	const currency = template.ticketCurrency.toUpperCase() || 'USD';
	return `${new Intl.NumberFormat([], { style: 'currency', currency }).format(template.ticketPriceCents / 100)} ${currency}`;
}

export function sortContacts(contacts: ContactDTO[]) {
	return [...contacts].sort((left, right) => left.displayName.localeCompare(right.displayName) || new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime());
}

export function sortCommitments(commitments: CommitmentDTO[]) {
	const statusOrder: Record<CommitmentDTO['status'], number> = { open: 0, done: 1, cancelled: 2 };

	return [...commitments].sort((left, right) => {
		const statusDelta = statusOrder[left.status] - statusOrder[right.status];
		if (statusDelta !== 0) return statusDelta;

		const leftDue = left.dueAt ? new Date(left.dueAt).getTime() : Number.POSITIVE_INFINITY;
		const rightDue = right.dueAt ? new Date(right.dueAt).getTime() : Number.POSITIVE_INFINITY;
		if (leftDue !== rightDue) return leftDue - rightDue;

		return new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime();
	});
}

export function commitmentStatusLabel(status: CommitmentDTO['status']) {
	switch (status) {
		case 'open':
			return 'Open';
		case 'done':
			return 'Done';
		case 'cancelled':
			return 'Cancelled';
	}
}

export function commitmentStatusTone(status: CommitmentDTO['status']) {
	switch (status) {
		case 'open':
			return 'border-amber-400/20 bg-amber-400/10 text-amber-200';
		case 'done':
			return 'border-emerald-400/20 bg-emerald-400/10 text-emerald-200';
		case 'cancelled':
			return 'border-rose-400/20 bg-rose-400/10 text-rose-200';
	}
}

export function eventCountLabel(event: EventDTO) {
	return `Reserved ${event.reservedCount} / Checked in ${event.checkedInCount}`;
}

export function eventStatusLabel(status: EventStatus) {
	switch (status) {
		case 'draft':
			return 'Draft';
		case 'published':
			return 'Live';
		case 'end_of_night':
			return 'Closed';
	}
}

export function eventStatusTone(status: EventStatus) {
	switch (status) {
		case 'draft':
			return 'border-amber-400/25 bg-amber-400/10 text-amber-200';
		case 'published':
			return 'border-emerald-400/25 bg-emerald-400/10 text-emerald-200';
		case 'end_of_night':
			return 'border-fuchsia-400/25 bg-fuchsia-400/10 text-fuchsia-200';
	}
}

export function eventStatusSurface(status: EventStatus) {
	switch (status) {
		case 'draft':
			return 'border-amber-400/20 bg-amber-400/[0.06]';
		case 'published':
			return 'border-emerald-400/20 bg-emerald-400/[0.06]';
		case 'end_of_night':
			return 'border-fuchsia-400/20 bg-fuchsia-400/[0.06]';
	}
}

export function eventStatusSummary(status: EventStatus) {
	switch (status) {
		case 'draft':
			return 'Keep shaping the page, then publish when it is ready.';
		case 'published':
			return 'Live now. Keep the Door open and wrap when the room closes.';
		case 'end_of_night':
			return 'Closed out. Review the report and prep the next one.';
	}
}

export function staffingStatusCopy(event: EventDTO) {
	const staffingTotal = event.staffingOpenCount + event.staffingAssignedCount + event.staffingCompletedCount + event.staffingCancelledCount;
	const unresolvedCount = event.staffingOpenCount + event.staffingAssignedCount;

	if (staffingTotal === 0) {
		return 'No staffing items yet.';
	}

	if (unresolvedCount > 0) {
		return 'Unresolved staffing remains before closeout.';
	}

	return 'All staffing complete.';
}

export function archiveLearningLoopCopy(archives: WorkspaceArchiveSummaryDTO[]) {
	const hasArchives = archives.length > 0;
	const hasNotes = archives.some((archive) => archive.noteCount > 0);
	const hasSeededDraft = archives.some((archive) => Boolean(archive.seededEventId));

	if (!hasArchives) {
		return 'Closed events will become private workspace memory here.';
	}

	if (!hasNotes) {
		return 'Open an archive and capture the first lesson.';
	}

	if (!hasSeededDraft) {
		return 'Use lessons to seed the next draft.';
	}

	return 'Review the seeded draft before publishing.';
}

export function deleteTemplateState(current: TemplateDeletionState, templateID: string): TemplateDeletionResult {
	return {
		templates: current.templates?.filter((template) => template.id !== templateID) ?? null,
		editingTemplateId: current.editingTemplateId === templateID ? null : current.editingTemplateId,
		templateDeletingId: current.templateDeletingId === templateID ? null : current.templateDeletingId,
		templateNotice: 'Template deleted.',
		resetTemplateEditor: current.editingTemplateId === templateID,
	};
}
