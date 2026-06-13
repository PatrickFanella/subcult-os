import { renderToString } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import App from './App';

function renderAt(pathname: string) {
  vi.stubGlobal('window', {
    history: { pushState: () => undefined },
    location: { pathname, search: '' },
  });

  return renderToString(<App />);
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('App routes', () => {
  it('renders the workspace route by default', () => {
    expect(renderAt('/')).toContain('Workspace');
  });

  it('renders the auth route', () => {
    const rendered = renderAt('/login');
    expect(rendered).toContain('Access');
    expect(rendered).toContain('Sign in');
  });

  it('renders the public event route', () => {
    expect(renderAt('/e/night-market')).toContain('Reserve');
  });

  it('renders the event editor route', () => {
    expect(renderAt('/events/new')).toContain('Event editor');
  });

  it('renders the door route', () => {
    expect(renderAt('/door/event-1')).toContain('Door');
  });

  it('renders the ticket route', () => {
    expect(renderAt('/tickets/ticket-123')).toContain('Ticket');
  });
});
