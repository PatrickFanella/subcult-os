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

function makeUseStateImplementation(values: unknown[] = []) {
  return ((initial: unknown) => {
    if (values.length > 0) {
      return [values.shift(), vi.fn()];
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

  it('renders the workspace archive affordance for closed events', () => {
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
      false,
    ]);

    expect(rendered).toContain('Archive ready after closeout');
    expect(rendered).toContain('Open archive');
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

    const rendered = renderWithState('/e/night-market', <PublicEventView slug="night-market" />, [event, '', '', false, false, null, null]);

    expect(rendered).toContain('Buy ticket');
    expect(rendered).toContain('$18.00');
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
