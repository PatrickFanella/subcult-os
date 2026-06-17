import { api, postJSON } from '@/api/client';
import type { TicketDTO } from '@/api/types';

export function searchDoorTickets(eventID: string, query: string) {
  return api<TicketDTO[]>(`/api/events/${encodeURIComponent(eventID)}/door/tickets?query=${encodeURIComponent(query.trim())}`);
}

export function checkInTicket(eventID: string, code: string) {
  return postJSON<TicketDTO>(`/api/events/${encodeURIComponent(eventID)}/door/check-ins`, { code });
}
