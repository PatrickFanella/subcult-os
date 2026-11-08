import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { api, deleteJSON, postJSON } from '../api';
import type { AvailabilityPoll, PollOption } from '../modules/availabilityPolls/model';
import { pollInstantFromInput, pollOptionLabel } from '../modules/availabilityPolls/model';

export function PollOptionTime({ option }: { option: PollOption }) {
  const label = pollOptionLabel(option);
  return <><span>{label.local}</span><span className="block text-xs text-fg-muted">{label.event}</span></>;
}

export function PollResults({ poll, privateNotes = false }: { poll: AvailabilityPoll; privateNotes?: boolean }) {
  return <div className="overflow-x-auto"><table className="w-full text-left text-sm"><caption className="py-2 text-left">Answers: Yes / If need be / No</caption><thead><tr><th className="p-2">Option</th><th className="p-2">Yes</th><th className="p-2">If need be</th><th className="p-2">No</th></tr></thead><tbody>
    {poll.options.map((option, index) => <tr key={option.id} className="border-t border-stroke-subtle"><th scope="row" className="p-2 font-normal"><strong>{index + 1}. </strong><PollOptionTime option={option} />{poll.chosenOptionId === option.id ? <strong className="block">Chosen</strong> : null}</th><td className="p-2">{poll.totals[option.id]?.yes ?? 0}</td><td className="p-2">{poll.totals[option.id]?.if_need_be ?? 0}</td><td className="p-2">{poll.totals[option.id]?.no ?? 0}</td></tr>)}
  </tbody></table>
    {poll.responses?.length ? <table className="mt-4 w-full text-left text-sm"><caption className="text-left">Respondents</caption><thead><tr><th className="p-2">Name</th>{poll.options.map((o, i) => <th className="p-2" key={o.id}>Option {i + 1}</th>)}{privateNotes ? <th className="p-2">Private note</th> : null}</tr></thead><tbody>{poll.responses.map((response, i) => <tr className="border-t border-stroke-subtle" key={i}><th className="p-2 font-normal" scope="row">{response.displayName}</th>{poll.options.map((o) => <td className="p-2" key={o.id}>{response.answers[o.id] === 'yes' ? 'Yes' : response.answers[o.id] === 'if_need_be' ? 'If need be' : 'No'}</td>)}{privateNotes ? <td className="p-2 whitespace-pre-wrap">{response.privateNote || '—'}</td> : null}</tr>)}</tbody></table> : null}
  </div>;
}

export function AvailabilityPollsPanel({ eventId, draft, disabled, onChoose }: { eventId: string; draft: boolean; disabled: boolean; onChoose: (option: PollOption) => Promise<void> }) {
  const [polls, setPolls] = useState<AvailabilityPoll[]>([]);
  const [prompt, setPrompt] = useState('');
  const [options, setOptions] = useState([{ start: '', end: '' }, { start: '', end: '' }]);
  const [showNames, setShowNames] = useState(false);
  const [closesAt, setClosesAt] = useState('');
  const [shareUrl, setShareUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const path = `/api/events/${eventId}/polls`;
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  useEffect(() => { let active = true; api<AvailabilityPoll[]>(path).then((rows) => { if (active) setPolls(rows); }).catch((err) => { if (active) setError(err instanceof Error ? err.message : 'Unable to load polls'); }); return () => { active = false; }; }, [path]);
  async function refresh() { setPolls(await api<AvailabilityPoll[]>(path)); }
  async function create(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError(null); setShareUrl('');
    try {
      const created = await postJSON<{ url: string }>(path, { prompt, showNames, closesAt: closesAt ? pollInstantFromInput(closesAt) : null, options: options.map((o) => ({ startsAt: pollInstantFromInput(o.start), endsAt: o.end ? pollInstantFromInput(o.end) : null, timezone: zone })) });
      setShareUrl(created.url); setPrompt(''); await refresh();
    } catch (err) { setError(err instanceof Error ? err.message : 'Unable to create poll'); } finally { setBusy(false); }
  }
  async function act(poll: AvailabilityPoll, remove: boolean) {
    if (remove && !window.confirm('Delete this poll and all of its responses?')) return;
    setBusy(true); setError(null);
    try { if (remove) await deleteJSON(`${path}/${poll.id}`); else await postJSON(`${path}/${poll.id}/close`, {}); await refresh(); setShareUrl(''); }
    catch (err) { setError(err instanceof Error ? err.message : 'Unable to change poll'); } finally { setBusy(false); }
  }
  async function choose(option: PollOption) {
    if (disabled || busy || !draft) return;
    setBusy(true); setError(null);
    try { await onChoose(option); await refresh(); } catch (err) { setError(err instanceof Error ? err.message : 'Unable to choose option'); } finally { setBusy(false); }
  }
  return <section id="availability-polls" className="space-y-4 border border-stroke-subtle bg-surface-panel p-6">
    <h2 className="text-2xl font-bold">Availability polls</h2><p className="text-sm text-fg-secondary">Ask crew and performers which date or call time works. Share the poll privately; the link grants access to its answers.</p>
    {error ? <p role="alert" className="text-status-danger">{error}</p> : null}
    {!draft ? <p>Polls are read-only after publication. Use the lifecycle/reschedule flow to change the event time.</p> : null}
    {disabled && draft ? <p className="text-status-warning">Save event changes before choosing a poll option.</p> : null}
    {polls.map((poll) => <article key={poll.id} className="space-y-3 border border-stroke-subtle p-4"><h3 className="text-xl font-bold break-words">{poll.prompt}</h3><p>{poll.status === 'closed' ? 'Closed' : 'Open'} · {poll.showNames ? 'Respondent names shared' : 'Respondent names hidden'}</p>{poll.closesAt ? <p>Closes {new Date(poll.closesAt).toLocaleString()}</p> : null}<PollResults poll={poll} privateNotes />
      {draft ? <div className="flex flex-wrap gap-2">{poll.options.map((option, index) => <button className="btn-secondary px-3 py-2" key={option.id} disabled={disabled || busy} onClick={() => void choose(option)}>Use option {index + 1}</button>)}{poll.status === 'open' ? <button className="btn-secondary px-3 py-2" disabled={busy} onClick={() => void act(poll, false)}>Close poll</button> : null}<button className="btn-danger px-3 py-2" disabled={busy} onClick={() => void act(poll, true)}>Delete poll</button></div> : null}
    </article>)}
    {shareUrl ? <div className="space-y-2 border border-stroke-subtle p-4" aria-live="polite"><h3 className="font-bold">Save this poll link now</h3><p className="text-sm">Shown only once. Keep it private; names are {showNames ? 'visible' : 'hidden'} to other respondents.</p><textarea className="field min-h-28" aria-label="New poll link" readOnly value={shareUrl} /><a className="btn-secondary px-3 py-2" href={shareUrl} target="_blank" rel="noreferrer">Open poll</a></div> : null}
    {draft ? <form onSubmit={create} className="space-y-4"><h3 className="text-xl font-bold">New poll</h3><label className="block">Prompt<input className="field mt-1" required maxLength={500} value={prompt} onChange={(e) => setPrompt(e.target.value)} /></label><p className="text-sm">Enter dates in this device’s timezone: {zone}. Respondents also see their own local time. Check both offsets when a date crosses daylight saving time.</p>
      {options.map((option, index) => <fieldset className="min-w-0 space-y-2 border border-stroke-subtle p-3" key={index}><legend>Option {index + 1}</legend><label className="block">Start<input type="datetime-local" required className="field mt-1" value={option.start} onChange={(e) => setOptions((rows) => rows.map((row, i) => i === index ? { ...row, start: e.target.value } : row))} /></label><label className="block">End (optional)<input type="datetime-local" className="field mt-1" value={option.end} onChange={(e) => setOptions((rows) => rows.map((row, i) => i === index ? { ...row, end: e.target.value } : row))} /></label>{options.length > 2 ? <button className="btn-ghost px-3 py-2" type="button" onClick={() => setOptions((rows) => rows.filter((_, i) => i !== index))}>Remove option {index + 1}</button> : null}</fieldset>)}
      <button type="button" className="btn-secondary px-3 py-2" disabled={options.length >= 12} onClick={() => setOptions((rows) => [...rows, { start: '', end: '' }])}>Add option</button>
      <label className="block">Closes at (optional)<input type="datetime-local" className="field mt-1" value={closesAt} onChange={(e) => setClosesAt(e.target.value)} /></label>
      <label className="flex items-center gap-3"><input type="checkbox" checked={showNames} onChange={(e) => setShowNames(e.target.checked)} />Show respondent names to other respondents (off by default)</label>
      <button className="btn-primary px-4 py-3" disabled={busy}>Create availability poll</button>
    </form> : null}
  </section>;
}
