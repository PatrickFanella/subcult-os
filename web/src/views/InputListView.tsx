import { TechSheetsPanel } from '../components/TechSheetsPanel';

export function InputListView({ eventId }: { eventId: string }) {
  return <main className="min-h-screen px-4 py-6 text-fg-primary sm:px-6"><div className="mx-auto max-w-6xl space-y-4"><a className="underline underline-offset-4" href="/workspace">Back to workspace</a><TechSheetsPanel key={eventId} eventId={eventId} /></div></main>;
}
