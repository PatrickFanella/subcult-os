import { useState, type ReactNode } from 'react';
import { Platform, Text, TextInput, View, type StyleProp, type TextInputProps, type ViewStyle } from 'react-native';
import { useThemeTokens } from '@/theme/ThemeProvider';
import { terminalFontFamily } from '@/theme/terminal';

type FieldProps = TextInputProps & {
  label?: string;
  // Leading icon rendered inside the field frame (for example, a search glyph).
  icon?: ReactNode;
  // Layout for the label + input block.
  containerStyle?: StyleProp<ViewStyle>;
};

export function Field({ label, icon, containerStyle, multiline, style, accessibilityLabel, onFocus, onBlur, ...props }: FieldProps) {
  const tokens = useThemeTokens();
  // With an icon the frame is a View, so it carries the focus ring for the inner input.
  const [focused, setFocused] = useState(false);
  const fontFamily = terminalFontFamily(Platform.OS);
  const inputStyle = [
    { fontFamily, color: tokens.color.text.primary },
    multiline ? { minHeight: 132, paddingVertical: 12, lineHeight: 21, textAlignVertical: 'top' as const } : null,
    style,
  ];
  const input = (
    <TextInput
      placeholderTextColor={tokens.color.text.muted}
      accessibilityLabel={accessibilityLabel ?? label}
      {...props}
      multiline={multiline}
      onFocus={(event) => { setFocused(true); onFocus?.(event); }}
      onBlur={(event) => { setFocused(false); onBlur?.(event); }}
      className={icon ? 'flex-1 text-body-lg outline-none' : 'field'}
      style={inputStyle}
    />
  );

  return (
    <View className="gap-2" style={containerStyle}>
      {label ? <Text className="text-fg-muted text-label font-bold uppercase" style={{ fontFamily, letterSpacing: 1.1 }}>{label}</Text> : null}
      {icon ? (
        <View style={{
          minHeight: tokens.size.field,
          flexDirection: 'row',
          alignItems: 'center',
          gap: 10,
          paddingRight: focused ? 15 : 16,
          borderWidth: focused ? 2 : 1,
          borderColor: focused ? tokens.color.border.focus : tokens.color.border.strong,
          paddingLeft: focused ? 15 : 16,
          backgroundColor: tokens.color.surface.inset,
        }}>
          {icon}
          {input}
        </View>
      ) : input}
    </View>
  );
}
