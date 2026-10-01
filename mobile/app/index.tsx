import { useThemeTokens, useThemedStyles } from '@/theme/ThemeProvider';
import type { Tokens } from '@/theme/tokens';
import { Link } from 'expo-router';
import { Calendar, MapPin } from 'lucide-react-native';
import { useEffect, useState } from 'react';
import { Image, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { listPublicEvents } from '@/api/events';
import type { PublicEventSummaryDTO } from '@/api/types';
import { eventArtwork } from '@/data/eventArtwork';
import { AppChrome } from '@/ui/AppChrome';
import {
	discoveryEmptyBody,
	discoveryEmptyTitle,
	discoveryErrorCopy,
	discoveryLoadingCopy,
	discoveryPricingLabel,
	discoverySearchPlaceholder,
	discoverySubtitle,
	discoveryViewEventLabel,
	formatDiscoveryDate,
	formatDiscoveryTime,
} from '@/modules/discovery/discoveryModel';

export default function DiscoveryFeedScreen() {
  const tokens = useThemeTokens();
  const styles = useThemedStyles(createStyles);

	const [events, setEvents] = useState<PublicEventSummaryDTO[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [searchQuery, setSearchQuery] = useState('');
	const [viewportHeight, setViewportHeight] = useState(0);

	useEffect(() => {
		let cancelled = false;

		async function load() {
			setLoading(true);
			setError(null);

			try {
				const loaded = await listPublicEvents(searchQuery);
				if (!cancelled) {
					setEvents(loaded);
				}
			} catch (caught) {
				if (!cancelled) {
					setError(caught instanceof Error ? caught.message : 'Unable to load events');
					setEvents([]);
				}
			} finally {
				if (!cancelled) {
					setLoading(false);
				}
			}
		}

		void load();

		return () => {
			cancelled = true;
		};
	}, [searchQuery]);

	const searchBar = (
		<View style={styles.searchWrap}>
			<TextInput
				value={searchQuery}
				onChangeText={setSearchQuery}
				placeholder={discoverySearchPlaceholder}
				placeholderTextColor={tokens.color.text.muted}
				style={styles.searchInput}
				returnKeyType="search"
			/>
		</View>
	);

	if (loading) {
		return (
			<AppChrome>
				{searchBar}
				<View style={styles.centerState}>
					<Text style={styles.centerTitle}>{discoveryLoadingCopy}</Text>
				</View>
			</AppChrome>
		);
	}

	if (error) {
		return (
			<AppChrome>
				{searchBar}
				<View style={styles.centerState}>
					<Text style={styles.centerTitle}>Could not load Events</Text>
					<Text style={styles.centerBody}>{discoveryErrorCopy(error)}</Text>
				</View>
			</AppChrome>
		);
	}

	if (events.length === 0) {
		return (
			<AppChrome>
				{searchBar}
				<View style={styles.centerState}>
					<Text style={styles.centerTitle}>{discoveryEmptyTitle(searchQuery)}</Text>
					<Text style={styles.centerBody}>{discoveryEmptyBody(searchQuery)}</Text>
				</View>
			</AppChrome>
		);
	}

	return (
		<AppChrome>
			{searchBar}
			<ScrollView
				pagingEnabled
				showsVerticalScrollIndicator={false}
				style={styles.scroll}
				onLayout={(event) => setViewportHeight(event.nativeEvent.layout.height)}
			>
				{events.map((event) => (
					<View key={event.id} style={[styles.slide, viewportHeight ? { height: viewportHeight } : styles.slideFallback]}>
						<Image source={{ uri: event.imageUrl || eventArtwork(event.publicSlug) }} style={styles.image} resizeMode="cover" />
						<View style={styles.gradient} />
						<View style={styles.copy}>
							<View style={styles.pillRow}>
								<Text style={styles.pill}>{event.workspaceName}</Text>
								<Text style={styles.pill}>{discoveryPricingLabel(event)}</Text>
							</View>
							<Text style={styles.title}>{event.title}</Text>
							<Text style={styles.subtitle}>{discoverySubtitle(event)}</Text>
							<View style={styles.metaRow}>
								<View style={styles.metaItem}>
									<Calendar size={16} color="rgba(255,255,255,0.72)" />
									<Text style={styles.metaText}>{formatDiscoveryDate(event.startsAt)} • {formatDiscoveryTime(event.startsAt)}</Text>
								</View>
								<View style={styles.metaItem}>
									<MapPin size={16} color="rgba(255,255,255,0.72)" />
									<Text style={styles.metaText}>{event.locationDisplay}</Text>
								</View>
							</View>
							<Link href={{ pathname: '/event-detail', params: { slug: event.publicSlug } }} style={styles.button}>
								{discoveryViewEventLabel}
							</Link>
						</View>
					</View>
				))}
			</ScrollView>
		</AppChrome>
	);
}

const createStyles = (tokens: Tokens) => StyleSheet.create({
	scroll: { flex: 1, backgroundColor: tokens.color.surface.immersive },
	searchWrap: {
		position: 'absolute',
		top: 16,
		left: 16,
		right: 16,
		zIndex: 10,
		borderRadius: 18,
		backgroundColor: tokens.color.surface.panel,
		padding: 6,
		shadowColor: tokens.color.surface.immersive,
		shadowOpacity: 0.14,
		shadowRadius: 18,
		shadowOffset: { width: 0, height: 8 },
	},
	searchInput: { borderRadius: 14, backgroundColor: tokens.color.surface.panel, color: tokens.color.text.primary, fontSize: 16, fontWeight: '700', paddingHorizontal: 14, paddingVertical: 12 },
	slide: { position: 'relative', width: '100%', backgroundColor: tokens.color.surface.immersive },
	slideFallback: { minHeight: 640 },
	image: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, width: '100%', height: '100%' },
	gradient: { position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: 'rgba(0,0,0,0.38)' },
	copy: { position: 'absolute', left: 0, right: 0, bottom: 0, padding: 24, paddingBottom: 48 },
	pillRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginBottom: 12 },
	pill: {
		paddingHorizontal: 12,
		paddingVertical: 6,
		backgroundColor: 'rgba(255,255,255,0.20)',
		borderRadius: tokens.radius.pill,
		color: tokens.color.text.onImmersive,
		fontSize: tokens.type['label'],
		fontWeight: '600',
		textTransform: 'uppercase',
		letterSpacing: 1.2,
		overflow: 'hidden',
	},
	title: { color: tokens.color.text.onImmersive, fontSize: tokens.type['display'], fontWeight: '700', letterSpacing: -1.2, lineHeight: 42, marginBottom: 4 },
	subtitle: { color: 'rgba(255,255,255,0.82)', fontSize: 18, fontWeight: '300', marginBottom: 16 },
	metaRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 16, marginBottom: 24 },
	metaItem: { flexDirection: 'row', alignItems: 'center', gap: 6 },
	metaText: { color: 'rgba(255,255,255,0.72)', fontSize: 14, fontWeight: '600' },
	button: {
		width: '100%',
		backgroundColor: tokens.color.surface.panel,
		color: tokens.color.text.primary,
		paddingVertical: 16,
		borderRadius: tokens.radius.control,
		textAlign: 'center',
		fontSize: 18,
		fontWeight: '700',
		overflow: 'hidden',
	},
	centerState: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 28, backgroundColor: tokens.color.surface.panel },
	centerTitle: { color: tokens.color.text.primary, fontSize: 24, fontWeight: '700', textAlign: 'center', marginBottom: 8 },
	centerBody: { color: tokens.color.text.muted, fontSize: tokens.type['body'], lineHeight: 22, textAlign: 'center' },
});
