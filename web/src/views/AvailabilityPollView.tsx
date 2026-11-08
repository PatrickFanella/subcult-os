import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { PollOptionTime, PollResults } from '../components/AvailabilityPollsPanel';
import type { AvailabilityPoll, PollAnswer, PollResponse } from '../modules/availabilityPolls/model';
import { pollEditCapability, pollRequest } from '../modules/availabilityPolls/model';

export function AvailabilityPollView({ pollId }: { pollId: string }) {
  const token = typeof window === 'undefined' ? '' : window.location.hash?.slice(1) ?? '';
  const [edit] = useState(() => typeof window === 'undefined' ? { token: '', retained: false } : pollEditCapability(pollId));
  const [poll, setPoll] = useState<AvailabilityPoll | null>(null);
  const [form, setForm] = useState<PollResponse>({ displayName: '', privateNote: '', answers: {} });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    const robots = document.createElement('meta'); robots.name = 'robots'; robots.content = 'noindex,nofollow';
    const referrer = document.createElement('meta'); referrer.name = 'referrer'; referrer.content = 'no-referrer';
    document.head.append(robots, referrer);
    let active = true;
    if (token) pollRequest<AvailabilityPoll>(pollId, token, edit.token).then((result) => { if (active) { setPoll(result); if (result.ownResponse) setForm(result.ownResponse); } }).catch((err) => { if (active) setError(err instanceof Error ? err.message : 'Poll unavailable'); });
    return () => { active = false; robots.remove(); referrer.remove(); };
  }, [pollId, token, edit.token]);
  async function submit(e: FormEvent) {
    e.preventDefault(); if (busy) return; setBusy(true); setNotice(null); setError(null);
    try { await pollRequest(pollId, token, edit.token, form); setPoll(await pollRequest<AvailabilityPoll>(pollId, token, edit.token)); setNotice('Answers saved. Return in this browser to change them while the poll is open.'); }
    catch (err) { setError(err instanceof Error ? err.message : 'Unable to save answers'); }
    finally { setBusy(false); }
  }
  return <main className="mx-auto max-w-3xl space-y-6 p-4 sm:p-6">
    <header><p className="text-sm text-fg-muted">Availability poll</p><h1 className="mt-2 text-3xl font-bold break-words">{poll?.prompt ?? 'Availability'}</h1><p className="mt-2">No account needed. The organizer sees your name, answers and private note. {poll?.showNames ? 'Other respondents can see names and answers.' : 'Other respondents see answer totals without names.'}</p></header>
    {!token ? <p role="alert">Poll unavailable. Open the complete link shared by the organizer.</p> : null}
    {error ? <p role="alert" className="text-status-danger">{error}</p> : null}
    {notice ? <p role="status" className="text-status-success">{notice}</p> : null}
    {poll ? <>
      <section className="border border-stroke-subtle bg-surface-panel p-4"><h2 className="text-xl font-bold">{poll.status === 'closed' ? 'Poll closed' : 'Current answers'}</h2>{poll.closesAt ? <p>Closes {new Date(poll.closesAt).toLocaleString()}</p> : null}<PollResults poll={poll} /></section>
      {poll.status === 'open' ? <form className="space-y-4 border border-stroke-subtle bg-surface-panel p-4" onSubmit={submit}><h2 className="text-xl font-bold">Your answers</h2>
        {!edit.retained ? <p className="text-status-warning">This browser cannot save edit access. Keep this tab open to change your answers.</p> : null}
        <label className="block">Name<input required maxLength={80} className="field mt-1" autoComplete="off" value={form.displayName} onChange={(e) => setForm((v) => ({ ...v, displayName: e.target.value }))} /></label>
        {poll.options.map((option, index) => <fieldset className="min-w-0 space-y-2 border border-stroke-subtle p-3" key={option.id}><legend>Option {index + 1}</legend><PollOptionTime option={option} /><label className="block mt-2">Availability for option {index + 1}<select required className="field mt-1" value={form.answers[option.id] ?? ''} onChange={(e) => setForm((v) => ({ ...v, answers: { ...v.answers, [option.id]: e.target.value as PollAnswer } }))}><option value="">Choose an answer</option><option value="yes">Yes</option><option value="if_need_be">If need be</option><option value="no">No</option></select></label></fieldset>)}
        <label className="block">Private note to the organizer (optional)<textarea maxLength={1000} className="field mt-1 min-h-24" value={form.privateNote ?? ''} onChange={(e) => setForm((v) => ({ ...v, privateNote: e.target.value }))} /></label>
        <button className="btn-primary px-4 py-3" disabled={busy}>{busy ? 'Saving…' : poll.ownResponse ? 'Update my answers' : 'Save my answers'}</button>
      </form> : <p>New answers and edits are closed. {poll.chosenOptionId ? 'The organizer chose an option above.' : 'The organizer has not chosen an option.'}</p>}
    </> : null}
  </main>;
}
