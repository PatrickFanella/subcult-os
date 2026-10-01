import { useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { router, useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';

import { useAuth } from '@/auth/AuthContext';
import { PrimaryButton } from '@/ui/PrimaryButton';

export default function VerifyEmailScreen() {

  const styles = useThemedStyles(createStyles);

  const params = useLocalSearchParams<{ token?: string }>();
  const token = typeof params.token === 'string' ? params.token : '';
  const { verifyEmail } = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      setError('This verification link is missing its token.');
      return;
    }
    void verifyEmail(token)
      .then(() => router.replace('/staff'))
      .catch((caught) => setError(caught instanceof Error ? caught.message : 'Unable to verify email'));
  }, [token]);

  return (
    <View style={styles.screen}>
      <Text style={styles.kicker}>Account verification</Text>
      <Text style={styles.title}>{error ? 'Link not accepted' : 'Verifying your email…'}</Text>
      <Text style={styles.body}>{error ?? 'This should only take a moment.'}</Text>
      {error ? (
        <PrimaryButton onPress={() => router.replace('/login')} style={styles.button} label="Return to sign in" />
      ) : null}
    </View>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', backgroundColor: tokens.color.surface.panel, padding: 24, gap: 14 },
  kicker: { color: tokens.color.text.muted, textTransform: 'uppercase', letterSpacing: 1.5, fontSize: tokens.type['label'], fontWeight: '700' },
  title: { color: tokens.color.text.primary, fontSize: 34, fontWeight: '700', letterSpacing: -1 },
  body: { color: tokens.color.text.secondary, fontSize: 16, lineHeight: 23 },
  button: { alignSelf: 'flex-start' },
});
