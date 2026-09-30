import { tokens } from '@/theme/tokens';
import { Link } from 'expo-router';
import { ChevronLeft, RefreshCcw, Settings } from 'lucide-react-native';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { getMe, getMobileAuthDebug } from '@/api/auth';
import { getSessionDebugState } from '@/api/client';
import { listWorkspaceEvents } from '@/api/staff';
import { apiConfig } from '@/config/api';
import { useAuth } from '@/auth/AuthContext';
import { loadStoredSessionCookie } from '@/auth/sessionCookieStore';
import { safeBack } from '@/navigation/safeBack';

export default function SettingsScreen() {
  const { user, refresh } = useAuth();
  const [debugOutput, setDebugOutput] = useState<string>('');
  const [debugging, setDebugging] = useState(false);

  async function runAuthDebug() {
    setDebugging(true);
    setDebugOutput('Checking…');
    try {
      const serverDebug = await getMobileAuthDebug();
      const current = await getMe();
      const storedSession = await loadStoredSessionCookie();
      const sessionState = await getSessionDebugState();
      const lines = [
        `stored session: ${storedSession ? 'yes' : 'no'}`,
        `memory access token: ${sessionState.hasAccessToken ? 'yes' : 'no'}`,
        `memory refresh token: ${sessionState.hasRefreshToken ? 'yes' : 'no'}`,
        `server access token: ${serverDebug.hasAccessToken ? 'yes' : 'no'}`,
        `server refresh token: ${serverDebug.hasRefreshToken ? 'yes' : 'no'}`,
        `server token recognized: ${serverDebug.recognized ? 'yes' : 'no'}`,
        `/api/me: ${current.email}`,
        `workspaces: ${current.workspaces.length}`,
      ];
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
      const storedSession = await loadStoredSessionCookie();
      const sessionState = await getSessionDebugState();
      let serverLines: string[] = [];
      try {
        const serverDebug = await getMobileAuthDebug();
        serverLines = [
          `server access token: ${serverDebug.hasAccessToken ? 'yes' : 'no'}`,
          `server refresh token: ${serverDebug.hasRefreshToken ? 'yes' : 'no'}`,
          `server token recognized: ${serverDebug.recognized ? 'yes' : 'no'}`,
        ];
      } catch (debugError) {
        serverLines = [`server debug failed: ${debugError instanceof Error ? debugError.message : 'unknown error'}`];
      }
      setDebugOutput([
        `stored session: ${storedSession ? 'yes' : 'no'}`,
        `memory access token: ${sessionState.hasAccessToken ? 'yes' : 'no'}`,
        `memory refresh token: ${sessionState.hasRefreshToken ? 'yes' : 'no'}`,
        ...serverLines,
        `/api/me failed: ${caught instanceof Error ? caught.message : 'unknown error'}`,
      ].join('\n'));
    } finally {
      setDebugging(false);
    }
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/profile')} style={styles.backButton}><ChevronLeft size={24} color={tokens.color.text.primary} /></Pressable>
        <View><Text style={styles.kicker}>App</Text><Text style={styles.title}>Settings</Text></View>
      </View>
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.panel}>
          <View style={styles.panelHeader}><Settings size={22} color={tokens.color.text.primary} /><Text style={styles.panelTitle}>Connection</Text></View>
          <Text style={styles.label}>API URL</Text>
          <Text style={styles.mono}>{apiConfig.baseUrl}</Text>
          <Text style={styles.body}>For phone testing, this should be your computer LAN URL, not localhost.</Text>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Session</Text>
          <Text style={styles.body}>{user ? `Signed in as ${user.email}` : 'Not signed in.'}</Text>
          <Pressable onPress={() => void refresh()} style={styles.actionButton}><RefreshCcw size={18} color={tokens.color.surface.panel} /><Text style={styles.actionButtonText}>Refresh session</Text></Pressable>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Auth debug</Text>
          <Text style={styles.body}>Checks `/api/me` and event access for each workspace using the same mobile session as Staff.</Text>
          <Pressable onPress={() => void runAuthDebug()} style={styles.actionButton}><RefreshCcw size={18} color={tokens.color.surface.panel} /><Text style={styles.actionButtonText}>{debugging ? 'Checking…' : 'Run auth check'}</Text></Pressable>
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
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 24, paddingTop: 56, paddingBottom: 12 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: tokens.color.surface.inset, alignItems: 'center', justifyContent: 'center' },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '800' },
  title: { fontSize: tokens.type['title'], fontWeight: '900', letterSpacing: -1, color: tokens.color.text.primary, marginTop: 6 },
  content: { padding: 24, paddingTop: 8, paddingBottom: 40, gap: 18 },
  panel: { backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.card, padding: 18, gap: 10 },
  panelHeader: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  panelTitle: { color: tokens.color.text.primary, fontSize: 20, fontWeight: '900' },
  label: { color: tokens.color.text.muted, fontSize: tokens.type['label'], textTransform: 'uppercase', letterSpacing: 1.1, fontWeight: '900' },
  mono: { color: tokens.color.text.primary, fontFamily: 'monospace', fontWeight: '800' },
  body: { color: tokens.color.text.secondary, lineHeight: 21, fontWeight: '600' },
  actionButton: { alignSelf: 'flex-start', backgroundColor: tokens.color.action.primary, borderRadius: tokens.radius.control, paddingHorizontal: 16, paddingVertical: 12, flexDirection: 'row', alignItems: 'center', gap: 8 },
  actionButtonText: { color: tokens.color.text.inverse, fontWeight: '900' },
  debugBox: { backgroundColor: tokens.color.surface.panel, borderRadius: 14, padding: 12, color: tokens.color.text.primary, fontFamily: 'monospace', lineHeight: 20 },
  linkText: { color: tokens.color.text.primary, fontWeight: '900' },
});
