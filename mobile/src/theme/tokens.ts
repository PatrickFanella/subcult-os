export const tokens = {
  color: {
    surface: {
      canvas: '#09090b',
      default: '#09090b',
      panel: '#18181b',
      elevated: '#27272a',
      inset: '#0f1115',
      scrim: 'rgba(0, 0, 0, 0.62)',
    },
    text: {
      primary: '#fafafa',
      secondary: '#d4d4d8',
      muted: '#a1a1aa',
      inverse: '#09090b',
    },
    border: {
      subtle: 'rgba(255,255,255,0.10)',
      strong: 'rgba(255,255,255,0.18)',
      focus: '#fbbf24',
    },
    accent: {
      warm: '#f59e0b',
      cool: '#22d3ee',
      brand: '#d946ef',
    },
    action: {
      primary: '#fbbf24',
      secondary: '#22d3ee',
      brand: '#d946ef',
    },
    status: {
      success: '#34d399',
      warning: '#fbbf24',
      danger: '#fb7185',
      info: '#38bdf8',
    },
  },
  space: {
    1: 4,
    2: 8,
    3: 12,
    4: 16,
    5: 24,
    6: 32,
    7: 48,
  },
  radius: {
    card: 22,
    panel: 30,
    pill: 999,
  },
  size: {
    tap: 48,
    field: 56,
  },
} as const;

export type Tokens = typeof tokens;
