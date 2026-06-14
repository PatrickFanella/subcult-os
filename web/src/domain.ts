export type WorkspaceRole = 'owner' | 'member';

export type EventStatus = 'draft' | 'published' | 'end_of_night';

export interface CurrentUserDTO {
  id: string;
  email: string;
  displayName: string | null;
  workspaces: WorkspaceSummaryDTO[];
}

export interface WorkspaceSummaryDTO {
  id: string;
  name: string;
  role: WorkspaceRole;
}

export interface WorkspaceDTO {
  id: string;
  name: string;
  role: WorkspaceRole;
}

export interface CurrentWorkspaceDTO extends WorkspaceDTO {
  members: MemberDTO[];
  invitations: InvitationDTO[];
}

export interface MemberDTO {
  id: string;
  email: string;
  displayName: string | null;
  role: WorkspaceRole;
}

export interface EventDTO {
  id: string;
  workspaceId: string;
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  ticketAllocation: number;
  pricingMode: 'free' | 'fixed';
  ticketPriceCents: number;
  ticketCurrency: string;
  reservedCount: number;
  checkedInCount: number;
  status: EventStatus;
  publicSlug: string | null;
  publicUrl: string | null;
}

export interface PublicEventDTO extends EventDTO {
  remainingTickets: number;
  isFull: boolean;
}

export interface EventRoleDTO {
  id: string;
  eventId: string;
  name: string;
  description: string;
  capacity: number;
  public: boolean;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface EventRoleApplicationDTO {
  id: string;
  eventId: string;
  roleId: string;
  applicantName: string;
  applicantEmail: string;
  message: string;
  status: 'submitted' | 'under_review' | 'accepted' | 'waitlisted' | 'rejected' | 'withdrawn' | 'confirmed';
  reviewedByPersonId?: string | null;
  reviewedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface EventParticipantDTO {
  applicationId: string;
  roleId: string;
  roleName: string;
  applicantName: string;
  applicantEmail: string;
  status: 'accepted' | 'confirmed';
  updatedAt: string;
}

export interface TicketDTO {
  id: string;
  eventId: string;
  email: string;
  displayName: string | null;
  code: string;
  status: 'reserved' | 'checked_in';
  paymentStatus: 'free' | 'pending' | 'paid' | 'cancelled';
  amountCents: number;
  currency: string;
  checkedInAt: string | null;
}

export interface PaidReservationDTO {
  ticketId: string;
  checkoutSessionId: string;
  checkoutUrl: string;
}

export interface TicketReservationDTO extends TicketDTO {
  ticketUrl: string;
}

export interface InvitationCreatedDTO {
  id: string;
  workspaceId: string;
  email: string;
  role: WorkspaceRole;
  token: string;
}

export interface InvitationDTO {
  id: string;
  email: string;
  displayName?: string | null;
  role: WorkspaceRole;
  token?: string;
  acceptedAt: string | null;
}

export interface EventReportDTO {
  id: string;
  eventId: string;
  title: string;
  startsAt: string;
  publicUrl: string;
  ticketAllocation: number;
  ticketsReserved: number;
  ticketsCheckedIn: number;
  noShows: number;
  settlementSummary?: EventSettlementSummaryDTO;
  generatedAt: string;
  generatedByMemberEmail: string;
}

export interface EventArchiveDTO {
  id: string;
  eventId: string;
  reportId: string;
  settlementId: string;
  seededEventId?: string;
  status: 'private';
  noteCount: number;
  participants: EventArchiveParticipantDTO[];
  notes: EventArchiveNoteDTO[];
  createdAt: string;
  updatedAt: string;
}

export interface EventArchiveParticipantDTO {
  id: string;
  archiveId: string;
  sourceApplicationId: string;
  roleName: string;
  participantName: string;
  status: 'accepted' | 'confirmed';
  createdAt: string;
}

export interface WorkspaceArchiveSummaryDTO {
  id: string;
  eventId: string;
  title: string;
  startsAt: string;
  locationDisplay: string;
  noteCount: number;
  reportId: string;
  settlementId: string;
  seededEventId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface EventArchiveNoteDTO {
  id: string;
  archiveId: string;
  body: string;
  createdByPersonId: string;
  createdAt: string;
}

export interface EventSettlementSummaryDTO {
  currency: string;
  grossPaidRevenueCents: number;
  paidTicketCount: number;
  pendingTicketCount: number;
  cancelledTicketCount: number;
  freeTicketCount: number;
  reservedCount: number;
}

export interface EventSettlementAdjustmentDTO {
  id: string;
  settlementId: string;
  amountCents: number;
  label: string;
  reason: string;
  createdByPersonId: string;
  createdAt: string;
}

export interface EventSettlementDTO {
  id: string;
  eventId: string;
  currency: string;
  grossPaidRevenueCents: number;
  paidTicketCount: number;
  pendingTicketCount: number;
  cancelledTicketCount: number;
  freeTicketCount: number;
  reservedCount: number;
  adjustmentTotalCents: number;
  netTotalCents: number;
  status: 'open' | 'finalized';
  generatedAt: string;
  finalizedAt?: string;
  finalizedByPersonId?: string;
  adjustments: EventSettlementAdjustmentDTO[];
}

export interface DevEmailOutboxMessageDTO {
  id: string;
  recipientEmail: string;
  subject: string;
  body: string;
  relatedType: string;
  relatedId: string | null;
  createdAt: string;
}
