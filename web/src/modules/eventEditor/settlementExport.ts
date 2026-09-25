import { apiBlob } from '../../api';
import type { WorkspaceRole } from '../../domain';

const settlementExportFilename = 'event-settlement.csv';

export async function downloadSettlementExport(eventID: string) {
	const csv = await apiBlob(`/api/events/${encodeURIComponent(eventID)}/exports/settlement.csv`, {
		headers: { Accept: 'text/csv' },
	});
	const objectURL = URL.createObjectURL(csv);
	try {
		const link = document.createElement('a');
		link.href = objectURL;
		link.download = settlementExportFilename;
		link.hidden = true;
		document.body.appendChild(link);
		link.click();
		link.remove();
	} finally {
		URL.revokeObjectURL(objectURL);
  }
}

export function canDownloadSettlementExport(workspaceID: string | undefined, eventWorkspaceID: string | undefined, role: WorkspaceRole | undefined) {
  return workspaceID === eventWorkspaceID && (role === 'owner' || role === 'finance');
}
