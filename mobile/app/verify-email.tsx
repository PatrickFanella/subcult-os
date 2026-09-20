import { router, useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { useAuth } from '@/auth/AuthContext';

export default function VerifyEmailScreen() {
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
        <Pressable onPress={() => router.replace('/login')} style={styles.button}>
          <Text style={styles.buttonText}>Return to sign in</Text>
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', backgroundColor: '#ffffff', padding: 24, gap: 14 },
  kicker: { color: '#2563eb', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { color: '#171717', fontSize: 34, fontWeight: '800', letterSpacing: -1 },
  body: { color: '#525252', fontSize: 16, lineHeight: 23 },
  button: { alignSelf: 'flex-start', borderRadius: 16, backgroundColor: '#171717', paddingHorizontal: 18, paddingVertical: 14 },
  buttonText: { color: '#ffffff', fontWeight: '800' },
});
