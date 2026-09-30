import * as React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { CurrentWorkspaceDTO, MemberDTO, WorkspaceRole } from '../domain';
import { MemberRolesView } from './MemberRolesView';

vi.mock('react', async () => {
  const actual = await vi.importActual<typeof import('react')>('react');
  return { ...actual, useState: vi.fn((initial: unknown) => [typeof initial === 'function' ? (initial as () => unknown)() : initial, vi.fn()]) };
});
const useState = vi.mocked(React.useState);
afterEach(() => vi.clearAllMocks());
const member: MemberDTO = { id: 'membership-a', email: 'synthetic@example.test', displayName: 'Synthetic member', role: 'crew', accessState: 'active' };
function render(role: WorkspaceRole, overrides: Partial<MemberDTO> = {}, remaining: unknown[] = []) {
  const workspace: CurrentWorkspaceDTO = { id: 'workspace-a', name: 'Synthetic workspace', role, members: [{ ...member, ...overrides }], invitations: [] };
  const values: unknown[] = [workspace, false, null, {}, null, false, '', ...remaining];
  useState.mockImplementation(((initial: unknown) => [values.length ? values.shift() : typeof initial === 'function' ? (initial as () => unknown)() : initial, vi.fn()]) as unknown as typeof React.useState);
  return renderToStaticMarkup(<MemberRolesView workspaceId="workspace-a" />);
}
describe('member role page authority', () => {
  it.each<WorkspaceRole>(['member', 'crew', 'door', 'finance', 'organizer'])('keeps %s access read-only', role => {
    const html = render(role);
    expect(html).toContain('Synthetic member');
    expect(html).toContain('read-only');
    expect(html).not.toContain('<form');
  });
  it('offers all five assignable roles to owners with capability review and an explicit role action', () => {
    const html = render('owner');
    expect(html).toContain('Role to assign');
    for (const role of ['owner', 'organizer', 'finance', 'door', 'crew']) expect(html).toContain(`value="${role}"`);
    expect(html).not.toContain('value="member"');
    expect(html).toContain('Assign Crew role');
    expect(html).toContain('no door or finance permissions');
    expect(html).toContain('Keep at least one active owner');
  });
  it.each<MemberDTO['accessState']>(['expired', 'revoked', undefined])('does not offer role changes as restoration of %s membership', accessState => {
    const html = render('owner', { accessState });
    expect(html).toContain('Synthetic member');
    expect(html).not.toContain('<form');
  });
  it('keeps an unrecognized configured role unqualified', () => {
    const html = render('owner', { role: 'future-role' as WorkspaceRole });
    expect(html).toContain('Unknown role');
    expect(html).not.toContain('<form');
  });
  it('removes private roster content when no workspace is available', () => {
    const values: unknown[] = [null, false, 'Access unavailable', {}, null, true, ''];
    useState.mockImplementation(((initial: unknown) => [values.length ? values.shift() : initial, vi.fn()]) as unknown as typeof React.useState);
    const html = renderToStaticMarkup(<MemberRolesView workspaceId="workspace-a" />);
    expect(html).toContain('Access unavailable');
    expect(html).not.toContain(member.email);
    expect(html).not.toContain('<form');
    expect(html).toContain('Refresh workspace');
  });
});
