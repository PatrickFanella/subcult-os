import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { EventInputListDTO } from '../domain';
import { CombinedInputList } from './TechSheetsPanel';

it('shows both acts’ requirements for a shared physical channel', () => {
  const input = { id: 'a', label: 'Kick', sourceType: 'mic' as const, micDi: 'D112', stand: 'Short', phantomPower: false, notes: 'Change stand after first set', sharedKey: 'kick' };
  const list: EventInputListDTO = { eventId: 'event', eventTitle: 'Sound check', canEdit: false, orderVersion: 1, acts: [], channels: [{ number: 1, sharedKey: 'kick', uses: [{ ...input, actName: 'Act A' }, { ...input, id: 'b', actName: 'Act B', micDi: 'Beta 91', phantomPower: true, notes: 'Check phantom before changeover' }] }] };
  const html = renderToStaticMarkup(<CombinedInputList list={list} />);
  for (const text of ['1 channels', 'Act A: Kick', 'Act B: Kick', 'D112', 'Beta 91', '48V', 'Off', 'Change stand after first set', 'Check phantom before changeover']) expect(html).toContain(text);
  expect(html.match(/<tbody><tr/g)?.length).toBe(1);
});

describe('input text safety', () => {
  it('escapes operator-supplied notes and act labels', () => {
    const list: EventInputListDTO = { eventId: 'e', eventTitle: 'e', canEdit: false, orderVersion: 0, acts: [], channels: [{ number: 1, sharedKey: '', uses: [{ id: 'i', actName: '<script>bad</script>', label: 'Mic', sourceType: 'mic', micDi: '', stand: '', phantomPower: false, notes: '<img src=x onerror=bad()>', sharedKey: '' }] }] };
    const html = renderToStaticMarkup(<CombinedInputList list={list} />);
    expect(html).not.toContain('<script>'); expect(html).not.toContain('<img src=x');
    expect(html).toContain('&lt;script&gt;');
  });
});
