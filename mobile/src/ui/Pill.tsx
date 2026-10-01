import type { PropsWithChildren } from 'react';
import { Platform, Text, type StyleProp, type TextStyle } from 'react-native';
import { useThemeTokens } from '@/theme/ThemeProvider';
import { terminalFontFamily } from '@/theme/terminal';
import type { Tokens } from '@/theme/tokens';

export type PillTone = 'neutral' | 'accent' | 'success' | 'warning' | 'danger' | 'info';

type PillProps = PropsWithChildren<{
  tone?: PillTone;
  // Layout only (alignSelf, margins); colors come from the tone.
  style?: StyleProp<TextStyle>;
}>;

function toneColors(tone: PillTone, { color }: Tokens) {
  switch (tone) {
    case 'accent': return { color: color.action.primary, backgroundColor: color.surface.inset, borderColor: color.action.primary };
    case 'success':
    case 'warning':
    case 'danger':
    case 'info': return { color: color.status[tone], backgroundColor: color.statusSurface[tone], borderColor: color.status[tone] };
    default: return { color: color.text.muted, backgroundColor: color.surface.inset, borderColor: color.border.subtle };
  }
}

// Square, uppercase status label. Always pair the tone with words.
export function Pill({ children, tone = 'neutral', style }: PillProps) {
  const tokens = useThemeTokens();
  return (
    <Text className="chip" style={[toneColors(tone, tokens), { fontFamily: terminalFontFamily(Platform.OS), borderRadius: 0, overflow: 'hidden', flexShrink: 0 }, style]}>
      {children}
    </Text>
  );
}
