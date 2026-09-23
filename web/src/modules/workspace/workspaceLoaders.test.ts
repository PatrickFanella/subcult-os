import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, api } from '../../api';
import type { CurrentUserDTO } from '../../domain';
import { loadCurrentWorkspace, loadWorkspaceOverview, selectWorkspace } from './workspaceLoaders';

vi.mock('../../api', async () => ({...await vi.importActual('../../api'), api: vi.fn()}));
afterEach(() => vi.clearAllMocks());
const selected = { id: 'chosen', name: 'Chosen room', role: 'owner', members: [], invitations: [] };
const fallback = { ...selected, id: 'fallback', name: 'Fallback room' };
const user = {id:'person',email:'synthetic@example.test',displayName:'Operator',workspaces:[fallback]} as CurrentUserDTO;
const failure = (status: number) => new ApiError(status, `Synthetic ${status}`, {});

describe('workspace selection', () => {
  it('keeps the explicitly selected workspace', async () => {
    vi.mocked(api).mockResolvedValueOnce(selected);
    expect(await selectWorkspace(user,'chosen')).toEqual({workspace:selected,notice:null});
    expect(api).toHaveBeenCalledExactlyOnceWith('/api/workspaces/chosen');
  });
  it.each([403,404])('falls back only for unavailable selection %s',async(status)=>{
    vi.mocked(api).mockRejectedValueOnce(failure(status)).mockResolvedValueOnce(fallback);
    expect(await selectWorkspace(user,'chosen')).toMatchObject({workspace:fallback,notice:expect.any(String)});
  });
  it.each([401,429,500,503])('does not switch on %s',async(status)=>{
    const error=failure(status);vi.mocked(api).mockRejectedValueOnce(error);
    await expect(selectWorkspace(user,'chosen')).rejects.toBe(error);
    expect(api).toHaveBeenCalledTimes(1);
  });
  it('does not switch on transport failure',async()=>{
    const error=new TypeError('network unavailable');vi.mocked(api).mockRejectedValueOnce(error);
    await expect(selectWorkspace(user,'chosen')).rejects.toBe(error);
    expect(api).toHaveBeenCalledTimes(1);
  });
  it('loads authoritative fallback rather than inventing empty membership',async()=>{
    vi.mocked(api).mockRejectedValueOnce(failure(404)).mockResolvedValueOnce(fallback);
    expect(await selectWorkspace(user,null)).toEqual({workspace:fallback,notice:null});
    expect(api).toHaveBeenLastCalledWith('/api/workspaces/fallback');
  });
  it('propagates current-workspace failures except absence',async()=>{
    vi.mocked(api).mockRejectedValueOnce(failure(500));
    await expect(loadCurrentWorkspace()).rejects.toMatchObject({status:500});
  });
});

describe('workspace overview',()=>{
  it.each(['events','archives','contacts','commitments'])('does not invent empty %s on outage',async(panel)=>{
    const error=failure(503);
    vi.mocked(api).mockImplementation(async(path)=>{if(path.endsWith('/'+panel))throw error;return [] as never;});
    await expect(loadWorkspaceOverview('chosen','')).rejects.toBe(error);
    expect(vi.mocked(api).mock.calls.every(([path])=>path.startsWith('/api/workspaces/chosen/'))).toBe(true);
  });
  it('preserves permitted empty data and explicit private-panel denial',async()=>{
    vi.mocked(api).mockImplementation(async(path)=>{if(path.endsWith('/contacts')||path.endsWith('/commitments'))throw failure(403);return [] as never;});
    expect(await loadWorkspaceOverview('chosen','')).toEqual({events:[],archives:[],contacts:{data:null,denied:true},commitments:{data:null,denied:true}});
  });
});
