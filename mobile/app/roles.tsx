import { useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Plus, Users } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { createEventRole, listEventRoleApplications, listEventRoles, reviewEventRoleApplication } from '@/api/staff';
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
  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const [roles, setRoles] = useState<EventRoleDTO[]>([]);
  const [applications, setApplications] = useState<EventRoleApplicationDTO[]>([]);
  const [loading, setLoading] = useState(Boolean(eventID));
  const [error, setError] = useState<string | null>(eventID ? null : 'Missing event ID. Open Roles from Staff mode.');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [capacity, setCapacity] = useState('1');
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

  async function createRole() {
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
      const created = await createEventRole(eventID, { name: trimmedName, description: description.trim(), capacity: nextCapacity, public: true });
      setRoles((current) => [created, ...current]);
      setName('');
      setDescription('');
      setCapacity('1');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to create role');
    } finally {
      setCreating(false);
    }
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
          <ChevronLeft size={24} color="#171717" />
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
          <Text style={styles.panelTitle}>Create public role</Text>
          <TextInput value={name} onChangeText={setName} placeholder="Door volunteer, performer, vendor…" placeholderTextColor="#a3a3a3" style={styles.input} />
          <TextInput value={capacity} onChangeText={(value) => setCapacity(value.replace(/[^0-9]/g, ''))} placeholder="Capacity" placeholderTextColor="#a3a3a3" keyboardType="number-pad" style={styles.input} />
          <TextInput value={description} onChangeText={setDescription} placeholder="What should applicants know?" placeholderTextColor="#a3a3a3" multiline style={[styles.input, styles.textArea]} />
          <Pressable disabled={creating} onPress={() => void createRole()} style={[styles.primaryButton, creating && styles.disabled]}>
            <Plus size={18} color="#ffffff" /><Text style={styles.primaryButtonText}>{creating ? 'Creating…' : 'Create role'}</Text>
          </Pressable>
        </View>

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Open roles</Text>
          {roles.length === 0 ? <Text style={styles.message}>No roles yet.</Text> : null}
          {roles.map((role) => (
            <View key={role.id} style={styles.roleCard}>
              <View style={styles.roleIcon}><Users size={18} color="#171717" /></View>
              <View style={styles.roleCopy}>
                <Text style={styles.roleName}>{role.name}</Text>
                <Text style={styles.roleMeta}>{role.capacity} spots · {role.public ? 'public' : 'private'} · {role.active ? 'active' : 'inactive'}</Text>
                {role.description ? <Text style={styles.roleDescription}>{role.description}</Text> : null}
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

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff', padding: 24, paddingTop: 56 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, marginBottom: 24 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '900', letterSpacing: -1, color: '#171717', marginTop: 6 },
  content: { gap: 22, paddingBottom: 48 },
  panel: { backgroundColor: '#f5f5f5', borderRadius: 28, padding: 18, gap: 12 },
  panelTitle: { color: '#171717', fontSize: 20, fontWeight: '900' },
  input: { minHeight: 52, borderRadius: 16, backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 14, paddingVertical: 12, fontWeight: '700' },
  textArea: { minHeight: 104, textAlignVertical: 'top', lineHeight: 20 },
  primaryButton: { minHeight: 52, borderRadius: 16, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center', flexDirection: 'row', gap: 8 },
  primaryButtonText: { color: '#ffffff', fontWeight: '900' },
  disabled: { opacity: 0.45 },
  section: { gap: 12 },
  sectionTitle: { color: '#171717', fontSize: 20, fontWeight: '900' },
  message: { color: '#737373', fontWeight: '700', lineHeight: 20 },
  error: { color: '#dc2626', fontWeight: '800', lineHeight: 20 },
  roleCard: { backgroundColor: '#ffffff', borderRadius: 22, borderWidth: 1, borderColor: '#f0f0f0', padding: 16, flexDirection: 'row', gap: 14 },
  roleIcon: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  roleCopy: { flex: 1 },
  roleName: { color: '#171717', fontSize: 18, fontWeight: '900' },
  roleMeta: { color: '#737373', fontSize: 12, fontWeight: '700', marginTop: 3 },
  roleDescription: { color: '#525252', lineHeight: 20, marginTop: 8 },
  applicationCard: { backgroundColor: '#fafafa', borderRadius: 22, padding: 16, gap: 12 },
  applicationHeader: { flexDirection: 'row', justifyContent: 'space-between', gap: 12 },
  applicantName: { color: '#171717', fontSize: 18, fontWeight: '900' },
  statusPill: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 10, paddingVertical: 6, borderRadius: 999, overflow: 'hidden', fontSize: 11, fontWeight: '900', textTransform: 'uppercase' },
  applicationMessage: { color: '#525252', lineHeight: 20 },
  actionsRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  actionButton: { backgroundColor: '#ffffff', borderRadius: 999, paddingHorizontal: 12, paddingVertical: 8, borderWidth: 1, borderColor: '#e5e5e5' },
  actionButtonText: { color: '#171717', fontSize: 12, fontWeight: '900' },
});
