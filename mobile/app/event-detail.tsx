import { router, useLocalSearchParams } from 'expo-router';
import { Calendar, ChevronLeft, MapPin, Share2, Ticket as TicketIcon } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Image, Linking, Pressable, ScrollView, Share, StyleSheet, Text, TextInput, View } from 'react-native';

import { formatCurrency, formatDate, formatTime, pricingLabel } from '@/api/format';
import { createPaidReservation, getPublicEvent, reserveFreeTicket } from '@/api/events';
import type { PublicEventDTO } from '@/api/types';
import { eventArtwork } from '@/data/eventArtwork';
import { safeBack } from '@/navigation/safeBack';
import { savePendingPaidTicket, saveTicketToWallet } from '@/tickets/walletStore';

export default function EventDetailScreen() {
  const params = useLocalSearchParams<{ slug?: string }>();
  const slug = typeof params.slug === 'string' ? params.slug : '';
  const [event, setEvent] = useState<PublicEventDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [reserving, setReserving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!slug) {
        setError('Missing event slug');
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const loaded = await getPublicEvent(slug);
        if (!cancelled) {
          setEvent(loaded);
        }
      } catch (caught) {
        if (!cancelled) {
          setError(caught instanceof Error ? caught.message : 'Unable to load event');
          setEvent(null);
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
  }, [slug]);

  async function handleReserve() {
    if (!event || !event.publicSlug) {
      return;
    }

    const trimmedEmail = email.trim();
    const trimmedDisplayName = displayName.trim();
    if (!trimmedEmail || !trimmedEmail.includes('@')) {
      setError('Enter a valid email address for the ticket.');
      return;
    }

    setReserving(true);
    setError(null);
    try {
      if (event.pricingMode === 'fixed') {
        const checkout = await createPaidReservation(event.publicSlug, {
          email: trimmedEmail,
          displayName: trimmedDisplayName || undefined,
        });
        await savePendingPaidTicket({
          code: checkout.ticketCode,
          email: trimmedEmail,
          displayName: trimmedDisplayName || null,
          ticketUrl: checkout.ticketUrl,
          checkoutSessionId: checkout.checkoutSessionId,
        });
        await Linking.openURL(checkout.checkoutUrl);
        return;
      }

      const ticket = await reserveFreeTicket(event.publicSlug, {
        email: trimmedEmail,
        displayName: trimmedDisplayName || undefined,
      });
      await saveTicketToWallet(ticket);
      router.push({ pathname: '/ticket', params: { code: ticket.code } });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to reserve ticket');
    } finally {
      setReserving(false);
    }
  }

  async function shareEvent() {
    if (!event?.publicUrl) return;
    await Share.share({
      title: event.title,
      message: `${event.title}\n${event.publicUrl}`,
      url: event.publicUrl,
    });
  }

  if (loading) {
    return <CenteredState title="Loading event…" />;
  }

  if (!event) {
    return <CenteredState title="Could not load event" body={error ?? 'Event not found'} />;
  }

  const price = pricingLabel(event.pricingMode, event.ticketPriceCents, event.ticketCurrency);
  const soldOut = event.isFull;

  return (
    <View style={styles.screen}>
      <ScrollView style={styles.scroller} contentContainerStyle={styles.scrollContent}>
        <View style={styles.hero}>
          <Image source={{ uri: event.imageUrl || eventArtwork(event.publicSlug ?? event.id) }} style={styles.heroImage} resizeMode="cover" />
          <View style={styles.heroOverlay} />
          <View style={styles.topBar}>
            <Pressable onPress={() => safeBack('/')} style={styles.roundButton}>
              <ChevronLeft size={24} color="#ffffff" />
            </Pressable>
            <Pressable onPress={() => void shareEvent()} style={styles.roundButton}>
              <Share2 size={20} color="#ffffff" />
            </Pressable>
          </View>
        </View>
        <View style={styles.content}>
          <Text style={styles.organizer}>{price}</Text>
          <Text style={styles.title}>{event.title}</Text>
          <Text style={styles.subtitle}>{event.publicDescription || 'Published event'}</Text>
          <View style={styles.infoList}>
            <InfoRow icon={<Calendar size={20} color="#171717" />} title={formatDate(event.startsAt)} detail={formatTime(event.startsAt) || 'Doors soon'} />
            <InfoRow icon={<MapPin size={20} color="#171717" />} title={event.locationDisplay.split(',')[0] || 'Venue'} detail={event.locationDisplay.split(',')[1]?.trim() || 'Local Venue'} />
          </View>
          <View style={styles.aboutBlock}>
            <Text style={styles.aboutTitle}>About</Text>
            <Text style={styles.aboutText}>{event.publicDescription || 'More details soon.'}</Text>
          </View>
          <View style={styles.formCard}>
            <Text style={styles.aboutTitle}>{event.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve ticket'}</Text>
            <Text style={styles.formHelp}>Enter the email where your ticket should be sent.</Text>
            <TextInput
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              keyboardType="email-address"
              placeholder="Email address"
              placeholderTextColor="#a3a3a3"
              style={styles.input}
            />
            <TextInput
              value={displayName}
              onChangeText={setDisplayName}
              placeholder="Display name (optional)"
              placeholderTextColor="#a3a3a3"
              style={styles.input}
            />
            {error ? <Text style={styles.errorText}>{error}</Text> : null}
          </View>
        </View>
      </ScrollView>
      <View style={styles.stickyBar}>
        <View>
          <Text style={styles.priceLabel}>{soldOut ? 'Status' : 'Starting from'}</Text>
          <Text style={styles.price}>{soldOut ? 'Sold out' : price}</Text>
          {!soldOut ? <Text style={styles.remaining}>{event.remainingTickets} remaining</Text> : null}
        </View>
        <Pressable disabled={soldOut || reserving} onPress={handleReserve} style={[styles.ticketButton, (soldOut || reserving) && styles.ticketButtonDisabled]}>
          <TicketIcon size={20} color="#ffffff" />
          <Text style={styles.ticketButtonText}>{reserving ? 'Working…' : event.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve'}</Text>
        </Pressable>
      </View>
    </View>
  );
}

function CenteredState({ title, body }: { title: string; body?: string }) {
  return (
    <View style={styles.centerState}>
      <Text style={styles.centerTitle}>{title}</Text>
      {body ? <Text style={styles.centerBody}>{body}</Text> : null}
    </View>
  );
}

function InfoRow({ icon, title, detail }: { icon: React.ReactNode; title: string; detail: string }) {
  return (
    <View style={styles.infoRow}>
      <View style={styles.infoIcon}>{icon}</View>
      <View>
        <Text style={styles.infoTitle}>{title}</Text>
        <Text style={styles.infoDetail}>{detail}</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  scroller: { flex: 1 },
  scrollContent: { paddingBottom: 158 },
  hero: { width: '100%', height: '45%', minHeight: 350, position: 'relative' },
  heroImage: { width: '100%', height: '100%' },
  heroOverlay: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: 'rgba(0,0,0,0.25)' },
  topBar: { position: 'absolute', top: 48, left: 0, right: 0, paddingHorizontal: 16, flexDirection: 'row', justifyContent: 'space-between', zIndex: 10 },
  roundButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: 'rgba(255,255,255,0.20)', alignItems: 'center', justifyContent: 'center' },
  content: { flex: 1, paddingHorizontal: 24, marginTop: -40, zIndex: 10, paddingBottom: 24 },
  organizer: { fontSize: 14, fontWeight: '700', letterSpacing: 1.1, color: '#737373', textTransform: 'uppercase' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 4, marginBottom: 8 },
  subtitle: { fontSize: 18, color: '#737373', marginBottom: 24 },
  infoList: { gap: 16, marginBottom: 32 },
  infoRow: { flexDirection: 'row', alignItems: 'center', gap: 16 },
  infoIcon: { width: 48, height: 48, borderRadius: 16, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  infoTitle: { fontWeight: '700', color: '#404040' },
  infoDetail: { fontSize: 14, color: '#737373' },
  aboutBlock: { marginBottom: 24 },
  aboutTitle: { fontSize: 18, fontWeight: '800', marginBottom: 8, color: '#171717' },
  aboutText: { color: '#525252', lineHeight: 22 },
  formCard: { gap: 12, backgroundColor: '#f5f5f5', borderRadius: 24, padding: 18, marginBottom: 32 },
  formHelp: { color: '#737373', lineHeight: 20 },
  input: { minHeight: 52, borderRadius: 16, backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 14, fontSize: 15, fontWeight: '600' },
  errorText: { color: '#dc2626', fontWeight: '600', lineHeight: 20 },
  stickyBar: { position: 'absolute', left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(255,255,255,0.96)', borderTopWidth: 1, borderTopColor: '#f5f5f5', padding: 16, paddingBottom: 32, flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  priceLabel: { color: '#737373', fontSize: 14, fontWeight: '600' },
  price: { color: '#171717', fontSize: 24, fontWeight: '800' },
  remaining: { color: '#737373', fontSize: 12, fontWeight: '600' },
  ticketButton: { backgroundColor: '#171717', paddingHorizontal: 22, paddingVertical: 16, borderRadius: 16, overflow: 'hidden', flexDirection: 'row', alignItems: 'center', gap: 8 },
  ticketButtonDisabled: { opacity: 0.45 },
  ticketButtonText: { color: '#ffffff', fontWeight: '700' },
  centerState: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 28, backgroundColor: '#ffffff' },
  centerTitle: { color: '#171717', fontSize: 24, fontWeight: '800', textAlign: 'center', marginBottom: 8 },
  centerBody: { color: '#737373', fontSize: 15, lineHeight: 22, textAlign: 'center' },
});
