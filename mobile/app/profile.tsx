import { Link } from 'expo-router';
import { LogOut, Settings, UserCircle } from 'lucide-react-native';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { useAuth } from '@/auth/AuthContext';
import { AppChrome } from '@/ui/AppChrome';

export default function ProfileScreen() {
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
            <Pressable style={styles.rowButton}><Settings size={20} color="#171717" /><Text style={styles.rowButtonText}>App settings</Text></Pressable>
          </Link>
          <Pressable onPress={() => void signOut()} style={styles.rowButton}><LogOut size={20} color="#dc2626" /><Text style={[styles.rowButtonText, styles.dangerText]}>Sign out</Text></Pressable>
        </View>
      </ScrollView>
    </AppChrome>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#f5f5f5' },
  content: { padding: 24, paddingTop: 64, paddingBottom: 32, gap: 18 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: '#ffffff' },
  centerTitle: { color: '#171717', fontSize: 24, fontWeight: '900', textAlign: 'center' },
  centerBody: { color: '#737373', textAlign: 'center', lineHeight: 21, marginTop: 8 },
  primaryLink: { marginTop: 16, backgroundColor: '#171717', color: '#ffffff', paddingHorizontal: 22, paddingVertical: 14, borderRadius: 16, overflow: 'hidden', fontWeight: '900' },
  headerCard: { backgroundColor: '#ffffff', borderRadius: 30, padding: 24, alignItems: 'center', gap: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)', elevation: 2 },
  avatar: { width: 76, height: 76, borderRadius: 38, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center', marginBottom: 6 },
  avatarText: { color: '#ffffff', fontWeight: '900', fontSize: 24 },
  title: { color: '#171717', fontSize: 28, fontWeight: '900', letterSpacing: -0.8, textAlign: 'center' },
  email: { color: '#737373', fontWeight: '700' },
  panel: { backgroundColor: '#ffffff', borderRadius: 24, padding: 18, gap: 12 },
  sectionTitle: { color: '#171717', fontSize: 20, fontWeight: '900' },
  muted: { color: '#737373', fontWeight: '700' },
  workspaceRow: { paddingVertical: 10, borderTopWidth: 1, borderTopColor: '#f5f5f5' },
  workspaceName: { color: '#171717', fontWeight: '900', fontSize: 16 },
  workspaceRole: { color: '#737373', textTransform: 'uppercase', letterSpacing: 1, fontSize: 12, fontWeight: '800', marginTop: 4 },
  rowButton: { minHeight: 52, borderRadius: 16, backgroundColor: '#f5f5f5', paddingHorizontal: 14, flexDirection: 'row', alignItems: 'center', gap: 10 },
  rowButtonText: { color: '#171717', fontWeight: '900' },
  dangerText: { color: '#dc2626' },
});
