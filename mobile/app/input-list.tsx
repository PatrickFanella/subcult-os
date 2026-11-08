import { useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { api } from '@/api/client';
import type { EventInputListDTO } from '@/api/types';
import { safeBack } from '@/navigation/safeBack';
import { useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';

export default function InputListScreen() {
  const styles = useThemedStyles(createStyles);
  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventId = typeof params.eventId === 'string' ? params.eventId : '';
  const [list, setList] = useState<EventInputListDTO | null>(null);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    let active = true; setList(null); setError('');
    if (!eventId) { setError('Open the input list from an event’s run of show.'); return; }
    api<EventInputListDTO>(`/api/events/${eventId}/input-list`).then((data) => { if (active) setList(data); }).catch((err) => { if (active) setError(err instanceof Error ? err.message : 'Unable to load input list'); });
    return () => { active = false; };
  }, [eventId, revision]);
  return <View style={styles.screen}><ScrollView contentContainerStyle={styles.content}><Pressable accessibilityRole="button" style={styles.button} onPress={() => safeBack('/staff')}><Text style={styles.link}>Back to run of show</Text></Pressable><Text accessibilityRole="header" style={styles.title}>{list?.eventTitle || 'Input list'}</Text><Text style={styles.muted}>Private workspace tech sheets. Edit and print from the event’s web editor.</Text><Pressable accessibilityRole="button" onPress={() => setRevision((value) => value + 1)} style={styles.button}><Text style={styles.link}>Reload input list</Text></Pressable>{error ? <Text accessibilityRole="alert" style={styles.error}>{error}</Text> : !list ? <Text style={styles.text}>Loading…</Text> : <><Text style={styles.text}>{list.channels.length} channels. Shared keys combine one physical channel; check differences before repatching.</Text>{list.channels.map((channel) => <View style={styles.card} key={channel.number}><Text style={styles.heading}>Channel {channel.number}{channel.sharedKey ? ` · Shared: ${channel.sharedKey}` : ''}</Text>{channel.uses.map((use) => <View key={use.id} style={styles.use}><Text style={styles.text}>{use.actName}: {use.label}</Text><Text style={styles.muted}>{use.sourceType} · {use.micDi || 'Mic / DI unspecified'} · {use.stand || 'Stand unspecified'} · {use.phantomPower ? '48V' : 'Phantom off'}</Text>{use.notes ? <Text style={styles.text}>{use.notes}</Text> : null}</View>)}</View>)}{list.acts.map((act) => <View key={act.actId} style={styles.card}><Text style={styles.heading}>{act.actName}</Text><Text style={styles.text}>Monitors: {act.monitorNotes || '—'}</Text><Text style={styles.text}>Brings: {act.backlineBrought || '—'}</Text><Text style={styles.text}>Needs: {act.backlineNeeded || '—'}</Text>{act.stagePlotUrl ? <Text style={styles.muted}>Private stage plot available in the authenticated web input list.</Text> : null}</View>)}{!list.acts.length ? <Text style={styles.text}>No acts on this bill yet.</Text> : null}</>}</ScrollView></View>;
}

function createStyles(tokens: Tokens) {
  return StyleSheet.create({
    screen: { flex: 1, backgroundColor: tokens.color.surface.canvas },
    content: { padding: 20, paddingTop: 60, paddingBottom: 48, gap: 16 },
    title: { fontSize: 26, fontWeight: '700', color: tokens.color.text.primary },
    heading: { fontSize: 17, fontWeight: '700', color: tokens.color.text.primary },
    text: { fontSize: 15, color: tokens.color.text.primary },
    muted: { fontSize: 14, color: tokens.color.text.muted },
    link: { fontSize: 16, color: tokens.color.text.primary, textDecorationLine: 'underline' },
    button: { minHeight: 44, justifyContent: 'center' },
    card: { padding: 16, gap: 10, borderWidth: 1, borderColor: tokens.color.border.subtle, backgroundColor: tokens.color.surface.panel },
    use: { gap: 6 },
    error: { color: tokens.color.status.danger },
  });
}
