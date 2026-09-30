import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { unknownAccessEntry } from '../modules/eventAccess/eventAccessModel';
import { AccessEntrySummary } from './EventAccessView';

describe('access information display', () => {
  it('does not translate unknown into a refusal or assurance', () => {
    const html = renderToStaticMarkup(<AccessEntrySummary entry={unknownAccessEntry('entry')} evaluatedAt="2026-10-01T00:00:00Z" />);
    expect(html).toContain('Unknown');
    expect(html).toContain('Unknown does not mean no');
    expect(html).not.toContain('Organizer assertion');
  });
  it('labels expired information separately from the retained original assertion', () => {
    const entry = { ...unknownAccessEntry('entry'), evaluatedAt: '2026-10-01T00:00:00Z', value: 'yes' as const, expiresAt: '2026-09-01T00:00:00Z', reviewedAt: '2026-08-01T00:00:00Z', sourceKind: 'organizer_assertion' as const, sourceReference: 'Synthetic entrance plan' };
    const html = renderToStaticMarkup(<AccessEntrySummary entry={entry} evaluatedAt="2026-08-30T00:00:00Z" />);
    expect(html).toContain('Unknown');
    expect(html).toContain('review expired');
    expect(html).toContain('previous value was yes');
    expect(html).toContain('Organizer assertion');
    expect(html).toContain('Synthetic entrance plan');
  });
  it('distinguishes event observation from venue inheritance and escapes source content', () => {
    const entry = { ...unknownAccessEntry('sensory'), value: 'known' as const, effectiveValue: 'known' as const, sourceKind: 'event_observation' as const, sourceReference: '<script>source</script>', details: 'Synthetic sound check', reviewedAt: '2026-09-30T00:00:00Z' };
    const html = renderToStaticMarkup(<AccessEntrySummary entry={entry} evaluatedAt="2026-10-01T00:00:00Z" />);
    expect(html).toContain('Event-specific observation');
    expect(html).toContain('event-specific');
    expect(html).toContain('&lt;script&gt;');
    expect(html).not.toContain('<script>');
  });
});
