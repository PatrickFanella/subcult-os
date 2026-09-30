import { createContext, useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from 'react';
import { Platform } from 'react-native';
import * as SecureStore from 'expo-secure-store';
import { Uniwind, useUniwind } from 'uniwind';
import { darkTokens, tokens, type Tokens } from './tokens';

type Appearance = 'system' | 'light' | 'dark';
const key = 'subcult.appearance';
const parse = (value: unknown): Appearance => value === 'dark' || value === 'light' ? value : 'system';
const ThemeContext = createContext({ preference: 'system' as Appearance, setPreference: (_value: Appearance) => {} });

export function ThemeProvider({ children }: PropsWithChildren) {
  const [preference, setPreferenceState] = useState<Appearance>('system');
  const changed = useRef(false);
  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const value = Platform.OS === 'web' ? localStorage.getItem(key) : await SecureStore.getItemAsync(key);
        if (alive && !changed.current) { const next = parse(value); setPreferenceState(next); Uniwind.setTheme(next); }
      } catch { /* System appearance remains available without storage. */ }
    };
    void load();
    return () => { alive = false; };
  }, []);
  const setPreference = (next: Appearance) => {
    changed.current = true;
    setPreferenceState(next);
    Uniwind.setTheme(next);
    try {
      if (Platform.OS === 'web') localStorage.setItem(key, next);
      else void SecureStore.setItemAsync(key, next).catch(() => {});
    } catch { /* The current session still changes appearance. */ }
  };
  return <ThemeContext.Provider value={{ preference, setPreference }}>{children}</ThemeContext.Provider>;
}
export function useAppearance() { return useContext(ThemeContext); }
export function useThemeTokens(): Tokens {
  const { theme } = useUniwind();
  return theme === 'dark' ? darkTokens : tokens;
}
export function useThemedStyles<T>(factory: (tokens: Tokens) => T): T {
  const selected = useThemeTokens();
  return useMemo(() => factory(selected), [factory, selected]);
}
