import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { router } from 'expo-router';
import { Search, Ticket } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { loadSavedTickets, type SavedTicket } from '@/tickets/walletStore';
import { AppChrome } from '@/ui/AppChrome';
import { Field } from '@/ui/Field';
import { Pill } from '@/ui/Pill';
import { PrimaryButton } from '@/ui/PrimaryButton';
import { ticketJourneySavedTicketStatus, ticketWalletEmptyCopy } from '@/modules/tickets/ticketJourney';

export default function TicketsScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const [code, setCode] = useState('');
  const [savedTickets, setSavedTickets] = useState<SavedTicket[]>([]);
  const [loaded, setLoaded] = useState(false);
  const normalizedCode = code.trim();

  useEffect(() => {
    let cancelled = false;
    async function loadWallet() {
      try {
        const loaded = await loadSavedTickets();
        if (!cancelled) setSavedTickets(loaded);
      } finally {
        if (!cancelled) setLoaded(true);
      }
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
          <Field
            icon={<Search size={20} color={tokens.color.text.muted} />}
            value={code}
            onChangeText={setCode}
            placeholder="Ticket code"
            autoCapitalize="characters"
            autoCorrect={false}
            onSubmitEditing={openTicket}
          />
          <PrimaryButton disabled={!normalizedCode} onPress={openTicket} icon={<Ticket size={20} color={tokens.color.text.inverse} />} label="Open ticket" />
        </View>

        <View style={styles.walletSection}>
          <Text style={styles.sectionTitle}>Saved tickets</Text>
          {savedTickets.length === 0 ? (
            <View style={styles.emptyCard}>
              <View style={styles.emptyIcon}><Ticket size={30} color={tokens.color.text.primary} /></View>
              <Text style={styles.emptyTitle}>{ticketWalletEmptyCopy(loaded)}</Text>
              <Text style={styles.emptyBody}>{loaded ? 'Reserve or look up a ticket and it will stay here for fast access.' : 'Loading saved tickets…'}</Text>
            </View>
          ) : savedTickets.map((ticket) => (
            <Pressable key={ticket.code} onPress={() => router.push({ pathname: '/ticket', params: { code: ticket.code } })} style={styles.savedTicketCard}>
              <View style={styles.savedTicketCopy}>
                <Text style={styles.savedTicketName}>{ticket.displayName || ticket.email}</Text>
                <Text style={styles.savedTicketCode}>{ticket.code}</Text>
              </View>
              <View style={styles.savedTicketMetaBlock}>
                <Pill tone={ticket.paymentStatus === 'pending' ? 'warning' : 'success'} style={styles.savedTicketStatus}>{ticketJourneySavedTicketStatus(ticket)}</Pill>
                {ticket.paymentStatus === 'pending' ? <Text style={styles.pendingHint}>Tap to refresh</Text> : null}
              </View>
            </Pressable>
          ))}
        </View>
      </ScrollView>
    </AppChrome>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.inset },
  content: { padding: 24, paddingTop: 64, paddingBottom: 32, gap: 18 },
  pageTitle: { fontSize: tokens.type['title'], fontWeight: '700', letterSpacing: -1, color: tokens.color.text.primary },
  pageBody: { color: tokens.color.text.muted, lineHeight: 22, marginBottom: 10 },
  lookupCard: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.panel, padding: 20, gap: 14, boxShadow: '0 2px 8px rgba(0,0,0,0.08)', elevation: 2 },
  walletSection: { gap: 12 },
  sectionTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '700', marginTop: 6 },
  savedTicketCard: { backgroundColor: tokens.color.surface.panel, borderRadius: 22, padding: 18, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 12, overflow: 'hidden' },
  savedTicketCopy: { flex: 1, minWidth: 0 },
  savedTicketName: { color: tokens.color.text.primary, fontSize: tokens.type['body-lg'], fontWeight: '700', marginBottom: 4 },
  savedTicketCode: { color: tokens.color.text.muted, fontFamily: 'monospace', letterSpacing: 1.5 },
  savedTicketMetaBlock: { alignItems: 'flex-end', gap: 4, flexShrink: 0, maxWidth: 132 },
  savedTicketStatus: { alignSelf: 'flex-end', textAlign: 'center' },
  pendingHint: { color: tokens.color.text.muted, fontSize: 11, fontWeight: '700' },
  emptyCard: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.panel, padding: 24, alignItems: 'center', gap: 10 },
  emptyIcon: { width: 64, height: 64, borderRadius: tokens.radius.hero, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center', marginBottom: 4 },
  emptyTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '700' },
  emptyBody: { color: tokens.color.text.muted, textAlign: 'center', lineHeight: 21 },
});
