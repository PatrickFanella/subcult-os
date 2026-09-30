import { describe, expect, it } from 'vitest';
import { noticeOutcomeGuidance, noticeOutcomeSummary } from './noticeOutcomes';

describe('notice outcome guidance', () => {
  it('distinguishes an unsent queue from a retry with uncertain prior acceptance', () => {
    expect(noticeOutcomeGuidance('pending', 0)).toBe('Waiting for the mail worker.');
    expect(noticeOutcomeGuidance('pending', 1)).toContain('earlier attempt may have been accepted');
    expect(noticeOutcomeGuidance('quarantined', 2)).toContain('will not retry automatically');
    expect(noticeOutcomeGuidance('accepted', 1)).toContain('Delivery feedback is separate');
    expect(noticeOutcomeGuidance('suppressed', 2)).toContain('blocked');
  });
  it('summarizes queue states without counting acceptance as delivery', () => {
    expect(noticeOutcomeSummary([
      { email: 'one', status: 'accepted', attempts: 1, feedback: 'unknown' },
      { email: 'two', status: 'accepted', attempts: 1, feedback: 'delivered' },
      { email: 'three', status: 'quarantined', attempts: 2, feedback: 'unknown' },
    ])).toBe('2 accepted · 1 quarantined');
    expect(noticeOutcomeSummary([])).toBe('No recipients recorded.');
  });
});
