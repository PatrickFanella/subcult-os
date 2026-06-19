import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '../api';
import type { PublicEventSummaryDTO } from '../domain';
import {
	discoveryBadgeLabel,
	discoveryBrowseLabel,
	discoveryDescription,
	discoveryEmptyBody,
	discoveryEmptyTitle,
	discoveryErrorCopy,
	discoveryLoadingCopy,
	discoveryPricingLabel,
	discoveryRemainingLabel,
	discoveryScopeLabel,
	discoverySearchLabel,
	discoverySearchPlaceholder,
	discoveryTitle,
	discoveryViewEventLabel,
	formatDiscoveryDateTime,
	getRequestedDiscoveryQuery,
} from '../modules/discovery/discoveryModel';
import {
	publicCardClass,
	publicEyebrowClass,
	publicMutedTextClass,
	publicPageInnerClass,
	publicPageShellClass,
	publicPrimaryButtonClass,
	publicSecondaryButtonClass,
	publicStatusPillClass,
} from '../modules/publicUi/publicUi';

export function DiscoverView() {
	const [events, setEvents] = useState<PublicEventSummaryDTO[] | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [searchQuery, setSearchQuery] = useState(() => getRequestedDiscoveryQuery().trim());
	const requestSeq = useRef(0);

	async function loadEvents(nextQuery: string, syncUrl = false) {
		const normalizedQuery = nextQuery.trim();
		const requestID = ++requestSeq.current;

		if (syncUrl && typeof window !== 'undefined') {
			const suffix = normalizedQuery ? `?q=${encodeURIComponent(normalizedQuery)}` : '';
			window.history.pushState({}, '', `/discover${suffix}`);
		}

		setLoading(true);
		setError(null);

		try {
			const path = normalizedQuery ? `/api/public/events?q=${encodeURIComponent(normalizedQuery)}` : '/api/public/events';
			const loaded = await api<PublicEventSummaryDTO[]>(path);
			if (requestSeq.current !== requestID) {
				return;
			}
			setEvents(loaded ?? []);
		} catch (caught) {
			if (requestSeq.current !== requestID) {
				return;
			}
			setError(caught instanceof Error ? caught.message : 'Unable to load published events');
			setEvents([]);
		} finally {
			if (requestSeq.current === requestID) {
				setLoading(false);
			}
		}
	}

	useEffect(() => {
		void loadEvents(searchQuery);
	}, []);

	function handleSubmit(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		const nextQuery = searchQuery.trim();
		setSearchQuery(nextQuery);
		void loadEvents(nextQuery, true);
	}

	function handleReset() {
		setSearchQuery('');
		void loadEvents('', true);
	}

	return (
		<main className={publicPageShellClass}>
			<section className={`${publicPageInnerClass} max-w-6xl`}>
				<header className="rounded-[32px] border border-neutral-200 bg-white p-6 shadow-sm sm:p-8">
					<p className={publicEyebrowClass}>{discoveryBrowseLabel}</p>
					<div className="mt-3 flex flex-wrap items-center gap-2">
						<span className={publicStatusPillClass('success')}>{discoveryBadgeLabel}</span>
						<span className={publicStatusPillClass()}>{discoveryScopeLabel}</span>
					</div>

					<h1 className="mt-5 text-4xl font-black tracking-tight text-[#171717] sm:text-5xl">{discoveryTitle}</h1>
					<p className="mt-3 max-w-2xl text-base leading-7 text-neutral-600">{discoveryDescription}</p>

					<form className="mt-6 flex flex-col gap-3 rounded-[24px] border border-neutral-200 bg-neutral-50 p-4 sm:flex-row sm:items-end" onSubmit={handleSubmit}>
						<label className="flex-1 space-y-2">
							<span className={publicEyebrowClass}>{discoverySearchLabel}</span>
							<input
								className="w-full rounded-full border border-neutral-300 bg-white px-4 py-3 text-sm text-[#171717] outline-none transition placeholder:text-neutral-400 focus:border-[#171717]"
								type="search"
								value={searchQuery}
								onChange={(event) => setSearchQuery(event.target.value)}
								placeholder={discoverySearchPlaceholder}
							/>
						</label>
						<div className="flex gap-3 sm:shrink-0">
							<button className={publicPrimaryButtonClass} type="submit">
								Search
							</button>
							<button className={publicSecondaryButtonClass} type="button" onClick={() => handleReset()}>
								Reset
							</button>
						</div>
					</form>
				</header>

				{loading ? <div className={`${publicCardClass} ${publicMutedTextClass}`}>{discoveryLoadingCopy}</div> : null}

				{error ? (
					<p aria-live="polite" className="rounded-[24px] border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-medium text-rose-700">
						{discoveryErrorCopy(error)}
					</p>
				) : null}

				{!loading && !error && events?.length === 0 ? (
					<div className={publicCardClass}>
						<h2 className="text-xl font-black text-[#171717]">{discoveryEmptyTitle(searchQuery)}</h2>
						<p className={`mt-2 ${publicMutedTextClass}`}>{discoveryEmptyBody(searchQuery)}</p>
					</div>
				) : null}

				{!loading && !error && events && events.length > 0 ? (
					<div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
						{events.map((event) => (
							<article key={event.id} className="overflow-hidden rounded-[28px] border border-neutral-200 bg-white shadow-sm">
								{event.imageUrl ? <img className="h-48 w-full object-cover" src={event.imageUrl} alt="" /> : null}

								<div className="flex h-full flex-col p-5">
									<div className="flex items-start justify-between gap-3">
										<div>
											<p className={publicEyebrowClass}>{formatDiscoveryDateTime(event.startsAt)}</p>
											<h2 className="mt-2 text-2xl font-black leading-tight text-[#171717]">{event.title}</h2>
										</div>
										<span className={publicStatusPillClass(event.isFull ? 'danger' : 'neutral')}>{discoveryRemainingLabel(event)}</span>
									</div>

									<div className="mt-4 grid gap-3 text-sm text-neutral-700">
										<div className="rounded-[20px] bg-neutral-50 p-4">
											<p className={publicEyebrowClass}>Hosted by</p>
											<p className="mt-2 font-bold text-[#171717]">{event.workspaceName}</p>
										</div>
										<div className="rounded-[20px] bg-neutral-50 p-4">
											<p className={publicEyebrowClass}>Location</p>
											<p className="mt-2 font-bold text-[#171717]">{event.locationDisplay}</p>
										</div>
										<div className="rounded-[20px] bg-neutral-50 p-4">
											<p className={publicEyebrowClass}>Description</p>
											<p className="mt-2 leading-6 text-neutral-600">{event.publicDescription || 'No public description provided.'}</p>
										</div>
										<div className="grid gap-3 sm:grid-cols-2">
											<div className="rounded-[20px] bg-neutral-50 p-4">
												<p className={publicEyebrowClass}>Pricing</p>
												<p className="mt-2 font-bold text-[#171717]">{discoveryPricingLabel(event)}</p>
											</div>
											<div className="rounded-[20px] bg-neutral-50 p-4">
												<p className={publicEyebrowClass}>Remaining tickets</p>
												<p className={`mt-2 font-bold ${event.isFull ? 'text-rose-700' : 'text-[#171717]'}`}>{event.remainingTickets}</p>
											</div>
										</div>
									{event.applicationsOpen ? (
										<div className="rounded-[20px] border border-emerald-200 bg-emerald-50 p-4">
											<p className="text-xs font-black uppercase tracking-[0.24em] text-emerald-700">Applications</p>
											<p className="mt-2 font-bold text-emerald-800">Applications open</p>
										</div>
									) : null}
								</div>

									<a className={`${publicPrimaryButtonClass} mt-5 w-full`} href={event.publicUrl}>
										{discoveryViewEventLabel}
									</a>
								</div>
							</article>
						))}
					</div>
				) : null}
			</section>
		</main>
	);
}
