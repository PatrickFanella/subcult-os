import type { LifecycleIntentActionDTO } from '../domain';

export function LifecycleActionRow({ action }: { action: LifecycleIntentActionDTO }) {
  const labels = { public_record: 'Public record review', provider_ticket: 'Ticket provider review', operational_notice: 'Operational notice draft', refund: 'Refund review' };
  const localQueue = action.destination === 'notice_email_outbox';
  const destinations: Record<string, string> = { public_record: 'Public record', ticket_provider: 'Ticket provider', operational_notice: 'Notice draft', refund_provider: 'Refund provider', notice_email_outbox: 'Local email queue' };
  const label = localQueue ? 'Listing notice queue' : labels[action.actionKind];
  const status = localQueue && action.status === 'succeeded' ? 'Queued locally' : action.status;
  return <li className="bg-surface-inset p-3">
    <p>{label}: <strong>{status}</strong> · attempts {action.attemptCount}</p>
    <p className="mt-1 text-xs text-fg-secondary">Destination: {destinations[action.destination] ?? 'Other destination'} · {action.dispatchApproved ? 'Dispatch approved' : 'Draft; dispatch not approved'}</p>
    {localQueue && action.status === 'succeeded' && <p className="mt-1 text-xs text-fg-secondary">Queue creation completed. Check the recipient outcomes below for delivery status.</p>}
    {action.status === 'unknown' && <p className="mt-1 text-status-warning">Reconcile this destination before retrying; the outcome is unknown.</p>}
    {action.nextAttemptAt && <p className="mt-1 text-xs text-fg-secondary">Next attempt: <time dateTime={action.nextAttemptAt}>{action.nextAttemptAt}</time></p>}
    {action.finishedAt && <p className="mt-1 text-xs text-fg-secondary">Action finished: <time dateTime={action.finishedAt}>{action.finishedAt}</time></p>}
  </li>;
}
