import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { Link } from 'expo-router';
import { LogOut, Settings } from 'lucide-react-native';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { useAuth } from '@/auth/AuthContext';
import { AppChrome } from '@/ui/AppChrome';
import { PrimaryButton } from '@/ui/PrimaryButton';

export default function ProfileScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const { user, loading, signOut } = useAuth();
  const initials = (user?.displayName || user?.email || 'ME').slice(0, 2).toUpperCase();

  if (loading) {
    return <AppChrome><View style={styles.center}><Text style={styles.centerTitle}>Loading profile…</Text></View></AppChrome>;
  }

  if (!user) {
    return (
      <AppChrome>
        <View style={styles.center}>
          <Text style={styles.centerTitle}>Profile requires sign in</Text>
          <Text style={styles.centerBody}>Sign in to manage your account and app settings.</Text>
          <Link href={{ pathname: '/login', params: { next: '/profile' } }} style={styles.primaryLink}>Sign in</Link>
        </View>
      </AppChrome>
    );
  }

  return (
    <AppChrome>
      <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
        <View style={styles.headerCard}>
          <View style={styles.avatar}><Text style={styles.avatarText}>{initials}</Text></View>
          <Text style={styles.title}>{user.displayName || 'SUBCULT user'}</Text>
          <Text style={styles.email}>{user.email}</Text>
        </View>

        <View style={styles.panel}>
          <Text style={styles.sectionTitle}>Workspaces</Text>
          {user.workspaces.length === 0 ? <Text style={styles.muted}>No workspaces yet.</Text> : user.workspaces.map((workspace) => (
            <View key={workspace.id} style={styles.workspaceRow}>
              <View><Text style={styles.workspaceName}>{workspace.name}</Text><Text style={styles.workspaceRole}>{workspace.role}</Text></View>
            </View>
          ))}
        </View>

        <View style={styles.panel}>
          <Link href="/settings" asChild>
            <Pressable style={styles.rowButton}><Settings size={20} color={tokens.color.text.primary} /><Text style={styles.rowButtonText}>App settings</Text></Pressable>
          </Link>
          <PrimaryButton variant="danger" onPress={() => void signOut()} icon={<LogOut size={20} color={tokens.color.status.danger} />} label="Sign out" />
        </View>
      </ScrollView>
    </AppChrome>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.inset },
  content: { padding: 24, paddingTop: 64, paddingBottom: 32, gap: 18 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: tokens.color.surface.panel },
  centerTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '700', textAlign: 'center' },
  centerBody: { color: tokens.color.text.muted, textAlign: 'center', lineHeight: 21, marginTop: 8 },
  primaryLink: { marginTop: 16, backgroundColor: tokens.color.action.primary, color: tokens.color.text.inverse, paddingHorizontal: 22, paddingVertical: 14, borderRadius: tokens.radius.control, overflow: 'hidden', fontWeight: '700' },
  headerCard: { backgroundColor: tokens.color.surface.panel, borderRadius: 30, padding: 24, alignItems: 'center', gap: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)', elevation: 2 },
  avatar: { width: 76, height: 76, borderRadius: 38, backgroundColor: tokens.color.action.primary, alignItems: 'center', justifyContent: 'center', marginBottom: 6 },
  avatarText: { color: tokens.color.text.inverse, fontWeight: '700', fontSize: 24 },
  title: { color: tokens.color.text.primary, fontSize: 28, fontWeight: '700', letterSpacing: -0.8, textAlign: 'center' },
  email: { color: tokens.color.text.muted, fontWeight: '700' },
  panel: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.card, padding: 18, gap: 12 },
  sectionTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '700' },
  muted: { color: tokens.color.text.muted, fontWeight: '700' },
  workspaceRow: { paddingVertical: 10, borderTopWidth: 1, borderTopColor: tokens.color.surface.inset },
  workspaceName: { color: tokens.color.text.primary, fontWeight: '700', fontSize: 16 },
  workspaceRole: { color: tokens.color.text.muted, textTransform: 'uppercase', letterSpacing: 1, fontSize: tokens.type['label'], fontWeight: '700', marginTop: 4 },
  rowButton: { minHeight: 52, borderRadius: tokens.radius.control, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 14, flexDirection: 'row', alignItems: 'center', gap: 10 },
  rowButtonText: { color: tokens.color.text.primary, fontWeight: '700' },
});
