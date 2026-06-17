import { api } from '@/api/client';
import type { TicketDTO } from '@/api/types';

export function getTicket(code: string) {
  return api<TicketDTO>(`/api/tickets/${encodeURIComponent(code.trim())}`);
}
