import { useThemeTokens } from '@/theme/ThemeProvider';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import '@/global.css';
import { AuthProvider } from '@/auth/AuthContext';
import { ThemeProvider } from '@/theme/ThemeProvider';

export default function RootLayout() {
  const tokens = useThemeTokens();

  return (
    <ThemeProvider><SafeAreaProvider>
      <AuthProvider>
        <StatusBar style="auto" />
        <Stack
          screenOptions={{
            headerStyle: { backgroundColor: tokens.color.surface.panel },
            headerTintColor: tokens.color.text.primary,
            headerTitleStyle: { fontWeight: '800' },
            contentStyle: { backgroundColor: tokens.color.surface.panel },
          }}
        >
          <Stack.Screen name="index" options={{ headerShown: false }} />
          <Stack.Screen name="event-detail" options={{ headerShown: false }} />
          <Stack.Screen name="staff" options={{ headerShown: false }} />
          <Stack.Screen name="run-of-show" options={{ headerShown: false }} />
          <Stack.Screen name="roles" options={{ headerShown: false }} />
          <Stack.Screen name="scanner" options={{ headerShown: false }} />
          <Stack.Screen name="event-dashboard" options={{ headerShown: false }} />
          <Stack.Screen name="event-edit" options={{ headerShown: false }} />
          <Stack.Screen name="readiness" options={{ headerShown: false }} />
          <Stack.Screen name="door" options={{ headerShown: false }} />
          <Stack.Screen name="ticket" options={{ headerShown: false }} />
          <Stack.Screen name="tickets" options={{ headerShown: false }} />
          <Stack.Screen name="profile" options={{ headerShown: false }} />
          <Stack.Screen name="settings" options={{ headerShown: false }} />
          <Stack.Screen name="login" options={{ headerShown: false }} />
          <Stack.Screen name="verify-email" options={{ headerShown: false }} />
          <Stack.Screen name="recover-password" options={{ headerShown: false }} />
          <Stack.Screen name="workspace-create" options={{ headerShown: false }} />
        </Stack>
      </AuthProvider>
    </SafeAreaProvider></ThemeProvider>
  );
}
