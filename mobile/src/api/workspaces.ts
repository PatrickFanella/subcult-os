import { postJSON } from '@/api/client';
import type { WorkspaceSummaryDTO } from '@/api/types';

export function createWorkspace(body: { name: string }) {
  return postJSON<WorkspaceSummaryDTO>('/api/workspaces', body);
}
