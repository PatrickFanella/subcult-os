import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';

import '@/global.css';
import { AuthProvider } from '@/auth/AuthContext';

export default function RootLayout() {
  return (
    <AuthProvider>
      <StatusBar style="auto" />
      <Stack
        screenOptions={{
          headerStyle: { backgroundColor: '#ffffff' },
          headerTintColor: '#171717',
          headerTitleStyle: { fontWeight: '800' },
          contentStyle: { backgroundColor: '#ffffff' },
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
        <Stack.Screen name="door" options={{ headerShown: false }} />
        <Stack.Screen name="ticket" options={{ headerShown: false }} />
        <Stack.Screen name="tickets" options={{ headerShown: false }} />
        <Stack.Screen name="login" options={{ headerShown: false }} />
      </Stack>
    </AuthProvider>
  );
}
