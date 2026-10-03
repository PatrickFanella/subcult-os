import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { router, useLocalSearchParams } from 'expo-router';
import { Calendar, ChevronLeft, MapPin, Share2, Ticket as TicketIcon } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Image, Linking, Pressable, ScrollView, Share, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { formatDate, formatTime, pricingLabel } from '@/api/format';
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
import { Field } from '@/ui/Field';
import { Pill } from '@/ui/Pill';
import { PrimaryButton } from '@/ui/PrimaryButton';

export default function EventDetailScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

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
              <ChevronLeft size={24} color={tokens.color.text.onImmersive} />
            </Pressable>
            <Pressable onPress={() => void shareEvent()} style={styles.roundButton}>
              <Share2 size={20} color={tokens.color.text.onImmersive} />
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
            <InfoRow icon={<Calendar size={20} color={tokens.color.text.primary} />} title={formatDate(event.startsAt)} detail={formatTime(event.startsAt) || 'Time to be announced'} />
            <InfoRow icon={<MapPin size={20} color={tokens.color.text.primary} />} title={event.locationDisplay.split(',')[0] || 'Location to be announced'} detail={event.locationDisplay.split(',')[1]?.trim() || 'No further address listed'} />
          </View>
          <View style={styles.aboutBlock}>
            <Text style={styles.aboutTitle}>About</Text>
            <Text style={styles.aboutText}>{event.publicDescription || 'More details soon.'}</Text>
          </View>
          <View style={styles.participationCard}>
            <Text style={styles.aboutTitle}>Participation</Text>
            <Text style={styles.formHelp}>Public roles are open for applications. Applying does not affect your ticket.</Text>
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
                        <Pill>{publicRoleAvailabilityLabel(role.capacity)}</Pill>
                      </View>
                      <Field
                        value={draft.applicantName}
                        onChangeText={(value) => updateRoleDraft(role.id, (current) => ({ ...current, applicantName: value, submitted: false, error: null }))}
                        placeholder="Applicant name"
                        editable={!draft.submitting && !draft.submitted}
                      />
                      <Field
                        value={draft.applicantEmail}
                        onChangeText={(value) => updateRoleDraft(role.id, (current) => ({ ...current, applicantEmail: value, submitted: false, error: null }))}
                        autoCapitalize="none"
                        keyboardType="email-address"
                        placeholder="Applicant email"
                        editable={!draft.submitting && !draft.submitted}
                      />
                      <Field
                        value={draft.message}
                        onChangeText={(value) => updateRoleDraft(role.id, (current) => ({ ...current, message: value, submitted: false, error: null }))}
                        multiline
                        placeholder="Message (optional)"
                        style={styles.messageInput}
                        editable={!draft.submitting && !draft.submitted}
                      />
                      <PrimaryButton variant="secondary" disabled={!canSubmit} busy={draft.submitting} onPress={() => void handleRoleSubmit(role)} label={publicRoleApplicationButtonLabel(draft.submitting, draft.submitted)} />
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
                <Field
                  value={email}
                  onChangeText={setEmail}
                  autoCapitalize="none"
                  keyboardType="email-address"
                  placeholder="Email address"
                />
                <Field
                  value={displayName}
                  onChangeText={setDisplayName}
                  placeholder="Display name (optional)"
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
          <PrimaryButton disabled={stickyCtaDisabled} onPress={handleReserve} icon={<TicketIcon size={20} color={tokens.color.text.inverse} />} label={stickyCtaLabel} />
        </View>
      </View>
    </View>
  );
}

function CenteredState({ title, body }: { title: string; body?: string }) {

  const styles = useThemedStyles(createStyles);

  return (
    <View style={styles.centerState}>
      <Text style={styles.centerTitle}>{title}</Text>
      {body ? <Text style={styles.centerBody}>{body}</Text> : null}
    </View>
  );
}

function InfoRow({ icon, title, detail }: { icon: React.ReactNode; title: string; detail: string }) {

  const styles = useThemedStyles(createStyles);

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

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel },
  scroller: { flex: 1 },
  scrollContent: {},
  hero: { width: '100%', height: '45%', minHeight: 350, position: 'relative' },
  heroImage: { width: '100%', height: '100%' },
  heroOverlay: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: 'rgba(0,0,0,0.08)' },
  topBar: { position: 'absolute', top: 48, left: 0, right: 0, paddingHorizontal: 16, flexDirection: 'row', justifyContent: 'space-between', zIndex: 10 },
  roundButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: 'rgba(255,255,255,0.20)', alignItems: 'center', justifyContent: 'center' },
  content: { flex: 1, paddingHorizontal: 20, marginTop: -72, zIndex: 10, paddingBottom: 32 },
  titleCard: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.panel, padding: 20, marginBottom: 22, boxShadow: '0 2px 12px rgba(0,0,0,0.12)', elevation: 2 },
  organizer: { fontSize: 14, fontWeight: '700', letterSpacing: 1.1, color: tokens.color.text.muted, textTransform: 'uppercase' },
  title: { fontSize: tokens.type['title'], fontWeight: '700', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 4, marginBottom: 8 },
  subtitle: { fontSize: 18, color: tokens.color.text.muted, marginBottom: 24 },
  infoList: { gap: 16, marginBottom: 32 },
  infoRow: { flexDirection: 'row', alignItems: 'center', gap: 16 },
  infoIcon: { width: 48, height: 48, borderRadius: tokens.radius.control, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  infoTitle: { fontWeight: '700', color: tokens.color.text.secondary },
  infoDetail: { fontSize: 14, color: tokens.color.text.muted },
  aboutBlock: { marginBottom: 24 },
  aboutTitle: { fontSize: 18, fontWeight: '700', marginBottom: 8, color: tokens.color.text.primary },
  aboutText: { color: tokens.color.text.secondary, lineHeight: 22 },
  participationCard: { gap: 12, backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.card, padding: 18, marginBottom: 24 },
  roleList: { gap: 14 },
  roleCard: { gap: 12, backgroundColor: tokens.color.surface.panel, borderRadius: 20, padding: 14 },
  roleHeader: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 },
  roleTitleWrap: { flex: 1 },
  roleName: { color: tokens.color.text.primary, fontSize: 16, fontWeight: '700' },
  roleDescription: { color: tokens.color.text.muted, lineHeight: 20, marginTop: 4 },
  formCard: { gap: 12, backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.card, padding: 18, marginBottom: 48 },
  formHelp: { color: tokens.color.text.muted, lineHeight: 20 },
  messageInput: { minHeight: 96 },
  signedInCard: { backgroundColor: tokens.color.surface.panel, borderRadius: 18, padding: 14, gap: 3 },
  signedInLabel: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '700', textTransform: 'uppercase', letterSpacing: 1 },
  signedInName: { color: tokens.color.text.primary, fontSize: 16, fontWeight: '700' },
  signedInEmail: { color: tokens.color.text.muted, fontWeight: '700' },
  errorText: { color: tokens.color.status.danger, fontWeight: '600', lineHeight: 20 },
  successText: { color: tokens.color.status.success, fontWeight: '700', lineHeight: 20 },
  stickyBar: { position: 'absolute', left: 0, right: 0, bottom: 0, backgroundColor: tokens.color.surface.panel, borderTopWidth: 1, borderTopColor: tokens.color.surface.inset, paddingTop: 14, paddingHorizontal: 16, flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', boxShadow: '0 -2px 10px rgba(0,0,0,0.08)', elevation: 8 },
  priceLabel: { color: tokens.color.text.muted, fontSize: 14, fontWeight: '600' },
  price: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '700' },
  remaining: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '600' },
  stickyAction: { alignItems: 'flex-end', gap: 6 },
  stickyWarning: { color: tokens.color.status.danger, fontSize: tokens.type['label'], fontWeight: '700' },
  centerState: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 28, backgroundColor: tokens.color.surface.panel },
  centerTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '700', textAlign: 'center', marginBottom: 8 },
  centerBody: { color: tokens.color.text.muted, fontSize: tokens.type['body'], lineHeight: 22, textAlign: 'center' },
});
