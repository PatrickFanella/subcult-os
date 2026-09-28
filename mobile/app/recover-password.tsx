import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import * as authAPI from '@/api/auth';
import { useAuth } from '@/auth/AuthContext';
import { newPasswordProblem, recoveryRequestNotice } from '@/modules/auth/recoveryModel';

// Opened from a recovery email link (https://<web host>/recover-password?token=…)
// or from sign in without a token, where it requests a new link instead.
export default function RecoverPasswordScreen() {
  const params = useLocalSearchParams<{ token?: string }>();
  const token = typeof params.token === 'string' ? params.token : '';
  const { completeRecovery } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [working, setWorking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  async function requestLink() {
    const trimmedEmail = email.trim();
    setError(null);
    setNotice(null);
    if (!trimmedEmail.includes('@')) {
      setError('Enter a valid email address.');
      return;
    }
    setWorking(true);
    try {
      await authAPI.requestRecovery(trimmedEmail);
      setNotice(recoveryRequestNotice(trimmedEmail));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to request a recovery link');
    } finally {
      setWorking(false);
    }
  }

  async function resetPassword() {
    setError(null);
    const problem = newPasswordProblem(password, confirmation);
    if (problem) {
      setError(problem);
      return;
    }
    setWorking(true);
    try {
      await completeRecovery(token, password);
      // Replace the route so the one-use token does not stay in history.
      router.replace({ pathname: '/login', params: { notice: 'recovered' } } as never);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to reset password');
    } finally {
      setWorking(false);
    }
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
      <Text style={styles.kicker}>Account recovery</Text>
      <Text style={styles.title}>{token ? 'Choose a new password' : 'Reset your password'}</Text>
      <Text style={styles.body}>
        {token
          ? 'Saving a new password signs this account out everywhere, including this device.'
          : 'Enter the email address for your account and we will send a recovery link.'}
      </Text>

      <View style={styles.card}>
        {token ? (
          <>
            <TextInput
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoComplete="new-password"
              textContentType="newPassword"
              placeholder="New password"
              placeholderTextColor="#a3a3a3"
              style={styles.input}
            />
            <TextInput
              value={confirmation}
              onChangeText={setConfirmation}
              secureTextEntry
              autoComplete="new-password"
              textContentType="newPassword"
              placeholder="Confirm new password"
              placeholderTextColor="#a3a3a3"
              style={styles.input}
            />
          </>
        ) : (
          <TextInput
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="email-address"
            autoComplete="email"
            textContentType="emailAddress"
            placeholder="Email address"
            placeholderTextColor="#a3a3a3"
            style={styles.input}
          />
        )}
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}
        <Pressable disabled={working} onPress={token ? resetPassword : requestLink} style={[styles.button, working && styles.buttonDisabled]}>
          <Text style={styles.buttonText}>{working ? 'Working…' : token ? 'Save new password' : 'Send recovery link'}</Text>
        </Pressable>
      </View>

      <Pressable onPress={() => router.replace('/login')}>
        <Text style={styles.link}>Return to sign in</Text>
      </Pressable>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  content: { flexGrow: 1, justifyContent: 'center', padding: 24, gap: 14 },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { color: '#171717', fontSize: 36, fontWeight: '800', letterSpacing: -1.2 },
  body: { color: '#737373', fontSize: 16, lineHeight: 23, marginBottom: 12 },
  card: { marginTop: 8, gap: 12, backgroundColor: '#f5f5f5', borderRadius: 28, padding: 20 },
  input: { minHeight: 54, borderRadius: 16, backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 14, fontSize: 15, fontWeight: '600' },
  error: { color: '#dc2626', fontWeight: '700', lineHeight: 20 },
  notice: { color: '#166534', fontWeight: '700', lineHeight: 20 },
  button: { minHeight: 56, borderRadius: 18, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center', marginTop: 4 },
  buttonDisabled: { opacity: 0.55 },
  buttonText: { color: '#ffffff', fontWeight: '800', fontSize: 16 },
  link: { color: '#2563eb', fontWeight: '800', fontSize: 15 },
});
