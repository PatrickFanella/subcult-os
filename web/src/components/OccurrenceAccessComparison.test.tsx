import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { EventAccessRevisionDTO, OccurrenceAccessComparisonDTO, VenueAccessRevisionDTO } from '../domain';
import { accessTopics, unknownAccessEntry } from '../modules/eventAccess/eventAccessModel';
import { AccessComparisonResult } from './OccurrenceAccessComparison';

function fixture(): OccurrenceAccessComparisonDTO {
 const eventEntries: EventAccessRevisionDTO[] = accessTopics.map(({ topic }) => ({ ...unknownAccessEntry(topic), scope: 'event', sourceKind: 'unknown' }));
 const venueEntries: VenueAccessRevisionDTO[] = accessTopics.map(({ topic }) => ({ ...unknownAccessEntry(topic, 'venue'), scope: 'venue', sourceKind: 'unknown' }));
 return { eventId:'event',workspaceId:'workspace',evaluatedAt:'2026-10-01T00:00:00Z',occurrence:{id:'occurrence',name:'Synthetic occurrence',startsAt:'2026-12-01T12:00:00Z',status:'scheduled',updatedAt:'2026-09-30T00:00:00Z',placeId:'venue'},event:{eventId:'event',evaluatedAt:'2026-10-01T00:00:00Z',entries:eventEntries},venue:{placeId:'venue',placeName:'Synthetic venue',evaluatedAt:'2026-10-01T00:00:00Z',entries:venueEntries} };
}
describe('occurrence access comparison',()=>{
 it('keeps a positive venue assertion beside unknown event information',()=>{
  const comparison=fixture();comparison.venue!.entries[0]={...comparison.venue!.entries[0],value:'yes',effectiveValue:'yes',sourceKind:'venue_observation',sourceReference:'Synthetic venue visit',reviewedAt:'2026-09-30T00:00:00Z'};
  const html=renderToStaticMarkup(<AccessComparisonResult comparison={comparison}/>);
  expect(html).toContain('Venue observation');expect(html).toContain('Venue information');expect(html).toContain('Event information');expect(html).toContain('Unknown does not mean no');expect(html).toContain('Occurrence revision:');expect(html).not.toContain('<form');
 });
 it('shows a missing linked venue as unknown rather than an assurance',()=>{
  const comparison=fixture();comparison.venue=null;delete comparison.occurrence.placeId;
  const html=renderToStaticMarkup(<AccessComparisonResult comparison={comparison}/>);
  expect(html).toContain('no linked venue');expect(html).toContain('Venue conditions are unknown');expect(html).not.toContain('href=');expect((html.match(/Unknown does not mean no/g)??[]).length).toBe(12);
 });
 it('retains expired provenance and separates event observations',()=>{
  const comparison=fixture();comparison.venue!.entries[0]={...comparison.venue!.entries[0],value:'yes',effectiveValue:'unknown',needsReview:true,sourceKind:'organizer_assertion',reviewedAt:'2026-08-01T00:00:00Z',expiresAt:'2026-09-01T00:00:00Z',sourceReference:'Synthetic old plan'};
  comparison.event.entries[0]={...comparison.event.entries[0],value:'no',effectiveValue:'no',sourceKind:'event_observation',sourceReference:'Synthetic event arrangement',reviewedAt:'2026-09-30T00:00:00Z'};
  const html=renderToStaticMarkup(<AccessComparisonResult comparison={comparison}/>);
  expect(html).toContain('review expired');expect(html).toContain('previous value was yes');expect(html).toContain('Event-specific observation');expect(html).toContain('Synthetic old plan');expect(html).toContain('Synthetic event arrangement');
 });
 it('escapes names and source text while linking to the correct private venue',()=>{
  const comparison=fixture();comparison.venue!.placeName='<script>venue</script>';comparison.venue!.entries[0]={...comparison.venue!.entries[0],value:'yes',effectiveValue:'yes',sourceKind:'venue_observation',sourceReference:'<img onerror=alert(1)>'};
  const html=renderToStaticMarkup(<AccessComparisonResult comparison={comparison}/>);
  expect(html).toContain('/workspace/workspace/places/venue/access-info');expect(html).toContain('&lt;script&gt;');expect(html).toContain('&lt;img');expect(html).not.toContain('<script>');expect(html).not.toContain('<img');
 });
});
