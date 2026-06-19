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

export function publicRoleApplicationButtonLabel(submitting: boolean, submitted: boolean) {
	if (submitted) return 'Submitted';
	return submitting ? 'Submitting…' : 'Submit application';
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
