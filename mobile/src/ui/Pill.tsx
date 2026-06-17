import type { PropsWithChildren } from 'react';
import { Text } from 'react-native';

type PillTone = 'neutral' | 'accent' | 'success' | 'warning' | 'info';

type PillProps = PropsWithChildren<{
  tone?: PillTone;
}>;

const toneClasses: Record<PillTone, string> = {
  neutral: 'text-fg-muted',
  accent: 'text-action-primary',
  success: 'text-status-success',
  warning: 'text-status-warning',
  info: 'text-status-info',
};

export function Pill({ children, tone = 'neutral' }: PillProps) {
  return <Text className={['chip', toneClasses[tone]].join(' ')}>{children}</Text>;
}
