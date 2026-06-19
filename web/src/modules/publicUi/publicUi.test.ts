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
		expect(publicPageShellClass).toBe('min-h-screen bg-[#f5f5f5] text-[#171717]');
		expect(publicPageInnerClass).toBe('mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8');
		expect(publicCardClass).toBe('rounded-[28px] border border-neutral-200 bg-white p-5 shadow-sm');
		expect(publicHeroCardClass).toBe('overflow-hidden rounded-[32px] border border-neutral-200 bg-white shadow-sm');
		expect(publicPrimaryButtonClass).toBe(
			'inline-flex items-center justify-center rounded-full bg-[#171717] px-5 py-3 text-sm font-bold text-white transition hover:bg-black disabled:cursor-not-allowed disabled:bg-neutral-300',
		);
		expect(publicSecondaryButtonClass).toBe(
			'inline-flex items-center justify-center rounded-full border border-neutral-300 bg-white px-5 py-3 text-sm font-bold text-[#171717] transition hover:bg-neutral-50',
		);
		expect(publicMutedTextClass).toBe('text-sm text-neutral-600');
		expect(publicEyebrowClass).toBe('text-xs font-black uppercase tracking-[0.24em] text-neutral-500');
	});

	it('provides stable status pill tones', () => {
		expect(publicStatusPillClass('success')).toBe('rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700');
		expect(publicStatusPillClass('warning')).toBe('rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs font-bold text-amber-700');
		expect(publicStatusPillClass('danger')).toBe('rounded-full border border-rose-200 bg-rose-50 px-3 py-1 text-xs font-bold text-rose-700');
		expect(publicStatusPillClass()).toBe('rounded-full border border-neutral-200 bg-neutral-100 px-3 py-1 text-xs font-bold text-neutral-700');
	});
});
