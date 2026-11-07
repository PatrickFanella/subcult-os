import { afterEach, describe, expect, it, vi } from 'vitest';

const { apiBlob, downloadBlob } = vi.hoisted(() => ({ apiBlob: vi.fn(), downloadBlob: vi.fn() }));
vi.mock('../../api', () => ({ apiBlob }));
vi.mock('../eventEditor/settlementExport', () => ({ downloadBlob }));

import { downloadShiftCalendar } from './shiftCalendarExport';

describe('downloadShiftCalendar', () => {
  afterEach(() => {
    apiBlob.mockReset();
    downloadBlob.mockReset();
  });

  it('downloads the private calendar attachment through the credentialed client', async () => {
    const calendar = new Blob(['BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n'], { type: 'text/calendar' });
    apiBlob.mockResolvedValue(calendar);

    await downloadShiftCalendar();

    expect(apiBlob).toHaveBeenCalledWith('/api/me/participant-portal/shifts.ics', { headers: { Accept: 'text/calendar' } });
    expect(downloadBlob).toHaveBeenCalledWith(calendar, 'subcult-os-shifts.ics');
  });

  it('does not save a file when the request is denied', async () => {
    apiBlob.mockRejectedValue(Object.assign(new Error('unauthorized'), { status: 401 }));

    await expect(downloadShiftCalendar()).rejects.toThrow('unauthorized');
    expect(downloadBlob).not.toHaveBeenCalled();
  });
});
