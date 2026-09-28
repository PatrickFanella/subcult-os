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
  screen: { flex: 1, backgroundColor: '#ffffff' },
  content: { flexGrow: 1, justifyContent: 'center', padding: 24, gap: 14 },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { color: '#171717', fontSize: 36, fontWeight: '800', letterSpacing: -1.2 },
  body: { color: '#737373', fontSize: 16, lineHeight: 23, marginBottom: 12 },
  tabs: { flexDirection: 'row', gap: 8, backgroundColor: '#f5f5f5', borderRadius: 999, padding: 5, alignSelf: 'flex-start' },
  tab: { paddingHorizontal: 16, paddingVertical: 9, borderRadius: 999 },
  tabActive: { backgroundColor: '#171717' },
  tabText: { color: '#737373', fontWeight: '800' },
  tabTextActive: { color: '#ffffff' },
  card: { marginTop: 8, gap: 12, backgroundColor: '#f5f5f5', borderRadius: 28, padding: 20 },
  input: { minHeight: 54, borderRadius: 16, backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 14, fontSize: 15, fontWeight: '600' },
  error: { color: '#dc2626', fontWeight: '700', lineHeight: 20 },
  notice: { color: '#166534', fontWeight: '700', lineHeight: 20 },
  button: { minHeight: 56, borderRadius: 18, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center', marginTop: 4 },
  buttonDisabled: { opacity: 0.55 },
  buttonText: { color: '#ffffff', fontWeight: '800', fontSize: 16 },
  link: { color: '#2563eb', fontWeight: '800', fontSize: 15, textAlign: 'center' },
});
