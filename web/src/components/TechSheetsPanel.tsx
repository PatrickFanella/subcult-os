import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { api, apiBlob, postJSON } from '../api';
import type { ActInputDTO, ActTechSheetDTO, EventDTO, EventInputListDTO, EventOccurrenceDTO } from '../domain';

const message = (err: unknown) => err instanceof Error ? err.message : 'Unable to update tech sheets';
const basePath = (eventId: string) => `/api/events/${eventId}`;
const jsonWrite = <T,>(path: string, method: string, body: unknown) => api<T>(path, { method, body: JSON.stringify(body) });

export function CombinedInputList({ list }: { list: EventInputListDTO }) {
  return <div className="overflow-x-auto"><table className="w-full text-left text-sm"><caption className="py-3 text-left">{list.channels.length} channels · shared keys combine one channel across acts; confirm any equipment or phantom-power changes.</caption><thead><tr>{['Ch', 'Act / input', 'Source', 'Mic / DI', 'Stand', '48V', 'Notes'].map((label) => <th key={label} className="p-2">{label}</th>)}</tr></thead><tbody>
    {list.channels.map((channel) => <tr key={channel.number} className="border-t border-stroke-subtle"><th scope="row" className="p-2 align-top">{channel.number}{channel.sharedKey ? <span className="block text-xs font-normal">Shared: {channel.sharedKey}</span> : null}</th><td className="p-2 align-top">{channel.uses.map((use) => <p key={use.id}>{use.actName}: {use.label}</p>)}</td>{(['sourceType', 'micDi', 'stand', 'phantomPower', 'notes'] as const).map((field) => <td key={field} className="p-2 align-top whitespace-pre-wrap">{channel.uses.map((use) => <p key={use.id}>{field === 'phantomPower' ? use.phantomPower ? '48V' : 'Off' : use[field] || '—'}</p>)}</td>)}</tr>)}
  </tbody></table>{!list.acts.length ? <p className="text-fg-secondary">No acts on this bill yet.</p> : null}</div>;
}

function StagePlot({ url }: { url: string }) {
  const [image, setImage] = useState('');
  const [error, setError] = useState('');
  useEffect(() => () => { if (image) URL.revokeObjectURL(image); }, [image]);
  async function open() {
    setImage(''); setError('');
    try { setImage(URL.createObjectURL(await apiBlob(url))); } catch (err) { setError(message(err)); }
  }
  return <div className="space-y-2"><button type="button" className="btn-secondary px-3 py-2" onClick={() => void open()}>View private stage plot</button>{error ? <p role="alert">{error}</p> : null}{image ? <img className="max-h-96 max-w-full object-contain" src={image} alt="Private stage plot supplied for this act" /> : null}</div>;
}

function TechSheetEditor({ eventId, sheet, refresh, remove }: { eventId: string; sheet: ActTechSheetDTO; refresh: () => Promise<void>; remove: () => Promise<void> }) {
  const [draft, setDraft] = useState(sheet);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [clearStagePlot, setClearStagePlot] = useState(false);
  const path = `${basePath(eventId)}/tech-sheets/${sheet.actId}`;
  function updateInput(index: number, patch: Partial<ActInputDTO>) { setDraft((value) => ({ ...value, inputs: value.inputs.map((input, i) => i === index ? { ...input, ...patch } : input) })); }
  function moveInput(index: number, delta: number) {
    setDraft((value) => { const inputs = [...value.inputs]; [inputs[index], inputs[index + delta]] = [inputs[index + delta], inputs[index]]; return { ...value, inputs }; });
  }
  async function action(run: () => Promise<unknown>) {
    setBusy(true); setError('');
    try { await run(); await refresh(); } catch (err) { setError(message(err)); } finally { setBusy(false); }
  }
  function save(e: FormEvent) { e.preventDefault(); void action(() => jsonWrite(path, 'PUT', { expectedVersion: sheet.version, monitorNotes: draft.monitorNotes, backlineBrought: draft.backlineBrought, backlineNeeded: draft.backlineNeeded, inputs: draft.inputs, clearStagePlot })); }
  async function upload(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const data = new FormData(e.currentTarget); data.set('expectedVersion', String(sheet.version));
    await action(() => api(path + '/stage-plot', { method: 'POST', body: data }));
  }
  return <details className="min-w-0 border border-stroke-subtle p-4"><summary className="cursor-pointer font-bold">{sheet.actName} · {sheet.inputs.length} inputs · version {sheet.version}{sheet.attached ? '' : ' · removed from bill'}</summary>
    {error ? <p role="alert" className="my-3 text-status-danger">{error} Your unsaved form is still here. Reload the input list before retrying a conflict.</p> : null}
    {sheet.attached ? <form onSubmit={save} className="mt-4 space-y-4"><fieldset disabled={busy} className="min-w-0 space-y-4">
      <p className="text-sm text-fg-secondary">Give inputs the same shared key across acts only when they use the same physical channel. Different requirements remain visible in the combined list.</p>
      {draft.inputs.map((input, index) => <fieldset key={input.id} className="min-w-0 space-y-3 border border-stroke-subtle p-3"><legend>Input {index + 1}</legend><div className="grid gap-3 sm:grid-cols-2">
        <label>Instrument / label<input className="field mt-1" required maxLength={100} value={input.label} onChange={(e) => updateInput(index, { label: e.target.value })} /></label>
        <label>Source<select className="field mt-1" value={input.sourceType} onChange={(e) => updateInput(index, { sourceType: e.target.value as ActInputDTO['sourceType'] })}><option value="mic">Mic</option><option value="di">DI</option><option value="line">Line</option></select></label>
        <label>Mic / DI<input className="field mt-1" maxLength={100} value={input.micDi} onChange={(e) => updateInput(index, { micDi: e.target.value })} /></label>
        <label>Stand<input className="field mt-1" maxLength={100} value={input.stand} onChange={(e) => updateInput(index, { stand: e.target.value })} /></label>
        <label>Shared key (optional)<input className="field mt-1" maxLength={40} pattern="[a-z0-9][a-z0-9_-]*" placeholder="e.g. shared-kick" value={input.sharedKey} onChange={(e) => updateInput(index, { sharedKey: e.target.value })} /></label>
        <label className="flex items-center gap-3"><input type="checkbox" checked={input.phantomPower} onChange={(e) => updateInput(index, { phantomPower: e.target.checked })} />Requires phantom power (48V)</label>
      </div><label className="block">Input notes<textarea className="field mt-1" maxLength={500} value={input.notes} onChange={(e) => updateInput(index, { notes: e.target.value })} /></label><div className="flex flex-wrap gap-2"><button type="button" className="btn-secondary px-3 py-2" disabled={index === 0} onClick={() => moveInput(index, -1)}>Move input {index + 1} up</button><button type="button" className="btn-secondary px-3 py-2" disabled={index === draft.inputs.length - 1} onClick={() => moveInput(index, 1)}>Move input {index + 1} down</button><button type="button" className="btn-ghost px-3 py-2" onClick={() => setDraft((value) => ({ ...value, inputs: value.inputs.filter((_, i) => i !== index) }))}>Remove input {index + 1}</button></div></fieldset>)}
      <button type="button" className="btn-secondary px-3 py-2" disabled={draft.inputs.length >= 64} onClick={() => setDraft((value) => ({ ...value, inputs: [...value.inputs, { id: crypto.randomUUID(), label: '', sourceType: 'mic', micDi: '', stand: '', phantomPower: false, notes: '', sharedKey: '' }] }))}>Add input</button>
      {(['monitorNotes', 'backlineBrought', 'backlineNeeded'] as const).map((field, i) => <label className="block" key={field}>{['Monitor needs', 'Backline brought', 'Backline needed'][i]}<textarea className="field mt-1" maxLength={2000} value={draft[field]} onChange={(e) => setDraft((value) => ({ ...value, [field]: e.target.value }))} /></label>)}
      {sheet.stagePlotUrl ? <label className="flex items-center gap-3"><input type="checkbox" checked={clearStagePlot} onChange={(e) => setClearStagePlot(e.target.checked)} />Remove stage plot on save</label> : null}
      <button className="btn-primary px-4 py-3" disabled={busy}>{busy ? 'Saving…' : 'Save tech sheet'}</button>
    </fieldset></form> : <div className="mt-3 space-y-2"><p>Retained for review. Reattach the act to restore it to the list, or explicitly delete this sheet.</p><p className="whitespace-pre-wrap">Monitors: {sheet.monitorNotes || '—'}<br />Brings: {sheet.backlineBrought || '—'}<br />Needs: {sheet.backlineNeeded || '—'}</p>{sheet.inputs.map((input) => <p key={input.id}>{input.label} · {input.micDi} · {input.notes}</p>)}</div>}
    {sheet.stagePlotUrl ? <div className="mt-4"><StagePlot url={sheet.stagePlotUrl} /></div> : null}
    {sheet.attached && sheet.version > 0 ? <form onSubmit={(e) => void upload(e)} className="mt-4 space-y-3"><p className="text-sm">Save input edits before uploading a stage plot. Uploading reloads this sheet.</p><label className="block">Private stage plot (PNG / JPEG, up to 4 MiB)<input name="image" type="file" accept="image/png,image/jpeg" required className="block max-w-full" disabled={busy} /></label><button className="btn-secondary px-3 py-2" disabled={busy}>Upload stage plot</button></form> : null}
    <div className="mt-4 flex flex-wrap gap-2">{sheet.attached ? <button type="button" className="btn-ghost px-3 py-2" disabled={busy} onClick={() => { if (window.confirm(`Remove ${sheet.actName} from this event’s performer credits? Its sheet will be kept for review.`)) void action(remove); }}>Remove act from bill</button> : null}{sheet.version > 0 ? <button type="button" className="btn-danger px-3 py-2" disabled={busy} onClick={() => { if (window.confirm(`Permanently delete ${sheet.actName}’s tech sheet and stage plot?`)) void action(() => jsonWrite(path, 'DELETE', { expectedVersion: sheet.version, confirmed: true })); }}>Delete tech sheet</button> : null}</div>
  </details>;
}

function BillControls({ event, list, refresh }: { event: EventDTO; list: EventInputListDTO; refresh: () => Promise<void> }) {
  const [profiles, setProfiles] = useState<Array<{ id: string; kind: string; displayName: string }>>([]);
  const [occurrences, setOccurrences] = useState<EventOccurrenceDTO[]>([]);
  const [actId, setActId] = useState('');
  const [occurrenceId, setOccurrenceId] = useState('');
  const [name, setName] = useState('');
  const [sourceId, setSourceId] = useState('');
  const [events, setEvents] = useState<EventDTO[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const root = basePath(event.id);
  useEffect(() => { let active = true; Promise.all([api<typeof profiles>(`/api/workspaces/${event.workspaceId}/profiles`), api<EventOccurrenceDTO[]>(root + '/occurrences'), api<EventDTO[]>(`/api/workspaces/${event.workspaceId}/events`)]).then(([acts, dates, allEvents]) => { if (active) { setProfiles(acts.filter((a) => a.kind === 'act')); setOccurrences(dates); setOccurrenceId(dates[0]?.id ?? ''); setEvents(allEvents.filter((e) => e.id !== event.id)); } }).catch((err) => { if (active) setError(message(err)); }); return () => { active = false; }; }, [event.id, event.workspaceId, root]);
  async function action(run: () => Promise<unknown>) { setBusy(true); setError(''); try { await run(); await refresh(); } catch (err) { setError(message(err)); } finally { setBusy(false); } }
  return <details className="border border-stroke-subtle p-4"><summary className="cursor-pointer font-bold">Manage bill / copy sheets</summary>{error ? <p role="alert" className="my-3 text-status-danger">{error}</p> : null}<div className="mt-4 space-y-4">
    {!occurrences.length ? <div className="space-y-2"><p>Acts need a dated listing for their performer credits. This creates a private draft listing using the saved event title and time.</p><button type="button" className="btn-secondary px-3 py-2" disabled={busy} onClick={() => void action(async () => { const created = await postJSON<EventOccurrenceDTO>(root + '/occurrences', { name: event.title, startsAt: event.startsAt, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone }); setOccurrences([created]); setOccurrenceId(created.id); })}>Create draft listing for bill</button></div> : <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); void action(() => postJSON(`${root}/occurrences/${occurrenceId}/credits`, { profileId: actId, role: 'performer', sortOrder: list.acts.length })); }}><label className="block">Listing<select className="field mt-1" required value={occurrenceId} onChange={(e) => setOccurrenceId(e.target.value)}>{occurrences.map((o) => <option value={o.id} key={o.id}>{o.name}</option>)}</select></label><label className="block">Act<select className="field mt-1" required value={actId} onChange={(e) => setActId(e.target.value)}><option value="">Choose an act</option>{profiles.filter((a) => !list.acts.some((s) => s.actId === a.id)).map((a) => <option key={a.id} value={a.id}>{a.displayName}</option>)}</select></label><button className="btn-secondary px-3 py-2" disabled={busy || !actId}>Add act to bill</button></form>}
    <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); void action(async () => { const created = await postJSON<(typeof profiles)[number]>(`/api/workspaces/${event.workspaceId}/profiles`, { kind: 'act', displayName: name }); setProfiles((rows) => [...rows, created]); setActId(created.id); setName(''); }); }}><label className="block">New act name<input className="field mt-1" required maxLength={200} value={name} onChange={(e) => setName(e.target.value)} /></label><button className="btn-secondary px-3 py-2" disabled={busy}>Create act profile</button></form>
    {event.status === 'draft' ? <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); if (window.confirm('Copy sheets for matching acts into this draft? Existing sheets stay as they are. Stage plots are excluded.')) void action(() => postJSON(root + '/tech-sheets/copy', { sourceEventId: sourceId, confirmed: true })); }}><label className="block">Copy from event<select className="field mt-1" required value={sourceId} onChange={(e) => setSourceId(e.target.value)}><option value="">Choose an event</option>{events.map((source) => <option key={source.id} value={source.id}>{source.title}</option>)}</select></label><p className="text-sm">Explicitly copy input and backline notes for matching acts already on this bill. Stage plots are excluded. Templates and archive reuse start without tech sheets.</p><button className="btn-secondary px-3 py-2" disabled={busy || !sourceId}>Copy matching tech sheets</button></form> : null}
  </div></details>;
}

export function TechSheetsPanel({ eventId, event }: { eventId: string; event?: EventDTO }) {
  const [list, setList] = useState<EventInputListDTO | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const root = basePath(eventId);
  useEffect(() => { let active = true; api<EventInputListDTO>(root + '/input-list').then((data) => { if (active) setList(data); }).catch((err) => { if (active) setError(message(err)); }); return () => { active = false; }; }, [root]);
  async function refresh() { const data = await api<EventInputListDTO>(root + '/input-list'); setList(data); setError(''); }
  async function reorder(index: number, delta: number) {
    if (!list) return; const ids = list.acts.map((act) => act.actId); [ids[index], ids[index + delta]] = [ids[index + delta], ids[index]];
    setBusy(true); setError(''); try { await jsonWrite(root + '/input-list/order', 'PUT', { expectedVersion: list.orderVersion, actIds: ids }); await refresh(); } catch (err) { setError(message(err)); } finally { setBusy(false); }
  }
  async function print() {
    const popup = window.open('', '_blank');
    if (!popup) { setError('Allow a new tab to open the printable input list.'); return; }
    popup.opener = null;
    try { const url = URL.createObjectURL(await apiBlob(root + '/input-list?format=print')); popup.location.href = url; window.setTimeout(() => URL.revokeObjectURL(url), 60_000); } catch (err) { popup.close(); setError(message(err)); }
  }
  return <section id="tech-sheets" className="min-w-0 space-y-4 border border-stroke-subtle bg-surface-panel p-4 sm:p-6"><div className="flex flex-wrap items-center justify-between gap-3"><h2 className="text-2xl font-bold">{list?.eventTitle ? `${list.eventTitle} · ` : ''}Input list & tech sheets</h2><div className="flex flex-wrap gap-2"><button className="btn-secondary px-3 py-2" onClick={() => void refresh().catch((err) => { setList(null); setError(message(err)); })}>Reload input list</button><button className="btn-secondary px-3 py-2" disabled={!list} onClick={() => void print()}>Open printable input list</button></div></div><p className="text-sm text-fg-secondary">Private to active workspace operators. Keep printed and exported copies private.</p>{error ? <p role="alert" className="text-status-danger">{error}</p> : null}{!list && !error ? <p>Loading input list…</p> : null}
    {list ? <><CombinedInputList list={list} />{list.acts.map((sheet, index) => <div key={sheet.actId} className="min-w-0 space-y-2">{list.canEdit ? <div className="flex flex-wrap gap-2"><button className="btn-secondary px-3 py-2" disabled={busy || index === 0} onClick={() => void reorder(index, -1)}>Move {sheet.actName} up</button><button className="btn-secondary px-3 py-2" disabled={busy || index === list.acts.length - 1} onClick={() => void reorder(index, 1)}>Move {sheet.actName} down</button></div> : null}{list.canEdit ? <TechSheetEditor key={`${sheet.actId}:${sheet.version}:${sheet.attached}`} eventId={eventId} sheet={sheet} refresh={refresh} remove={() => jsonWrite<void>(root + '/tech-bill/' + sheet.actId, 'DELETE', {}).then(() => undefined)} /> : <article className="space-y-2 border border-stroke-subtle p-4"><h3 className="font-bold">{sheet.actName}</h3><p className="whitespace-pre-wrap">Monitors: {sheet.monitorNotes || '—'}<br />Brings: {sheet.backlineBrought || '—'}<br />Needs: {sheet.backlineNeeded || '—'}</p>{sheet.stagePlotUrl ? <StagePlot url={sheet.stagePlotUrl} /> : null}</article>}</div>)}{list.canEdit && list.retained?.length ? <div className="space-y-3"><h3 className="font-bold">Removed acts · retained sheets</h3>{list.retained.map((sheet) => <TechSheetEditor key={`${sheet.actId}:${sheet.version}`} eventId={eventId} sheet={sheet} refresh={refresh} remove={async () => undefined} />)}</div> : null}{list.canEdit && event ? <BillControls event={event} list={list} refresh={refresh} /> : null}</> : null}
  </section>;
}
