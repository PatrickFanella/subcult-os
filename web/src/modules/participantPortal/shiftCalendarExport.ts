import { apiBlob } from '../../api';
import { downloadBlob } from '../eventEditor/settlementExport';

export const shiftCalendarExportPath = '/api/me/participant-portal/shifts.ics';
export const shiftCalendarExportFilename = 'subcult-os-shifts.ics';

export async function downloadShiftCalendar() {
  const calendar = await apiBlob(shiftCalendarExportPath, { headers: { Accept: 'text/calendar' } });
  downloadBlob(calendar, shiftCalendarExportFilename);
}
