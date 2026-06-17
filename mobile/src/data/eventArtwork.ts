const fallbackImages = [
  'https://images.unsplash.com/photo-1459749411175-04bf5292ceea?auto=format&fit=crop&w=1080&q=80',
  'https://images.unsplash.com/photo-1470225620780-dba8ba36b745?auto=format&fit=crop&w=1080&q=80',
  'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?auto=format&fit=crop&w=1080&q=80',
  'https://images.unsplash.com/photo-1571204829887-3b8d69e4094d?auto=format&fit=crop&w=1080&q=80',
];

export function eventArtwork(seed: string | null | undefined) {
  if (!seed) {
    return fallbackImages[0];
  }

  const total = Array.from(seed).reduce((sum, char) => sum + char.charCodeAt(0), 0);
  return fallbackImages[total % fallbackImages.length];
}
