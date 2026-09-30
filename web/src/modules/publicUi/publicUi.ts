export const publicPageShellClass = 'min-h-screen bg-surface-canvas text-fg-primary';

export const publicPageInnerClass = 'mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8';

export const publicCardClass = 'rounded-panel border border-stroke-subtle bg-surface-panel p-5 shadow-sm';

export const publicHeroCardClass = 'overflow-hidden rounded-hero border border-stroke-subtle bg-surface-panel shadow-sm';

export const publicPrimaryButtonClass =
	'btn-primary inline-flex items-center justify-center px-5 py-3 text-sm transition';

export const publicSecondaryButtonClass =
	'btn-secondary inline-flex items-center justify-center px-5 py-3 text-sm transition';

export const publicMutedTextClass = 'text-sm text-fg-secondary';

export const publicEyebrowClass = 'text-xs font-black uppercase tracking-[0.24em] text-fg-muted';

export function publicStatusPillClass(tone: 'neutral' | 'success' | 'warning' | 'danger' = 'neutral') {
	switch (tone) {
		case 'success':
			return 'rounded-full border border-status-success/20 bg-status-surface-success px-3 py-1 text-xs font-bold text-status-success';
		case 'warning':
			return 'rounded-full border border-status-warning/20 bg-status-surface-warning px-3 py-1 text-xs font-bold text-status-warning';
		case 'danger':
			return 'rounded-full border border-status-danger/20 bg-status-surface-danger px-3 py-1 text-xs font-bold text-status-danger';
		default:
			return 'rounded-full border border-stroke-subtle bg-surface-inset px-3 py-1 text-xs font-bold text-fg-secondary';
	}
}
