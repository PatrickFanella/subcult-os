export interface TicketWalletSnapshotInput {
	code: string;
	email: string;
	displayName: string | null;
	status: string;
	paymentStatus: string;
	ticketUrl?: string;
	checkoutSessionId?: string;
}

export interface TicketWalletSnapshot extends TicketWalletSnapshotInput {
	savedAt: string;
}

export function buildTicketWalletSnapshot(input: TicketWalletSnapshotInput, savedAt = new Date().toISOString()): TicketWalletSnapshot {
	return {
		...input,
		savedAt,
	};
}

export function ticketJourneyDisplayName(ticket: Pick<TicketWalletSnapshotInput, 'displayName' | 'email'>) {
	return ticket.displayName ?? ticket.email;
}

export function formatTicketCode(code: string) {
	const chunks: string[] = [];
	for (let index = 0; index < code.length; index += 4) {
		chunks.push(code.slice(index, index + 4));
	}
	return chunks.join(' ');
}

export function ticketJourneyStatusLabel(status: string) {
	return status === 'checked_in' ? 'Checked in' : 'Reserved';
}

export function ticketJourneyTicketBadge(status: string) {
	return status === 'checked_in' ? 'Checked in' : 'Admit one';
}

export function ticketJourneyDoorBadge(status: string) {
	return status === 'checked_in' ? 'Checked in' : 'Ready';
}

export function ticketJourneyDoorResultLabel(status: string) {
	return status === 'checked_in' ? 'Checked in' : 'Ready';
}

export function ticketJourneyArrivalNotes(paymentStatus: string) {
	return paymentStatus === 'pending'
		? 'Payment is still pending. Refresh after checkout completes; the door will only accept paid/free tickets.'
		: 'Show this QR code or ticket code at the door. Staff scanners read the ticket code embedded in the QR pass.';
}

export function ticketJourneySavedTicketStatus(ticket: Pick<TicketWalletSnapshotInput, 'status' | 'paymentStatus'>) {
	return ticket.paymentStatus === 'pending' ? 'payment pending' : ticket.status;
}
