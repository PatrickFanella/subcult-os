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

export interface TicketDTO {
  id: string;
  eventId: string;
  email: string;
  displayName: string | null;
  code: string;
  status: 'reserved' | 'checked_in';
  checkedInAt: string | null;
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
  generatedAt: string;
  generatedByMemberEmail: string;
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
