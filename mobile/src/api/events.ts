import { api, postJSON } from '@/api/client';
import type { PaidReservationDTO, PublicEventDTO, PublicEventSummaryDTO, TicketReservationDTO } from '@/api/types';

export function listPublicEvents(query = '') {
  const normalized = query.trim();
  const suffix = normalized ? `?q=${encodeURIComponent(normalized)}` : '';
  return api<PublicEventSummaryDTO[]>(`/api/public/events${suffix}`);
}

export function getPublicEvent(slug: string) {
  return api<PublicEventDTO>(`/api/public/events/${encodeURIComponent(slug)}`);
}

export function reserveFreeTicket(slug: string, body: { email: string; displayName?: string }) {
  return postJSON<TicketReservationDTO>(`/api/public/events/${encodeURIComponent(slug)}/reservations`, body);
}

export function createPaidReservation(slug: string, body: { email: string; displayName?: string }) {
  return postJSON<PaidReservationDTO>(`/api/public/events/${encodeURIComponent(slug)}/paid-reservations`, body);
}
