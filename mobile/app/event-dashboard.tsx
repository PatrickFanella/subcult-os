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
          <ChevronLeft size={24} color="#171717" />
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
                <View style={styles.editButtonInner}><Pencil size={16} color="#ffffff" /><Text style={styles.editButtonText}>Edit event</Text></View>
              </Link>
              <Link href={{ pathname: '/readiness', params: { eventId: event.id } }} style={styles.lightButton}>
                <View style={styles.lightButtonInner}><ClipboardCheck size={16} color="#2563eb" /><Text style={styles.lightButtonText}>Readiness</Text></View>
              </Link>
              {event.publicUrl ? (
                <Pressable onPress={() => void Linking.openURL(event.publicUrl!)} style={styles.lightButton}>
                  <View style={styles.lightButtonInner}><ExternalLink size={16} color="#2563eb" /><Text style={styles.lightButtonText}>Preview</Text></View>
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
                <View style={styles.iconBubble}><Icon size={20} color="#171717" /></View>
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

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff', padding: 24, paddingTop: 56 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 6 },
  content: { gap: 20, paddingBottom: 32 },
  message: { color: '#737373', fontWeight: '700' },
  error: { color: '#dc2626', fontWeight: '700', lineHeight: 20 },
  heroCard: { backgroundColor: '#f5f5f5', borderRadius: 28, padding: 22 },
  heroLabel: { color: '#737373', fontSize: 12, fontWeight: '700', marginBottom: 8 },
  eventTitle: { color: '#171717', fontSize: 24, fontWeight: '800', letterSpacing: -0.6 },
  eventMeta: { color: '#737373', marginTop: 6 },
  closeoutCue: { color: '#166534', backgroundColor: '#f0fdf4', borderRadius: 14, marginTop: 12, paddingHorizontal: 12, paddingVertical: 10, fontWeight: '700', lineHeight: 20 },
  actionRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 10, marginTop: 16 },
  editButton: { alignSelf: 'flex-start', backgroundColor: '#171717', borderRadius: 16, overflow: 'hidden', paddingHorizontal: 16, paddingVertical: 12 },
  editButtonInner: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  editButtonText: { color: '#ffffff', fontWeight: '900' },
  lightButton: { alignSelf: 'flex-start', backgroundColor: '#eff6ff', borderRadius: 16, overflow: 'hidden', paddingHorizontal: 16, paddingVertical: 12 },
  lightButtonInner: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  lightButtonText: { color: '#2563eb', fontWeight: '900' },
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 14 },
  statCard: { width: '47.5%', backgroundColor: '#ffffff', borderRadius: 22, borderWidth: 1, borderColor: '#f0f0f0', padding: 18, boxShadow: '0 3px 8px rgba(0,0,0,0.05)', elevation: 1 },
  iconBubble: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center', marginBottom: 16 },
  statValue: { color: '#171717', fontSize: 30, fontWeight: '800' },
  statLabel: { color: '#737373', fontSize: 13, fontWeight: '600' },
  panel: { backgroundColor: '#fafafa', borderRadius: 22, padding: 20 },
  panelTitle: { color: '#171717', fontSize: 20, fontWeight: '800', marginBottom: 8 },
  panelBody: { color: '#525252', lineHeight: 22 },
});
