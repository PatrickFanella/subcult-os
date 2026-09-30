import { ActivityIndicator, Pressable, Text } from 'react-native';
import { useThemeTokens } from '@/theme/ThemeProvider';

type PrimaryButtonProps = {
  label: string;
  onPress?: () => void;
  disabled?: boolean;
  busy?: boolean;
};

export function PrimaryButton({ label, onPress, disabled = false, busy = false }: PrimaryButtonProps) {
  const tokens = useThemeTokens();

  return (
    <Pressable onPress={onPress} disabled={disabled || busy} accessibilityRole="button"
      accessibilityState={{ disabled: disabled || busy, busy }}
      className="btn-primary flex-row gap-2 active:opacity-80" style={disabled || busy ? { opacity: 0.5 } : undefined}>
      {busy ? <ActivityIndicator color={tokens.color.text.inverse} /> : null}
      <Text className="text-fg-inverse text-body font-black">{label}</Text>
    </Pressable>
  );
}
