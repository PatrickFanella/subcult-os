import { useLocalSearchParams } from 'expo-router';
import { Calendar, ChevronLeft, MapPin, Share2 } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Pressable, ScrollView, Share, StyleSheet, Text, View } from 'react-native';
import QRCode from 'react-native-qrcode-svg';

import { getTicket } from '@/api/tickets';
import type { TicketDTO } from '@/api/types';
import { safeBack } from '@/navigation/safeBack';
import { saveTicketToWallet } from '@/tickets/walletStore';

export default function TicketScreen() {
  const params = useLocalSearchParams<{ code?: string; checkout?: string }>();
  const code = typeof params.code === 'string' ? params.code.trim() : '';
  const checkoutState = typeof params.checkout === 'string' ? params.checkout : '';
  const [ticket, setTicket] = useState<TicketDTO | null>(null);
  const [loading, setLoading] = useState(Boolean(code));
  const [error, setError] = useState<string | null>(code ? null : 'Enter a ticket code from the Tickets tab.');

  async function loadTicket() {
    if (!code) {
      setTicket(null);
      setError('Enter a ticket code from the Tickets tab.');
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const loaded = await getTicket(code);
      setTicket(loaded);
      await saveTicketToWallet(loaded);
    } catch (caught) {
      setTicket(null);
      setError(caught instanceof Error ? caught.message : 'Unable to load ticket');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!code) {
        setTicket(null);
        setError('Enter a ticket code from the Tickets tab.');
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const loaded = await getTicket(code);
        if (!cancelled) {
          setTicket(loaded);
          await saveTicketToWallet(loaded);
        }
      } catch (caught) {
        if (!cancelled) {
          setTicket(null);
          setError(caught instanceof Error ? caught.message : 'Unable to load ticket');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [code]);

  async function shareTicket() {
    if (!ticket?.ticketUrl) return;
    await Share.share({
      title: `Ticket ${ticket.code}`,
      message: `Ticket ${ticket.code}: ${ticket.ticketUrl}`,
      url: ticket.ticketUrl,
    });
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/tickets')} style={styles.backButton}>
          <ChevronLeft size={24} color="#171717" />
        </Pressable>
        <Text style={styles.title}>Ticket</Text>
      </View>

      <ScrollView contentContainerStyle={styles.content}>
        {loading ? <Text style={styles.stateText}>Loading ticket…</Text> : null}
        {error ? <Text style={styles.errorText}>{error}</Text> : null}
        {checkoutState === 'success' ? <Text style={styles.successText}>Checkout complete. If payment still shows pending, refresh in a moment while Stripe confirms.</Text> : null}
        {checkoutState === 'cancelled' ? <Text style={styles.errorText}>Checkout was cancelled. Your ticket is not paid yet.</Text> : null}

        <View style={styles.ticketCard}>
          <View style={styles.cardHeader}>
            <Text style={styles.statusPill}>{ticket?.status === 'checked_in' ? 'Checked in' : 'Admit one'}</Text>
            <Pressable disabled={!ticket?.ticketUrl} onPress={() => void shareTicket()} style={styles.circleBadge}><Share2 size={22} color="#171717" /></Pressable>
          </View>
          <Text style={styles.eventTitle}>{ticket ? `Ticket ${ticket.code}` : 'Ticket lookup'}</Text>
          <Text style={styles.subtitle}>{ticket?.displayName || ticket?.email || 'Load a ticket by code'}</Text>

          <View style={styles.metaList}>
            <View style={styles.metaRow}><Calendar size={18} color="#a3a3a3" /><Text style={styles.metaText}>Payment: {ticket?.paymentStatus ?? '—'}</Text></View>
            <View style={styles.metaRow}><MapPin size={18} color="#a3a3a3" /><Text style={styles.metaText}>Status: {ticket?.status ?? '—'}</Text></View>
          </View>

          <View style={styles.dashedRule} />

          <View style={styles.qrBlock}>
            <View style={styles.qrBox}>
              {ticket ? <QRCode value={ticket.code} size={168} color="#171717" backgroundColor="#ffffff" /> : <Text style={styles.qrPlaceholder}>QR</Text>}
            </View>
            <Text style={styles.ticketCode}>{ticket?.code ?? 'NO-CODE'}</Text>
          </View>
        </View>

        <View style={styles.notesCard}>
          <Text style={styles.notesTitle}>Arrival notes</Text>
          <Text style={styles.notesText}>{ticket?.paymentStatus === 'pending' ? 'Payment is still pending. Refresh after checkout completes; the door will only accept paid/free tickets.' : 'Show this QR code or ticket code at the door. Staff scanners read the ticket code embedded in the QR pass.'}</Text>
          {ticket ? <Pressable onPress={() => void loadTicket()} style={styles.refreshButton}><Text style={styles.refreshButtonText}>Refresh ticket</Text></Pressable> : null}
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#f5f5f5', padding: 24, paddingTop: 56 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#ffffff', alignItems: 'center', justifyContent: 'center' },
  title: { fontSize: 28, fontWeight: '800', letterSpacing: -0.8, color: '#171717' },
  content: { gap: 20, paddingBottom: 32 },
  stateText: { color: '#737373', fontWeight: '700' },
  errorText: { color: '#dc2626', fontWeight: '700', lineHeight: 20 },
  successText: { color: '#16a34a', fontWeight: '800', lineHeight: 20 },
  ticketCard: { backgroundColor: '#ffffff', borderRadius: 32, padding: 24, boxShadow: '0 2px 8px rgba(0,0,0,0.08)', elevation: 2 },
  cardHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 18 },
  statusPill: { alignSelf: 'flex-start', backgroundColor: '#f5f5f5', color: '#171717', paddingHorizontal: 12, paddingVertical: 6, borderRadius: 999, overflow: 'hidden', fontSize: 12, fontWeight: '800', textTransform: 'uppercase', letterSpacing: 1 },
  circleBadge: { width: 48, height: 48, borderRadius: 24, backgroundColor: '#ffffff', alignItems: 'center', justifyContent: 'center', boxShadow: '0 1px 4px rgba(0,0,0,0.08)', elevation: 1 },
  eventTitle: { fontSize: 26, fontWeight: '800', letterSpacing: -0.7, color: '#171717', marginBottom: 4 },
  subtitle: { color: '#737373', marginBottom: 24, fontSize: 14 },
  metaList: { gap: 12, marginBottom: 24 },
  metaRow: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  metaText: { color: '#404040', fontWeight: '600', fontSize: 14 },
  dashedRule: { borderTopWidth: 2, borderStyle: 'dashed', borderColor: '#e5e5e5', marginVertical: 20 },
  qrBlock: { alignItems: 'center', justifyContent: 'center', paddingVertical: 10 },
  qrBox: { width: 208, height: 208, borderRadius: 20, backgroundColor: '#ffffff', borderWidth: 1, borderColor: '#e5e5e5', alignItems: 'center', justifyContent: 'center', marginBottom: 16 },
  qrPlaceholder: { color: '#737373', fontWeight: '900', letterSpacing: 4 },
  ticketCode: { fontFamily: 'monospace', color: '#737373', letterSpacing: 3, fontSize: 14 },
  notesCard: { backgroundColor: '#ffffff', borderRadius: 22, padding: 20 },
  notesTitle: { color: '#171717', fontSize: 20, fontWeight: '800', marginBottom: 8 },
  notesText: { color: '#525252', lineHeight: 22 },
  refreshButton: { alignSelf: 'flex-start', marginTop: 14, backgroundColor: '#171717', borderRadius: 14, paddingHorizontal: 16, paddingVertical: 12 },
  refreshButtonText: { color: '#ffffff', fontWeight: '900' },
});
