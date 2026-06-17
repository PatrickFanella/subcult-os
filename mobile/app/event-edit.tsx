import { useLocalSearchParams, useRouter } from 'expo-router';
import * as ImagePicker from 'expo-image-picker';
import { ChevronLeft } from 'lucide-react-native';
import type { ComponentProps } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { Image, KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { createEvent, getEvent, publishEvent, updateEvent, uploadEventImage, type EventWritePayload } from '@/api/staff';
import type { EventDTO, PricingMode } from '@/api/types';
import { useAuth } from '@/auth/AuthContext';
import { safeBack } from '@/navigation/safeBack';
import { storeSelectedEventID } from '@/staff/selectionStore';

type FormState = {
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  imageUrl: string;
  ticketAllocation: string;
  pricingMode: PricingMode;
  ticketPriceDollars: string;
  ticketCurrency: string;
};

const emptyForm: FormState = {
  title: '',
  startsAt: defaultStartsAtInput(),
  publicDescription: '',
  locationDisplay: '',
  imageUrl: '',
  ticketAllocation: '50',
  pricingMode: 'free',
  ticketPriceDollars: '0.00',
  ticketCurrency: 'USD',
};

export default function EventEditScreen() {
  const router = useRouter();
  const params = useLocalSearchParams<{ workspaceId?: string; eventId?: string }>();
  const workspaceID = typeof params.workspaceId === 'string' ? params.workspaceId : '';
  const eventID = typeof params.eventId === 'string' ? params.eventId : '';
  const editing = Boolean(eventID);
  const { user, loading: authLoading } = useAuth();
  const [form, setForm] = useState<FormState>(emptyForm);
  const [event, setEvent] = useState<EventDTO | null>(null);
  const [loading, setLoading] = useState(editing);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [selectedImage, setSelectedImage] = useState<ImagePicker.ImagePickerAsset | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const targetWorkspaceID = workspaceID || event?.workspaceId || user?.workspaces[0]?.id || '';
  const workspace = user?.workspaces.find((candidate) => candidate.id === targetWorkspaceID) ?? user?.workspaces[0] ?? null;
  const canPublish = event?.status === 'draft';

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!eventID) return;
      setLoading(true);
      setError(null);
      try {
        const loaded = await getEvent(eventID);
        if (!cancelled) {
          setEvent(loaded);
          setForm(formFromEvent(loaded));
          setSelectedImage(null);
        }
      } catch (caught) {
        if (!cancelled) setError(caught instanceof Error ? caught.message : 'Unable to load event');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [eventID]);

  const preview = useMemo(() => buildPayload(form), [form]);

  function updateField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  async function save() {
    setSaving(true);
    setError(null);
    setNotice(null);
    try {
      const payload = requirePayload(form);
      let saved = editing ? await updateEvent(eventID, payload) : await createEvent(targetWorkspaceID, payload);
      if (selectedImage) {
        saved = await uploadEventImage(saved.id, selectedImage);
        setSelectedImage(null);
      }
      setEvent(saved);
      setForm(formFromEvent(saved));
      await storeSelectedEventID(saved.id);
      setNotice(editing ? 'Event updated.' : 'Draft event created.');
      if (!editing) router.replace({ pathname: '/event-edit', params: { eventId: saved.id, workspaceId: saved.workspaceId } });
      return saved;
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to save event');
      return null;
    } finally {
      setSaving(false);
    }
  }

  async function chooseImage() {
    setError(null);
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsEditing: true,
      aspect: [16, 9],
      quality: 0.9,
    });
    if (!result.canceled && result.assets[0]) {
      const asset = result.assets[0];
      setSelectedImage(asset);
      updateField('imageUrl', asset.uri);
      setNotice('Image selected. Save the event to upload it.');
    }
  }

  async function saveAndPublish() {
    const saved = await save();
    if (!saved) return;
    setPublishing(true);
    setError(null);
    try {
      const published = await publishEvent(saved.id);
      setEvent(published);
      setForm(formFromEvent(published));
      setNotice('Event published.');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to publish event');
    } finally {
      setPublishing(false);
    }
  }

  if (authLoading) return <CenteredState title="Checking session…" />;
  if (!user) return <CenteredState title="Sign in required" body="Organizer tools require a workspace account." />;
  if (!workspace) return <CenteredState title="No workspace found" body="Create or join a workspace before creating events." />;
  if (!editing && !targetWorkspaceID) return <CenteredState title="Missing workspace" body="Open event creation from Staff mode." />;

  return (
    <KeyboardAvoidingView style={styles.screen} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.header}>
        <Pressable onPress={() => safeBack('/staff')} style={styles.backButton}>
          <ChevronLeft size={24} color="#171717" />
        </Pressable>
        <View style={styles.headerCopy}>
          <Text style={styles.kicker}>{editing ? 'Edit event' : 'Create event'}</Text>
          <Text style={styles.title}>{editing ? 'Event setup' : 'New event'}</Text>
          <Text style={styles.subtitle}>{workspace.name}</Text>
        </View>
      </View>

      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        {loading ? <Text style={styles.message}>Loading event…</Text> : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}

        <View style={styles.panel}>
          <Text style={styles.sectionTitle}>Public basics</Text>
          <LabeledInput label="Title" value={form.title} onChangeText={(value) => updateField('title', value)} placeholder="Warehouse Frequencies" />
          <LabeledInput label="Starts at" value={form.startsAt} onChangeText={(value) => updateField('startsAt', value)} placeholder="2026-06-19 21:00" />
          <LabeledInput label="Location" value={form.locationDisplay} onChangeText={(value) => updateField('locationDisplay', value)} placeholder="Pilsen, Chicago" />
          <View style={styles.imagePickerBlock}>
            {form.imageUrl ? <Image source={{ uri: form.imageUrl }} style={styles.imagePreview} resizeMode="cover" /> : <View style={styles.imagePlaceholder}><Text style={styles.imagePlaceholderText}>No hero image</Text></View>}
            <Pressable onPress={() => void chooseImage()} style={styles.imageButton}>
              <Text style={styles.imageButtonText}>{selectedImage ? 'Choose different image' : 'Choose image'}</Text>
            </Pressable>
          </View>
          <LabeledInput label="Hero image URL" value={form.imageUrl} onChangeText={(value) => { setSelectedImage(null); updateField('imageUrl', value); }} placeholder="https://…" autoCapitalize="none" keyboardType="url" />
          <LabeledInput label="Description" value={form.publicDescription} onChangeText={(value) => updateField('publicDescription', value)} placeholder="What should attendees know?" multiline />
        </View>

        <View style={styles.panel}>
          <Text style={styles.sectionTitle}>Tickets</Text>
          <LabeledInput label="Capacity" value={form.ticketAllocation} onChangeText={(value) => updateField('ticketAllocation', value.replace(/[^0-9]/g, ''))} keyboardType="number-pad" />
          <View style={styles.segmentRow}>
            <SegmentButton label="Free" selected={form.pricingMode === 'free'} onPress={() => updateField('pricingMode', 'free')} />
            <SegmentButton label="Fixed price" selected={form.pricingMode === 'fixed'} onPress={() => updateField('pricingMode', 'fixed')} />
          </View>
          {form.pricingMode === 'fixed' ? (
            <LabeledInput label="Ticket price" value={form.ticketPriceDollars} onChangeText={(value) => updateField('ticketPriceDollars', value)} keyboardType="decimal-pad" placeholder="18.00" />
          ) : null}
          <Text style={styles.previewText}>{preview ? `${preview.ticketAllocation} tickets · ${preview.pricingMode === 'free' ? 'Free' : `$${(preview.ticketPriceCents / 100).toFixed(2)}`} · ${preview.ticketCurrency}` : 'Complete ticket fields to preview.'}</Text>
        </View>

        <View style={styles.actionBar}>
          <Pressable onPress={() => void save()} disabled={saving || publishing} style={[styles.primaryButton, (saving || publishing) && styles.disabledButton]}>
            <Text style={styles.primaryButtonText}>{saving ? 'Saving…' : 'Save draft'}</Text>
          </Pressable>
          <Pressable onPress={() => void saveAndPublish()} disabled={saving || publishing || (editing && !canPublish)} style={[styles.secondaryButton, (saving || publishing || (editing && !canPublish)) && styles.disabledButton]}>
            <Text style={styles.secondaryButtonText}>{publishing ? 'Publishing…' : canPublish || !editing ? 'Save & publish' : 'Published'}</Text>
          </Pressable>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

function LabeledInput({ label, multiline, style, ...props }: { label: string } & ComponentProps<typeof TextInput>) {
  return (
    <View style={styles.fieldBlock}>
      <Text style={styles.label}>{label}</Text>
      <TextInput
        {...props}
        multiline={multiline}
        placeholderTextColor="#a3a3a3"
        style={[styles.input, multiline && styles.textArea, style]}
      />
    </View>
  );
}

function SegmentButton({ label, selected, onPress }: { label: string; selected: boolean; onPress: () => void }) {
  return (
    <Pressable onPress={onPress} style={[styles.segmentButton, selected && styles.segmentButtonActive]}>
      <Text style={[styles.segmentButtonText, selected && styles.segmentButtonTextActive]}>{label}</Text>
    </Pressable>
  );
}

function CenteredState({ title, body }: { title: string; body?: string }) {
  return (
    <View style={styles.centered}>
      <Text style={styles.centeredTitle}>{title}</Text>
      {body ? <Text style={styles.centeredBody}>{body}</Text> : null}
    </View>
  );
}

function formFromEvent(event: EventDTO): FormState {
  return {
    title: event.title,
    startsAt: toLocalInput(event.startsAt),
    publicDescription: event.publicDescription,
    locationDisplay: event.locationDisplay,
    imageUrl: event.imageUrl ?? '',
    ticketAllocation: String(event.ticketAllocation),
    pricingMode: event.pricingMode,
    ticketPriceDollars: (event.ticketPriceCents / 100).toFixed(2),
    ticketCurrency: event.ticketCurrency || 'USD',
  };
}

function buildPayload(form: FormState): EventWritePayload | null {
  const startsAt = parseStartsAt(form.startsAt);
  const ticketAllocation = Number.parseInt(form.ticketAllocation, 10);
  if (!startsAt || Number.isNaN(ticketAllocation)) return null;
  const ticketPriceCents = form.pricingMode === 'free' ? 0 : Math.round(Number.parseFloat(form.ticketPriceDollars || '0') * 100);
  if (Number.isNaN(ticketPriceCents)) return null;
  return {
    title: form.title.trim(),
    startsAt,
    publicDescription: form.publicDescription.trim(),
    locationDisplay: form.locationDisplay.trim(),
    imageUrl: form.imageUrl.trim(),
    ticketAllocation,
    pricingMode: form.pricingMode,
    ticketPriceCents,
    ticketCurrency: form.ticketCurrency.trim().toUpperCase() || 'USD',
  };
}

function requirePayload(form: FormState): EventWritePayload {
  const payload = buildPayload(form);
  if (!payload) throw new Error('Check the date, capacity, and price fields.');
  if (!payload.title || !payload.publicDescription || !payload.locationDisplay) throw new Error('Title, description, and location are required.');
  if (payload.ticketAllocation <= 0) throw new Error('Capacity must be greater than zero to publish.');
  return payload;
}

function parseStartsAt(value: string) {
  const normalized = value.trim().replace(' ', 'T');
  const date = new Date(normalized);
  return Number.isNaN(date.getTime()) ? null : date.toISOString();
}

function toLocalInput(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16).replace('T', ' ');
}

function defaultStartsAtInput() {
  const date = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
  date.setMinutes(0, 0, 0);
  return toLocalInput(date.toISOString());
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#ffffff' },
  header: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 24, paddingTop: 56, paddingBottom: 12 },
  backButton: { width: 40, height: 40, borderRadius: 20, backgroundColor: '#f5f5f5', alignItems: 'center', justifyContent: 'center' },
  headerCopy: { flex: 1 },
  kicker: { alignSelf: 'flex-start', color: '#2563eb', backgroundColor: '#eff6ff', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 999, overflow: 'hidden', textTransform: 'uppercase', letterSpacing: 1.5, fontSize: 12, fontWeight: '800' },
  title: { fontSize: 32, fontWeight: '800', letterSpacing: -1, color: '#171717', marginTop: 6 },
  subtitle: { color: '#737373', fontWeight: '700', marginTop: 4 },
  content: { gap: 18, padding: 24, paddingTop: 8, paddingBottom: 40 },
  panel: { backgroundColor: '#fafafa', borderRadius: 28, padding: 20, gap: 14 },
  sectionTitle: { color: '#171717', fontSize: 20, fontWeight: '800' },
  imagePickerBlock: { gap: 10 },
  imagePreview: { width: '100%', height: 180, borderRadius: 20, backgroundColor: '#e5e5e5' },
  imagePlaceholder: { width: '100%', height: 180, borderRadius: 20, backgroundColor: '#eeeeee', alignItems: 'center', justifyContent: 'center' },
  imagePlaceholderText: { color: '#737373', fontWeight: '800' },
  imageButton: { alignSelf: 'flex-start', backgroundColor: '#171717', borderRadius: 14, paddingHorizontal: 16, paddingVertical: 12 },
  imageButtonText: { color: '#ffffff', fontWeight: '900' },
  fieldBlock: { gap: 8 },
  label: { color: '#737373', fontSize: 12, fontWeight: '800', textTransform: 'uppercase', letterSpacing: 1.1 },
  input: { minHeight: 52, borderRadius: 16, borderWidth: 1, borderColor: '#e5e5e5', backgroundColor: '#ffffff', color: '#171717', paddingHorizontal: 16, paddingVertical: 12, fontWeight: '700' },
  textArea: { minHeight: 132, textAlignVertical: 'top', lineHeight: 21 },
  segmentRow: { flexDirection: 'row', gap: 10 },
  segmentButton: { flex: 1, borderRadius: 16, backgroundColor: '#ffffff', borderWidth: 1, borderColor: '#e5e5e5', paddingVertical: 14, alignItems: 'center' },
  segmentButtonActive: { backgroundColor: '#171717', borderColor: '#171717' },
  segmentButtonText: { color: '#171717', fontWeight: '800' },
  segmentButtonTextActive: { color: '#ffffff' },
  previewText: { color: '#525252', fontWeight: '700', lineHeight: 20 },
  actionBar: { gap: 12 },
  primaryButton: { minHeight: 56, borderRadius: 18, backgroundColor: '#171717', alignItems: 'center', justifyContent: 'center' },
  primaryButtonText: { color: '#ffffff', fontWeight: '900', fontSize: 16 },
  secondaryButton: { minHeight: 56, borderRadius: 18, backgroundColor: '#eff6ff', alignItems: 'center', justifyContent: 'center' },
  secondaryButtonText: { color: '#2563eb', fontWeight: '900', fontSize: 16 },
  disabledButton: { opacity: 0.45 },
  message: { color: '#737373', fontWeight: '700' },
  error: { color: '#dc2626', fontWeight: '800', lineHeight: 20 },
  notice: { color: '#16a34a', fontWeight: '800', lineHeight: 20 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#ffffff', padding: 24 },
  centeredTitle: { color: '#171717', fontSize: 24, fontWeight: '900', textAlign: 'center' },
  centeredBody: { color: '#737373', textAlign: 'center', marginTop: 8, lineHeight: 21 },
});
