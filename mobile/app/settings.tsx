import { Link } from 'expo-router';
import { ChevronLeft, RefreshCcw, Settings } from 'lucide-react-native';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { getMe } from '@/api/auth';
import { listWorkspaceEvents } from '@/api/staff';
import { apiConfig } from '@/config/api';
import { useAuth } from '@/auth/AuthContext';
import { safeBack } from '@/navigation/safeBack';

export default function SettingsScreen() {
  const { user, refresh } = useAuth();
  const [debugOutput, setDebugOutput] = useState<string>('');
  const [debugging, setDebugging] = useState(false);

  async function runAuthDebug() {
    setDebugging(true);
    setDebugOutput('Checking…');
    try {
      const current = await getMe();
      const lines = [`/api/me: ${current.email}`, `workspaces: ${current.workspaces.length}`];
      for (const workspace of current.workspaces) {
        try {
          const events = await listWorkspaceEvents(workspace.id);
          lines.push(`${workspace.name}: ${events.length} event(s)`);
        } catch (caught) {
          lines.push(`${workspace.name}: ${caught instanceof Error ? caught.message : 'failed'}`);
        }
      }
      setDebugOutput(lines.join('\n'));
    } catch (caught) {
      setDebugOutput(`/api/me failed: ${caught instanceof Error ? caught.message : 'unknown error'}`);
    } finally {
      setDebugging(false);
    }
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/profile')} style={styles.backButton}><ChevronLeft size={24} color="#171717" /></Pressable>
        <View><Text style={styles.kicker}>App</Text><Text style={styles.title}>Settings</Text></View>
      </View>
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.panel}>
          <View style={styles.panelHeader}><Settings size={22} color="#171717" /><Text style={styles.panelTitle}>Connection</Text></View>
          <Text style={styles.label}>API URL</Text>
          <Text style={styles.mono}>{apiConfig.baseUrl}</Text>
          <Text style={styles.body}>For phone testing, this should be your computer LAN URL, not localhost.</Text>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Session</Text>
          <Text style={styles.body}>{user ? `Signed in as ${user.email}` : 'Not signed in.'}</Text>
          <Pressable onPress={() => void refresh()} style={styles.actionButton}><RefreshCcw size={18} color="#ffffff" /><Text style={styles.actionButtonText}>Refresh session</Text></Pressable>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Auth debug</Text>
          <Text style={styles.body}>Checks `/api/me` and event access for each workspace using the same mobile session as Staff.</Text>
          <Pressable onPress={() => void runAuthDebug()} style={styles.actionButton}><RefreshCcw size={18} color="#ffffff" /><Text style={styles.actionButtonText}>{debugging ? 'Checking…' : 'Run auth check'}</Text></Pressable>
          {debugOutput ? <Text style={styles.debugBox}>{debugOutput}</Text> : null}
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Testing notes</Text>
          <Text style={styles.body}>This app is pinned to Expo SDK 54 for App Store / Play Store Expo Go compatibility during rehearsal.</Text>
          <Link href="/profile" style={styles.linkText}>Back to profile</Link>
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 24, paddingTop: 56, paddingBottom: 12 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '900', letterSpacing: -1, color: '#171717', marginTop: 6 },
  content: { padding: 24, paddingTop: 8, paddingBottom: 40, gap: 18 },
  panel: { backgroundColor: '#fafafa', borderRadius: 24, padding: 18, gap: 10 },
  panelHeader: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  panelTitle: { color: '#171717', fontSize: 20, fontWeight: '900' },
  label: { color: '#737373', fontSize: 12, textTransform: 'uppercase', letterSpacing: 1.1, fontWeight: '900' },
  mono: { color: '#171717', fontFamily: 'monospace', fontWeight: '800' },
  body: { color: '#525252', lineHeight: 21, fontWeight: '600' },
  actionButton: { alignSelf: 'flex-start', backgroundColor: '#171717', borderRadius: 16, paddingHorizontal: 16, paddingVertical: 12, flexDirection: 'row', alignItems: 'center', gap: 8 },
  actionButtonText: { color: '#ffffff', fontWeight: '900' },
  debugBox: { backgroundColor: '#ffffff', borderRadius: 14, padding: 12, color: '#171717', fontFamily: 'monospace', lineHeight: 20 },
  linkText: { color: '#2563eb', fontWeight: '900' },
});
