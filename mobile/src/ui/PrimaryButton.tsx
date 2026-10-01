import type { ReactNode } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, type StyleProp, type ViewStyle } from 'react-native';
import { useThemeTokens } from '@/theme/ThemeProvider';
import { terminalFontFamily } from '@/theme/terminal';

type ButtonVariant = 'primary' | 'secondary' | 'danger';

type PrimaryButtonProps = {
  label: string;
  onPress?: () => void;
  disabled?: boolean;
  busy?: boolean;
  variant?: ButtonVariant;
  icon?: ReactNode;
  accessibilityLabel?: string;
  // Layout only (flex, alignSelf, margins); colors come from the variant.
  style?: StyleProp<ViewStyle>;
};

const frameClasses: Record<ButtonVariant, string> = {
  primary: 'btn-primary',
  secondary: 'btn-secondary',
  danger: 'btn-danger',
};

const labelClasses: Record<ButtonVariant, string> = {
  primary: 'text-fg-inverse',
  secondary: 'text-fg-primary',
  danger: 'text-status-danger',
};

export function PrimaryButton({ label, onPress, disabled = false, busy = false, variant = 'primary', icon, accessibilityLabel, style }: PrimaryButtonProps) {
  const tokens = useThemeTokens();
  const inactive = disabled || busy;
  const spinnerColor = variant === 'primary' ? tokens.color.text.inverse : variant === 'danger' ? tokens.color.status.danger : tokens.color.text.primary;

  return (
    <Pressable onPress={onPress} disabled={inactive} accessibilityRole="button" accessibilityLabel={accessibilityLabel}
      accessibilityState={{ disabled: inactive, busy }}
      className={`${frameClasses[variant]} flex-row gap-2 active:opacity-80`} style={[inactive ? { opacity: 0.5 } : null, style]}>
      {busy ? <ActivityIndicator color={spinnerColor} /> : icon}
      <Text className={`${labelClasses[variant]} text-body font-bold text-center`} style={{ fontFamily: terminalFontFamily(Platform.OS), textTransform: 'uppercase', flexShrink: 1 }}>{label}</Text>
    </Pressable>
  );
}
