import { describe, expect, it } from 'vitest';

import {
	publicCardClass,
	publicEyebrowClass,
	publicHeroCardClass,
	publicMutedTextClass,
	publicPageInnerClass,
	publicPageShellClass,
	publicPrimaryButtonClass,
	publicSecondaryButtonClass,
	publicStatusPillClass,
} from './publicUi';

describe('publicUi', () => {
	it('uses app-parity light public surfaces', () => {
		expect(publicPageShellClass).toBe('min-h-screen bg-surface-canvas text-fg-primary');
		expect(publicPageInnerClass).toBe('mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8');
		expect(publicCardClass).toBe('rounded-panel border border-stroke-subtle bg-surface-panel p-5');
		expect(publicHeroCardClass).toBe('overflow-hidden rounded-hero border border-stroke-subtle bg-surface-panel');
		expect(publicPrimaryButtonClass).toBe(
			'btn-primary inline-flex items-center justify-center px-5 py-3 text-sm transition',
		);
		expect(publicSecondaryButtonClass).toBe(
			'btn-secondary inline-flex items-center justify-center px-5 py-3 text-sm transition',
		);
		expect(publicMutedTextClass).toBe('text-sm text-fg-secondary');
		expect(publicEyebrowClass).toBe('text-xs font-bold uppercase tracking-[0.24em] text-fg-muted');
	});

	it('provides stable status pill tones', () => {
		expect(publicStatusPillClass('success')).toBe('border border-status-success/20 bg-status-surface-success px-3 py-1 text-xs font-bold text-status-success');
		expect(publicStatusPillClass('warning')).toBe('border border-status-warning/20 bg-status-surface-warning px-3 py-1 text-xs font-bold text-status-warning');
		expect(publicStatusPillClass('danger')).toBe('border border-status-danger/20 bg-status-surface-danger px-3 py-1 text-xs font-bold text-status-danger');
		expect(publicStatusPillClass()).toBe('border border-stroke-subtle bg-surface-inset px-3 py-1 text-xs font-bold text-fg-secondary');
	});
});
