import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { createWorkspace } from '@/api/workspaces';
import { useAuth } from '@/auth/AuthContext';
import { safeBack } from '@/navigation/safeBack';
import { storeSelectedWorkspaceID } from '@/staff/selectionStore';

export default function WorkspaceCreateScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

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
          <ChevronLeft size={24} color={tokens.color.text.primary} />
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

  const styles = useThemedStyles(createStyles);

  return (
    <View style={styles.centered}>
      <Text style={styles.centeredTitle}>{title}</Text>
      {body ? <Text style={styles.centeredBody}>{body}</Text> : null}
    </View>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 24, paddingTop: 56, paddingBottom: 12 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  headerCopy: { flex: 1 },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '800' },
  title: { fontSize: tokens.type['title'], fontWeight: '800', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 6 },
  subtitle: { color: tokens.color.text.muted, fontWeight: '700', marginTop: 4, lineHeight: 21 },
  content: { gap: 18, padding: 24, paddingTop: 8, paddingBottom: 40 },
  panel: { backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.panel, padding: 20, gap: 12 },
  sectionTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '800' },
  input: { minHeight: tokens.size.field, borderRadius: tokens.radius.control, borderWidth: 1, borderColor: tokens.color.border.subtle, backgroundColor: tokens.color.surface.panel, color: tokens.color.text.primary, paddingHorizontal: 16, fontSize: 16, fontWeight: '800' },
  helper: { color: tokens.color.text.muted, lineHeight: 20, fontWeight: '600' },
  error: { color: tokens.color.status.danger, fontWeight: '800', lineHeight: 20 },
  primaryButton: { minHeight: tokens.size.field, borderRadius: 18, backgroundColor: tokens.color.action.primary, alignItems: 'center', justifyContent: 'center' },
  primaryButtonText: { color: tokens.color.text.inverse, fontWeight: '900', fontSize: 16 },
  disabledButton: { opacity: 0.45 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: tokens.color.surface.panel, padding: 24 },
  centeredTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '900', textAlign: 'center' },
  centeredBody: { color: tokens.color.text.muted, textAlign: 'center', marginTop: 8, lineHeight: 21 },
});
