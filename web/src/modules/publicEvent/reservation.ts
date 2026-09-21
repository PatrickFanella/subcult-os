import { api, postJSON } from '../../api';
import type { PublicEventDTO, TicketReservationDTO } from '../../domain';

export async function reserveFreeTicket(slug: string, guest: { email: string; displayName?: string }) {
  const ticket = await postJSON<TicketReservationDTO>(`/api/public/events/${slug}/reservations`, guest);
  // Read authoritative inventory: retries may return an existing reservation,
  // and other guests may have reserved concurrently. Never decrement locally.
  const event = await api<PublicEventDTO>(`/api/public/events/${slug}`).catch(() => null);
  // Inventory refresh failure must not turn an issued ticket into a failed RSVP.
  return { ticket, event };
}
