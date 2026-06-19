export const publicPageShellClass = 'min-h-screen bg-[#f5f5f5] text-[#171717]';

export const publicPageInnerClass = 'mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8';

export const publicCardClass = 'rounded-[28px] border border-neutral-200 bg-white p-5 shadow-sm';

export const publicHeroCardClass = 'overflow-hidden rounded-[32px] border border-neutral-200 bg-white shadow-sm';

export const publicPrimaryButtonClass =
	'inline-flex items-center justify-center rounded-full bg-[#171717] px-5 py-3 text-sm font-bold text-white transition hover:bg-black disabled:cursor-not-allowed disabled:bg-neutral-300';

export const publicSecondaryButtonClass =
	'inline-flex items-center justify-center rounded-full border border-neutral-300 bg-white px-5 py-3 text-sm font-bold text-[#171717] transition hover:bg-neutral-50';

export const publicMutedTextClass = 'text-sm text-neutral-600';

export const publicEyebrowClass = 'text-xs font-black uppercase tracking-[0.24em] text-neutral-500';

export function publicStatusPillClass(tone: 'neutral' | 'success' | 'warning' | 'danger' = 'neutral') {
	switch (tone) {
		case 'success':
			return 'rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700';
		case 'warning':
			return 'rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs font-bold text-amber-700';
		case 'danger':
			return 'rounded-full border border-rose-200 bg-rose-50 px-3 py-1 text-xs font-bold text-rose-700';
		default:
			return 'rounded-full border border-neutral-200 bg-neutral-100 px-3 py-1 text-xs font-bold text-neutral-700';
	}
}
