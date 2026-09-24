import { Children, isValidElement } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { renderToString } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DiscoveryOccurrencesSection } from './DiscoverView';
import type { PublicDiscoveryOccurrenceDTO } from '../domain';

// This repository's convention for interactive-component tests (see
// IdentityActionView.test.tsx) mocks React's hooks so a function component
// can be invoked directly as a plain function and its returned element tree
// walked for handler props, without a jsdom/testing-library dependency.
// Escape-to-close is wired through a real `document.addEventListener`
// effect and is therefore not exercised here; see the "Known limits"
// section of docs/development/discovery-ux.md for what only a real browser
// check can confirm.
const SKIP = Symbol('skip-state');

function makeUseStateImplementation(values: unknown[] = []) {
	const setters: Array<ReturnType<typeof vi.fn>> = [];
	const impl = ((initial: unknown) => {
		const setter = vi.fn();
		setters.push(setter);
		if (values.length > 0) {
			const next = values.shift();
			if (next === SKIP) {
				return [typeof initial === 'function' ? (initial as () => unknown)() : initial, setter];
			}
			return [next, setter];
		}
		return [typeof initial === 'function' ? (initial as () => unknown)() : initial, setter];
	}) as unknown as typeof import('react').useState;
	return { impl, setters };
}

let currentUseState: typeof import('react').useState = ((initial: unknown) => [initial, vi.fn()]) as never;

vi.mock('react', async () => {
	const actual = await vi.importActual<typeof import('react')>('react');
	return {
		...actual,
		useState: (initial: unknown) => currentUseState(initial),
		useEffect: () => undefined,
		useRef: (initial: unknown) => ({ current: initial }),
	};
});

vi.mock('../api', () => ({ api: vi.fn() }));

afterEach(() => {
	vi.unstubAllGlobals();
	vi.clearAllMocks();
});

type AnyElement = ReactElement<Record<string, unknown>>;

function findAll(node: ReactNode, predicate: (el: AnyElement) => boolean, out: AnyElement[] = []): AnyElement[] {
	for (const child of Children.toArray(node)) {
		if (!isValidElement<{ children?: ReactNode }>(child)) continue;
		const element = child as AnyElement;
		if (predicate(element)) out.push(element);
		if (element.props.children) findAll(element.props.children as ReactNode, predicate, out);
	}
	return out;
}

function findOne(node: ReactNode, predicate: (el: AnyElement) => boolean): AnyElement {
	const found = findAll(node, predicate);
	if (found.length === 0) throw new Error('element not found');
	return found[0];
}

function renderSection(stateValues: unknown[]) {
	const { impl, setters } = makeUseStateImplementation([...stateValues]);
	currentUseState = impl;
	const element = DiscoveryOccurrencesSection();
	return { element, setters };
}

function occurrence(overrides: Partial<PublicDiscoveryOccurrenceDTO> = {}): PublicDiscoveryOccurrenceDTO {
	return {
		uri: 'at://did:plc:abc/tv.subcult.event.occurrence/1',
		source: { did: 'did:plc:abc', uri: 'at://did:plc:abc/tv.subcult.event.occurrence/1' },
		name: 'Test Occurrence',
		startsAt: '2026-10-01T20:00:00.000Z',
		status: 'scheduled',
		projectionStatus: 'active',
		handoff: { kind: 'none', reason: 'no_mapping' },
		...overrides,
	};
}

describe('DiscoveryOccurrencesSection states', () => {
	it('renders the loading state', () => {
		const { element } = renderSection([null, true, null, null]);
		const html = renderToString(element);
		expect(html).toContain('Loading discovery occurrences');
	});

	it('renders the empty state', () => {
		const { element } = renderSection([[], false, null, null]);
		const html = renderToString(element);
		expect(html).toContain('No discovery occurrences yet');
	});

	it('renders the error state', () => {
		const { element } = renderSection([[], false, 'Network down', null]);
		const html = renderToString(element);
		expect(html).toContain('Could not load discovery occurrences: Network down');
	});

	it('renders occurrence cards in a mobile-first single-column layout', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night' }), occurrence({ uri: 'uri-b', name: 'Beta Night' })];
		const { element } = renderSection([list, false, null, null]);
		const html = renderToString(element);
		expect(html).toContain('Alpha Night');
		expect(html).toContain('Beta Night');
		// Base class is single-column (no explicit column count until the
		// sm: breakpoint), matching the 360px narrow-screen requirement;
		// full media-query evaluation still needs a real browser (see the
		// "Known limits" note in the test file header).
		expect(html).toContain('grid-cols-1');
		expect(html).toContain('sm:grid-cols-2');
	});

	it('opens the detail view on Enter/Space and never crashes on other keys', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night' })];
		const { element, setters } = renderSection([list, false, null, null]);
		const card = findOne(element, (el) => el.type === 'article' && el.props.role === 'button');
		const onKeyDown = card.props.onKeyDown as (event: { key: string; preventDefault: () => void }) => void;

		onKeyDown({ key: 'Tab', preventDefault: vi.fn() });
		const selectedURISetter = setters[3];
		expect(selectedURISetter).not.toHaveBeenCalled();

		onKeyDown({ key: 'Enter', preventDefault: vi.fn() });
		expect(selectedURISetter).toHaveBeenCalledWith('uri-a');

		onKeyDown({ key: ' ', preventDefault: vi.fn() });
		expect(selectedURISetter).toHaveBeenCalledWith('uri-a');
	});

	it('opens the detail view when a card is clicked', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night' })];
		const { element, setters } = renderSection([list, false, null, null]);
		const card = findOne(element, (el) => el.type === 'article' && el.props.role === 'button');
		(card.props.onClick as () => void)();
		expect(setters[3]).toHaveBeenCalledWith('uri-a');
	});

	it('shows the local reservation handoff for a bookable occurrence', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night', handoff: { kind: 'local', eventSlug: 'alpha-night' } })];
		const { element } = renderSection([list, false, null, 'uri-a']);
		const html = renderToString(element);
		expect(html).toContain('href="/e/alpha-night"');
		expect(html).toContain('Reserve');
	});

	it('shows a clear unavailable state when handoff is none', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night', handoff: { kind: 'none', reason: 'no_mapping' } })];
		const { element } = renderSection([list, false, null, 'uri-a']);
		const html = renderToString(element);
		expect(html).toContain('No reservation destination has been linked for this occurrence yet.');
		expect(html).not.toContain('href="/e/');
	});

	it('shows a clear unavailable state when the projection marked the occurrence deleted/unavailable', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night', projectionStatus: 'deleted', handoff: { kind: 'local', eventSlug: 'alpha-night' } })];
		const { element } = renderSection([list, false, null, 'uri-a']);
		const html = renderToString(element);
		expect(html).toContain('This occurrence is no longer available from its source.');
		expect(html).not.toContain('href="/e/alpha-night"');
	});

	it('closes the detail view when the close button is activated', () => {
		const list = [occurrence({ uri: 'uri-a', name: 'Alpha Night' })];
		const { element, setters } = renderSection([list, false, null, 'uri-a']);
		const closeButton = findOne(element, (el) => el.type === 'button' && el.props.children === 'Close');
		(closeButton.props.onClick as () => void)();
		expect(setters[3]).toHaveBeenCalledWith(null);
	});

	it('plots occurrences with public coordinates and omits ones without', () => {
		const list = [
			occurrence({ uri: 'uri-with-location', location: { name: 'Venue', latitude: '41.8', longitude: '-87.6' } }),
			occurrence({ uri: 'uri-without-location' }),
		];
		const { element } = renderSection([list, false, null, null]);
		const html = renderToString(element);
		expect(html).toContain('discovery-map-point-uri-with-location');
		expect(html).not.toContain('discovery-map-point-uri-without-location');
	});
});
