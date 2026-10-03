// Subcults terminal treatment for native StyleSheet factories: one small corner radius
// on every frame and monospace text. Exact Space Mono loading is a separate device task.
import { tokens } from './tokens';

export const terminalFontFamily = (os: string) => (os === 'ios' ? 'Courier' : 'monospace');

const radiusKeys = [
  'borderRadius',
  'borderTopLeftRadius',
  'borderTopRightRadius',
  'borderBottomLeftRadius',
  'borderBottomRightRadius',
  'borderTopStartRadius',
  'borderTopEndRadius',
  'borderBottomStartRadius',
  'borderBottomEndRadius',
];
// `color` and these keys only apply to Text and TextInput styles.
const textKeys = ['fontSize', 'fontWeight', 'fontFamily', 'color', 'letterSpacing', 'lineHeight', 'textTransform'];

export function terminalStyles<T>(styles: T, os: string): T {
  if (!styles || typeof styles !== 'object') return styles;
  const entries = Object.entries(styles).map(([name, value]) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return [name, value];
    const style = { ...value } as Record<string, unknown>;
    // Avatars keep their portrait shape; every other rounded frame uses the shared corner.
    if (!name.toLowerCase().includes('avatar')) {
      for (const key of radiusKeys) if (key in style) style[key] = tokens.radius.card;
    }
    if (textKeys.some((key) => key in style)) {
      style.fontFamily = terminalFontFamily(os);
      // The terminal type scale stops at bold.
      if (style.fontWeight && Number(style.fontWeight) > 700) style.fontWeight = '700';
    }
    return [name, style];
  });
  return Object.fromEntries(entries) as T;
}
