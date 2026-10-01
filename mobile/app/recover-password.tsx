import { useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import * as authAPI from '@/api/auth';
import { useAuth } from '@/auth/AuthContext';
import { newPasswordProblem, recoveryRequestNotice } from '@/modules/auth/recoveryModel';
import { Field } from '@/ui/Field';
import { PrimaryButton } from '@/ui/PrimaryButton';

// Opened from a recovery email link (https://<web host>/recover-password?token=…)
// or from sign in without a token, where it requests a new link instead.
export default function RecoverPasswordScreen() {

  const styles = useThemedStyles(createStyles);

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
          : 'Enter your account email to request a recovery link.'}
      </Text>

      <View style={styles.card}>
        {token ? (
          <>
            <Field
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoComplete="new-password"
              textContentType="newPassword"
              placeholder="New password"
            />
            <Field
              value={confirmation}
              onChangeText={setConfirmation}
              secureTextEntry
              autoComplete="new-password"
              textContentType="newPassword"
              placeholder="Confirm new password"
            />
          </>
        ) : (
          <Field
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="email-address"
            autoComplete="email"
            textContentType="emailAddress"
            placeholder="Email address"
          />
        )}
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}
        <PrimaryButton busy={working} onPress={token ? resetPassword : requestLink} style={styles.button} label={working ? 'Working…' : token ? 'Save new password' : 'Request recovery link'} />
      </View>

      <Pressable onPress={() => router.replace('/login')}>
        <Text style={styles.link}>Return to sign in</Text>
      </Pressable>
    </ScrollView>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.panel },
  content: { flexGrow: 1, justifyContent: 'center', padding: 24, gap: 14 },
  kicker: { alignSelf: 'flex-start', color: tokens.color.text.muted, backgroundColor: tokens.color.surface.inset, paddingHorizontal: 8, paddingVertical: 4, borderRadius: tokens.radius.pill, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '700' },
  title: { color: tokens.color.text.primary, fontSize: 36, fontWeight: '700', letterSpacing: -1.2 },
  body: { color: tokens.color.text.muted, fontSize: 16, lineHeight: 23, marginBottom: 12 },
  card: { marginTop: 8, gap: 12, backgroundColor: tokens.color.surface.inset, borderRadius: tokens.radius.panel, padding: 20 },
  error: { color: tokens.color.status.danger, fontWeight: '700', lineHeight: 20 },
  notice: { color: tokens.color.status.success, fontWeight: '700', lineHeight: 20 },
  button: { marginTop: 4 },
  link: { color: tokens.color.text.primary, fontWeight: '700', fontSize: tokens.type['body'] },
});
