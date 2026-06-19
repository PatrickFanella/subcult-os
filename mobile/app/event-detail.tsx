import { router, useLocalSearchParams } from 'expo-router';
import { Calendar, ChevronLeft, MapPin, Share2, Ticket as TicketIcon } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Image, Linking, Pressable, ScrollView, Share, StyleSheet, Text, TextInput, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { formatCurrency, formatDate, formatTime, pricingLabel } from '@/api/format';
import { createPaidReservation, getPublicEvent, listPublicEventRoles, reserveFreeTicket, submitPublicRoleApplication } from '@/api/events';
import type { EventRoleDTO, PublicEventDTO } from '@/api/types';
import { useAuth } from '@/auth/AuthContext';
import { eventArtwork } from '@/data/eventArtwork';
import { publicEventPrimaryActionLabel, publicEventStickyCtaHint } from '@/modules/events/publicEventConversionModel';
import {
  emptyPublicRoleApplicationDraft,
  publicRoleApplicationButtonLabel,
  publicRoleApplicationStatusCopy,
  publicRoleAvailabilityLabel,
  publicRoleCanSubmit,
  validatePublicRoleApplicationDraft,
  type PublicRoleApplicationDraft,
} from '@/modules/discovery/publicEventRolesModel';
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
  const [roles, setRoles] = useState<EventRoleDTO[] | null>(null);
  const [roleError, setRoleError] = useState<string | null>(null);
  const [roleDrafts, setRoleDrafts] = useState<Record<string, PublicRoleApplicationDraft>>({});
  const insets = useSafeAreaInsets();
  const { user } = useAuth();
  const reservationEmail = user?.email ?? email.trim();
  const reservationDisplayName = (user?.displayName ?? displayName).trim();

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
      setRoleError(null);
      setRoles(null);
      setRoleDrafts({});

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

  useEffect(() => {
    if (!event || event.publicSlug !== slug) {
      return;
    }

    let cancelled = false;

    async function loadRoles() {
      setRoleError(null);
      setRoles(null);

      try {
        const loadedRoles = await listPublicEventRoles(slug);
        if (!cancelled) {
          setRoles(loadedRoles);
        }
      } catch (caught) {
        if (!cancelled) {
          setRoles([]);
          setRoleError(caught instanceof Error ? caught.message : 'Unable to load public roles');
        }
      }
    }

    void loadRoles();

    return () => {
      cancelled = true;
    };
  }, [event, slug]);

  function updateRoleDraft(roleID: string, updater: (draft: PublicRoleApplicationDraft) => PublicRoleApplicationDraft) {
    setRoleDrafts((current) => {
      const draft = current[roleID] ?? emptyPublicRoleApplicationDraft();
      return { ...current, [roleID]: updater(draft) };
    });
  }

  async function handleRoleSubmit(role: EventRoleDTO) {
    if (!event?.publicSlug) {
      return;
    }

    const draft = roleDrafts[role.id] ?? emptyPublicRoleApplicationDraft();
    const validationError = validatePublicRoleApplicationDraft(draft);
    if (validationError) {
      updateRoleDraft(role.id, (current) => ({ ...current, error: validationError, submitted: false }));
      return;
    }

    updateRoleDraft(role.id, (current) => ({ ...current, submitting: true, error: null }));
    try {
      const submitted = await submitPublicRoleApplication(event.publicSlug, {
        roleId: role.id,
        applicantName: draft.applicantName.trim(),
        applicantEmail: draft.applicantEmail.trim(),
        message: draft.message.trim(),
      });
      updateRoleDraft(role.id, (current) => ({
        ...current,
        applicantName: submitted.applicantName,
        applicantEmail: submitted.applicantEmail,
        message: submitted.message,
        submitting: false,
        submitted: true,
        error: null,
      }));
    } catch (caught) {
      updateRoleDraft(role.id, (current) => ({
        ...current,
        submitting: false,
        submitted: false,
        error: caught instanceof Error ? caught.message : 'Unable to submit application',
      }));
    }
  }

  async function handleReserve() {
    if (!event || !event.publicSlug) {
      return;
    }

    if (!reservationEmail || !reservationEmail.includes('@')) {
      setError('Enter a valid email address for the ticket.');
      return;
    }

    setReserving(true);
    setError(null);
    try {
      if (event.pricingMode === 'fixed') {
        const checkout = await createPaidReservation(event.publicSlug, {
          email: reservationEmail,
          displayName: reservationDisplayName || undefined,
        });
        await savePendingPaidTicket({
          code: checkout.ticketCode,
          email: reservationEmail,
          displayName: reservationDisplayName || null,
          ticketUrl: checkout.ticketUrl,
          checkoutSessionId: checkout.checkoutSessionId,
        });
        await Linking.openURL(checkout.checkoutUrl);
        return;
      }

      const ticket = await reserveFreeTicket(event.publicSlug, {
        email: reservationEmail,
        displayName: reservationDisplayName || undefined,
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
  const stickyCtaLabel = publicEventPrimaryActionLabel(event, reserving);
  const stickyCtaHint = publicEventStickyCtaHint(event, price);
  const stickyCtaDisabled = event.isFull || reserving;

  return (
    <View style={styles.screen}>
      <ScrollView style={styles.scroller} contentContainerStyle={[styles.scrollContent, { paddingBottom: 172 + insets.bottom }]}> 
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
          <View style={styles.titleCard}>
            <Text style={styles.organizer}>{price}</Text>
            <Text style={styles.title}>{event.title}</Text>
            <Text style={styles.subtitle}>{event.publicDescription || 'Published event'}</Text>
          </View>
          <View style={styles.infoList}>
            <InfoRow icon={<Calendar size={20} color="#171717" />} title={formatDate(event.startsAt)} detail={formatTime(event.startsAt) || 'Doors soon'} />
            <InfoRow icon={<MapPin size={20} color="#171717" />} title={event.locationDisplay.split(',')[0] || 'Venue'} detail={event.locationDisplay.split(',')[1]?.trim() || 'Local Venue'} />
          </View>
          <View style={styles.aboutBlock}>
            <Text style={styles.aboutTitle}>About</Text>
            <Text style={styles.aboutText}>{event.publicDescription || 'More details soon.'}</Text>
          </View>
          <View style={styles.participationCard}>
            <Text style={styles.aboutTitle}>Participation</Text>
            <Text style={styles.formHelp}>Public roles are open for short applications. Your ticket flow stays the same.</Text>
            {roleError ? <Text style={styles.errorText}>{roleError}</Text> : null}
            {roles === null ? (
              <Text style={styles.formHelp}>Loading participation roles…</Text>
            ) : roles.length === 0 ? (
              <Text style={styles.formHelp}>No public roles available right now.</Text>
            ) : (
              <View style={styles.roleList}>
                {roles.map((role) => {
                  const draft = roleDrafts[role.id] ?? emptyPublicRoleApplicationDraft();
                  const canSubmit = publicRoleCanSubmit(draft.submitting, draft.submitted);
                  const statusCopy = publicRoleApplicationStatusCopy(draft.submitted, draft.error);
                  return (
                    <View key={role.id} style={styles.roleCard}>
                      <View style={styles.roleHeader}>
                        <View style={styles.roleTitleWrap}>
                          <Text style={styles.roleName}>{role.name}</Text>
                          <Text style={styles.roleDescription}>{role.description || 'No description provided.'}</Text>
                        </View>
						<Text style={styles.rolePill}>{publicRoleAvailabilityLabel(role.capacity)}</Text>
                      </View>
                      <TextInput
                        value={draft.applicantName}
                        onChangeText={(value) => updateRoleDraft(role.id, (current) => ({ ...current, applicantName: value, submitted: false, error: null }))}
                        placeholder="Applicant name"
                        placeholderTextColor="#a3a3a3"
                        style={styles.input}
                        editable={!draft.submitting && !draft.submitted}
                      />
                      <TextInput
                        value={draft.applicantEmail}
                        onChangeText={(value) => updateRoleDraft(role.id, (current) => ({ ...current, applicantEmail: value, submitted: false, error: null }))}
                        autoCapitalize="none"
                        keyboardType="email-address"
                        placeholder="Applicant email"
                        placeholderTextColor="#a3a3a3"
                        style={styles.input}
                        editable={!draft.submitting && !draft.submitted}
                      />
                      <TextInput
                        value={draft.message}
                        onChangeText={(value) => updateRoleDraft(role.id, (current) => ({ ...current, message: value, submitted: false, error: null }))}
                        multiline
                        placeholder="Message (optional)"
                        placeholderTextColor="#a3a3a3"
                        style={[styles.input, styles.messageInput]}
                        editable={!draft.submitting && !draft.submitted}
                      />
                      <Pressable disabled={!canSubmit} onPress={() => void handleRoleSubmit(role)} style={[styles.roleButton, !canSubmit && styles.roleButtonDisabled]}>
                        <Text style={styles.roleButtonText}>{publicRoleApplicationButtonLabel(draft.submitting, draft.submitted)}</Text>
                      </Pressable>
                      <Text style={draft.error ? styles.errorText : draft.submitted ? styles.successText : styles.formHelp}>{statusCopy}</Text>
                    </View>
                  );
                })}
              </View>
            )}
          </View>
          <View style={styles.formCard}>
            <Text style={styles.aboutTitle}>{event.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve ticket'}</Text>
            {user ? (
              <View style={styles.signedInCard}>
                <Text style={styles.signedInLabel}>Reserving as</Text>
                <Text style={styles.signedInName}>{user.displayName || user.email}</Text>
                <Text style={styles.signedInEmail}>{user.email}</Text>
              </View>
            ) : (
              <>
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
              </>
            )}
            {error ? <Text style={styles.errorText}>{error}</Text> : null}
          </View>
        </View>
      </ScrollView>
      <View style={[styles.stickyBar, { paddingBottom: Math.max(insets.bottom, 14) }]}> 
        <View>
          <Text style={styles.priceLabel}>{event.isFull ? 'Status' : 'Starting from'}</Text>
          <Text style={styles.price}>{event.isFull ? 'Sold out' : price}</Text>
          <Text style={styles.remaining}>{stickyCtaHint}</Text>
        </View>
        <View style={styles.stickyAction}>
          {event.isFull ? <Text style={styles.stickyWarning}>No tickets remain.</Text> : null}
          <Pressable disabled={stickyCtaDisabled} onPress={handleReserve} style={[styles.ticketButton, stickyCtaDisabled && styles.ticketButtonDisabled]}>
            <TicketIcon size={20} color="#ffffff" />
            <Text style={styles.ticketButtonText}>{stickyCtaLabel}</Text>
          </Pressable>
        </View>
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
  scrollContent: {},
  hero: { width: '100%', height: '45%', minHeight: 350, position: 'relative' },
  heroImage: { width: '100%', height: '100%' },
  heroOverlay: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: 'rgba(0,0,0,0.08)' },
  topBar: { position: 'absolute', top: 48, left: 0, right: 0, paddingHorizontal: 16, flexDirection: 'row', justifyContent: 'space-between', zIndex: 10 },
  roundButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: 'rgba(255,255,255,0.20)', alignItems: 'center', justifyContent: 'center' },
  content: { flex: 1, paddingHorizontal: 20, marginTop: -72, zIndex: 10, paddingBottom: 32 },
  titleCard: { backgroundColor: 'rgba(255,255,255,0.96)', borderRadius: 28, padding: 20, marginBottom: 22, boxShadow: '0 2px 12px rgba(0,0,0,0.12)', elevation: 2 },
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
  participationCard: { gap: 12, backgroundColor: '#f5f5f5', borderRadius: 24, padding: 18, marginBottom: 24 },
  roleList: { gap: 14 },
  roleCard: { gap: 12, backgroundColor: '#ffffff', borderRadius: 20, padding: 14 },
  roleHeader: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 },
  roleTitleWrap: { flex: 1 },
  roleName: { color: '#171717', fontSize: 16, fontWeight: '900' },
  roleDescription: { color: '#737373', lineHeight: 20, marginTop: 4 },
  rolePill: { color: '#525252', backgroundColor: '#f5f5f5', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 6, fontSize: 12, fontWeight: '900', overflow: 'hidden' },
  formCard: { gap: 12, backgroundColor: '#f5f5f5', borderRadius: 24, padding: 18, marginBottom: 48 },
  formHelp: { color: '#737373', lineHeight: 20 },
  input: { minHeight: 52, borderRadius: 16, backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 14, fontSize: 15, fontWeight: '600' },
  messageInput: { minHeight: 96, paddingTop: 14, textAlignVertical: 'top' },
  roleButton: { alignItems: 'center', justifyContent: 'center', minHeight: 48, borderRadius: 16, borderWidth: 1, borderColor: '#d4d4d4', backgroundColor: '#ffffff' },
  roleButtonDisabled: { opacity: 0.55 },
  roleButtonText: { color: '#171717', fontWeight: '800' },
  signedInCard: { backgroundColor: '#ffffff', borderRadius: 18, padding: 14, gap: 3 },
  signedInLabel: { color: '#737373', fontSize: 12, fontWeight: '900', textTransform: 'uppercase', letterSpacing: 1 },
  signedInName: { color: '#171717', fontSize: 16, fontWeight: '900' },
  signedInEmail: { color: '#737373', fontWeight: '700' },
  errorText: { color: '#dc2626', fontWeight: '600', lineHeight: 20 },
  successText: { color: '#047857', fontWeight: '700', lineHeight: 20 },
  stickyBar: { position: 'absolute', left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(255,255,255,0.97)', borderTopWidth: 1, borderTopColor: '#f5f5f5', paddingTop: 14, paddingHorizontal: 16, flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', boxShadow: '0 -2px 10px rgba(0,0,0,0.08)', elevation: 8 },
  priceLabel: { color: '#737373', fontSize: 14, fontWeight: '600' },
  price: { color: '#171717', fontSize: 24, fontWeight: '800' },
  remaining: { color: '#737373', fontSize: 12, fontWeight: '600' },
  stickyAction: { alignItems: 'flex-end', gap: 6 },
  stickyWarning: { color: '#dc2626', fontSize: 12, fontWeight: '700' },
  ticketButton: { backgroundColor: '#171717', paddingHorizontal: 22, paddingVertical: 16, borderRadius: 16, overflow: 'hidden', flexDirection: 'row', alignItems: 'center', gap: 8 },
  ticketButtonDisabled: { opacity: 0.45 },
  ticketButtonText: { color: '#ffffff', fontWeight: '700' },
  centerState: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 28, backgroundColor: '#ffffff' },
  centerTitle: { color: '#171717', fontSize: 24, fontWeight: '800', textAlign: 'center', marginBottom: 8 },
  centerBody: { color: '#737373', fontSize: 15, lineHeight: 22, textAlign: 'center' },
});
