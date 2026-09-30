import type { LifecycleNoticeDTO } from '../../domain';

export function noticeOutcomeGuidance(status: string, attempts: number): string {
  switch (status) {
    case 'held': return 'Sending is disabled; this message remains held.';
    case 'pending': return attempts > 0 ? 'A retry is pending. An earlier attempt may have been accepted; check provider records before any correction.' : 'Waiting for the mail worker.';
    case 'leased': return 'The worker is processing this message. Refresh later; avoid a duplicate notice.';
    case 'accepted': return 'The provider accepted this message. Delivery feedback is separate.';
    case 'quarantined': return 'Acceptance may be uncertain. Check provider records before deciding on a correction; this message will not retry automatically.';
    case 'failed': return 'Automatic attempts have stopped. Review provider records and the current listing before deciding on a correction.';
    case 'suppressed': return 'Further attempts are blocked by recipient suppression.';
    case 'withheld_authority': return 'Sending was withheld because listing or recipient authority changed.';
    case 'withheld_consent': return 'Sending was withheld because consent no longer permits it.';
    default: return 'Unrecognized delivery state. Refresh and inspect the server record before taking action.';
  }
}

export function noticeOutcomeSummary(recipients: LifecycleNoticeDTO['recipients']): string {
  if (recipients.length === 0) return 'No recipients recorded.';
  const counts = new Map<string, number>();
  for (const recipient of recipients) counts.set(recipient.status, (counts.get(recipient.status) ?? 0) + 1);
  return [...counts].sort(([a], [b]) => a.localeCompare(b)).map(([status, count]) => `${count} ${status.replaceAll('_', ' ')}`).join(' · ');
}
