import { tokens } from '@/theme/tokens';
import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { useAuth } from '@/auth/AuthContext';

type Mode = 'login' | 'signup';

export default function LoginScreen() {
  const params = useLocalSearchParams<{ next?: string; mode?: Mode; notice?: string }>();
  const next = typeof params.next === 'string' && params.next.startsWith('/') ? params.next : '/staff';
  const [mode, setMode] = useState<Mode>(params.mode === 'signup' ? 'signup' : 'login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(
    params.notice === 'recovered' ? 'Password updated. Every session was signed out; sign in with the new password.' : null,
  );
  const { signIn, signUp, loading } = useAuth();

  async function submit() {
    const trimmedEmail = email.trim();
    const trimmedDisplayName = displayName.trim();
    setError(null);
    setNotice(null);

    if (!trimmedEmail || !trimmedEmail.includes('@')) {
      setError('Enter a valid email address.');
      return;
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }

    try {
      if (mode === 'signup') {
        const result = await signUp(trimmedEmail, password, trimmedDisplayName || undefined);
        setNotice(`Check ${result.email} for a verification link before signing in.`);
        return;
      } else {
        await signIn(trimmedEmail, password);
      }
      router.replace(next as never);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to continue');
    }
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
      <Text style={styles.kicker}>Operator access</Text>
      <Text style={styles.title}>{mode === 'signup' ? 'Create account' : 'Sign in'}</Text>
      <Text style={styles.body}>Sign in to unlock staff mode, door lookup, and live check-in tools.</Text>

      <View style={styles.tabs}>
        <Pressable onPress={() => setMode('login')} style={[styles.tab, mode === 'login' && styles.tabActive]}>
          <Text style={[styles.tabText, mode === 'login' && styles.tabTextActive]}>Sign in</Text>
        </Pressable>
        <Pressable onPress={() => setMode('signup')} style={[styles.tab, mode === 'signup' && styles.tabActive]}>
          <Text style={[styles.tabText, mode === 'signup' && styles.tabTextActive]}>Create</Text>
        </Pressable>
      </View>

      <View style={styles.card}>
        {mode === 'signup' ? (
          <TextInput
            value={displayName}
            onChangeText={setDisplayName}
            placeholder="Display name"
            placeholderTextColor="#a3a3a3"
            style={styles.input}
          />
        ) : null}
        <TextInput
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="email-address"
          placeholder="Email address"
          placeholderTextColor="#a3a3a3"
          style={styles.input}
        />
        <TextInput
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          placeholder="Password"
          placeholderTextColor="#a3a3a3"
          style={styles.input}
        />
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}
        <Pressable disabled={loading} onPress={submit} style={[styles.button, loading && styles.buttonDisabled]}>
          <Text style={styles.buttonText}>{loading ? 'Working…' : mode === 'signup' ? 'Create account' : 'Sign in'}</Text>
        </Pressable>
        {mode === 'login' ? (
          <Pressable onPress={() => router.push('/recover-password')}>
            <Text style={styles.link}>Forgot password?</Text>
          </Pressable>
        ) : null}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel },
  content: { flexGrow: 1, justifyContent: 'center', padding: 24, gap: 14 },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '800' },
  title: { color: tokens.color.text.primary, fontSize: 36, fontWeight: '800', letterSpacing: -1.2 },
  body: { color: tokens.color.text.muted, fontSize: 16, lineHeight: 23, marginBottom: 12 },
  tabs: { flexDirection: 'row', gap: 8, backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.pill, padding: 5, alignSelf: 'flex-start' },
  tab: { paddingHorizontal: 16, paddingVertical: 9, borderRadius: tokens.radius.pill },
  tabActive: { backgroundColor: tokens.color.action.primary },
  tabText: { color: tokens.color.text.muted, fontWeight: '800' },
  tabTextActive: { color: tokens.color.text.inverse },
  card: { marginTop: 8, gap: 12, backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.panel, padding: 20 },
  input: { minHeight: 54, borderRadius: tokens.radius.control, backgroundColor: tokens.color.surface.panel, color: tokens.color.text.primary, paddingHorizontal: 14, fontSize: tokens.type['body'], fontWeight: '600' },
  error: { color: tokens.color.status.danger, fontWeight: '700', lineHeight: 20 },
  notice: { color: tokens.color.status.success, fontWeight: '700', lineHeight: 20 },
  button: { minHeight: tokens.size.field, borderRadius: 18, backgroundColor: tokens.color.action.primary, alignItems: 'center', justifyContent: 'center', marginTop: 4 },
  buttonDisabled: { opacity: 0.55 },
  buttonText: { color: tokens.color.text.inverse, fontWeight: '800', fontSize: 16 },
  link: { color: tokens.color.text.primary, fontWeight: '800', fontSize: tokens.type['body'], textAlign: 'center' },
});
