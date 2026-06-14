import { AuthView } from './views/AuthView';
import { DiscoverView } from './views/DiscoverView';
import { DoorView } from './views/DoorView';
import { EventEditorView } from './views/EventEditorView';
import { InviteView } from './views/InviteView';
import { PublicEventView } from './views/PublicEventView';
import { TicketView } from './views/TicketView';
import { WorkspaceView } from './views/WorkspaceView';

function getPathname() {
  if (typeof window === 'undefined') {
    return '/';
  }

  return window.location.pathname;
}

function getSegment(pathname: string, index: number) {
  return pathname.split('/')[index] ?? '';
}

export default function App() {
  const pathname = getPathname();

  if (pathname === '/login' || pathname === '/signup' || pathname === '/auth') {
    return <AuthView />;
  }

  if (pathname === '/discover') {
    return <DiscoverView />;
  }

  if (pathname.startsWith('/e/')) {
    return <PublicEventView slug={getSegment(pathname, 2)} />;
  }

  if (pathname.startsWith('/public/events/')) {
    return <PublicEventView slug={getSegment(pathname, 3)} />;
  }

  if (pathname.startsWith('/tickets/')) {
    return <TicketView code={getSegment(pathname, 2)} />;
  }

  if (pathname.startsWith('/door/')) {
    return <DoorView eventId={getSegment(pathname, 2)} />;
  }

  if (pathname.startsWith('/invite/')) {
    return <InviteView token={getSegment(pathname, 2)} />;
  }

  if (pathname.startsWith('/events/')) {
    return <EventEditorView eventId={getSegment(pathname, 2)} />;
  }

  if (pathname === '/workspace' || pathname === '/') {
    return <WorkspaceView />;
  }

  return <WorkspaceView />;
}
