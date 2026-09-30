import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { CameraView, useCameraPermissions, type BarcodeScanningResult } from 'expo-camera';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronDown, ShieldCheck, Zap } from 'lucide-react-native';
import { useState } from 'react';
import { ImageBackground, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';

import { checkInTicket } from '@/api/door';

export default function ScannerScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

  const params = useLocalSearchParams<{ eventId?: string }>();
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const [code, setCode] = useState('');
  const [scanState, setScanState] = useState<'idle' | 'success' | 'error'>('idle');
  const [message, setMessage] = useState('Point the camera at a QR code or enter a code manually.');
  const [loading, setLoading] = useState(false);
  const [permission, requestPermission] = useCameraPermissions();

  async function checkIn(nextCode = code) {
    const trimmed = nextCode.trim();
    if (!eventID) {
      setScanState('error');
      setMessage('Missing event ID. Open Scanner from Staff mode.');
      return;
    }
    if (!trimmed) {
      setScanState('error');
      setMessage('Enter a ticket code first.');
      return;
    }

    setLoading(true);
    setScanState('idle');
    setCode(trimmed);
    try {
      const ticket = await checkInTicket(eventID, trimmed);
      setScanState('success');
      setMessage(`${ticket.displayName ?? 'Guest'} • ${ticket.code}`);
    } catch (caught) {
      setScanState('error');
      setMessage(caught instanceof Error ? caught.message : 'Unable to check in ticket');
    } finally {
      setLoading(false);
    }
  }

  function handleBarcodeScanned(result: BarcodeScanningResult) {
    if (loading || scanState === 'success') return;
    void checkIn(result.data);
  }

  return (
    <View style={styles.screen}>
      <ImageBackground
        source={{ uri: 'https://images.unsplash.com/photo-1571204829887-3b8d69e4094d?auto=format&fit=crop&w=1080&q=80' }}
        style={styles.background}
        imageStyle={styles.backgroundImage}
      />
      <View style={styles.topBar}>
        <Pressable onPress={() => router.replace('/staff')} style={styles.roundButton}>
          <ChevronDown size={24} color={tokens.color.text.onImmersive} />
        </Pressable>
        <Text style={styles.scannerPill}>SCANNER</Text>
        <Pressable style={styles.roundButton}>
          <Zap size={20} color={tokens.color.text.onImmersive} />
        </Pressable>
      </View>

      <View style={styles.center}>
        <Pressable onPress={() => void checkIn()} style={styles.viewfinder}>
          {permission?.granted ? (
            <CameraView
              style={styles.camera}
              facing="back"
              barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
              onBarcodeScanned={handleBarcodeScanned}
            />
          ) : (
            <View style={styles.permissionPanel}>
              <Text style={styles.permissionTitle}>Camera access</Text>
              <Text style={styles.permissionText}>Allow camera access to scan ticket QR codes.</Text>
              <Pressable onPress={requestPermission} style={styles.permissionButton}>
                <Text style={styles.permissionButtonText}>Enable camera</Text>
              </Pressable>
            </View>
          )}
          <View style={[styles.corner, styles.topLeft]} />
          <View style={[styles.corner, styles.topRight]} />
          <View style={[styles.corner, styles.bottomLeft]} />
          <View style={[styles.corner, styles.bottomRight]} />
          {scanState === 'idle' ? <View style={styles.scanLine} /> : null}
          {scanState === 'success' ? (
            <View style={styles.successOverlay}>
              <View style={styles.successIcon}><ShieldCheck size={40} color={tokens.color.status.success} /></View>
              <Text style={styles.approved}>Approved</Text>
              <Text style={styles.approvedSub}>{message}</Text>
            </View>
          ) : null}
          {scanState === 'error' ? (
            <View style={styles.errorOverlay}>
              <Text style={styles.approved}>Denied</Text>
              <Text style={styles.approvedSub}>{message}</Text>
            </View>
          ) : null}
        </Pressable>
      </View>

      <View style={styles.footer}>
        <TextInput
          value={code}
          onChangeText={setCode}
          placeholder="Ticket code"
          placeholderTextColor="rgba(255,255,255,0.55)"
          autoCapitalize="characters"
          autoCorrect={false}
          style={styles.input}
          onSubmitEditing={() => void checkIn()}
        />
        <Text style={styles.footerText}>{loading ? 'Checking in…' : message}</Text>
      </View>
    </View>
  );
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: tokens.color.surface.immersive, position: 'relative' },
  background: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, opacity: 0.4 },
  backgroundImage: { resizeMode: 'cover' },
  topBar: { position: 'absolute', top: 48, left: 0, right: 0, paddingHorizontal: 16, zIndex: 20, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  roundButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: 'rgba(255,255,255,0.20)', alignItems: 'center', justifyContent: 'center' },
  scannerPill: { color: tokens.color.text.onImmersive, backgroundColor: 'rgba(255,255,255,0.20)', paddingHorizontal: 16, paddingVertical: 7, borderRadius: tokens.radius.pill, overflow: 'hidden', fontSize: 14, fontWeight: '700', letterSpacing: 1.4 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 32, zIndex: 10 },
  viewfinder: { width: '100%', aspectRatio: 1, borderWidth: 2, borderColor: 'rgba(255,255,255,0.30)', borderRadius: 40, position: 'relative' },
  camera: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, borderRadius: 38, overflow: 'hidden' },
  permissionPanel: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, borderRadius: 38, backgroundColor: 'rgba(0,0,0,0.45)', alignItems: 'center', justifyContent: 'center', padding: 22 },
  permissionTitle: { color: tokens.color.text.onImmersive, fontSize: 22, fontWeight: '800', marginBottom: 8 },
  permissionText: { color: 'rgba(255,255,255,0.78)', textAlign: 'center', lineHeight: 20, marginBottom: 16 },
  permissionButton: { backgroundColor: tokens.color.surface.panel, borderRadius: tokens.radius.pill, paddingHorizontal: 18, paddingVertical: 10 },
  permissionButtonText: { color: tokens.color.text.primary, fontWeight: '800' },
  corner: { position: 'absolute', width: 48, height: 48, borderColor: tokens.color.surface.panel },
  topLeft: { top: -4, left: -4, borderTopWidth: 4, borderLeftWidth: 4, borderTopLeftRadius: 40 },
  topRight: { top: -4, right: -4, borderTopWidth: 4, borderRightWidth: 4, borderTopRightRadius: 40 },
  bottomLeft: { bottom: -4, left: -4, borderBottomWidth: 4, borderLeftWidth: 4, borderBottomLeftRadius: 40 },
  bottomRight: { bottom: -4, right: -4, borderBottomWidth: 4, borderRightWidth: 4, borderBottomRightRadius: 40 },
  scanLine: { position: 'absolute', left: '5%', right: '5%', top: '50%', height: 2, backgroundColor: tokens.color.status.success, borderRadius: tokens.radius.pill },
  successOverlay: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, borderRadius: 38, backgroundColor: 'rgba(34,197,94,0.90)', alignItems: 'center', justifyContent: 'center', padding: 22 },
  errorOverlay: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, borderRadius: 38, backgroundColor: 'rgba(239,68,68,0.90)', alignItems: 'center', justifyContent: 'center', padding: 22 },
  successIcon: { width: 80, height: 80, borderRadius: 40, backgroundColor: tokens.color.surface.panel, alignItems: 'center', justifyContent: 'center', marginBottom: 16 },
  approved: { color: tokens.color.text.onImmersive, fontSize: 24, fontWeight: '800', textAlign: 'center' },
  approvedSub: { color: 'rgba(255,255,255,0.82)', fontWeight: '600', textAlign: 'center' },
  footer: { gap: 12, padding: 32, paddingBottom: 48, zIndex: 10 },
  input: { minHeight: 54, borderRadius: 18, backgroundColor: 'rgba(255,255,255,0.18)', color: tokens.color.text.onImmersive, paddingHorizontal: 16, fontWeight: '800', letterSpacing: 1.5 },
  footerText: { textAlign: 'center', color: 'rgba(255,255,255,0.70)', fontSize: 14, fontWeight: '600' },
});
