import { router } from 'expo-router';
import { Search, Ticket } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { loadSavedTickets, type SavedTicket } from '@/tickets/walletStore';
import { AppChrome } from '@/ui/AppChrome';
import { ticketJourneySavedTicketStatus } from '@/modules/tickets/ticketJourney';

export default function TicketsScreen() {
  const [code, setCode] = useState('');
  const [savedTickets, setSavedTickets] = useState<SavedTicket[]>([]);
  const normalizedCode = code.trim();

  useEffect(() => {
    let cancelled = false;
    async function loadWallet() {
      const loaded = await loadSavedTickets();
      if (!cancelled) setSavedTickets(loaded);
    }
    void loadWallet();
    return () => {
      cancelled = true;
    };
  }, []);

  function openTicket() {
    if (!normalizedCode) {
      return;
    }
    router.push({ pathname: '/ticket', params: { code: normalizedCode } });
  }

  return (
    <AppChrome>
      <ScrollView style={styles.screen} contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.pageTitle}>My Tickets</Text>
        <Text style={styles.pageBody}>Enter a ticket code from a reservation email, or open a saved pending checkout after returning from Stripe.</Text>

        <View style={styles.lookupCard}>
          <View style={styles.inputRow}>
            <Search size={20} color="#a3a3a3" />
            <TextInput
              value={code}
              onChangeText={setCode}
              placeholder="Ticket code"
              placeholderTextColor="#a3a3a3"
              autoCapitalize="characters"
              autoCorrect={false}
              style={styles.input}
              onSubmitEditing={openTicket}
            />
          </View>
          <Pressable disabled={!normalizedCode} onPress={openTicket} style={[styles.primaryButton, !normalizedCode && styles.primaryButtonDisabled]}>
            <Ticket size={20} color="#ffffff" />
            <Text style={styles.primaryButtonText}>Open ticket</Text>
          </Pressable>
        </View>

        <View style={styles.walletSection}>
          <Text style={styles.sectionTitle}>Saved tickets</Text>
          {savedTickets.length === 0 ? (
            <View style={styles.emptyCard}>
              <View style={styles.emptyIcon}><Ticket size={30} color="#171717" /></View>
              <Text style={styles.emptyTitle}>No saved tickets yet</Text>
              <Text style={styles.emptyBody}>Reserve or look up a ticket and it will stay here for fast access.</Text>
            </View>
          ) : savedTickets.map((ticket) => (
            <Pressable key={ticket.code} onPress={() => router.push({ pathname: '/ticket', params: { code: ticket.code } })} style={styles.savedTicketCard}>
              <View style={styles.savedTicketCopy}>
                <Text style={styles.savedTicketName}>{ticket.displayName || ticket.email}</Text>
                <Text style={styles.savedTicketCode}>{ticket.code}</Text>
              </View>
              <View style={styles.savedTicketMetaBlock}>
                <Text style={[styles.savedTicketStatus, ticket.paymentStatus === 'pending' && styles.savedTicketStatusPending]}>{ticketJourneySavedTicketStatus(ticket)}</Text>
                {ticket.paymentStatus === 'pending' ? <Text style={styles.pendingHint}>Tap to refresh</Text> : null}
              </View>
            </Pressable>
          ))}
        </View>
      </ScrollView>
    </AppChrome>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#f5f5f5' },
  content: { padding: 24, paddingTop: 64, paddingBottom: 32, gap: 18 },
  pageTitle: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717' },
  pageBody: { color: '#737373', lineHeight: 22, marginBottom: 10 },
  lookupCard: { backgroundColor: '#ffffff', borderRadius: 28, padding: 20, gap: 14, boxShadow: '0 2px 8px rgba(0,0,0,0.08)', elevation: 2 },
  inputRow: { minHeight: 56, borderRadius: 18, backgroundColor: '#f5f5f5', paddingHorizontal: 16, flexDirection: 'row', alignItems: 'center', gap: 10 },
  input: { flex: 1, color: '#171717', fontSize: 16, fontWeight: '700', letterSpacing: 1 },
  primaryButton: { minHeight: 56, borderRadius: 18, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center', flexDirection: 'row', gap: 8 },
  primaryButtonDisabled: { opacity: 0.45 },
  primaryButtonText: { color: '#ffffff', fontWeight: '800', fontSize: 16 },
  walletSection: { gap: 12 },
  sectionTitle: { color: '#171717', fontSize: 20, fontWeight: '800', marginTop: 6 },
  savedTicketCard: { backgroundColor: '#ffffff', borderRadius: 22, padding: 18, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 12, overflow: 'hidden' },
  savedTicketCopy: { flex: 1, minWidth: 0 },
  savedTicketName: { color: '#171717', fontSize: 17, fontWeight: '800', marginBottom: 4 },
  savedTicketCode: { color: '#737373', fontFamily: 'monospace', letterSpacing: 1.5 },
  savedTicketMetaBlock: { alignItems: 'flex-end', gap: 4, flexShrink: 0, maxWidth: 132 },
  savedTicketStatus: { color: '#22c55e', backgroundColor: '#ecfdf5', paddingHorizontal: 9, paddingVertical: 6, borderRadius: 999, overflow: 'hidden', fontSize: 11, fontWeight: '800', textTransform: 'uppercase', textAlign: 'center' },
  savedTicketStatusPending: { color: '#d97706', backgroundColor: '#fffbeb' },
  pendingHint: { color: '#a3a3a3', fontSize: 11, fontWeight: '700' },
  emptyCard: { backgroundColor: '#ffffff', borderRadius: 28, padding: 24, alignItems: 'center', gap: 10 },
  emptyIcon: { width: 64, height: 64, borderRadius: 32, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center', marginBottom: 4 },
  emptyTitle: { color: '#171717', fontSize: 20, fontWeight: '800' },
  emptyBody: { color: '#737373', textAlign: 'center', lineHeight: 21 },
});
