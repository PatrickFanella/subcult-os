import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { Link, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, ClipboardCheck, ExternalLink, Pencil, ShieldCheck, Ticket, Users } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Linking, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { formatDate, formatTime } from '@/api/format';
import { getEvent } from '@/api/staff';
import type { EventDTO } from '@/api/types';
import { safeBack } from '@/navigation/safeBack';
import { mobileCloseoutHandoffCopy, mobileCloseoutStatusLabel } from '@/modules/settlement/settlementModel';

export default function EventDashboardScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const [event, setEvent] = useState<EventDTO | null>(null);
  const [loading, setLoading] = useState(Boolean(eventID));
  const [error, setError] = useState<string | null>(eventID ? null : 'Missing event ID. Open Dashboard from Staff mode.');

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!eventID) return;
      setLoading(true);
      setError(null);
      try {
        const loaded = await getEvent(eventID);
        if (!cancelled) setEvent(loaded);
      } catch (caught) {
        if (!cancelled) setError(caught instanceof Error ? caught.message : 'Unable to load event');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [eventID]);

  const stats = [
    { label: 'Reserved', value: event?.reservedCount ?? '—', icon: Ticket },
    { label: 'Checked in', value: event?.checkedInCount ?? '—', icon: ShieldCheck },
    { label: 'Staff open', value: event?.staffingOpenCount ?? '—', icon: Users },
  ];

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/staff')} style={styles.backButton}>
          <ChevronLeft size={24} color={tokens.color.text.primary} />
        </Pressable>
        <View>
          <Text style={styles.kicker}>Live Event</Text>
          <Text style={styles.title}>Dashboard</Text>
        </View>
      </View>

      <ScrollView contentContainerStyle={styles.content}>
        {loading ? <Text style={styles.message}>Loading event…</Text> : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}
        <View style={styles.heroCard}>
          <Text style={styles.heroLabel}>Active Event</Text>
          <Text style={styles.eventTitle}>{event?.title ?? 'No event loaded'}</Text>
          <Text style={styles.eventMeta}>{event ? `${formatDate(event.startsAt)} • ${formatTime(event.startsAt)} · ${event.locationDisplay}` : 'Open from Staff mode'}</Text>
          {event?.status === 'end_of_night' ? <Text style={styles.closeoutCue}>{mobileCloseoutStatusLabel(event.status)} · {mobileCloseoutHandoffCopy(event.status)}</Text> : null}
          {event ? (
            <View style={styles.actionRow}>
              <Link href={{ pathname: '/event-edit', params: { eventId: event.id, workspaceId: event.workspaceId } }} style={styles.editButton}>
                <View style={styles.editButtonInner}><Pencil size={16} color={tokens.color.text.inverse} /><Text style={styles.editButtonText}>Edit event</Text></View>
              </Link>
              <Link href={{ pathname: '/readiness', params: { eventId: event.id } }} style={styles.lightButton}>
                <View style={styles.lightButtonInner}><ClipboardCheck size={16} color={tokens.color.status.info} /><Text style={styles.lightButtonText}>Readiness</Text></View>
              </Link>
              {event.publicUrl ? (
                <Pressable onPress={() => void Linking.openURL(event.publicUrl!)} style={styles.lightButton}>
                  <View style={styles.lightButtonInner}><ExternalLink size={16} color={tokens.color.status.info} /><Text style={styles.lightButtonText}>Preview</Text></View>
                </Pressable>
              ) : null}
            </View>
          ) : null}
        </View>

        <View style={styles.grid}>
          {stats.map((stat) => {
            const Icon = stat.icon;
            return (
              <View key={stat.label} style={styles.statCard}>
                <View style={styles.iconBubble}><Icon size={20} color={tokens.color.text.primary} /></View>
                <Text style={styles.statValue}>{stat.value}</Text>
                <Text style={styles.statLabel}>{stat.label}</Text>
              </View>
            );
          })}
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Ticket flow</Text>
          <Text style={styles.panelBody}>{event ? `${Math.max(event.ticketAllocation - event.reservedCount, 0)} of ${event.ticketAllocation} tickets remain.` : 'Ticket metrics load after selecting an event.'}</Text>
        </View>
      </ScrollView>
    </View>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel, padding: 24, paddingTop: 56 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '700' },
  title: { fontSize: tokens.type['title'], fontWeight: '700', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 6 },
  content: { gap: 20, paddingBottom: 32 },
  message: { color: tokens.color.text.muted, fontWeight: '700' },
  error: { color: tokens.color.status.danger, fontWeight: '700', lineHeight: 20 },
  heroCard: { backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.panel, padding: 22 },
  heroLabel: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '700', marginBottom: 8 },
  eventTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '700', letterSpacing: -0.6 },
  eventMeta: { color: tokens.color.text.muted, marginTop: 6 },
  closeoutCue: { color: tokens.color.status.success, backgroundColor: tokens.color.statusSurface.success, borderRadius: 14, marginTop: 12, paddingHorizontal: 12, paddingVertical: 10, fontWeight: '700', lineHeight: 20 },
  actionRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 10, marginTop: 16 },
  editButton: { alignSelf: 'flex-start', backgroundColor: tokens.color.action.primary, borderRadius: tokens.radius.control, overflow: 'hidden', paddingHorizontal: 16, paddingVertical: 12 },
  editButtonInner: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  editButtonText: { color: tokens.color.text.inverse, fontWeight: '700' },
  lightButton: { alignSelf: 'flex-start', backgroundColor: tokens.color.statusSurface.info, borderRadius: tokens.radius.control, overflow: 'hidden', paddingHorizontal: 16, paddingVertical: 12 },
  lightButtonInner: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  lightButtonText: { color: tokens.color.text.primary, fontWeight: '700' },
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 14 },
  statCard: { width: '47.5%', backgroundColor: tokens.color.surface.panel, borderRadius: 22, borderWidth: 1, borderColor: tokens.color.surface.inset, padding: 18, boxShadow: '0 3px 8px rgba(0,0,0,0.05)', elevation: 1 },
  iconBubble: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center', marginBottom: 16 },
  statValue: { color: tokens.color.text.primary, fontSize: 30, fontWeight: '700' },
  statLabel: { color: tokens.color.text.muted, fontSize: 13, fontWeight: '600' },
  panel: { backgroundColor: tokens.color.surface.inset, borderRadius: 22, padding: 20 },
  panelTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '700', marginBottom: 8 },
  panelBody: { color: tokens.color.text.secondary, lineHeight: 22 },
});
