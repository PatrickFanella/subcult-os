import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Plus, Users, X } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { createEventRole, listEventRoleApplications, listEventRoles, reviewEventRoleApplication, updateEventRole } from '@/api/staff';
import type { EventRoleApplicationDTO, EventRoleApplicationStatus, EventRoleDTO } from '@/api/types';
import { safeBack } from '@/navigation/safeBack';

const reviewActions: { label: string; status: EventRoleApplicationStatus }[] = [
  { label: 'Review', status: 'under_review' },
  { label: 'Accept', status: 'accepted' },
  { label: 'Waitlist', status: 'waitlisted' },
  { label: 'Reject', status: 'rejected' },
  { label: 'Confirm', status: 'confirmed' },
];

export default function RolesScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const [roles, setRoles] = useState<EventRoleDTO[]>([]);
  const [applications, setApplications] = useState<EventRoleApplicationDTO[]>([]);
  const [loading, setLoading] = useState(Boolean(eventID));
  const [error, setError] = useState<string | null>(eventID ? null : 'Missing event ID. Open Roles from Staff mode.');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [capacity, setCapacity] = useState('1');
  const [isPublic, setIsPublic] = useState(true);
  const [isActive, setIsActive] = useState(true);
  const [editingRoleID, setEditingRoleID] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [updatingID, setUpdatingID] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!eventID) return;
      setLoading(true);
      setError(null);
      try {
        const [loadedRoles, loadedApplications] = await Promise.all([listEventRoles(eventID), listEventRoleApplications(eventID)]);
        if (!cancelled) {
          setRoles(loadedRoles);
          setApplications(loadedApplications);
        }
      } catch (caught) {
        if (!cancelled) setError(caught instanceof Error ? caught.message : 'Unable to load roles');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [eventID]);

  async function saveRole() {
    const trimmedName = name.trim();
    const nextCapacity = Number.parseInt(capacity, 10);
    if (!trimmedName) {
      setError('Role name is required.');
      return;
    }
    if (Number.isNaN(nextCapacity) || nextCapacity < 0) {
      setError('Capacity must be zero or greater.');
      return;
    }
    setCreating(true);
    setError(null);
    try {
      if (editingRoleID) {
        const updated = await updateEventRole(eventID, editingRoleID, { name: trimmedName, description: description.trim(), capacity: nextCapacity, public: isPublic, active: isActive });
        setRoles((current) => current.map((role) => (role.id === updated.id ? updated : role)));
        stopEditingRole();
      } else {
        const created = await createEventRole(eventID, { name: trimmedName, description: description.trim(), capacity: nextCapacity, public: isPublic });
        setRoles((current) => [created, ...current]);
        resetRoleForm();
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to save role');
    } finally {
      setCreating(false);
    }
  }

  function startEditingRole(role: EventRoleDTO) {
    setEditingRoleID(role.id);
    setName(role.name);
    setDescription(role.description);
    setCapacity(String(role.capacity));
    setIsPublic(role.public);
    setIsActive(role.active);
    setError(null);
  }

  function stopEditingRole() {
    setEditingRoleID(null);
    resetRoleForm();
  }

  function resetRoleForm() {
    setName('');
    setDescription('');
    setCapacity('1');
    setIsPublic(true);
    setIsActive(true);
  }

  async function review(application: EventRoleApplicationDTO, status: EventRoleApplicationStatus) {
    if (updatingID) return;
    setUpdatingID(application.id);
    setError(null);
    try {
      const updated = await reviewEventRoleApplication(eventID, application.id, status);
      setApplications((current) => current.map((candidate) => (candidate.id === updated.id ? updated : candidate)));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to update application');
    } finally {
      setUpdatingID(null);
    }
  }

  function roleName(roleID: string) {
    return roles.find((role) => role.id === roleID)?.name ?? 'Role';
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/staff')} style={styles.backButton}>
          <ChevronLeft size={24} color={tokens.color.text.primary} />
        </Pressable>
        <View>
          <Text style={styles.kicker}>Crew</Text>
          <Text style={styles.title}>Roles</Text>
        </View>
      </View>

      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        {loading ? <Text style={styles.message}>Loading roles…</Text> : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}

        <View style={styles.panel}>
          <View style={styles.panelHeaderRow}>
            <Text style={styles.panelTitle}>{editingRoleID ? 'Edit role' : 'Create role'}</Text>
            {editingRoleID ? (
              <Pressable onPress={stopEditingRole} style={styles.cancelEditButton}><X size={16} color={tokens.color.text.muted} /></Pressable>
            ) : null}
          </View>
          <TextInput value={name} onChangeText={setName} placeholder="Door volunteer, performer, vendor…" placeholderTextColor="#a3a3a3" style={styles.input} />
          <TextInput value={capacity} onChangeText={(value) => setCapacity(value.replace(/[^0-9]/g, ''))} placeholder="Capacity" placeholderTextColor="#a3a3a3" keyboardType="number-pad" style={styles.input} />
          <View style={styles.visibilityRow}>
            <VisibilityButton label="Public" selected={isPublic} onPress={() => setIsPublic(true)} />
            <VisibilityButton label="Private" selected={!isPublic} onPress={() => setIsPublic(false)} />
          </View>
          <Text style={styles.helpText}>{isPublic ? 'Public roles can appear on the attendee-facing event page.' : 'Private roles stay internal for organizer planning.'}</Text>
          {editingRoleID ? (
            <>
              <View style={styles.visibilityRow}>
                <VisibilityButton label="Active" selected={isActive} onPress={() => setIsActive(true)} />
                <VisibilityButton label="Inactive" selected={!isActive} onPress={() => setIsActive(false)} />
              </View>
              <Text style={styles.helpText}>{isActive ? 'Active roles can receive applications if public.' : 'Inactive roles stay in organizer history but are hidden from public application flow.'}</Text>
            </>
          ) : null}
          <TextInput value={description} onChangeText={setDescription} placeholder="What should applicants know?" placeholderTextColor="#a3a3a3" multiline style={[styles.input, styles.textArea]} />
          <Pressable disabled={creating} onPress={() => void saveRole()} style={[styles.primaryButton, creating && styles.disabled]}>
            <Plus size={18} color={tokens.color.text.inverse} /><Text style={styles.primaryButtonText}>{creating ? 'Saving…' : editingRoleID ? 'Save role' : 'Create role'}</Text>
          </Pressable>
        </View>

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Open roles</Text>
          {roles.length === 0 ? <Text style={styles.message}>No roles yet.</Text> : null}
          {roles.map((role) => (
            <View key={role.id} style={styles.roleCard}>
              <View style={styles.roleIcon}><Users size={18} color={tokens.color.text.primary} /></View>
              <View style={styles.roleCopy}>
                <Text style={styles.roleName}>{role.name}</Text>
                <Text style={styles.roleMeta}>{role.capacity} spots · {role.public ? 'public' : 'private'} · {role.active ? 'active' : 'inactive'}</Text>
                {role.description ? <Text style={styles.roleDescription}>{role.description}</Text> : null}
                <Pressable onPress={() => startEditingRole(role)} style={styles.editRoleButton}>
                  <Text style={styles.editRoleButtonText}>Edit role</Text>
                </Pressable>
              </View>
            </View>
          ))}
        </View>

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Applications</Text>
          {applications.length === 0 ? <Text style={styles.message}>No applications yet.</Text> : null}
          {applications.map((application) => (
            <View key={application.id} style={styles.applicationCard}>
              <View style={styles.applicationHeader}>
                <View>
                  <Text style={styles.applicantName}>{application.applicantName}</Text>
                  <Text style={styles.roleMeta}>{roleName(application.roleId)} · {application.applicantEmail}</Text>
                </View>
                <Text style={styles.statusPill}>{application.status}</Text>
              </View>
              {application.message ? <Text style={styles.applicationMessage}>{application.message}</Text> : null}
              <View style={styles.actionsRow}>
                {reviewActions.map((action) => (
                  <Pressable key={action.status} disabled={updatingID === application.id || application.status === action.status} onPress={() => void review(application, action.status)} style={[styles.actionButton, application.status === action.status && styles.disabled]}>
                    <Text style={styles.actionButtonText}>{updatingID === application.id ? 'Saving…' : action.label}</Text>
                  </Pressable>
                ))}
              </View>
            </View>
          ))}
        </View>
      </ScrollView>
    </View>
  );
}

function VisibilityButton({ label, selected, onPress }: { label: string; selected: boolean; onPress: () => void }) {

  const styles = useThemedStyles(createStyles);

  return (
    <Pressable onPress={onPress} style={[styles.visibilityButton, selected && styles.visibilityButtonActive]}>
      <Text style={[styles.visibilityButtonText, selected && styles.visibilityButtonTextActive]}>{label}</Text>
    </Pressable>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel, padding: 24, paddingTop: 56 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '800' },
  title: { fontSize: tokens.type['title'], fontWeight: '900', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 6 },
  content: { gap: 22, paddingBottom: 48 },
  panel: { backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.panel, padding: 18, gap: 12 },
  panelHeaderRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 12 },
  panelTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '900' },
  cancelEditButton: { width: 32, height: 32, borderRadius: tokens.radius.control, backgroundColor: tokens.color.surface.panel, alignItems: 'center', justifyContent: 'center' },
  input: { minHeight: 52, borderRadius: tokens.radius.control, backgroundColor: tokens.color.surface.panel, color: tokens.color.text.primary, paddingHorizontal: 14, paddingVertical: 12, fontWeight: '700' },
  textArea: { minHeight: 104, textAlignVertical: 'top', lineHeight: 20 },
  visibilityRow: { flexDirection: 'row', gap: 10 },
  visibilityButton: { flex: 1, borderRadius: tokens.radius.control, borderWidth: 1, borderColor: tokens.color.border.subtle, backgroundColor: tokens.color.surface.panel, paddingVertical: 13, alignItems: 'center' },
  visibilityButtonActive: { backgroundColor: tokens.color.action.primary, borderColor: tokens.color.text.primary },
  visibilityButtonText: { color: tokens.color.text.primary, fontWeight: '900' },
  visibilityButtonTextActive: { color: tokens.color.text.inverse },
  helpText: { color: tokens.color.text.muted, fontSize: tokens.type['label'], lineHeight: 18, fontWeight: '600' },
  primaryButton: { minHeight: 52, borderRadius: tokens.radius.control, backgroundColor: tokens.color.action.primary, alignItems: 'center', justifyContent: 'center', flexDirection: 'row', gap: 8 },
  primaryButtonText: { color: tokens.color.text.inverse, fontWeight: '900' },
  disabled: { opacity: 0.45 },
  section: { gap: 12 },
  sectionTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '900' },
  message: { color: tokens.color.text.muted, fontWeight: '700', lineHeight: 20 },
  error: { color: tokens.color.status.danger, fontWeight: '800', lineHeight: 20 },
  roleCard: { backgroundColor: tokens.color.surface.panel, borderRadius: 22, borderWidth: 1, borderColor: tokens.color.surface.inset, padding: 16, flexDirection: 'row', gap: 14 },
  roleIcon: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  roleCopy: { flex: 1 },
  roleName: { color: tokens.color.text.primary, fontSize: 18, fontWeight: '900' },
  roleMeta: { color: tokens.color.text.muted, fontSize: tokens.type['label'], fontWeight: '700', marginTop: 3 },
  roleDescription: { color: tokens.color.text.secondary, lineHeight: 20, marginTop: 8 },
  editRoleButton: { alignSelf: 'flex-start', marginTop: 10, borderRadius: tokens.radius.pill, borderWidth: 1, borderColor: tokens.color.border.subtle, backgroundColor: tokens.color.surface.panel, paddingHorizontal: 12, paddingVertical: 8 },
  editRoleButtonText: { color: tokens.color.text.primary, fontSize: tokens.type['label'], fontWeight: '900' },
  applicationCard: { backgroundColor: tokens.color.surface.inset, borderRadius: 22, padding: 16, gap: 12 },
  applicationHeader: { flexDirection: 'row', justifyContent: 'space-between', gap: 12 },
  applicantName: { color: tokens.color.text.primary, fontSize: 18, fontWeight: '900' },
  statusPill: { alignSelf: 'flex-start', color: tokens.color.status.info, backgroundColor: tokens.color.statusSurface.info, paddingHorizontal: 10, paddingVertical: 6, borderRadius: tokens.radius.pill, overflow: 'hidden', fontSize: 11, fontWeight: '900', textTransform: 'uppercase' },
  applicationMessage: { color: tokens.color.text.secondary, lineHeight: 20 },
  actionsRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  actionButton: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.pill, paddingHorizontal: 12, paddingVertical: 8, borderWidth: 1, borderColor: tokens.color.border.subtle },
  actionButtonText: { color: tokens.color.text.primary, fontSize: tokens.type['label'], fontWeight: '900' },
});
