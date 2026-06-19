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
                <Building2 size={18} color="#2563eb" />
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
          <DashboardCard to={{ pathname: '/event-edit', params: { workspaceId: workspace.id } }} icon={<CalendarPlus size={24} color="#ffffff" />} title="Create Event" subtitle="Draft & publish" primary />
          <DashboardCard to={activeEvent ? { pathname: '/event-edit', params: { eventId: activeEvent.id, workspaceId: workspace.id } } : { pathname: '/event-edit', params: { workspaceId: workspace.id } }} icon={<Pencil size={24} color="#171717" />} title="Edit Event" subtitle="Basics & tickets" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/readiness', params: { eventId: activeEvent.id } } : '/staff'} icon={<ClipboardCheck size={24} color="#ffffff" />} title="Readiness" subtitle="Setup checklist" primary />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/scanner', params: { eventId: activeEvent.id } } : '/staff'} icon={<QrCode size={24} color="#ffffff" />} title="Scan Tickets" subtitle="Run the door" primary />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/run-of-show', params: { eventId: activeEvent.id } } : '/staff'} icon={<ListChecks size={24} color="#171717" />} title="Run of Show" subtitle="Event timeline" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/roles', params: { eventId: activeEvent.id } } : '/staff'} icon={<UserPlus size={24} color="#171717" />} title="Roles" subtitle="Applicants" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/door', params: { eventId: activeEvent.id } } : '/staff'} icon={<Users size={24} color="#171717" />} title="Guest List" subtitle="VIP & Comp" />
          <DashboardCard disabled={!selectionReady} to={activeEvent ? { pathname: '/event-dashboard', params: { eventId: activeEvent.id } } : '/staff'} icon={<Mic2 size={24} color="#171717" />} title="Live Event" subtitle="Counters" />
        </View>

        <View style={styles.statsSection}>
          <Text style={styles.sectionLabel}>Quick Stats</Text>
          <View style={styles.statBox}>
            <View>
              <Text style={styles.statMuted}>Tickets</Text>
              <Text style={styles.statValue}>{remainingTickets ?? '—'} <Text style={styles.statTotal}>remaining</Text></Text>
            </View>
            <View style={styles.statPercent}><ShieldCheck size={16} color="#22c55e" /><Text style={styles.percentText}>{activeEvent ? activeEvent.status : '—'}</Text></View>
          </View>
          {eventError ? <Text style={styles.error}>{eventError}</Text> : null}
          <Text onPress={() => void signOut()} style={styles.signOut}>Sign out {user.email}</Text>
        </View>
      </ScrollView>
    </AppChrome>
  );
}

function WorkspaceChip({ workspace, selected, onPress }: { workspace: WorkspaceSummaryDTO; selected: boolean; onPress: () => void }) {
  return (
    <Pressable onPress={onPress} style={[styles.workspaceChip, selected && styles.workspaceChipActive]}>
      <Text style={[styles.workspaceChipTitle, selected && styles.workspaceChipTitleActive]}>{workspace.name}</Text>
      <Text style={[styles.workspaceChipMeta, selected && styles.workspaceChipMetaActive]}>{workspace.role}</Text>
    </Pressable>
  );
}

function CenteredStaffState({ title }: { title: string }) {
  return (
    <AppChrome>
      <View style={styles.authGate}><Text style={styles.emptyTitle}>{title}</Text></View>
    </AppChrome>
  );
}

function DashboardCard({ to, icon, title, subtitle, primary, disabled }: { to: Href; icon: React.ReactNode; title: string; subtitle: string; primary?: boolean; disabled?: boolean }) {
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

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  content: { padding: 24, paddingTop: 64, paddingBottom: 32 },
  authGate: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: '#ffffff' },
  authButton: { marginTop: 8, backgroundColor: '#171717', color: '#ffffff', paddingHorizontal: 22, paddingVertical: 14, borderRadius: 16, overflow: 'hidden', fontWeight: '800' },
  emptyTitle: { color: '#171717', fontSize: 24, fontWeight: '800', textAlign: 'center', marginBottom: 8 },
  emptyBody: { color: '#737373', textAlign: 'center', lineHeight: 21, marginBottom: 8 },
  emptyInline: { color: '#737373', fontWeight: '700' },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 32 },
  staffPill: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 8 },
  avatar: { width: 48, height: 48, borderRadius: 24, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center' },
  avatarText: { color: '#ffffff', fontWeight: '800' },
  activeEvent: { backgroundColor: '#f5f5f5', borderRadius: 28, padding: 20, marginBottom: 24, flexDirection: 'row', gap: 16, alignItems: 'center' },
  activeEventCopy: { flex: 1 },
  activeImage: { width: 64, height: 64, borderRadius: 16 },
  activeImagePlaceholder: { width: 64, height: 64, borderRadius: 16, backgroundColor: '#e5e5e5' },
  mutedTiny: { color: '#737373', fontSize: 12, fontWeight: '600', marginBottom: 4 },
  activeTitle: { color: '#171717', fontWeight: '800' },
  selectorSection: { marginBottom: 24 },
  selectorRow: { gap: 10, paddingRight: 24 },
  workspaceChip: { minWidth: 180, borderRadius: 20, borderWidth: 1, borderColor: '#e5e5e5', backgroundColor: '#ffffff', padding: 14 },
  workspaceChipActive: { borderColor: '#171717', backgroundColor: '#171717' },
  workspaceChipTitle: { color: '#171717', fontWeight: '800' },
  workspaceChipTitleActive: { color: '#ffffff' },
  workspaceChipMeta: { color: '#737373', fontSize: 12, fontWeight: '700', marginTop: 6, textTransform: 'uppercase', letterSpacing: 1 },
  workspaceChipMetaActive: { color: 'rgba(255,255,255,0.65)' },
  createWorkspaceChip: { minWidth: 180, borderRadius: 20, borderWidth: 1, borderColor: '#bfdbfe', backgroundColor: '#eff6ff', padding: 14, overflow: 'hidden' },
  createWorkspaceInner: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  createWorkspaceText: { color: '#2563eb', fontWeight: '900' },
  eventChip: { width: 190, borderRadius: 20, backgroundColor: '#f5f5f5', padding: 14 },
  eventChipActive: { backgroundColor: '#171717' },
  eventChipTitle: { color: '#171717', fontWeight: '800' },
  eventChipTitleActive: { color: '#ffffff' },
  eventChipMeta: { color: '#737373', fontSize: 12, fontWeight: '600', marginTop: 6 },
  eventChipMetaActive: { color: 'rgba(255,255,255,0.65)' },
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 16, marginBottom: 32 },
  dashboardCard: { width: '47.5%', height: 160, borderRadius: 24, padding: 20, justifyContent: 'space-between', overflow: 'hidden', borderWidth: 1, borderColor: 'transparent' },
  dashboardCardDisabled: { opacity: 1, backgroundColor: '#fafafa', borderColor: '#e5e5e5' },
  dashboardCardPrimary: { backgroundColor: '#000000' },
  dashboardCardNeutral: { backgroundColor: '#f5f5f5' },
  cardIcon: { width: 40, height: 40, borderRadius: 20, backgroundColor: 'rgba(255,255,255,0.20)', alignItems: 'center', justifyContent: 'center' },
  cardIconPrimary: { backgroundColor: 'rgba(255,255,255,0.18)' },
  cardTitle: { fontSize: 18, fontWeight: '800', color: '#171717', lineHeight: 22 },
  cardTitlePrimary: { color: '#ffffff' },
  cardSubtitle: { fontSize: 12, opacity: 0.7, marginTop: 4, color: '#171717' },
  cardSubtitlePrimary: { color: '#ffffff' },
  cardTextDisabled: { color: '#a3a3a3' },
  statsSection: { borderTopWidth: 1, borderTopColor: '#f5f5f5', paddingTop: 32 },
  sectionLabel: { fontSize: 14, fontWeight: '800', color: '#a3a3a3', textTransform: 'uppercase', letterSpacing: 1.2, marginBottom: 16 },
  statBox: { backgroundColor: '#fafafa', padding: 20, borderRadius: 16, flexDirection: 'row', alignItems: 'flex-end', justifyContent: 'space-between' },
  statMuted: { color: '#737373', fontSize: 14, marginBottom: 4 },
  statValue: { fontSize: 32, fontWeight: '800', color: '#171717' },
  statTotal: { fontSize: 16, fontWeight: '400', color: '#a3a3a3' },
  statPercent: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  percentText: { color: '#22c55e', fontWeight: '800', fontSize: 14 },
  error: { color: '#dc2626', fontWeight: '700', lineHeight: 20, marginTop: 12 },
  signOut: { color: '#737373', fontWeight: '700', textAlign: 'center', marginTop: 18 },
});
