import * as React from 'react';
import { renderToString } from 'react-dom/server';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { EventEditorView } from './views/EventEditorView';
import { PublicEventView } from './views/PublicEventView';
import { TicketView } from './views/TicketView';
import { WorkspaceView, normalizeCurrentWorkspace } from './views/WorkspaceView';

vi.mock('react', async () => {
  const actual = await vi.importActual<typeof import('react')>('react');

  return {
    ...actual,
    useState: vi.fn((initial: unknown) => [typeof initial === 'function' ? (initial as () => unknown)() : initial, vi.fn()]),
  };
});

const useStateMock = vi.mocked(React.useState);
const SKIP = Symbol('skip-state');

function skipStates(count: number) {
  return Array.from({ length: count }, () => SKIP);
}

function makeUseStateImplementation(values: unknown[] = []) {
  return ((initial: unknown) => {
    if (values.length > 0) {
      const next = values.shift();
      if (next === SKIP) {
        return [typeof initial === 'function' ? (initial as () => unknown)() : initial, vi.fn()];
      }

      return [next, vi.fn()];
    }

    return [typeof initial === 'function' ? (initial as () => unknown)() : initial, vi.fn()];
  }) as unknown as typeof React.useState;
}

function renderAt(pathname: string) {
  const url = new URL(pathname, 'http://example.test');

  vi.stubGlobal('window', {
    history: { pushState: () => undefined },
    location: { pathname: url.pathname, search: url.search },
  });

  return renderToString(<App />);
}

function renderWithState(pathname: string, element: React.ReactElement, stateValues: unknown[] = []) {
  const values = [...stateValues];
  const url = new URL(pathname, 'http://example.test');

  vi.stubGlobal('window', {
    history: { pushState: () => undefined },
    location: { pathname: url.pathname, search: url.search },
  });

  useStateMock.mockImplementation(makeUseStateImplementation(values));

  return renderToString(element);
}

beforeEach(() => {
  useStateMock.mockImplementation(makeUseStateImplementation());
});

afterEach(() => {
  useStateMock.mockReset();
  vi.unstubAllGlobals();
});

describe('App routes', () => {
  it('renders the workspace route by default', () => {
    const rendered = renderAt('/');
    expect(rendered).toContain('Operator home');
    expect(rendered).toContain('Run the room from one place');
    expect(rendered).toContain('Public discovery');
  });

  it('renders the workspace route with a selected workspace', () => {
    const rendered = renderAt('/workspace?workspaceId=workspace-1');
    expect(rendered).toContain('Workspace access');
    expect(rendered).toContain('One person can operate multiple Workspaces. Use this switcher to jump between them.');
  });

  it('renders the auth route', () => {
    const rendered = renderAt('/login');
    expect(rendered).toContain('Operator access');
    expect(rendered).toContain('Sign in');
    expect(rendered).toContain('Sign in to resume the workspace');
    expect(rendered).toContain('8+ characters');
  });

  it('renders the signup auth route', () => {
    const rendered = renderAt('/signup');
    expect(rendered).toContain('Join the room');
    expect(rendered).toContain('Create account');
  });

  it('preserves auth next links', () => {
    const rendered = renderAt('/login?next=/invite/test-token');
    expect(rendered).toContain('?next=%2Finvite%2Ftest-token');
    expect(rendered).toContain('Accepting an invitation? Sign in/sign up with the invited email.');
  });

  it('renders the discover loading state', () => {
    const rendered = renderAt('/discover');
    expect(rendered).toContain('Discover events');
    expect(rendered).toContain('Search published events');
    expect(rendered).toContain('Search');
    expect(rendered).toContain('Reset');
    expect(rendered).toContain('Loading published events…');
  });

  it('syncs the discover search query from the url', () => {
    const rendered = renderAt('/discover?q=Market');
    expect(rendered).toContain('Search published events');
    expect(rendered).toContain('value="Market"');
  });

  it('renders discover event cards', () => {
    const rendered = renderWithState('/discover', <App />, [
      [
        {
          id: 'event-1',
          title: 'Night Market',
          startsAt: '2026-06-14T23:00:00.000Z',
          publicDescription: 'Late set with food and music.',
          locationDisplay: 'The Hall',
          workspaceName: 'Signal Collective',
          pricingMode: 'fixed',
          ticketPriceCents: 1800,
          ticketCurrency: 'usd',
          remainingTickets: 12,
          isFull: false,
          applicationsOpen: true,
          status: 'published',
          publicSlug: 'night-market',
          publicUrl: '/e/night-market',
        },
        {
          id: 'event-2',
          title: 'Community Jam',
          startsAt: '2026-06-15T01:00:00.000Z',
          publicDescription: 'Free late-night hang.',
          locationDisplay: 'The Loft',
          workspaceName: 'Signal Collective',
          pricingMode: 'free',
          ticketPriceCents: 0,
          ticketCurrency: 'usd',
          remainingTickets: 0,
          isFull: true,
          applicationsOpen: false,
          status: 'published',
          publicSlug: 'community-jam',
          publicUrl: '/e/community-jam',
        },
      ],
      false,
      null,
    ]);

    expect(rendered).toContain('Discover events');
    expect(rendered).toContain('Night Market');
    expect(rendered).toContain('Hosted by');
    expect(rendered).toContain('Signal Collective');
    expect(rendered).toContain('$18.00');
    expect(rendered).toContain('12 tickets left');
    expect(rendered).toContain('Applications open');
    expect(rendered).toContain('Community Jam');
    expect(rendered).toContain('Free');
    expect(rendered).toContain('Sold out');
    expect(rendered).toContain('The Hall');
    expect(rendered).toContain('View event');
    expect(rendered).toContain('2026');
  });

  it('renders the discover empty state', () => {
    const rendered = renderWithState('/discover', <App />, [[], false, null]);
    expect(rendered).toContain('No published events are discoverable yet.');
  });

  it('renders the discover search empty state', () => {
    const rendered = renderWithState('/discover?q=market', <App />, [[], false, null, 'market']);
    expect(rendered).toContain('No events matched your search.');
    expect(rendered).toContain('Reset');
  });

  it('renders the discover error state', () => {
    const rendered = renderWithState('/discover', <App />, [[], false, 'Network down']);
    expect(rendered).toContain('Could not load published events.');
    expect(rendered).toContain('Network down');
  });

  it('renders the invite route', () => {
    const rendered = renderAt('/invite/test-token');
    expect(rendered).toContain('Invitation');
    expect(rendered).toContain('Accept your invite');
    expect(rendered).toContain('Checking invitation…');
    expect(rendered).toContain('Invite token');
    expect(rendered).toContain('test…oken');
    expect(rendered).toContain('Next steps');
    expect(rendered).toContain('Workspace');
    expect(rendered).toContain('Sign in');
    expect(rendered).toContain('Create account');
  });

  it('renders the public event route', () => {
    const rendered = renderAt('/e/night-market');
    expect(rendered).toContain('Free guest reservation');
    expect(rendered).toContain('No account needed');
    expect(rendered).toContain('Discover more events');
    expect(rendered).toContain('free ticket');
    expect(rendered).toContain('Email required');
  });

  it('renders the event editor route', () => {
    const rendered = renderAt('/events/new');
    expect(rendered).toContain('Event editor');
    expect(rendered).toContain('Events start inside a workspace');
  });

  it('renders the event editor pricing copy', () => {
    const rendered = renderWithState('/events/new?workspaceId=workspace-1', <EventEditorView eventId="new" />);
    expect(rendered).toContain('Pricing');
    expect(rendered).toContain('Free reservation');
    expect(rendered).toContain('Fixed paid ticket');
    expect(rendered).toContain('USD only');
  });

  it('renders the workspace archive section', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      staffingOpenCount: 1,
      staffingAssignedCount: 0,
      staffingCompletedCount: 0,
      staffingCancelledCount: 0,
      status: 'end_of_night',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/workspace?workspaceId=workspace-1', <WorkspaceView />, [
      {
        id: 'person-1',
        email: 'owner@example.com',
        displayName: 'Owner',
        workspaces: [{ id: 'workspace-1', name: 'Main Room', role: 'owner' }],
      },
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [event],
      [
        {
          id: 'archive-1',
          eventId: event.id,
          title: 'Night Market',
          startsAt: event.startsAt,
          locationDisplay: event.locationDisplay,
          noteCount: 1,
          reportId: 'report-1',
          settlementId: 'settlement-1',
          createdAt: '2026-06-14T03:00:00.000Z',
          updatedAt: '2026-06-14T03:10:00.000Z',
        },
      ],
      false,
    ]);

    expect(rendered).toContain('Workspace archive');
    expect(rendered).toContain('Use lessons to seed the next draft.');
    expect(rendered).toContain('Search archives');
    expect(rendered).toContain('Search titles, locations, or notes');
    expect(rendered).toContain('Search');
    expect(rendered).toContain('Reset');
    expect(rendered).toContain('Night Market');
    expect(rendered).toContain('1 note');
    expect(rendered).toContain('Open archive');
    expect(rendered).toContain('Seed next draft');
    expect(rendered).toContain('No seeded draft yet');
    expect(rendered).toContain('Unresolved staffing remains before closeout.');
  });

  it('renders empty archive search copy', () => {
    const rendered = renderWithState('/workspace?workspaceId=workspace-1&q=doors', <WorkspaceView />, [
      {
        id: 'person-1',
        email: 'owner@example.com',
        displayName: 'Owner',
        workspaces: [{ id: 'workspace-1', name: 'Main Room', role: 'owner' }],
      },
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [],
      [],
      false,
    ]);

    expect(rendered).toContain('No archives matched your search');
    expect(rendered).toContain('Try a different search or reset the filter to show every private archive.');
  });

  it('shows seeded draft links in the archive section', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      staffingOpenCount: 0,
      staffingAssignedCount: 0,
      staffingCompletedCount: 1,
      staffingCancelledCount: 0,
      status: 'end_of_night',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/workspace?workspaceId=workspace-1', <WorkspaceView />, [
      {
        id: 'person-1',
        email: 'owner@example.com',
        displayName: 'Owner',
        workspaces: [{ id: 'workspace-1', name: 'Main Room', role: 'owner' }],
      },
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
      [event],
      [
        {
          id: 'archive-1',
          eventId: event.id,
          title: 'Night Market',
          startsAt: event.startsAt,
          locationDisplay: event.locationDisplay,
          noteCount: 1,
          reportId: 'report-1',
          settlementId: 'settlement-1',
          seededEventId: 'event-2',
          createdAt: '2026-06-14T03:00:00.000Z',
          updatedAt: '2026-06-14T03:10:00.000Z',
        },
      ],
      false,
    ]);

    expect(rendered).toContain('Open seeded draft');
    expect(rendered).not.toContain('Seed next draft');
    expect(rendered).toContain('All staffing complete.');
  });

  it('renders contacts and commitments panels for owners', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      staffingOpenCount: 0,
      staffingAssignedCount: 0,
      staffingCompletedCount: 0,
      staffingCancelledCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/workspace?workspaceId=workspace-1', <WorkspaceView />, [
      {
        id: 'person-1',
        email: 'owner@example.com',
        displayName: 'Owner',
        workspaces: [{ id: 'workspace-1', name: 'Main Room', role: 'owner' }],
      },
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [event],
      [],
      false,
      ...skipStates(11),
      [
        {
          id: 'contact-1',
          workspaceId: 'workspace-1',
          displayName: 'Mira Door',
          email: 'mira@example.com',
          phone: '+15555550123',
          notes: 'Prefers late load-in',
          tags: ['door', 'trusted'],
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:10:00.000Z',
        },
      ],
      ...skipStates(3),
      [
        {
          id: 'commitment-1',
          workspaceId: 'workspace-1',
          eventId: event.id,
          title: 'Confirm projector',
          description: 'Keep private',
          dueAt: '2026-06-14T18:00:00.000Z',
          status: 'open',
          ownerPersonId: null,
          createdByPersonId: 'person-1',
          completedAt: null,
          completedByPersonId: null,
          createdAt: '2026-06-13T20:15:00.000Z',
          updatedAt: '2026-06-13T20:15:00.000Z',
        },
        {
          id: 'commitment-2',
          workspaceId: 'workspace-1',
          eventId: null,
          contactId: null,
          title: 'Follow up with vendor',
          description: '',
          dueAt: null,
          status: 'done',
          ownerPersonId: 'member-1',
          createdByPersonId: 'person-1',
          completedAt: '2026-06-13T21:00:00.000Z',
          completedByPersonId: 'person-1',
          createdAt: '2026-06-13T20:20:00.000Z',
          updatedAt: '2026-06-13T21:00:00.000Z',
        },
      ],
    ]);

    expect(rendered).toContain('Contacts');
    expect(rendered).toContain('Mira Door');
    expect(rendered).toContain('door');
    expect(rendered).toContain('trusted');
    expect(rendered).toContain('Private note:');
    expect(rendered).toContain('Commitments');
    expect(rendered).toContain('Confirm projector');
    expect(rendered).toContain('Follow up with vendor');
    expect(rendered).toContain('Mark done');
    expect(rendered).toContain('Add contact');
    expect(rendered).toContain('Add commitment');
  });

  it('hides owner-only workspace controls for members', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      staffingOpenCount: 0,
      staffingAssignedCount: 0,
      staffingCompletedCount: 0,
      staffingCancelledCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/workspace?workspaceId=workspace-1', <WorkspaceView />, [
      {
        id: 'person-1',
        email: 'member@example.com',
        displayName: 'Member',
        workspaces: [{ id: 'workspace-1', name: 'Main Room', role: 'member' }],
      },
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'member',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [event],
      [],
      false,
      ...skipStates(11),
      [
        {
          id: 'contact-1',
          workspaceId: 'workspace-1',
          displayName: 'Mira Door',
          email: 'mira@example.com',
          phone: '+15555550123',
          notes: 'Prefers late load-in',
          tags: ['door', 'trusted'],
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:10:00.000Z',
        },
      ],
      ...skipStates(3),
      [
        {
          id: 'commitment-1',
          workspaceId: 'workspace-1',
          eventId: event.id,
          title: 'Confirm projector',
          description: 'Keep private',
          dueAt: '2026-06-14T18:00:00.000Z',
          status: 'open',
          ownerPersonId: null,
          createdByPersonId: 'person-1',
          completedAt: null,
          completedByPersonId: null,
          createdAt: '2026-06-13T20:15:00.000Z',
          updatedAt: '2026-06-13T20:15:00.000Z',
        },
      ],
    ]);

    expect(rendered).toContain('Contacts');
    expect(rendered).toContain('Commitments');
    expect(rendered).toContain('Mira Door');
    expect(rendered).toContain('Confirm projector');
    expect(rendered).not.toContain('Add contact');
    expect(rendered).not.toContain('Add commitment');
    expect(rendered).not.toContain('Edit');
    expect(rendered).not.toContain('Mark done');
  });

  it('shows no staffing copy on event cards without staffing items', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      staffingOpenCount: 0,
      staffingAssignedCount: 0,
      staffingCompletedCount: 0,
      staffingCancelledCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/workspace?workspaceId=workspace-1', <WorkspaceView />, [
      {
        id: 'person-1',
        email: 'owner@example.com',
        displayName: 'Owner',
        workspaces: [{ id: 'workspace-1', name: 'Main Room', role: 'owner' }],
      },
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
      [event],
      [],
      false,
    ]);

    expect(rendered).toContain('No staffing items yet.');
  });

  it('locks event pricing after tickets exist', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1500,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [event, null, null, '', false, false, {
      title: event.title,
      startsAt: '2026-06-13T23:00',
      publicDescription: event.publicDescription,
      locationDisplay: event.locationDisplay,
      ticketAllocation: '100',
      pricingMode: 'fixed',
      ticketPriceDollars: '15.00',
    }, {
      title: event.title,
      startsAt: '2026-06-13T23:00',
      publicDescription: event.publicDescription,
      locationDisplay: event.locationDisplay,
      ticketAllocation: '100',
      pricingMode: 'fixed',
      ticketPriceDollars: '15.00',
    }, false, false, false, null, null]);

    expect(rendered).toContain('Pricing is locked once tickets exist or after the event closes.');
    expect(rendered).toContain('$15.00 USD');
  });

  it('renders the private applications review panel for owners', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      null,
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
      [
        {
          id: 'role-1',
          eventId: event.id,
          name: 'Performer',
          description: 'Play a 20-minute set.',
          capacity: 3,
          public: true,
          active: true,
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:00:00.000Z',
        },
      ],
      [
        {
          id: 'application-1',
          eventId: event.id,
          roleId: 'role-1',
          applicantName: 'Alex',
          applicantEmail: 'alex@example.com',
          message: 'Bring a keyboard.',
          status: 'submitted',
          createdAt: '2026-06-13T21:00:00.000Z',
          updatedAt: '2026-06-13T21:00:00.000Z',
        },
      ],
      {},
      null,
    ]);

    expect(rendered).toContain('Private review');
    expect(rendered).toContain('Performer');
    expect(rendered).toContain('Alex');
    expect(rendered).toContain('Bring a keyboard.');
    expect(rendered).toContain('Update status');
    expect(rendered).toContain('Submitted');
  });

  it('renders the participant roster panel for accepted applications', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      null,
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
      [
        {
          id: 'role-1',
          eventId: event.id,
          name: 'Performer',
          description: 'Play a 20-minute set.',
          capacity: 3,
          public: true,
          active: true,
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:00:00.000Z',
        },
      ],
      null,
      {},
      null,
      [
        {
          applicationId: 'application-1',
          roleId: 'role-1',
          roleName: 'Performer',
          applicantName: 'Alex',
          applicantEmail: 'alex@example.com',
          status: 'confirmed',
          updatedAt: '2026-06-13T22:00:00.000Z',
        },
      ],
      [
        {
          id: 'staffing-1',
          eventId: event.id,
          title: 'Load in',
          kind: 'task',
          notes: 'Bring the banner.',
          startsAt: '2026-06-13T21:00:00.000Z',
          endsAt: '2026-06-13T21:30:00.000Z',
          assignedPersonId: 'member-1',
          assigneeName: 'Morgan',
          status: 'assigned',
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:10:00.000Z',
        },
        {
          id: 'staffing-2',
          eventId: event.id,
          title: 'Door shift',
          kind: 'shift',
          notes: 'Front desk coverage.',
          startsAt: '2026-06-13T22:00:00.000Z',
          endsAt: '2026-06-13T23:00:00.000Z',
          assignedApplicationId: 'application-1',
          assigneeName: 'Alex',
          status: 'open',
          createdAt: '2026-06-13T20:15:00.000Z',
          updatedAt: '2026-06-13T20:15:00.000Z',
        },
        {
          id: 'staffing-3',
          eventId: event.id,
          title: 'Sound check',
          kind: 'task',
          notes: '',
          startsAt: null,
          endsAt: null,
          assigneeName: 'Morgan',
          status: 'completed',
          createdAt: '2026-06-13T20:20:00.000Z',
          updatedAt: '2026-06-13T22:30:00.000Z',
          completedAt: '2026-06-13T22:30:00.000Z',
          completedByPersonId: 'person-1',
        },
        {
          id: 'staffing-4',
          eventId: event.id,
          title: 'Decor setup',
          kind: 'shift',
          notes: 'Unused if vendor arrives early.',
          startsAt: null,
          endsAt: null,
          status: 'cancelled',
          createdAt: '2026-06-13T20:25:00.000Z',
          updatedAt: '2026-06-13T20:40:00.000Z',
        },
      ],
      false,
      {
        title: '',
        kind: 'task',
        notes: '',
        startsAt: '',
        endsAt: '',
      },
      null,
    ]);

    expect(rendered).toContain('Participant roster');
    expect(rendered).toContain('Accepted participants');
    expect(rendered).toContain('Performer');
    expect(rendered).toContain('Alex');
    expect(rendered).toContain('confirmed');
    expect(rendered).not.toContain('Bring a keyboard.');
    expect(rendered).toContain('Staffing board');
    expect(rendered).toContain('Add staffing item');
    expect(rendered).toContain('Assign to');
    expect(rendered).toContain('Clear assignee');
    expect(rendered).toContain('Mark completed');
    expect(rendered).toContain('Mark cancelled');
  });

  it('hides staffing mutation controls for members', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      null,
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'member',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [
        {
          id: 'role-1',
          eventId: event.id,
          name: 'Performer',
          description: 'Play a 20-minute set.',
          capacity: 3,
          public: true,
          active: true,
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:00:00.000Z',
        },
      ],
      null,
      {},
      null,
      [
        {
          applicationId: 'application-1',
          roleId: 'role-1',
          roleName: 'Performer',
          applicantName: 'Alex',
          applicantEmail: 'alex@example.com',
          status: 'confirmed',
          updatedAt: '2026-06-13T22:00:00.000Z',
        },
      ],
      [
        {
          id: 'staffing-1',
          eventId: event.id,
          title: 'Load in',
          kind: 'task',
          notes: 'Bring the banner.',
          startsAt: '2026-06-13T21:00:00.000Z',
          endsAt: '2026-06-13T21:30:00.000Z',
          assignedPersonId: 'member-1',
          assigneeName: 'Morgan',
          status: 'assigned',
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:10:00.000Z',
        },
      ],
      false,
      {
        title: '',
        kind: 'task',
        notes: '',
        startsAt: '',
        endsAt: '',
      },
      null,
    ]);

    expect(rendered).toContain('Staffing board');
    expect(rendered).toContain('Open');
    expect(rendered).toContain('Assigned');
    expect(rendered).toContain('Completed');
    expect(rendered).toContain('Cancelled');
    expect(rendered).not.toContain('Add staffing item');
    expect(rendered).not.toContain('Assign to');
    expect(rendered).not.toContain('Clear assignee');
    expect(rendered).not.toContain('Mark completed');
    expect(rendered).not.toContain('Mark cancelled');
  });

  it('renders event commitments for owners', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      null,
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [
        {
          id: 'role-1',
          eventId: event.id,
          name: 'Performer',
          description: 'Play a 20-minute set.',
          capacity: 3,
          public: true,
          active: true,
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:00:00.000Z',
        },
      ],
      null,
      {},
      null,
      null,
      null,
      false,
      {
        title: '',
        kind: 'task',
        notes: '',
        startsAt: '',
        endsAt: '',
      },
      null,
      ...skipStates(2),
      [
        {
          id: 'commitment-1',
          workspaceId: 'workspace-1',
          eventId: event.id,
          title: 'Confirm projector',
          description: 'Keep private',
          dueAt: '2026-06-14T18:00:00.000Z',
          status: 'open',
          ownerPersonId: 'member-1',
          createdByPersonId: 'person-1',
          completedAt: null,
          completedByPersonId: null,
          createdAt: '2026-06-13T20:15:00.000Z',
          updatedAt: '2026-06-13T20:15:00.000Z',
        },
      ],
    ]);

    expect(rendered).toContain('Event commitments');
    expect(rendered).toContain('Confirm projector');
    expect(rendered).toContain('Owner assigned');
    expect(rendered).toContain('Add commitment');
    expect(rendered).toContain('Mark done');
    expect(rendered).toContain('Reopen');
    expect(rendered).toContain('Cancel');
  });

  it('hides event commitment controls for members', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      null,
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'member',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      [
        {
          id: 'role-1',
          eventId: event.id,
          name: 'Performer',
          description: 'Play a 20-minute set.',
          capacity: 3,
          public: true,
          active: true,
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:00:00.000Z',
        },
      ],
      null,
      {},
      null,
      null,
      null,
      false,
      {
        title: '',
        kind: 'task',
        notes: '',
        startsAt: '',
        endsAt: '',
      },
      null,
      ...skipStates(2),
      [
        {
          id: 'commitment-1',
          workspaceId: 'workspace-1',
          eventId: event.id,
          title: 'Confirm projector',
          description: 'Keep private',
          dueAt: '2026-06-14T18:00:00.000Z',
          status: 'open',
          ownerPersonId: 'member-1',
          createdByPersonId: 'person-1',
          completedAt: null,
          completedByPersonId: null,
          createdAt: '2026-06-13T20:15:00.000Z',
          updatedAt: '2026-06-13T20:15:00.000Z',
        },
      ],
    ]);

    expect(rendered).toContain('Event commitments');
    expect(rendered).toContain('Confirm projector');
    expect(rendered).not.toContain('Add commitment');
    expect(rendered).not.toContain('Mark done');
    expect(rendered).not.toContain('Reopen');
  });

  it('renders notification activity for member views without private body text', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };
    const archiveNoteBody = 'Archive note body should stay private';
    const settlementInternal = 'Settlement internals should stay private';
    const applicationMessage = 'Role application message should stay private';
    const staffingNotes = 'Staffing notes should stay private';

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      {
        id: 'archive-1',
        eventId: event.id,
        reportId: 'report-1',
        settlementId: 'settlement-1',
        status: 'private',
        noteCount: 1,
        participants: [],
        staffingItems: [],
        notes: [
          {
            id: 'note-1',
            archiveId: 'archive-1',
            body: archiveNoteBody,
            createdByPersonId: 'owner-1',
            createdAt: '2026-06-13T22:25:00.000Z',
          },
        ],
        createdAt: '2026-06-13T22:00:00.000Z',
        updatedAt: '2026-06-13T22:25:00.000Z',
      },
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      {
        id: 'settlement-1',
        eventId: event.id,
        currency: 'usd',
        grossPaidRevenueCents: 3000,
        paidTicketCount: 12,
        pendingTicketCount: 3,
        cancelledTicketCount: 4,
        freeTicketCount: 5,
        reservedCount: 24,
        adjustmentTotalCents: -250,
        netTotalCents: 2750,
        status: 'finalized',
        generatedAt: '2026-06-13T22:30:00.000Z',
        finalizedAt: '2026-06-13T22:45:00.000Z',
        finalizedByPersonId: 'owner-1',
        adjustments: [
          {
            id: 'adjustment-1',
            settlementId: 'settlement-1',
            amountCents: -250,
            label: 'Cash drawer',
            reason: settlementInternal,
            createdByPersonId: 'member-1',
            createdAt: '2026-06-13T22:40:00.000Z',
          },
        ],
      },
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'member',
        members: [{ id: 'member-1', email: 'morgan@example.com', displayName: 'Morgan', role: 'member' }],
        invitations: [],
      },
      null,
      null,
      {},
      null,
      null,
      null,
      false,
      {
        title: '',
        kind: 'task',
        notes: '',
        startsAt: '',
        endsAt: '',
      },
      null,
      [
        {
          id: 'notification-1',
          eventId: event.id,
          recipientEmail: 'alex@example.com',
          notificationType: 'role_application.accepted',
          relatedType: 'role_application',
          relatedId: 'application-1',
          subject: 'Application accepted for Performer',
          preview: 'Application accepted for Performer',
          status: 'queued',
          createdAt: '2026-06-13T22:30:00.000Z',
        },
        {
          id: 'notification-2',
          eventId: event.id,
          recipientEmail: 'morgan@example.com',
          notificationType: 'staffing.assignment',
          relatedType: 'staffing_item',
          relatedId: 'staffing-1',
          subject: 'Assigned to Load in',
          preview: 'Assigned to Load in',
          status: 'queued',
          createdAt: '2026-06-13T22:35:00.000Z',
        },
      ],
      0,
    ]);

    const notificationStart = rendered.indexOf('Notification activity');
    const archiveStart = rendered.indexOf('Private archive');
    expect(notificationStart).toBeGreaterThanOrEqual(0);
    expect(archiveStart).toBeGreaterThan(notificationStart);

    const notificationSection = rendered.slice(notificationStart, archiveStart);

    expect(notificationSection).toContain('Notification activity');
    expect(notificationSection).toContain('2 queued notifications');
    expect(notificationSection).toContain('alex@example.com');
    expect(notificationSection).toContain('role_application.accepted');
    expect(notificationSection).toContain('Application accepted for Performer');
    expect(notificationSection).toContain('staffing.assignment');
    expect(notificationSection).toContain('morgan@example.com');
    expect(notificationSection).not.toContain(applicationMessage);
    expect(notificationSection).not.toContain(staffingNotes);
    expect(notificationSection).not.toContain(archiveNoteBody);
    expect(notificationSection).not.toContain(settlementInternal);
  });

  it('renders the end-of-night settlement summary report panel', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      status: 'end_of_night',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const report = {
      id: 'report-1',
      eventId: event.id,
      title: 'End of night report',
      startsAt: event.startsAt,
      publicUrl: event.publicUrl,
      ticketAllocation: event.ticketAllocation,
      ticketsReserved: 26,
      ticketsCheckedIn: 20,
      noShows: 6,
      settlementSummary: {
        currency: 'usd',
        grossPaidRevenueCents: 3000,
        paidTicketCount: 12,
        pendingTicketCount: 3,
        cancelledTicketCount: 4,
        freeTicketCount: 5,
        reservedCount: 24,
      },
      generatedAt: '2026-06-14T03:00:00.000Z',
      generatedByMemberEmail: 'operator@example.com',
    };

    const settlement = {
      id: 'settlement-1',
      eventId: event.id,
      currency: 'usd',
      grossPaidRevenueCents: 3000,
      paidTicketCount: 12,
      pendingTicketCount: 3,
      cancelledTicketCount: 4,
      freeTicketCount: 5,
      reservedCount: 24,
      adjustmentTotalCents: -250,
      netTotalCents: 2750,
      status: 'finalized',
      generatedAt: '2026-06-14T03:00:00.000Z',
      finalizedAt: '2026-06-14T03:30:00.000Z',
      finalizedByPersonId: 'owner-1',
      adjustments: [
        {
          id: 'adjustment-1',
          settlementId: 'settlement-1',
          amountCents: -250,
          label: 'Cash drawer',
          reason: 'Counted short at closeout',
          createdByPersonId: 'member-1',
          createdAt: '2026-06-14T03:15:00.000Z',
        },
      ],
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      report,
      {
        id: 'archive-1',
        eventId: event.id,
        reportId: 'report-1',
        settlementId: 'settlement-1',
        status: 'private',
        noteCount: 1,
        participants: [],
        staffingItems: [
          {
            id: 'staffing-1',
            archiveId: 'archive-1',
            sourceStaffingItemId: 'source-staffing-1',
            title: 'Door shift',
            kind: 'shift',
            status: 'open',
            assigneeName: 'Morgan',
            createdAt: '2026-06-14T03:05:00.000Z',
          },
        ],
        notes: [
          {
            id: 'note-1',
            archiveId: 'archive-1',
            body: 'Move doors earlier.',
            createdByPersonId: 'person-1',
            createdAt: '2026-06-14T03:10:00.000Z',
          },
        ],
        createdAt: '2026-06-14T03:00:00.000Z',
        updatedAt: '2026-06-14T03:10:00.000Z',
      },
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      settlement,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
    ]);

    expect(rendered).toContain('Settlement summary');
    expect(rendered).toContain('$30.00 USD');
    expect(rendered).toContain('Paid tickets');
    expect(rendered).toContain('12');
    expect(rendered).toContain('Pending tickets');
    expect(rendered).toContain('3');
    expect(rendered).toContain('Cancelled tickets');
    expect(rendered).toContain('4');
    expect(rendered).toContain('Free tickets');
    expect(rendered).toContain('5');
    expect(rendered).toContain('Reserved total');
    expect(rendered).toContain('24');
    expect(rendered).toContain('Settlement closeout');
    expect(rendered).toContain('Gross revenue');
    expect(rendered).toContain('$30.00 USD');
    expect(rendered).toContain('Adjustment total');
    expect(rendered).toContain('-$2.50 USD');
    expect(rendered).toContain('Net total');
    expect(rendered).toContain('$27.50 USD');
    expect(rendered).toContain('finalized (locked)');
    expect(rendered).toContain('Adjustments are locked after settlement finalization.');
    expect(rendered).not.toContain('Finalize settlement');
    expect(rendered).toContain('Cash drawer');
    expect(rendered).toContain('Counted short at closeout');
    expect(rendered).toContain('Created');
    expect(rendered).toContain('Updated');
    expect(rendered).toContain('Report');
    expect(rendered).toContain('report-1');
    expect(rendered).toContain('Settlement');
    expect(rendered).toContain('settlement-1');
    expect(rendered).toContain('Staffing memory');
    expect(rendered).toContain('Door shift');
    expect(rendered).toContain('Unresolved staffing should inform next draft planning.');
    expect(rendered).toContain('Back to workspace archive');
    expect(rendered).toContain('Private notes stay in the archive. The next draft starts clean.');
    expect(rendered).toContain('Use these notes while planning the next event.');
    expect(rendered).toContain('Seed next draft');
  });

  it('shows seeded draft links in the event archive panel', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      status: 'end_of_night',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      {
        id: 'archive-1',
        eventId: event.id,
        reportId: 'report-1',
        settlementId: 'settlement-1',
        seededEventId: 'event-2',
        status: 'private',
        noteCount: 0,
        participants: [],
        staffingItems: [],
        notes: [],
        createdAt: '2026-06-14T03:00:00.000Z',
        updatedAt: '2026-06-14T03:00:00.000Z',
      },
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
    ]);

    expect(rendered).toContain('Open seeded draft');
    expect(rendered).not.toContain('Seed next draft');
    expect(rendered).toContain('Capture one lesson before seeding the next draft.');
    expect(rendered).toContain('/workspace?workspaceId=workspace-1');
  });

  it('hides archive actions for non-owners', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      status: 'end_of_night',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      {
        id: 'archive-1',
        eventId: event.id,
        reportId: 'report-1',
        settlementId: 'settlement-1',
        status: 'private',
        noteCount: 0,
        participants: [],
        staffingItems: [],
        notes: [],
        createdAt: '2026-06-14T03:00:00.000Z',
        updatedAt: '2026-06-14T03:00:00.000Z',
      },
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'member',
        members: [],
        invitations: [],
      },
    ]);

    expect(rendered).toContain('Private archive');
    expect(rendered).not.toContain('Seed next draft');
    expect(rendered).not.toContain('Add lesson');
  });

  it('renders participant memory in the event archive panel', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 26,
      checkedInCount: 20,
      status: 'end_of_night',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
    };

    const rendered = renderWithState('/events/event-1?workspaceId=workspace-1', <EventEditorView eventId="event-1" />, [
      event,
      null,
      {
        id: 'archive-1',
        eventId: event.id,
        reportId: 'report-1',
        settlementId: 'settlement-1',
        status: 'private',
        noteCount: 0,
        participants: [
          {
            id: 'archive-participant-1',
            archiveId: 'archive-1',
            sourceApplicationId: 'application-1',
            roleName: 'Performer',
            participantName: 'Alex',
            status: 'accepted',
            createdAt: '2026-06-14T03:05:00.000Z',
          },
          {
            id: 'archive-participant-2',
            archiveId: 'archive-1',
            sourceApplicationId: 'application-2',
            roleName: 'Performer',
            participantName: 'Blair',
            status: 'confirmed',
            createdAt: '2026-06-14T03:06:00.000Z',
          },
        ],
        staffingItems: [],
        notes: [],
        createdAt: '2026-06-14T03:00:00.000Z',
        updatedAt: '2026-06-14T03:10:00.000Z',
      },
      '',
      false,
      false,
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      {
        title: event.title,
        startsAt: '2026-06-13T23:00',
        publicDescription: event.publicDescription,
        locationDisplay: event.locationDisplay,
        ticketAllocation: '100',
        pricingMode: 'fixed',
        ticketPriceDollars: '18.00',
      },
      false,
      false,
      false,
      null,
      null,
      null,
      {
        amountDollars: '',
        label: '',
        reason: '',
      },
      false,
      false,
      {
        id: 'workspace-1',
        name: 'Main Room',
        role: 'owner',
        members: [],
        invitations: [],
      },
    ]);

    expect(rendered).toContain('Participant memory');
    expect(rendered).toContain('Alex');
    expect(rendered).toContain('Blair');
    expect(rendered).toContain('accepted');
    expect(rendered).toContain('confirmed');
    expect(rendered).not.toContain('applicantEmail');
  });

  it('renders the public paid ticket CTA', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
      remainingTickets: 88,
      isFull: false,
    };

    const rendered = renderWithState('/e/night-market', <PublicEventView slug="night-market" />, [event, '', '', false, false, null, null, [], {}]);

    expect(rendered).toContain('Buy ticket');
    expect(rendered).toContain('$18.00');
    expect(rendered).toContain('Stripe Checkout');
  });

  it('renders public role application forms alongside ticket flow', () => {
    const event = {
      id: 'event-1',
      workspaceId: 'workspace-1',
      title: 'Night Market',
      startsAt: '2026-06-13T23:00:00.000Z',
      publicDescription: 'A late set.',
      locationDisplay: 'The Hall',
      ticketAllocation: 100,
      pricingMode: 'fixed',
      ticketPriceCents: 1800,
      ticketCurrency: 'usd',
      reservedCount: 12,
      checkedInCount: 0,
      status: 'published',
      publicSlug: 'night-market',
      publicUrl: '/e/night-market',
      remainingTickets: 88,
      isFull: false,
    };

    const rendered = renderWithState('/e/night-market', <PublicEventView slug="night-market" />, [
      event,
      '',
      '',
      false,
      false,
      null,
      null,
      [
        {
          id: 'role-1',
          eventId: event.id,
          name: 'Performer',
          description: 'Play a 20-minute set.',
          capacity: 3,
          public: true,
          active: true,
          createdAt: '2026-06-13T20:00:00.000Z',
          updatedAt: '2026-06-13T20:00:00.000Z',
        },
      ],
      {},
    ]);

    expect(rendered).toContain('Apply to participate');
    expect(rendered).toContain('Performer');
    expect(rendered).toContain('Applicant name');
    expect(rendered).toContain('Applicant email');
    expect(rendered).toContain('Message');
    expect(rendered).toContain('Submit application');
    expect(rendered).toContain('Buy ticket');
    expect(rendered).toContain('Stripe Checkout');
  });

  it('renders the workspace-backed new event flow', () => {
    const rendered = renderAt('/events/new?workspaceId=workspace-1');
    expect(rendered).toContain('Publish checklist');
    expect(rendered).toContain('Fill in details');
  });

  it('renders the door route', () => {
    const rendered = renderAt('/door/event-1');
    expect(rendered).toContain('Mobile check-in');
    expect(rendered).toContain('Exact code works');
    expect(rendered).toContain('Reset');
  });

  it('renders the ticket route', () => {
    const rendered = renderAt('/tickets/ticket-123');
    expect(rendered).toContain('Show this at the door');
    expect(rendered).toContain('Your reservation lives here');
  });

  it.each([
    [
      'free',
      {
        id: 'ticket-123',
        eventId: 'event-1',
        email: 'guest@example.com',
        displayName: 'Guest',
        code: 'ABCD1234',
        status: 'reserved',
        paymentStatus: 'free',
        amountCents: 0,
        currency: 'usd',
        checkedInAt: null,
      },
      'Free ticket',
      'No payment needed',
    ],
    [
      'paid',
      {
        id: 'ticket-123',
        eventId: 'event-1',
        email: 'guest@example.com',
        displayName: 'Guest',
        code: 'ABCD1234',
        status: 'reserved',
        paymentStatus: 'paid',
        amountCents: 1800,
        currency: 'usd',
        checkedInAt: null,
      },
      'Paid ticket',
      '$18.00',
    ],
    [
      'pending',
      {
        id: 'ticket-123',
        eventId: 'event-1',
        email: 'guest@example.com',
        displayName: 'Guest',
        code: 'ABCD1234',
        status: 'reserved',
        paymentStatus: 'pending',
        amountCents: 1800,
        currency: 'usd',
        checkedInAt: null,
      },
      'Payment pending',
      'Checkout may still be processing',
    ],
    [
      'cancelled',
      {
        id: 'ticket-123',
        eventId: 'event-1',
        email: 'guest@example.com',
        displayName: 'Guest',
        code: 'ABCD1234',
        status: 'reserved',
        paymentStatus: 'cancelled',
        amountCents: 1800,
        currency: 'usd',
        checkedInAt: null,
      },
      'Payment cancelled',
      'Finish checkout to activate this ticket',
    ],
  ])('renders the ticket payment %s banner', (_label, ticket, banner, summary) => {
    const rendered = renderWithState('/tickets/ticket-123', <TicketView code="ticket-123" />, [ticket, false, null]);

    expect(rendered).toContain(banner);
    expect(rendered).toContain(summary);
  });
});

describe('workspace response normalization', () => {
  it('treats null collection fields as empty arrays', () => {
    const workspace = normalizeCurrentWorkspace({
      id: 'workspace-1',
      name: 'Signal Collective',
      role: 'owner',
      members: null,
      invitations: null,
    });

    expect(workspace.members).toEqual([]);
    expect(workspace.invitations).toEqual([]);
  });
});
