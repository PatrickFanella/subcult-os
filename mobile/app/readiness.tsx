import { Link, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, CircleAlert, CircleCheck, ExternalLink, Image as ImageIcon, ListChecks, ShieldCheck, Ticket, UserPlus } from 'lucide-react-native';
import { useEffect, useMemo, useState } from 'react';
import { Linking, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { formatDate, formatTime } from '@/api/format';
import { createTestTicket, getEvent, listEventRoles, listEventStaffing } from '@/api/staff';
import type { EventDTO, EventRoleDTO, EventStaffingItemDTO } from '@/api/types';
import { safeBack } from '@/navigation/safeBack';
import { saveTicketToWallet } from '@/tickets/walletStore';

type ChecklistItem = {
  key: string;
  label: string;
  detail: string;
  done: boolean;
  actionLabel: string;
  href?: { pathname: string; params: Record<string, string> };
  externalUrl?: string;
  icon: typeof CircleCheck;
};

export default function ReadinessScreen() {
  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const [event, setEvent] = useState<EventDTO | null>(null);
  const [roles, setRoles] = useState<EventRoleDTO[]>([]);
  const [staffing, setStaffing] = useState<EventStaffingItemDTO[]>([]);
  const [loading, setLoading] = useState(Boolean(eventID));
  const [creatingTicket, setCreatingTicket] = useState(false);
  const [error, setError] = useState<string | null>(eventID ? null : 'Missing event ID. Open readiness from Staff mode.');
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!eventID) return;
      setLoading(true);
      setError(null);
      try {
        const [loadedEvent, loadedRoles, loadedStaffing] = await Promise.all([
          getEvent(eventID),
          listEventRoles(eventID).catch(() => []),
          listEventStaffing(eventID).catch(() => []),
        ]);
        if (!cancelled) {
          setEvent(loadedEvent);
          setRoles(loadedRoles);
          setStaffing(loadedStaffing);
        }
      } catch (caught) {
        if (!cancelled) setError(caught instanceof Error ? caught.message : 'Unable to load readiness checklist');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [eventID]);

  const checklist = useMemo(() => buildChecklist(event, roles, staffing), [event, roles, staffing]);
  const doneCount = checklist.filter((item) => item.done).length;
  const readinessPercent = checklist.length > 0 ? Math.round((doneCount / checklist.length) * 100) : 0;

  async function reserveTestTicket() {
    if (!event) return;
    setCreatingTicket(true);
    setError(null);
    setNotice(null);
    try {
      const ticket = await createTestTicket(event.id);
      await saveTicketToWallet(ticket);
      const refreshed = await getEvent(event.id);
      setEvent(refreshed);
      setNotice(`Test ticket created: ${ticket.code}`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to create test ticket');
    } finally {
      setCreatingTicket(false);
    }
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/staff')} style={styles.backButton}>
          <ChevronLeft size={24} color="#171717" />
        </Pressable>
        <View style={styles.headerCopy}>
          <Text style={styles.kicker}>Fake event setup</Text>
          <Text style={styles.title}>Readiness</Text>
          <Text style={styles.subtitle}>{event ? `${formatDate(event.startsAt)} • ${formatTime(event.startsAt)}` : 'Organizer checklist'}</Text>
        </View>
      </View>

      <ScrollView contentContainerStyle={styles.content}>
        {loading ? <Text style={styles.message}>Loading checklist…</Text> : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}

        <View style={styles.summaryCard}>
          <Text style={styles.summaryLabel}>Setup score</Text>
          <Text style={styles.summaryValue}>{readinessPercent}%</Text>
          <Text style={styles.summaryBody}>{event?.title ?? 'Select an event from Staff mode to see setup progress.'}</Text>
          <View style={styles.progressTrack}><View style={[styles.progressFill, { width: `${readinessPercent}%` }]} /></View>
          <Text style={styles.summaryMeta}>{doneCount} of {checklist.length} checks complete</Text>
        </View>

        <View style={styles.quickActions}>
          {event ? (
            <>
              <Link href={{ pathname: '/event-edit', params: { eventId: event.id, workspaceId: event.workspaceId } }} style={styles.quickAction}>Edit event</Link>
              <Pressable disabled={creatingTicket} onPress={() => void reserveTestTicket()} style={[styles.quickActionButton, creatingTicket && styles.disabledButton]}>
                <Text style={styles.quickActionButtonText}>{creatingTicket ? 'Creating…' : 'Create test ticket'}</Text>
              </Pressable>
              {event.publicSlug ? <Link href={{ pathname: '/event-detail', params: { slug: event.publicSlug } }} style={styles.quickAction}>Mobile preview</Link> : null}
              {event.publicUrl ? <Text onPress={() => void Linking.openURL(event.publicUrl!)} style={styles.quickAction}>Web preview</Text> : null}
            </>
          ) : null}
        </View>

        <View style={styles.panel}>
          <Text style={styles.sectionTitle}>Checklist</Text>
          {checklist.map((item) => <ChecklistRow key={item.key} item={item} />)}
        </View>

        <View style={styles.panel}>
          <Text style={styles.sectionTitle}>Rehearsal notes</Text>
          <Text style={styles.bodyText}>For the fake event, aim to finish setup, reserve one test ticket, open its QR pass, and confirm staff can find it from Door or Scanner.</Text>
        </View>
      </ScrollView>
    </View>
  );
}

function ChecklistRow({ item }: { item: ChecklistItem }) {
  const Icon = item.icon;
  const indicatorStyle = item.done ? styles.itemIconDone : styles.itemIconOpen;
  const iconColor = item.done ? '#16a34a' : '#f59e0b';
  const content = (
    <View style={styles.itemActionInner}>
      <Text style={styles.itemActionText}>{item.actionLabel}</Text>
      {item.externalUrl ? <ExternalLink size={14} color="#2563eb" /> : null}
    </View>
  );

  return (
    <View style={styles.itemRow}>
      <View style={[styles.itemIcon, indicatorStyle]}><Icon size={20} color={iconColor} /></View>
      <View style={styles.itemCopy}>
        <Text style={styles.itemLabel}>{item.label}</Text>
        <Text style={styles.itemDetail}>{item.detail}</Text>
        {item.href ? <Link href={item.href} style={styles.itemAction}>{content}</Link> : null}
        {item.externalUrl ? <Text onPress={() => void Linking.openURL(item.externalUrl!)} style={styles.itemAction}>{content}</Text> : null}
      </View>
    </View>
  );
}

function buildChecklist(event: EventDTO | null, roles: EventRoleDTO[], staffing: EventStaffingItemDTO[]): ChecklistItem[] {
  const editHref = event ? { pathname: '/event-edit', params: { eventId: event.id, workspaceId: event.workspaceId } } : undefined;
  const rolesHref = event ? { pathname: '/roles', params: { eventId: event.id } } : undefined;
  const staffingHref = event ? { pathname: '/run-of-show', params: { eventId: event.id } } : undefined;
  const doorHref = event ? { pathname: '/door', params: { eventId: event.id } } : undefined;
  const detailsComplete = Boolean(event?.title && event.publicDescription && event.locationDisplay && event.startsAt);
  const ticketingReady = Boolean(event && event.ticketAllocation > 0 && (event.pricingMode === 'free' || event.ticketPriceCents > 0));
  const hasHero = Boolean(event?.imageUrl);
  const published = event?.status === 'published';
  const hasTestTicket = Boolean(event && event.reservedCount > 0);
  const hasRoles = roles.some((role) => role.active);
  const hasRunOfShow = staffing.length > 0;
  const scannerReady = hasTestTicket;

  return [
    {
      key: 'details',
      label: 'Event details complete',
      detail: detailsComplete ? 'Title, date, description, and location are present.' : 'Add title, date, location, and attendee description.',
      done: detailsComplete,
      actionLabel: 'Edit basics',
      href: editHref,
      icon: detailsComplete ? CircleCheck : CircleAlert,
    },
    {
      key: 'image',
      label: 'Hero image uploaded',
      detail: hasHero ? 'Public event cards have an image.' : 'Upload or paste a hero image URL for attendee confidence.',
      done: hasHero,
      actionLabel: 'Manage image',
      href: editHref,
      icon: hasHero ? CircleCheck : ImageIcon,
    },
    {
      key: 'tickets',
      label: 'Ticket settings ready',
      detail: ticketingReady ? `${event?.ticketAllocation ?? 0} ${event?.pricingMode === 'free' ? 'free' : 'paid'} tickets configured.` : 'Set capacity and a valid free/fixed price.',
      done: ticketingReady,
      actionLabel: 'Edit tickets',
      href: editHref,
      icon: ticketingReady ? CircleCheck : Ticket,
    },
    {
      key: 'published',
      label: 'Public page published',
      detail: published ? 'The public event page is available.' : 'Publish when the fake event is ready for attendee testing.',
      done: published,
      actionLabel: published ? 'Open public page' : 'Publish event',
      href: published && event?.publicSlug ? { pathname: '/event-detail', params: { slug: event.publicSlug } } : editHref,
      externalUrl: published && event?.publicUrl ? event.publicUrl : undefined,
      icon: published ? CircleCheck : ExternalLink,
    },
    {
      key: 'test-ticket',
      label: 'Test ticket reserved',
      detail: hasTestTicket ? `${event?.reservedCount ?? 0} ticket(s) reserved for this event.` : 'Create a rehearsal ticket here or reserve from the public page.',
      done: hasTestTicket,
      actionLabel: event?.publicSlug ? 'Open attendee flow' : 'Publish first',
      href: event?.publicSlug ? { pathname: '/event-detail', params: { slug: event.publicSlug } } : editHref,
      icon: hasTestTicket ? CircleCheck : Ticket,
    },
    {
      key: 'roles',
      label: 'Roles configured',
      detail: hasRoles ? `${roles.length} role(s) configured.` : 'Create at least one role if this fake event includes collaborators/applicants.',
      done: hasRoles,
      actionLabel: 'Manage roles',
      href: rolesHref,
      icon: hasRoles ? CircleCheck : UserPlus,
    },
    {
      key: 'run-of-show',
      label: 'Run-of-show started',
      detail: hasRunOfShow ? `${staffing.length} timeline item(s) created.` : 'Add at least one setup or event-day task.',
      done: hasRunOfShow,
      actionLabel: 'Open run of show',
      href: staffingHref,
      icon: hasRunOfShow ? CircleCheck : ListChecks,
    },
    {
      key: 'scanner',
      label: 'Door rehearsal possible',
      detail: scannerReady ? 'A reserved ticket exists, so Door/Scanner can be tested.' : 'Reserve a test ticket before rehearsing door lookup/check-in.',
      done: scannerReady,
      actionLabel: 'Open Door',
      href: doorHref,
      icon: scannerReady ? CircleCheck : ShieldCheck,
    },
  ];
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 24, paddingTop: 56, paddingBottom: 12 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  headerCopy: { flex: 1 },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 6 },
  subtitle: { color: '#737373', fontWeight: '700', marginTop: 4 },
  content: { gap: 18, padding: 24, paddingTop: 8, paddingBottom: 40 },
  message: { color: '#737373', fontWeight: '700' },
  error: { color: '#dc2626', fontWeight: '800', lineHeight: 20 },
  notice: { color: '#16a34a', fontWeight: '800', lineHeight: 20 },
  summaryCard: { backgroundColor: '#171717', borderRadius: 28, padding: 22, gap: 10 },
  summaryLabel: { color: 'rgba(255,255,255,0.65)', fontSize: 12, fontWeight: '800', textTransform: 'uppercase', letterSpacing: 1.2 },
  summaryValue: { color: '#ffffff', fontSize: 48, fontWeight: '900', letterSpacing: -2 },
  summaryBody: { color: 'rgba(255,255,255,0.82)', fontWeight: '700', lineHeight: 21 },
  summaryMeta: { color: 'rgba(255,255,255,0.6)', fontWeight: '700' },
  progressTrack: { height: 8, borderRadius: 999, overflow: 'hidden', backgroundColor: 'rgba(255,255,255,0.14)' },
  progressFill: { height: '100%', borderRadius: 999, backgroundColor: '#22c55e' },
  quickActions: { flexDirection: 'row', flexWrap: 'wrap', gap: 10 },
  quickAction: { backgroundColor: '#eff6ff', color: '#2563eb', borderRadius: 999, overflow: 'hidden', paddingHorizontal: 14, paddingVertical: 10, fontWeight: '900' },
  quickActionButton: { backgroundColor: '#171717', borderRadius: 999, overflow: 'hidden', paddingHorizontal: 14, paddingVertical: 10 },
  quickActionButtonText: { color: '#ffffff', fontWeight: '900' },
  disabledButton: { opacity: 0.45 },
  panel: { backgroundColor: '#fafafa', borderRadius: 28, padding: 20, gap: 16 },
  sectionTitle: { color: '#171717', fontSize: 20, fontWeight: '800' },
  bodyText: { color: '#525252', lineHeight: 22, fontWeight: '600' },
  itemRow: { flexDirection: 'row', gap: 14, paddingVertical: 4 },
  itemIcon: { width: 42, height: 42, borderRadius: 21, alignItems: 'center', justifyContent: 'center' },
  itemIconDone: { backgroundColor: '#dcfce7' },
  itemIconOpen: { backgroundColor: '#fef3c7' },
  itemCopy: { flex: 1, gap: 5 },
  itemLabel: { color: '#171717', fontWeight: '900', fontSize: 16 },
  itemDetail: { color: '#737373', fontWeight: '600', lineHeight: 20 },
  itemAction: { alignSelf: 'flex-start', marginTop: 4, backgroundColor: '#ffffff', borderColor: '#e5e5e5', borderWidth: 1, borderRadius: 999, overflow: 'hidden', paddingHorizontal: 12, paddingVertical: 8 },
  itemActionInner: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  itemActionText: { color: '#2563eb', fontWeight: '900' },
});
