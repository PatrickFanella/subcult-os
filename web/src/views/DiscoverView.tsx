import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '../api';
import type { PublicEventSummaryDTO } from '../domain';
import {
	discoveryBadgeLabel,
	discoveryBrowseLabel,
	discoveryDescription,
	discoveryEmptyStateCopy,
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
		<main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
			<section className="mx-auto w-full max-w-6xl space-y-6">
				<header className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
					<p className="text-xs uppercase tracking-[0.35em] text-amber-300">{discoveryBrowseLabel}</p>
					<div className="mt-3 flex flex-wrap items-center gap-2">
						<span className="rounded-full border border-emerald-400/30 bg-emerald-500/10 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-emerald-200">
							{discoveryBadgeLabel}
						</span>
						<span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-zinc-200">
							{discoveryScopeLabel}
						</span>
					</div>

					<h1 className="mt-5 text-3xl font-semibold tracking-tight text-white sm:text-4xl">{discoveryTitle}</h1>
					<p className="mt-3 max-w-2xl text-sm leading-7 text-zinc-300 sm:text-base">{discoveryDescription}</p>

					<form className="mt-6 flex flex-col gap-3 rounded-2xl border border-white/10 bg-white/[0.03] p-4 sm:flex-row sm:items-end" onSubmit={handleSubmit}>
						<label className="flex-1 space-y-2">
							<span className="text-xs uppercase tracking-[0.25em] text-zinc-500">{discoverySearchLabel}</span>
							<input
								className="w-full rounded-2xl border border-white/10 bg-zinc-950/80 px-4 py-3 text-sm text-white outline-none ring-0 transition placeholder:text-zinc-500 focus:border-amber-300/40 focus:bg-zinc-900"
								type="search"
								value={searchQuery}
								onChange={(event) => setSearchQuery(event.target.value)}
								placeholder={discoverySearchPlaceholder}
							/>
						</label>
						<div className="flex gap-3 sm:shrink-0">
							<button className="rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200" type="submit">
								Search
							</button>
							<button className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 font-medium text-zinc-100 transition hover:bg-white/10" type="button" onClick={() => handleReset()}>
								Reset
							</button>
						</div>
					</form>
				</header>

				{loading ? <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">{discoveryLoadingCopy}</div> : null}

				{error ? (
					<p aria-live="polite" className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
						{discoveryErrorCopy(error)}
					</p>
				) : null}

				{!loading && !error && events?.length === 0 ? (
					<div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">
						{discoveryEmptyStateCopy(searchQuery)}
					</div>
				) : null}

				{!loading && !error && events && events.length > 0 ? (
					<div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
						{events.map((event) => (
							<article key={event.id} className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-5 shadow-lg shadow-black/20">
								<div className="flex items-start justify-between gap-3">
									<div>
										<p className="text-xs uppercase tracking-[0.3em] text-amber-300">{event.status}</p>
										<h2 className="mt-2 text-xl font-semibold text-white">{event.title}</h2>
									</div>
									<span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.25em] text-zinc-300">
										{discoveryRemainingLabel(event)}
									</span>
								</div>

								<div className="mt-4 grid gap-3 text-sm text-zinc-300">
									<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
										<p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Hosted by</p>
										<p className="mt-2 font-medium text-white">{event.workspaceName}</p>
									</div>
									<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
										<p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Date</p>
										<p className="mt-2 font-medium text-white">{formatDiscoveryDateTime(event.startsAt)}</p>
									</div>
									<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
										<p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Location</p>
										<p className="mt-2 font-medium text-white">{event.locationDisplay}</p>
									</div>
									<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
										<p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Description</p>
										<p className="mt-2 leading-6 text-zinc-300">{event.publicDescription || 'No public description provided.'}</p>
									</div>
									<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
										<p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Pricing</p>
										<p className="mt-2 font-medium text-white">{discoveryPricingLabel(event)}</p>
									</div>
									<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
										<p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Remaining tickets</p>
										<p className={`mt-2 font-medium ${event.isFull ? 'text-rose-200' : 'text-white'}`}>{event.remainingTickets}</p>
									</div>
									{event.applicationsOpen ? (
										<div className="rounded-2xl border border-emerald-400/30 bg-emerald-500/10 p-4">
											<p className="text-xs uppercase tracking-[0.2em] text-emerald-200">Applications</p>
											<p className="mt-2 font-medium text-white">Applications open</p>
										</div>
									) : null}
								</div>

								<a className="mt-5 inline-flex w-full items-center justify-center rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200" href={event.publicUrl}>
									{discoveryViewEventLabel}
								</a>
							</article>
						))}
					</div>
				) : null}
			</section>
		</main>
	);
}
