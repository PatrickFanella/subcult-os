export type Appearance = 'system' | 'light' | 'dark';
export const appearanceKey = 'subcult.appearance';
export function parseAppearance(value: unknown): Appearance {
  return value === 'light' || value === 'dark' ? value : 'system';
}
export function readAppearance(): Appearance {
  try { return parseAppearance(localStorage.getItem(appearanceKey)); } catch { return 'system'; }
}
export function setAppearance(value: Appearance) {
  document.documentElement.dataset.theme = value;
  try { localStorage.setItem(appearanceKey, value); } catch { /* The current session still changes theme. */ }
  window.dispatchEvent(new Event('subcult-appearance'));
}
export function subscribeAppearance(listener: () => void) {
  const sync = (event: StorageEvent) => {
    if (event.key !== null && event.key !== appearanceKey) return;
    document.documentElement.dataset.theme = readAppearance(); listener();
  };
  window.addEventListener('storage', sync);
  window.addEventListener('subcult-appearance', listener);
  return () => { window.removeEventListener('storage', sync); window.removeEventListener('subcult-appearance', listener); };
}
export function currentAppearance(): Appearance {
  return parseAppearance(document.documentElement.dataset.theme);
}
