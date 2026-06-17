import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { createWorkspace } from '@/api/workspaces';
import { useAuth } from '@/auth/AuthContext';
import { safeBack } from '@/navigation/safeBack';
import { storeSelectedWorkspaceID } from '@/staff/selectionStore';

export default function WorkspaceCreateScreen() {
  const params = useLocalSearchParams<{ next?: string }>();
  const next = typeof params.next === 'string' && params.next.startsWith('/') ? params.next : '/staff';
  const { user, loading: authLoading, refresh } = useAuth();
  const [name, setName] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    const trimmedName = name.trim();
    setError(null);
    if (!trimmedName) {
      setError('Workspace name is required.');
      return;
    }
    if (trimmedName.length < 3) {
      setError('Use at least 3 characters for the workspace name.');
      return;
    }

    setSaving(true);
    try {
      const workspace = await createWorkspace({ name: trimmedName });
      await storeSelectedWorkspaceID(workspace.id);
      await refresh();
      router.replace(next as never);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to create workspace');
    } finally {
      setSaving(false);
    }
  }

  if (authLoading) return <CenteredState title="Checking session…" />;
  if (!user) return <CenteredState title="Sign in required" body="Create an account before creating a workspace." />;

  return (
    <KeyboardAvoidingView style={styles.screen} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/staff')} style={styles.backButton}>
          <ChevronLeft size={24} color="#171717" />
        </Pressable>
        <View style={styles.headerCopy}>
          <Text style={styles.kicker}>Organizer setup</Text>
          <Text style={styles.title}>Create workspace</Text>
          <Text style={styles.subtitle}>A workspace owns events, staff, roles, and ticket operations.</Text>
        </View>
      </View>

      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <View style={styles.panel}>
          <Text style={styles.sectionTitle}>Workspace name</Text>
          <TextInput
            value={name}
            onChangeText={setName}
            autoCapitalize="words"
            placeholder="Signal Room Chicago"
            placeholderTextColor="#a3a3a3"
            style={styles.input}
          />
          <Text style={styles.helper}>Use the collective, venue, crew, or organizer name attendees and collaborators recognize.</Text>
          {error ? <Text style={styles.error}>{error}</Text> : null}
        </View>

        <Pressable disabled={saving} onPress={() => void submit()} style={[styles.primaryButton, saving && styles.disabledButton]}>
          <Text style={styles.primaryButtonText}>{saving ? 'Creating…' : 'Create workspace'}</Text>
        </Pressable>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

function CenteredState({ title, body }: { title: string; body?: string }) {
  return (
    <View style={styles.centered}>
      <Text style={styles.centeredTitle}>{title}</Text>
      {body ? <Text style={styles.centeredBody}>{body}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 24, paddingTop: 56, paddingBottom: 12 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  headerCopy: { flex: 1 },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 6 },
  subtitle: { color: '#737373', fontWeight: '700', marginTop: 4, lineHeight: 21 },
  content: { gap: 18, padding: 24, paddingTop: 8, paddingBottom: 40 },
  panel: { backgroundColor: '#fafafa', borderRadius: 28, padding: 20, gap: 12 },
  sectionTitle: { color: '#171717', fontSize: 20, fontWeight: '800' },
  input: { minHeight: 56, borderRadius: 16, borderWidth: 1, borderColor: '#e5e5e5', backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 16, fontSize: 16, fontWeight: '800' },
  helper: { color: '#737373', lineHeight: 20, fontWeight: '600' },
  error: { color: '#dc2626', fontWeight: '800', lineHeight: 20 },
  primaryButton: { minHeight: 56, borderRadius: 18, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center' },
  primaryButtonText: { color: '#ffffff', fontWeight: '900', fontSize: 16 },
  disabledButton: { opacity: 0.45 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#ffffff', padding: 24 },
  centeredTitle: { color: '#171717', fontSize: 24, fontWeight: '900', textAlign: 'center' },
  centeredBody: { color: '#737373', textAlign: 'center', marginTop: 8, lineHeight: 21 },
});
