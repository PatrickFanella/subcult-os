import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { Link, router, type Href } from 'expo-router';
import { Building2, CalendarPlus, ClipboardCheck, ListChecks, Mic2, Pencil, QrCode, ShieldCheck, UserPlus, Users } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Image, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { formatDate, formatTime } from '@/api/format';
import { listWorkspaceEvents } from '@/api/staff';
import type { EventDTO, WorkspaceSummaryDTO } from '@/api/types';
import { useAuth } from '@/auth/AuthContext';
import { eventArtwork } from '@/data/eventArtwork';
import { clearSelectedEventID, loadStaffSelection, storeSelectedEventID, storeSelectedWorkspaceID } from '@/staff/selectionStore';
import { nextSelectedEvent, selectedEventLabel, selectedWorkspaceLabel, staffSelectionEmptyCopy, staffSelectionReady } from '@/modules/staff/staffOperatorModel';
import { AppChrome } from '@/ui/AppChrome';

export default function StaffScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const { user, loading: authLoading, signOut } = useAuth();
  const [selectedWorkspaceID, setSelectedWorkspaceID] = useState<string | null>(null);
  const [events, setEvents] = useState<EventDTO[]>([]);
  const [selectedEventID, setSelectedEventID] = useState<string | null>(null);
  const [loadingEvents, setLoadingEvents] = useState(false);
  const [hydratingSelection, setHydratingSelection] = useState(true);
  const [eventError, setEventError] = useState<string | null>(null);

  const workspaceKey = user?.workspaces.map((workspace) => workspace.id).join('|') ?? '';
  const workspace = selectedWorkspaceID ? user?.workspaces.find((candidate) => candidate.id === selectedWorkspaceID) ?? null : null;

  useEffect(() => {
    if (!user) {
      setHydratingSelection(false);
      setSelectedWorkspaceID(null);
      setSelectedEventID(null);
      setEvents([]);
      return;
    }

    let cancelled = false;
    const workspaces = user.workspaces;

    async function hydrateSelection() {
      setHydratingSelection(true);
      const stored = await loadStaffSelection();
      if (cancelled) return;

      const storedWorkspace = stored.workspaceID && workspaces.some((candidate) => candidate.id === stored.workspaceID) ? stored.workspaceID : null;
      const nextWorkspaceID = storedWorkspace ?? workspaces[0]?.id ?? null;

      setSelectedWorkspaceID(nextWorkspaceID);
      setSelectedEventID(stored.eventID);
      setHydratingSelection(false);
    }

    void hydrateSelection();

    return () => {
      cancelled = true;
    };
  }, [user?.id, workspaceKey]);

  useEffect(() => {
    if (hydratingSelection) return;
    if (!workspace) {
      setEvents([]);
      setSelectedEventID(null);
      setLoadingEvents(false);
      return;
    }
    const workspaceID = workspace.id;
    let cancelled = false;

    async function loadEvents() {
      setLoadingEvents(true);
      setEventError(null);
      try {
        const loaded = await listWorkspaceEvents(workspaceID);
        if (!cancelled) {
          setEvents(loaded);
          setSelectedEventID((current) => {
            const nextID = nextSelectedEvent(current, loaded)?.id ?? null;
            if (nextID) void storeSelectedEventID(nextID);
            return nextID;
          });
        }
      } catch (caught) {
        if (!cancelled) {
          const message = caught instanceof Error ? caught.message : 'Unable to load workspace events';
          setEventError(message.toLowerCase().includes('forbidden') ? 'Forbidden: this account cannot load events for the selected workspace. Try another workspace or sign in again.' : message);
          setEvents([]);
          setSelectedEventID(null);
          void clearSelectedEventID();
        }
      } finally {
        if (!cancelled) setLoadingEvents(false);
      }
    }

    void loadEvents();
    return () => {
      cancelled = true;
    };
  }, [workspace?.id, hydratingSelection]);

  if (authLoading || hydratingSelection) return <CenteredStaffState title="Checking dashboard…" />;

  if (!user) {
    return (
      <AppChrome>
        <View style={styles.authGate}>
          <Text style={styles.emptyTitle}>Staff mode requires sign in</Text>
          <Text style={styles.emptyBody}>Sign in with a workspace account to use door and event operations.</Text>
          <Link href={{ pathname: '/login', params: { next: '/staff' } }} style={styles.authButton}>Sign in</Link>
        </View>
      </AppChrome>
    );
  }

  if (!workspace) {
    return (
      <AppChrome>
        <View style={styles.authGate}>
          <Text style={styles.emptyTitle}>No workspace found</Text>
          <Text style={styles.emptyBody}>Create a workspace to start drafting fake events, roles, and run-of-show items.</Text>
          <Link href={{ pathname: '/workspace-create', params: { next: '/staff' } }} style={styles.authButton}>Create workspace</Link>
          <Text onPress={() => void signOut()} style={styles.signOut}>Sign out {user.email}</Text>
        </View>
      </AppChrome>
    );
  }

  const activeEvent = events.find((event) => event.id === selectedEventID) ?? null;
  const selectionReady = staffSelectionReady(workspace, activeEvent);
  const activeEventDate = activeEvent ? `${formatDate(activeEvent.startsAt)} • ${formatTime(activeEvent.startsAt)}` : eventError ?? 'Create an event to use door tools';
  const initials = (user.displayName || user.email).slice(0, 2).toUpperCase();
  const remainingTickets = activeEvent ? Math.max(activeEvent.ticketAllocation - activeEvent.reservedCount, 0) : null;

  return (
    <AppChrome>
      <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
        <View style={styles.header}>
          <View>
            <Text style={styles.staffPill}>{selectedWorkspaceLabel(workspace)}</Text>
            <Text style={styles.title}>Dashboard</Text>
          </View>
          <Link href="/profile" asChild><Pressable style={styles.avatar}><Text style={styles.avatarText}>{initials}</Text></Pressable></Link>
        </View>

        <View style={styles.selectorSection}>
          <Text style={styles.sectionLabel}>Workspace</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.selectorRow}>
            {user.workspaces.map((candidate) => (
              <WorkspaceChip
                key={candidate.id}
                workspace={candidate}
                selected={candidate.id === workspace.id}
                onPress={() => {
                  setSelectedWorkspaceID(candidate.id);
                  setSelectedEventID(null);
                  setEvents([]);
                  setEventError(null);
                  void storeSelectedWorkspaceID(candidate.id);
                  void clearSelectedEventID();
                }}
              />
            ))}
            <Link href={{ pathname: '/workspace-create', params: { next: '/staff' } }} style={styles.createWorkspaceChip}>
              <View style={styles.createWorkspaceInner}>
                <Building2 size={18} color={tokens.color.status.info} />
                <Text style={styles.createWorkspaceText}>New workspace</Text>
              </View>
            </Link>
          </ScrollView>
        </View>

        <View style={styles.activeEvent}>
          {activeEvent ? <Image source={{ uri: activeEvent.imageUrl || eventArtwork(activeEvent.publicSlug ?? activeEvent.id) }} style={styles.activeImage} resizeMode="cover" /> : <View style={styles.activeImagePlaceholder} />}
          <View style={styles.activeEventCopy}>
            <Text style={styles.mutedTiny}>Active Event</Text>
            <Text style={styles.activeTitle}>{activeEvent ? selectedEventLabel(activeEvent) : (loadingEvents ? 'Loading events…' : 'No workspace events')}</Text>
            <Text style={styles.mutedTiny}>{activeEventDate}</Text>
          </View>
        </View>

        <View style={styles.selectorSection}>
          <Text style={styles.sectionLabel}>Event selector</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.selectorRow}>
            {events.map((event) => {
              const selected = event.id === activeEvent?.id;
              return (
                <Pressable key={event.id} onPress={() => {
                  setSelectedEventID(event.id);
                  void storeSelectedEventID(event.id);
                }} style={[styles.eventChip, selected && styles.eventChipActive]}>
                  <Text style={[styles.eventChipTitle, selected && styles.eventChipTitleActive]}>{event.title}</Text>
                  <Text style={[styles.eventChipMeta, selected && styles.eventChipMetaActive]}>{formatDate(event.startsAt)} · {event.status}</Text>
                </Pressable>
              );
            })}
            {!loadingEvents && events.length === 0 ? <Text style={styles.emptyInline}>{staffSelectionEmptyCopy(user.workspaces, events)}</Text> : null}
          </ScrollView>
        </View>

        <View style={styles.grid}>
          <DashboardCard to={{ pathname: '/event-edit', params: { workspaceId: workspace.id } }} icon={<CalendarPlus size={24} color={tokens.color.text.inverse} />} title="Create Event" subtitle="Draft & publish" primary />
          <DashboardCard to={activeEvent ? { pathname: '/event-edit', params: { eventId: activeEvent.id, workspaceId: workspace.id } } : { pathname: '/event-edit', params: { workspaceId: workspace.id } }} icon={<Pencil size={24} color={tokens.color.text.primary} />} title="Edit Event" subtitle="Basics & tickets" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/readiness', params: { eventId: activeEvent.id } } : '/staff'} icon={<ClipboardCheck size={24} color={tokens.color.text.inverse} />} title="Readiness" subtitle="Setup checklist" primary />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/scanner', params: { eventId: activeEvent.id } } : '/staff'} icon={<QrCode size={24} color={tokens.color.text.inverse} />} title="Scan Tickets" subtitle="Run the door" primary />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/run-of-show', params: { eventId: activeEvent.id } } : '/staff'} icon={<ListChecks size={24} color={tokens.color.text.primary} />} title="Run of Show" subtitle="Event timeline" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/roles', params: { eventId: activeEvent.id } } : '/staff'} icon={<UserPlus size={24} color={tokens.color.text.primary} />} title="Roles" subtitle="Applicants" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/door', params: { eventId: activeEvent.id } } : '/staff'} icon={<Users size={24} color={tokens.color.text.primary} />} title="Guest List" subtitle="VIP & Comp" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/event-dashboard', params: { eventId: activeEvent.id } } : '/staff'} icon={<Mic2 size={24} color={tokens.color.text.primary} />} title="Live Event" subtitle="Counters" />
        </View>

        <View style={styles.statsSection}>
          <Text style={styles.sectionLabel}>Quick Stats</Text>
          <View style={styles.statBox}>
            <View>
              <Text style={styles.statMuted}>Tickets</Text>
              <Text style={styles.statValue}>{remainingTickets ?? '—'} <Text style={styles.statTotal}>remaining</Text></Text>
            </View>
            <View style={styles.statPercent}><ShieldCheck size={16} color={tokens.color.status.success} /><Text style={styles.percentText}>{activeEvent ? activeEvent.status : '—'}</Text></View>
          </View>
          {eventError ? <Text style={styles.error}>{eventError}</Text> : null}
          <Text onPress={() => void signOut()} style={styles.signOut}>Sign out {user.email}</Text>
        </View>
      </ScrollView>
    </AppChrome>
  );
}

function WorkspaceChip({ workspace, selected, onPress }: { workspace: WorkspaceSummaryDTO; selected: boolean; onPress: () => void }) {

  const styles = useThemedStyles(createStyles);

  return (
    <Pressable onPress={onPress} style={[styles.workspaceChip, selected && styles.workspaceChipActive]}>
      <Text style={[styles.workspaceChipTitle, selected && styles.workspaceChipTitleActive]}>{workspace.name}</Text>
      <Text style={[styles.workspaceChipMeta, selected && styles.workspaceChipMetaActive]}>{workspace.role}</Text>
    </Pressable>
  );
}

function CenteredStaffState({ title }: { title: string }) {

  const styles = useThemedStyles(createStyles);

  return (
    <AppChrome>
      <View style={styles.authGate}><Text style={styles.emptyTitle}>{title}</Text></View>
    </AppChrome>
  );
}

function DashboardCard({ to, icon, title, subtitle, primary, disabled }: { to: Href; icon: React.ReactNode; title: string; subtitle: string; primary?: boolean; disabled?: boolean }) {

  const styles = useThemedStyles(createStyles);

  return (
    <Pressable disabled={disabled} onPress={() => router.push(to)} style={[styles.dashboardCard, primary ? styles.dashboardCardPrimary : styles.dashboardCardNeutral, disabled && styles.dashboardCardDisabled]}>
      <View style={[styles.cardIcon, primary && styles.cardIconPrimary]}>{icon}</View>
      <View>
        <Text style={[styles.cardTitle, primary && styles.cardTitlePrimary, disabled && styles.cardTextDisabled]}>{title}</Text>
        <Text style={[styles.cardSubtitle, primary && styles.cardSubtitlePrimary, disabled && styles.cardTextDisabled]}>{disabled ? 'Select an event first' : subtitle}</Text>
      </View>
    </Pressable>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel },
  content: { padding: 24, paddingTop: 64, paddingBottom: 32 },
  authGate: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: tokens.color.surface.panel },
  authButton: { marginTop: 8, backgroundColor: tokens.color.action.primary, color: tokens.color.text.inverse, paddingHorizontal: 22, paddingVertical: 14, borderRadius: tokens.radius.control, overflow: 'hidden', fontWeight: '700' },
  emptyTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '700', textAlign: 'center', marginBottom: 8 },
  emptyBody: { color: tokens.color.text.muted, textAlign: 'center', lineHeight: 21, marginBottom: 8 },
  emptyInline: { color: tokens.color.text.muted, fontWeight: '700' },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 32 },
  staffPill: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '700' },
  title: { fontSize: tokens.type['title'], fontWeight: '700', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 8 },
  avatar: { width: 48, height: 48, borderRadius: tokens.radius.card, backgroundColor: tokens.color.action.primary, alignItems: 'center', justifyContent: 'center' },
  avatarText: { color: tokens.color.text.inverse, fontWeight: '700' },
  activeEvent: { backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.panel, padding: 20, marginBottom: 24, flexDirection: 'row', gap: 16, alignItems: 'center' },
  activeEventCopy: { flex: 1 },
  activeImage: { width: 64, height: 64, borderRadius: 16 },
  activeImagePlaceholder: { width: 64, height: 64, borderRadius: tokens.radius.control, backgroundColor: tokens.color.border.subtle },
  mutedTiny: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '600', marginBottom: 4 },
  activeTitle: { color: tokens.color.text.primary, fontWeight: '700' },
  selectorSection: { marginBottom: 24 },
  selectorRow: { gap: 10, paddingRight: 24 },
  workspaceChip: { minWidth: 180, borderRadius: 20, borderWidth: 1, borderColor: tokens.color.border.subtle, backgroundColor: tokens.color.surface.panel, padding: 14 },
  workspaceChipActive: { borderColor: tokens.color.text.primary, backgroundColor: tokens.color.action.primary },
  workspaceChipTitle: { color: tokens.color.text.primary, fontWeight: '700' },
  workspaceChipTitleActive: { color: tokens.color.text.inverse },
  workspaceChipMeta: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '700', marginTop: 6, textTransform: 'uppercase', letterSpacing: 1 },
  workspaceChipMetaActive: { color: tokens.color.text.inverse },
  createWorkspaceChip: { minWidth: 180, borderRadius: 20, borderWidth: 1, borderColor: tokens.color.border.strong, backgroundColor: tokens.color.statusSurface.info, padding: 14, overflow: 'hidden' },
  createWorkspaceInner: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  createWorkspaceText: { color: tokens.color.text.primary, fontWeight: '700' },
  eventChip: { width: 190, borderRadius: 20, backgroundColor: tokens.color.surface.inset, padding: 14 },
  eventChipActive: { backgroundColor: tokens.color.action.primary },
  eventChipTitle: { color: tokens.color.text.primary, fontWeight: '700' },
  eventChipTitleActive: { color: tokens.color.text.inverse },
  eventChipMeta: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '600', marginTop: 6 },
  eventChipMetaActive: { color: tokens.color.text.inverse },
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 16, marginBottom: 32 },
  dashboardCard: { width: '47.5%', height: 160, borderRadius: tokens.radius.card, padding: 20, justifyContent: 'space-between', overflow: 'hidden', borderWidth: 1, borderColor: 'transparent' },
  dashboardCardDisabled: { opacity: 1, backgroundColor: tokens.color.surface.inset, borderColor: tokens.color.border.subtle },
  dashboardCardPrimary: { backgroundColor: tokens.color.surface.immersive },
  dashboardCardNeutral: { backgroundColor: tokens.color.surface.inset },
  cardIcon: { width: 40, height: 40, borderRadius: 20, backgroundColor: 'rgba(255,255,255,0.20)', alignItems: 'center', justifyContent: 'center' },
  cardIconPrimary: { backgroundColor: 'rgba(255,255,255,0.18)' },
  cardTitle: { fontSize: 18, fontWeight: '700', color: tokens.color.text.primary, lineHeight: 22 },
  cardTitlePrimary: { color: tokens.color.text.inverse },
  cardSubtitle: { fontSize: tokens.type['label'], opacity: 0.7, marginTop: 4, color: tokens.color.text.primary },
  cardSubtitlePrimary: { color: tokens.color.text.inverse },
  cardTextDisabled: { color: tokens.color.text.muted },
  statsSection: { borderTopWidth: 1, borderTopColor: tokens.color.surface.inset, paddingTop: 32 },
  sectionLabel: { fontSize: 14, fontWeight: '700', color: tokens.color.text.muted, textTransform: 'uppercase', letterSpacing: 1.2, marginBottom: 16 },
  statBox: { backgroundColor: tokens.color.surface.inset, padding: 20, borderRadius: tokens.radius.control, flexDirection: 'row', alignItems: 'flex-end', justifyContent: 'space-between' },
  statMuted: { color: tokens.color.text.muted, fontSize: 14, marginBottom: 4 },
  statValue: { fontSize: tokens.type['title'], fontWeight: '700', color: tokens.color.text.primary },
  statTotal: { fontSize: 16, fontWeight: '400', color: tokens.color.text.muted },
  statPercent: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  percentText: { color: tokens.color.status.success, fontWeight: '700', fontSize: 14 },
  error: { color: tokens.color.status.danger, fontWeight: '700', lineHeight: 20, marginTop: 12 },
  signOut: { color: tokens.color.text.muted, fontWeight: '700', textAlign: 'center', marginTop: 18 },
});
