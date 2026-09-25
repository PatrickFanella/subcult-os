import { api, postJSON } from '@/api/client';
import type { DoorTicketDTO } from '@/api/types';

export function searchDoorTickets(eventID: string, query: string) {
  return api<DoorTicketDTO[]>(`/api/events/${encodeURIComponent(eventID)}/door/tickets?query=${encodeURIComponent(query.trim())}`);
}

export function checkInTicket(eventID: string, code: string) {
  return postJSON<DoorTicketDTO>(`/api/events/${encodeURIComponent(eventID)}/door/check-ins`, { code });
}
