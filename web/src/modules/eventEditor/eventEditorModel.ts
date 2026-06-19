import type {
	CommitmentDTO,
	EventDTO,
	EventRoleApplicationDTO,
	EventRoleDTO,
	EventStaffingItemDTO,
	EventTemplateDTO,
} from '../../domain';
import { sortRunOfShowItems } from '../runOfShow/runOfShowModel';

export type FormState = {
	title: string;
	startsAt: string;
	publicDescription: string;
	locationDisplay: string;
	ticketAllocation: string;
	pricingMode: 'free' | 'fixed';
	ticketPriceDollars: string;
};

export type SettlementAdjustmentFormState = {
	amountDollars: string;
	label: string;
	reason: string;
};

export type CommitmentFormState = {
	title: string;
	description: string;
	dueAt: string;
};

export const applicationReviewStatusOptions: { value: EventRoleApplicationDTO['status']; label: string }[] = [
	{ value: 'submitted', label: 'Submitted' },
	{ value: 'under_review', label: 'Under review' },
	{ value: 'accepted', label: 'Accepted' },
	{ value: 'waitlisted', label: 'Waitlisted' },
	{ value: 'rejected', label: 'Rejected' },
	{ value: 'withdrawn', label: 'Withdrawn' },
	{ value: 'confirmed', label: 'Confirmed' },
];

export function isNewEvent(eventId: string) {
	return eventId === '' || eventId === 'new';
}

export function getWorkspaceId() {
	if (typeof window === 'undefined') {
		return '';
	}

	return new URLSearchParams(window.location.search).get('workspaceId') ?? '';
}

export function toInputValue(value: string) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return '';
	}

	const offset = date.getTimezoneOffset();
	return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}

export function fromInputValue(value: string) {
	return new Date(value).toISOString();
}

export function toRfc3339DateTime(value: string) {
	if (!value) {
		return undefined;
	}

	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

export function emptyForm(): FormState {
	return {
		title: '',
		startsAt: '',
		publicDescription: '',
		locationDisplay: '',
		ticketAllocation: '1',
		pricingMode: 'free',
		ticketPriceDollars: '0.00',
	};
}

export function formFromEvent(event: EventDTO): FormState {
	return {
		title: event.title,
		startsAt: toInputValue(event.startsAt),
		publicDescription: event.publicDescription,
		locationDisplay: event.locationDisplay,
		ticketAllocation: String(event.ticketAllocation),
		pricingMode: event.pricingMode,
		ticketPriceDollars: (event.ticketPriceCents / 100).toFixed(2),
	};
}

export function formsMatch(left: FormState, right: FormState) {
	return (
		left.title === right.title &&
		left.startsAt === right.startsAt &&
		left.publicDescription === right.publicDescription &&
		left.locationDisplay === right.locationDisplay &&
		left.ticketAllocation === right.ticketAllocation &&
		left.pricingMode === right.pricingMode &&
		left.ticketPriceDollars === right.ticketPriceDollars
	);
}

export function formatMoney(cents: number, currency: string) {
	const normalizedCurrency = currency.trim().toUpperCase() || 'USD';
	return `${new Intl.NumberFormat([], { style: 'currency', currency: normalizedCurrency }).format(cents / 100)} ${normalizedCurrency}`;
}

export function formatSignedMoney(cents: number, currency: string) {
	const sign = cents < 0 ? '-' : '+';
	return `${sign}${formatMoney(Math.abs(cents), currency)}`;
}

export function priceInCents(value: string) {
	const parsed = Number(value);
	if (Number.isNaN(parsed)) {
		return 0;
	}

	return Math.round(parsed * 100);
}

export function formatDateTime(value: string) {
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function emptySettlementAdjustmentForm(): SettlementAdjustmentFormState {
	return { amountDollars: '', label: '', reason: '' };
}

export function emptyCommitmentForm(): CommitmentFormState {
	return { title: '', description: '', dueAt: '' };
}

export function sortTemplates(templates: EventTemplateDTO[]) {
	return [...templates].sort(
		(left, right) => left.name.localeCompare(right.name) || new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
	);
}

export function sortCommitments(items: CommitmentDTO[]) {
	const statusOrder: Record<CommitmentDTO['status'], number> = { open: 0, done: 1, cancelled: 2 };

	return [...items].sort((left, right) => {
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

export function pricingSummary(event: EventDTO | null) {
	if (!event || event.pricingMode === 'free') {
		return 'Free reservation';
	}

	return formatMoney(event.ticketPriceCents, event.ticketCurrency);
}

export function buildPayload(form: FormState) {
	return {
		title: form.title.trim(),
		startsAt: fromInputValue(form.startsAt),
		publicDescription: form.publicDescription.trim(),
		locationDisplay: form.locationDisplay.trim(),
		ticketAllocation: Number(form.ticketAllocation),
		pricingMode: form.pricingMode,
		ticketPriceCents: form.pricingMode === 'fixed' ? priceInCents(form.ticketPriceDollars) : 0,
		ticketCurrency: 'usd',
	};
}

export function buildRoleNameById(roles: EventRoleDTO[] | null) {
	return new Map<string, string>((roles ?? []).map((role) => [role.id, role.name] as [string, string]));
}

export function buildStaffingCounts(items: EventStaffingItemDTO[] | null) {
	return (items ?? []).reduce(
		(counts, item) => ({
			...counts,
			[item.status]: counts[item.status] + 1,
		}),
		{ open: 0, assigned: 0, completed: 0, cancelled: 0 },
	);
}

export function buildStaffingGroups(items: EventStaffingItemDTO[] | null): Array<{ kind: 'task' | 'shift'; label: string; items: EventStaffingItemDTO[] }> {
	const staffingItems = items ?? [];
	const staffingTasks = sortRunOfShowItems(staffingItems.filter((item) => item.kind === 'task'));
	const staffingShifts = sortRunOfShowItems(staffingItems.filter((item) => item.kind === 'shift'));

	return [
		{ kind: 'task', label: 'Tasks', items: staffingTasks },
		{ kind: 'shift', label: 'Shifts', items: staffingShifts },
	];
}

export function buildCommitmentCounts(items: CommitmentDTO[] | null) {
	return (items ?? []).reduce(
		(counts, commitment) => ({
			...counts,
			[commitment.status]: counts[commitment.status] + 1,
		}),
		{ open: 0, done: 0, cancelled: 0 },
	);
}
