import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { EventRoleDTO } from '../domain';
import { EventRoleSetupPanel } from './EventRoleSetupPanel';

const role: EventRoleDTO = { id: 'role-a', eventId: 'event-a', name: 'Performer', description: 'Participant-facing details', capacity: 2, public: true, active: true, createdAt: '', updatedAt: '' };
describe('event role setup', () => {
  it('offers explicit private creation to an authorized open-event owner', () => {
    const html = renderToStaticMarkup(<EventRoleSetupPanel eventId="event-a" roles={[]} allowed onCreated={() => {}} />);
    expect(html).toContain('Add private role');
    expect(html).toContain('Accept public applications');
    expect(html).not.toContain('checked=""');
    expect(html).toContain('role="status"');
  });
  it('keeps role information readable without mutation controls for members or closed events', () => {
    const html = renderToStaticMarkup(<EventRoleSetupPanel eventId="event-a" roles={[role]} allowed={false} onCreated={() => {}} />);
    expect(html).toContain('Performer');
    expect(html).toContain('Public applications');
    expect(html).toContain('2 places');
    expect(html).not.toContain('<form');
  });
  it('does not render another event’s retained role records', () => {
    const html = renderToStaticMarkup(<EventRoleSetupPanel eventId="event-b" roles={[role]} allowed={false} onCreated={() => {}} />);
    expect(html).not.toContain('Performer');
    expect(html).not.toContain('Participant-facing details');
    expect(html).toContain('No participation roles yet.');
  });
});
