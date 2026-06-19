import { Link, useLocalSearchParams } from 'expo-router';
import { CheckCircle2, ChevronLeft, Search, Ticket, UserCheck } from 'lucide-react-native';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { checkInTicket, searchDoorTickets } from '@/api/door';
import type { TicketDTO } from '@/api/types';
import { useAuth } from '@/auth/AuthContext';
import { safeBack } from '@/navigation/safeBack';
import {
  ticketJourneyDisplayName,
  ticketJourneyDoorBadge,
  ticketJourneyDoorResultLabel,
} from '@/modules/tickets/ticketJourney';

export default function DoorScreen() {
  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const { user, loading: authLoading } = useAuth();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<TicketDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [checkingIn, setCheckingIn] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function search() {
    const trimmed = query.trim();
    setNotice(null);
    setError(null);
    if (!trimmed) {
      setNotice('Type an email, name, or exact ticket code.');
      return;
    }
    if (!eventID) {
      setError('Missing event ID. Open Door from Staff mode.');
      return;
    }

    setLoading(true);
    try {
      const loaded = await searchDoorTickets(eventID, trimmed);
      setResults(loaded);
      setNotice(loaded.length === 0 ? `No matches for “${trimmed}”.` : `${loaded.length} ticket${loaded.length === 1 ? '' : 's'} ready.`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to search tickets');
    } finally {
      setLoading(false);
    }
  }

  async function checkIn(ticket: TicketDTO) {
    if (!eventID) return;
    setCheckingIn(ticket.code);
    setNotice(null);
    setError(null);
    try {
      const updated = await checkInTicket(eventID, ticket.code);
      setResults((current) => current.map((item) => (item.code === updated.code ? updated : item)));
      setNotice(`${ticketJourneyDoorResultLabel(updated.status)} — ${ticketJourneyDisplayName(updated)}`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to check in ticket');
    } finally {
      setCheckingIn(null);
    }
  }

  if (authLoading) {
    return <CenteredDoorState title="Checking session…" />;
  }

  if (!user) {
    return (
      <View style={styles.authGate}>
        <Text style={styles.emptyTitle}>Door requires sign in</Text>
        <Text style={styles.emptyBody}>Sign in with a workspace account to search and check in tickets.</Text>
        <Link href={{ pathname: '/login', params: { next: eventID ? `/door?eventId=${eventID}` : '/staff' } }} style={styles.authButton}>Sign in</Link>
      </View>
    );
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/staff')} style={styles.backButton}>
          <ChevronLeft size={24} color="#171717" />
        </Pressable>
        <View>
          <Text style={styles.kicker}>Door Mode</Text>
          <Text style={styles.title}>Guest List</Text>
        </View>
      </View>

      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <View style={styles.searchBox}>
          <Search size={20} color="#a3a3a3" />
          <TextInput
            placeholder="Name, email, or ticket code"
            placeholderTextColor="#a3a3a3"
            style={styles.input}
            autoCapitalize="none"
            autoCorrect={false}
            value={query}
            onChangeText={setQuery}
            onSubmitEditing={search}
          />
        </View>
        <Pressable disabled={loading} onPress={search} style={[styles.primaryButton, loading && styles.primaryButtonDisabled]}>
          <Search size={20} color="#ffffff" />
          <Text style={styles.primaryButtonText}>{loading ? 'Searching…' : 'Search'}</Text>
        </Pressable>

        {error ? <Text style={styles.errorText}>{error}</Text> : null}
        {notice ? <View style={styles.notice}><CheckCircle2 size={20} color="#22c55e" /><Text style={styles.noticeText}>{notice}</Text></View> : null}

        {results.map((ticket) => (
          <View key={ticket.id} style={styles.resultCard}>
            <View style={styles.resultHeader}>
              <Text style={[styles.statusPill, ticket.status === 'checked_in' && styles.statusPillChecked]}>{ticketJourneyDoorBadge(ticket.status)}</Text>
              <Ticket size={20} color="#737373" />
            </View>
            <Text style={styles.guestName}>{ticketJourneyDisplayName(ticket)}</Text>
            <Text style={styles.guestMeta}>{ticket.code} · {ticket.paymentStatus}</Text>
            <View style={styles.divider} />
            <Pressable disabled={checkingIn === ticket.code} onPress={() => void checkIn(ticket)} style={styles.primaryButton}>
              <UserCheck size={20} color="#ffffff" />
              <Text style={styles.primaryButtonText}>{checkingIn === ticket.code ? 'Checking in…' : ticket.status === 'checked_in' ? 'Confirm checked in' : 'Check in guest'}</Text>
            </Pressable>
          </View>
        ))}
      </ScrollView>
    </View>
  );
}

function CenteredDoorState({ title }: { title: string }) {
  return <View style={styles.authGate}><Text style={styles.emptyTitle}>{title}</Text></View>;
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff', padding: 24, paddingTop: 56 },
  authGate: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: '#ffffff' },
  authButton: { marginTop: 8, backgroundColor: '#171717', color: '#ffffff', paddingHorizontal: 22, paddingVertical: 14, borderRadius: 16, overflow: 'hidden', fontWeight: '800' },
  emptyTitle: { color: '#171717', fontSize: 24, fontWeight: '800', textAlign: 'center', marginBottom: 8 },
  emptyBody: { color: '#737373', textAlign: 'center', lineHeight: 21, marginBottom: 8 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 6 },
  content: { gap: 16, paddingBottom: 32 },
  searchBox: { minHeight: 56, borderRadius: 18, backgroundColor: '#f5f5f5', paddingHorizontal: 16, flexDirection: 'row', alignItems: 'center', gap: 10 },
  input: { flex: 1, color: '#171717', fontSize: 16, fontWeight: '500' },
  resultCard: { backgroundColor: '#ffffff', borderRadius: 28, borderWidth: 1, borderColor: '#f0f0f0', padding: 22, boxShadow: '0 3px 10px rgba(0,0,0,0.06)', elevation: 2 },
  resultHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 },
  statusPill: { alignSelf: 'flex-start', color: '#171717', backgroundColor: '#f5f5f5', paddingHorizontal: 10, paddingVertical: 5, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1, fontSize: 12, fontWeight: '800' },
  statusPillChecked: { color: '#16a34a', backgroundColor: '#f0fdf4' },
  guestName: { fontSize: 26, fontWeight: '800', color: '#171717', letterSpacing: -0.7 },
  guestMeta: { color: '#737373', fontSize: 15, marginTop: 4 },
  divider: { borderTopWidth: 1, borderTopColor: '#f0f0f0', marginVertical: 20 },
  primaryButton: { minHeight: 56, borderRadius: 18, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center', flexDirection: 'row', gap: 8 },
  primaryButtonDisabled: { opacity: 0.55 },
  primaryButtonText: { color: '#ffffff', fontWeight: '800', fontSize: 16 },
  notice: { flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: '#fafafa', borderRadius: 18, padding: 16 },
  noticeText: { flex: 1, color: '#525252', lineHeight: 20 },
  errorText: { color: '#dc2626', fontWeight: '700', lineHeight: 20 },
});
