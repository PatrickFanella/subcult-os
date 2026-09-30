import type { EventWritePayload } from '@/api/staff';
import type { EventDTO, PricingMode } from '@/api/types';

export type EventEditImageAsset = {
	uri: string;
};

export type EventEditImagePickerResult<TAsset extends EventEditImageAsset = EventEditImageAsset> = {
	canceled: boolean;
	assets?: TAsset[] | null;
};

export type EventEditFormState = {
	title: string;
	startsAt: string;
	startsAtSource?: string;
	publicDescription: string;
	locationDisplay: string;
	imageUrl: string;
	ticketAllocation: string;
	pricingMode: PricingMode;
	ticketPriceDollars: string;
	ticketCurrency: string;
};

export type EventEditImageSelection<TAsset extends EventEditImageAsset = EventEditImageAsset> = {
	selectedImage: TAsset;
	imageUrl: string;
	notice: string;
};

export function createDefaultStartsAtInput() {
	const date = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
	date.setMinutes(0, 0, 0);
	return toLocalInput(date.toISOString());
}

export function createEmptyEventEditForm(): EventEditFormState {
	return {
		title: '',
		startsAt: createDefaultStartsAtInput(),
		publicDescription: '',
		locationDisplay: '',
		imageUrl: '',
		ticketAllocation: '50',
		pricingMode: 'free',
		ticketPriceDollars: '0.00',
		ticketCurrency: 'USD',
	};
}

export function createEventEditFormFromEvent(event: EventDTO): EventEditFormState {
	return {
		title: event.title,
		startsAt: toLocalInput(event.startsAt),
		startsAtSource: event.startsAt,
		publicDescription: event.publicDescription,
		locationDisplay: event.locationDisplay,
		imageUrl: event.imageUrl ?? '',
		ticketAllocation: String(event.ticketAllocation),
		pricingMode: event.pricingMode,
		ticketPriceDollars: (event.ticketPriceCents / 100).toFixed(2),
		ticketCurrency: event.ticketCurrency || 'USD',
	};
}

export function buildEventEditPayload(form: EventEditFormState): EventWritePayload | null {
	const source = form.startsAtSource;
	const hydratedInput = source ? toLocalInput(source) : '';
	// Unchanged local minute text must not reselect a DST offset or truncate
	// the server's seconds/fractions during an unrelated edit.
	const startsAt = source && hydratedInput && hydratedInput === form.startsAt ? source : parseEventEditStartsAt(form.startsAt);
	const ticketAllocation = Number.parseInt(form.ticketAllocation, 10);
	if (!startsAt || Number.isNaN(ticketAllocation)) return null;
	const ticketPriceCents = form.pricingMode === 'free' ? 0 : Math.round(Number.parseFloat(form.ticketPriceDollars || '0') * 100);
	if (Number.isNaN(ticketPriceCents)) return null;
	return {
		title: form.title.trim(),
		startsAt,
		publicDescription: form.publicDescription.trim(),
		locationDisplay: form.locationDisplay.trim(),
		imageUrl: form.imageUrl.trim(),
		ticketAllocation,
		pricingMode: form.pricingMode,
		ticketPriceCents,
		ticketCurrency: form.ticketCurrency.trim().toUpperCase() || 'USD',
	};
}

export function requireEventEditPayload(form: EventEditFormState): EventWritePayload {
	const payload = buildEventEditPayload(form);
	if (!payload) throw new Error('Check the date, capacity, and price fields.');
	if (!payload.title || !payload.publicDescription || !payload.locationDisplay) throw new Error('Title, description, and location are required.');
	if (payload.ticketAllocation <= 0) throw new Error('Capacity must be greater than zero to publish.');
	return payload;
}

export function eventEditReadinessWarnings(form: EventEditFormState, event: EventDTO | null) {
	const warnings: string[] = [];
	const payload = buildEventEditPayload(form);
	if (!payload?.title) warnings.push('Add an event title.');
	if (!payload?.startsAt) warnings.push('Use a valid start date/time.');
	if (!payload?.locationDisplay) warnings.push('Add a public location.');
	if (!payload?.publicDescription) warnings.push('Add an attendee-facing description.');
	if (!payload || payload.ticketAllocation <= 0) warnings.push('Set ticket capacity greater than zero.');
	if (payload?.pricingMode === 'fixed' && payload.ticketPriceCents <= 0) warnings.push('Set a paid ticket price greater than zero.');
	if (!form.imageUrl.trim() && !event?.imageUrl) warnings.push('Add a hero image for the public page.');
	return warnings;
}

export function eventEditPreviewText(form: EventEditFormState) {
	const payload = buildEventEditPayload(form);
	return payload ? `${payload.ticketAllocation} tickets · ${payload.pricingMode === 'free' ? 'Free' : `$${(payload.ticketPriceCents / 100).toFixed(2)}`} · ${payload.ticketCurrency}` : 'Complete ticket fields to preview.';
}

export function eventEditPublishHintTitle(warnings: string[]) {
	return warnings.length === 0 ? 'Ready to publish' : 'Before publishing';
}

export function eventEditPublishHintBody(warnings: string[]) {
	return warnings.length === 0 ? ['This event has the core fields needed for the fake-event rehearsal.'] : warnings.map((warning) => `• ${warning}`);
}

export function isEventEditSaveDisabled(saving: boolean, publishing: boolean) {
	return saving || publishing;
}

export function isEventEditPublishDisabled(editing: boolean, canPublish: boolean, saving: boolean, publishing: boolean) {
	return saving || publishing || (editing && !canPublish);
}

export function eventEditSaveButtonLabel(saving: boolean) {
	return saving ? 'Saving…' : 'Save draft';
}

export function eventEditPublishButtonLabel(editing: boolean, canPublish: boolean, publishing: boolean) {
	return publishing ? 'Publishing…' : canPublish || !editing ? 'Save & publish' : 'Published';
}

export function eventEditImageSelectionFromResult<TAsset extends EventEditImageAsset>(result: EventEditImagePickerResult<TAsset>): EventEditImageSelection<TAsset> | null {
	const selectedImage = result.assets?.[0];
	if (result.canceled || !selectedImage) return null;
	return {
		selectedImage,
		imageUrl: selectedImage.uri,
		notice: 'Image selected. Save the event to upload it.',
	};
}

function parseEventEditStartsAt(value: string) {
	const normalized = value.trim().replace(' ', 'T');
	const date = new Date(normalized);
	return Number.isNaN(date.getTime()) ? null : date.toISOString();
}

function toLocalInput(value: string) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	const offset = date.getTimezoneOffset();
	return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16).replace('T', ' ');
}
