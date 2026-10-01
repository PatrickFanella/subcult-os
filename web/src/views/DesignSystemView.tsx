import { useState } from 'react';
import { Button } from '../ui/Button';
import { Notice } from '../ui/Notice';
import { publicCardClass, publicEyebrowClass, publicPageInnerClass, publicPageShellClass, publicStatusPillClass } from '../modules/publicUi/publicUi';

export function DesignSystemView() {
  const [saved, setSaved] = useState(false);
  return (
    <main className={publicPageShellClass}>
      <div className={`${publicPageInnerClass} max-w-6xl`}>
        <header className="flex flex-wrap items-center justify-between gap-4 border-b border-stroke-subtle pb-5">
          <a href="/discover" className="text-lg font-black tracking-tight">subcult</a>
          <span className={publicEyebrowClass}>Design system · Terminal</span>
        </header>
        <section className="grid items-end gap-6 py-6 md:grid-cols-[2fr_1fr]">
          <div><p className={publicEyebrowClass}>Built around the room</p><h1 className="heading-1 mt-4 max-w-3xl">The artwork speaks.<br />The interface works.</h1></div>
          <p className="body-copy max-w-sm">Sharp frames. Purple actions. Monospace type. A shared foundation for finding an event, joining the crew, and running the door.</p>
        </section>
        <section className="grid gap-6 md:grid-cols-2" aria-label="Event and controls">
          <article className="overflow-hidden rounded-hero bg-surface-immersive text-fg-on-immersive">
            <img className="h-64 w-full object-cover" src="https://images.unsplash.com/photo-1459749411175-04bf5292ceea?auto=format&fit=crop&w=1080&q=80" alt="Stage lights above a crowd at a concert" />
            <div className="space-y-4 p-6"><p className="text-xs font-bold uppercase tracking-widest">Friday · Doors at 8 PM</p><h2 className="text-4xl font-extrabold tracking-tight">A night together</h2><p className="text-sm">Example event · Main room</p><a href="/discover" className="inline-flex min-h-touch items-center rounded-control border border-stroke-strong bg-surface-panel px-5 font-bold text-fg-primary">Find your next event</a></div>
          </article>
          <section className={publicCardClass} aria-labelledby="controls-title">
            <p className={publicEyebrowClass}>Controls</p><h2 id="controls-title" className="heading-2 mt-3">Clear next steps</h2>
            <div className="mt-6 flex flex-wrap gap-3"><Button onClick={() => setSaved(!saved)}>{saved ? 'Saved' : 'Save event'}</Button><Button variant="secondary" onClick={() => setSaved(false)}>Reset</Button><Button variant="secondary" onClick={() => { window.location.href = '/discover'; }}>Discover</Button></div>
            <div className="mt-3 flex flex-wrap gap-3"><Button disabled>Unavailable</Button><Button busy>Saving…</Button></div>
            <label className="mt-6 block space-y-2 text-sm font-bold" htmlFor="sample-name"><span>Display name</span><input id="sample-name" className="field" placeholder="How should we credit you?" autoComplete="off" /></label>
            <label className="mt-4 block space-y-2 text-sm font-bold" htmlFor="sample-disabled"><span>Read-only example</span><input id="sample-disabled" className="field" value="Main room" disabled /></label>
            <p className="body-small mt-4">Tab through the controls to review keyboard focus. Touch targets start at 48 px.</p>
            {saved ? <div className="mt-4"><Notice tone="success">Event saved for this preview.</Notice></div> : null}
          </section>
        </section>
        <section className="grid gap-6 md:grid-cols-2" aria-label="Type and feedback">
          <div className={publicCardClass}><p className={publicEyebrowClass}>Type & spacing</p><h2 className="heading-2 mt-3">A room for everyone</h2><p className="body-copy mt-4">Heavy headings establish the event. Plain, readable body text carries times, places, and instructions.</p><p className="body-small mt-3">Use the 4, 8, 12, 16, 24, 32, 48 spacing scale. Keep labels close to the controls they describe.</p><div className="mt-6 flex flex-wrap gap-2">{(['neutral', 'success', 'warning', 'danger'] as const).map((tone) => <span key={tone} className={publicStatusPillClass(tone)}>{tone === 'neutral' ? 'Draft' : tone === 'success' ? 'Ready' : tone === 'warning' ? 'Needs attention' : 'Unavailable'}</span>)}</div></div>
          <div className={`${publicCardClass} space-y-3`}><p className={publicEyebrowClass}>Feedback</p><Notice tone="success">Check-in complete. You’re ready to enter.</Notice><Notice tone="warning">You’re offline. Keep this screen open until sync completes.</Notice><Notice tone="danger">This ticket cannot be admitted. Ask the door lead for help.</Notice><Notice tone="info">No events yet. Published events will appear here.</Notice></div>
        </section>
        <footer className="body-small border-t border-stroke-subtle py-5">Subcult · Subcults terminal foundations, shared across web and native.</footer>
      </div>
    </main>
  );
}
