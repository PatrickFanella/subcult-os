export type PublicRoleApplicationDraft = {
	applicantName: string;
	applicantEmail: string;
	message: string;
	submitting: boolean;
	submitted: boolean;
	error: string | null;
};

export function emptyPublicRoleApplicationDraft(): PublicRoleApplicationDraft {
	return {
		applicantName: '',
		applicantEmail: '',
		message: '',
		submitting: false,
		submitted: false,
		error: null,
	};
}

export function publicRoleCapacityLabel(capacity: number) {
	return capacity > 0 ? `${capacity} ${capacity === 1 ? 'spot' : 'spots'}` : 'Open';
}

export function publicRoleAvailabilityLabel(capacity: number, filled?: number) {
	if (capacity <= 0) return 'Open application';
	if (filled === undefined) return `${capacity} ${capacity === 1 ? 'spot' : 'spots'} total`;
	const remaining = Math.max(capacity - filled, 0);
	if (remaining === 0) return 'Role full';
	return `${remaining} ${remaining === 1 ? 'spot' : 'spots'} open`;
}

export function publicRoleApplicationButtonLabel(submitting: boolean, submitted: boolean) {
	if (submitted) return 'Submitted';
	return submitting ? 'Submitting…' : 'Submit application';
}

export function publicRoleApplicationStatusCopy(submitted: boolean, error: string | null) {
	if (error) return error;
	return submitted ? 'Application sent. The host reviews it from their workspace.' : 'Tell the host what you would bring.';
}

export function publicRoleCanSubmit(submitting: boolean, submitted: boolean) {
	return !submitting && !submitted;
}

export function validatePublicRoleApplicationDraft(draft: Pick<PublicRoleApplicationDraft, 'applicantName' | 'applicantEmail' | 'message'>) {
	if (!draft.applicantName.trim()) {
		return 'Please enter your name.';
	}
	if (!draft.applicantEmail.trim() || !draft.applicantEmail.trim().includes('@')) {
		return 'Please enter a valid email address.';
	}
	if (Array.from(draft.message.trim()).length > 2000) {
		return 'Message must be 2000 characters or fewer.';
	}
	return null;
}
