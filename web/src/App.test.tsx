import { renderToString } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { normalizeCurrentWorkspace } from './views/WorkspaceView';

function renderAt(pathname: string) {
  const url = new URL(pathname, 'http://example.test');

  vi.stubGlobal('window', {
    history: { pushState: () => undefined },
    location: { pathname: url.pathname, search: url.search },
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

  it('preserves auth next links', () => {
    const rendered = renderAt('/login?next=/invite/test-token');
    expect(rendered).toContain('?next=%2Finvite%2Ftest-token');
  });

  it('renders the invite route', () => {
    const rendered = renderAt('/invite/test-token');
    expect(rendered).toContain('Invitation');
    expect(rendered).toContain('Accept your invite');
  });

  it('renders the public event route', () => {
    expect(renderAt('/e/night-market')).toContain('Reserve');
  });

  it('renders the event editor route', () => {
    const rendered = renderAt('/events/new');
    expect(rendered).toContain('Event editor');
    expect(rendered).toContain('Events start inside a workspace');
  });

  it('renders the workspace-backed new event flow', () => {
    const rendered = renderAt('/events/new?workspaceId=workspace-1');
    expect(rendered).toContain('Publish checklist');
    expect(rendered).toContain('Fill in details');
  });

  it('renders the door route', () => {
    expect(renderAt('/door/event-1')).toContain('Door');
  });

  it('renders the ticket route', () => {
    expect(renderAt('/tickets/ticket-123')).toContain('Ticket');
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
