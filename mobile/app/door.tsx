import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { Link, useLocalSearchParams } from 'expo-router';
import { CheckCircle2, ChevronLeft, Search, Ticket, UserCheck } from 'lucide-react-native';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { checkInTicket, searchDoorTickets } from '@/api/door';
import type { DoorTicketDTO } from '@/api/types';
import { useAuth } from '@/auth/AuthContext';
import { safeBack } from '@/navigation/safeBack';
import {
  ticketJourneyDoorBadge,
  ticketJourneyDoorResultLabel,
  doorCheckInButtonLabel,
} from '@/modules/tickets/ticketJourney';

export default function DoorScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const { user, loading: authLoading } = useAuth();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<DoorTicketDTO[]>([]);
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
      setResults([]);
      setError(caught instanceof Error ? caught.message : 'Unable to search tickets');
    } finally {
      setLoading(false);
    }
  }

  async function checkIn(ticket: DoorTicketDTO) {
    if (!eventID) return;
    setCheckingIn(ticket.code);
    setNotice(null);
    setError(null);
    try {
      const updated = await checkInTicket(eventID, ticket.code);
      setResults((current) => current.map((item) => (item.code === updated.code ? updated : item)));
      setNotice(`${ticketJourneyDoorResultLabel(updated.status)} — ${updated.displayName ?? 'Guest'}`);
    } catch (caught) {
      setResults([]);
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
          <ChevronLeft size={24} color={tokens.color.text.primary} />
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
          <Search size={20} color={tokens.color.text.inverse} />
          <Text style={styles.primaryButtonText}>{loading ? 'Searching…' : 'Search'}</Text>
        </Pressable>

        {error ? <Text style={styles.errorText}>{error}</Text> : null}
        {notice ? <View style={styles.notice}><CheckCircle2 size={20} color={tokens.color.status.success} /><Text style={styles.noticeText}>{notice}</Text></View> : null}

        {results.map((ticket) => (
          <View key={ticket.id} style={styles.resultCard}>
            <View style={styles.resultHeader}>
              <Text style={[styles.statusPill, ticket.status === 'checked_in' && styles.statusPillChecked]}>{ticketJourneyDoorBadge(ticket.status)}</Text>
              <Ticket size={20} color={tokens.color.text.muted} />
            </View>
            <Text style={styles.guestName}>{ticket.displayName ?? 'Guest'}</Text>
            <Text style={styles.guestMeta}>{ticket.code} · {ticket.admissionEligible ? 'Ready for entry' : 'Not eligible'}</Text>
            <View style={styles.divider} />
            <Pressable disabled={checkingIn === ticket.code || !ticket.admissionEligible} onPress={() => void checkIn(ticket)} style={styles.primaryButton}>
              <UserCheck size={20} color={tokens.color.text.inverse} />
              <Text style={styles.primaryButtonText}>{doorCheckInButtonLabel(checkingIn === ticket.code, ticket.status === 'checked_in')}</Text>
            </Pressable>
          </View>
        ))}
      </ScrollView>
    </View>
  );
}

function CenteredDoorState({ title }: { title: string }) {

  const styles = useThemedStyles(createStyles);

  return <View style={styles.authGate}><Text style={styles.emptyTitle}>{title}</Text></View>;
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel, padding: 24, paddingTop: 56 },
  authGate: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: tokens.color.surface.panel },
  authButton: { marginTop: 8, backgroundColor: tokens.color.action.primary, color: tokens.color.text.inverse, paddingHorizontal: 22, paddingVertical: 14, borderRadius: tokens.radius.control, overflow: 'hidden', fontWeight: '800' },
  emptyTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '800', textAlign: 'center', marginBottom: 8 },
  emptyBody: { color: tokens.color.text.muted, textAlign: 'center', lineHeight: 21, marginBottom: 8 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '800' },
  title: { fontSize: tokens.type['title'], fontWeight: '800', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 6 },
  content: { gap: 16, paddingBottom: 32 },
  searchBox: { minHeight: tokens.size.field, borderRadius: 18, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 16, flexDirection: 'row', alignItems: 'center', gap: 10 },
  input: { flex: 1, color: tokens.color.text.primary, fontSize: 16, fontWeight: '500' },
  resultCard: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.panel, borderWidth: 1, borderColor: tokens.color.surface.inset, padding: 22, boxShadow: '0 3px 10px rgba(0,0,0,0.06)', elevation: 2 },
  resultHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 },
  statusPill: { alignSelf: 'flex-start', color: tokens.color.text.primary, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 10, paddingVertical: 5, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1, fontSize: tokens.type['label'], fontWeight: '800' },
  statusPillChecked: { color: tokens.color.status.success, backgroundColor: tokens.color.statusSurface.success },
  guestName: { fontSize: 26, fontWeight: '800', color: tokens.color.text.primary, letterSpacing: -0.7 },
  guestMeta: { color: tokens.color.text.muted, fontSize: tokens.type['body'], marginTop: 4 },
  divider: { borderTopWidth: 1, borderTopColor: tokens.color.surface.inset, marginVertical: 20 },
  primaryButton: { minHeight: tokens.size.field, borderRadius: 18, backgroundColor: tokens.color.action.primary, alignItems: 'center', justifyContent: 'center', flexDirection: 'row', gap: 8 },
  primaryButtonDisabled: { opacity: 0.55 },
  primaryButtonText: { color: tokens.color.text.inverse, fontWeight: '800', fontSize: 16 },
  notice: { flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: tokens.color.surface.inset, borderRadius: 18, padding: 16 },
  noticeText: { flex: 1, color: tokens.color.text.secondary, lineHeight: 20 },
  errorText: { color: tokens.color.status.danger, fontWeight: '700', lineHeight: 20 },
});
