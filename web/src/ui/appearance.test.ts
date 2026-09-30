import { afterEach, describe, expect, it, vi } from 'vitest';
import { appearanceKey, currentAppearance, parseAppearance, readAppearance, setAppearance, subscribeAppearance } from './appearance';

afterEach(() => vi.unstubAllGlobals());
describe('appearance preference', () => {
  it('accepts explicit themes and defaults invalid preferences to system', () => {
    expect(parseAppearance('light')).toBe('light');
    expect(parseAppearance('dark')).toBe('dark');
    for (const value of [undefined, null, 'unknown', 'system']) expect(parseAppearance(value)).toBe('system');
  });
  it('changes the current theme even when persistence is unavailable', () => {
    const root = { dataset: { theme: 'system' } };
    const events = new EventTarget();
    vi.stubGlobal('document', { documentElement: root });
    vi.stubGlobal('window', events);
    vi.stubGlobal('localStorage', { getItem: () => { throw new Error('blocked'); }, setItem: () => { throw new Error('blocked'); } });
    const listener = vi.fn();
    const stop = subscribeAppearance(listener);
    expect(readAppearance()).toBe('system');
    setAppearance('dark');
    expect(currentAppearance()).toBe('dark');
    expect(listener).toHaveBeenCalledOnce();
    stop();
    setAppearance('light');
    expect(listener).toHaveBeenCalledOnce();
  });
  it('persists a selection and reads it on reload', () => {
    const values = new Map<string, string>();
    vi.stubGlobal('document', { documentElement: { dataset: {} } });
    vi.stubGlobal('window', new EventTarget());
    vi.stubGlobal('localStorage', { getItem: (key: string) => values.get(key), setItem: (key: string, value: string) => values.set(key, value) });
    setAppearance('light');
    expect(values.get(appearanceKey)).toBe('light');
    expect(readAppearance()).toBe('light');
    setAppearance('system');
    expect(readAppearance()).toBe('system');
  });
});
