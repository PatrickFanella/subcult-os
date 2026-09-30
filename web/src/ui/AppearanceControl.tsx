import { useSyncExternalStore } from 'react';
import { currentAppearance, parseAppearance, setAppearance, subscribeAppearance } from './appearance';

export function AppearanceControl() {
  const value = useSyncExternalStore(subscribeAppearance, currentAppearance, () => 'system');
  return <label className="flex items-center gap-2 text-sm text-fg-secondary">
    <span>Appearance</span>
    <select aria-label="Appearance" className="min-h-touch rounded-control border border-stroke-strong bg-surface-panel px-3 text-fg-primary" value={value} onChange={(event) => setAppearance(parseAppearance(event.target.value))}>
      <option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option>
    </select>
  </label>;
}
