import { afterEach, describe, expect, it, vi } from 'vitest';
import { changeWorkspaceMemberRole, editableMemberRole, loadMemberRoleWorkspace } from './memberRoleWrites';

afterEach(() => vi.unstubAllGlobals());
describe('scoped member role writes', () => {
  it('writes only role to the encoded workspace membership route and retains response metadata', async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'membership/a', role: 'door', expiresAt: '2026-11-01T00:00:00Z' })));
    vi.stubGlobal('fetch', fetch);
    const receipt = await changeWorkspaceMemberRole('workspace/a', 'membership/a', 'door');
    expect(receipt.expiresAt).toBe('2026-11-01T00:00:00Z');
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch.mock.calls[0][0]).toBe('/api/workspaces/workspace%2Fa/members/membership%2Fa');
    expect(fetch.mock.calls[0][1]).toMatchObject({ method: 'PATCH', credentials: 'include' });
    expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual({ role: 'door' });
  });
  it.each(['member', 'unknown', '', '__proto__'])('rejects non-assignable %s before any request', async role => {
    const fetch = vi.fn(); vi.stubGlobal('fetch', fetch);
    await expect(changeWorkspaceMemberRole('workspace-a', 'membership-a', role)).rejects.toThrow('assignable');
    expect(fetch).not.toHaveBeenCalled();
  });
  it.each([{ id: 'another-membership', role: 'door' }, { id: 'membership-a', role: 'owner' }])('rejects a mismatched mutation receipt', async receipt => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(receipt))));
    await expect(changeWorkspaceMemberRole('workspace-a', 'membership-a', 'door')).rejects.toThrow('did not match');
  });
  it.each([403, 409])('preserves server permission/last-owner rejection %s', async status => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'role change rejected' }), { status })));
    await expect(changeWorkspaceMemberRole('workspace-a', 'membership-a', 'door')).rejects.toMatchObject({ status, message: 'role change rejected' });
  });
  it('rejects another workspace read before it can populate the role page', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'workspace-b', members: [] }))));
    await expect(loadMemberRoleWorkspace('workspace-a')).rejects.toThrow('did not match');
  });
  it('maps the legacy alias to crew without making it an assignable server value', () => {
    expect(editableMemberRole('member')).toBe('crew');
    expect(editableMemberRole('finance')).toBe('finance');
  });
});
