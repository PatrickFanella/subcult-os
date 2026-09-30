// Matches the API minimum in handleCompleteRecovery.
export const minimumPasswordLength = 8;

export function newPasswordProblem(password: string, confirmation: string): string | null {
	if (password.length < minimumPasswordLength) return `Password must be at least ${minimumPasswordLength} characters.`;
	if (password !== confirmation) return 'The passwords do not match.';
	return null;
}

// The API answers every recovery request the same way so the form cannot be
// used to discover which addresses have accounts.
export function recoveryRequestNotice(email: string) {
	return `If ${email} belongs to a verified account, check its email for a recovery link. Email delivery is not confirmed.`;
}
