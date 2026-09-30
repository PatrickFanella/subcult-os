import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { LifecycleIntentActionDTO } from '../domain';
import { LifecycleActionRow } from './LifecycleActionRow';

const draft: LifecycleIntentActionDTO = { id: 'draft', actionKind: 'operational_notice', destination: 'operational_notice', status: 'pending', attemptCount: 0, dispatchApproved: false, createdAt: '2026-09-30T00:00:00Z', updatedAt: '2026-09-30T00:00:00Z' };

describe('lifecycle action outcomes', () => {
  it('distinguishes an unapproved draft from the completed local notice queue', () => {
    const html = renderToStaticMarkup(<ul><LifecycleActionRow action={draft} /><LifecycleActionRow action={{ ...draft, id: 'queue', destination: 'notice_email_outbox', dispatchApproved: true, status: 'succeeded', attemptCount: 1 }} /></ul>);
    expect(html).toContain('Operational notice draft');
    expect(html).toContain('Draft; dispatch not approved');
    expect(html).toContain('Listing notice queue');
    expect(html).toContain('Queued locally');
    expect(html).toContain('Check the recipient outcomes below for delivery status');
    expect(html).not.toContain('Delivered');
  });

  it('keeps an unknown provider outcome distinct from success and requires reconciliation', () => {
    const html = renderToStaticMarkup(<LifecycleActionRow action={{ ...draft, actionKind: 'refund', destination: 'refund_provider', status: 'unknown', dispatchApproved: true, attemptCount: 1 }} />);
    expect(html).toContain('Refund review');
    expect(html).toContain('Reconcile this destination before retrying');
    expect(html).not.toContain('Queued locally');
  });

  it('shows the recorded retry and completion timestamps without inventing an outcome', () => {
    const html = renderToStaticMarkup(<LifecycleActionRow action={{ ...draft, status: 'retryable', nextAttemptAt: '2026-10-01T00:00:00Z', finishedAt: '2026-09-30T00:01:00Z' }} />);
    expect(html).toContain('Next attempt:');
    expect(html).toContain('Action finished:');
    expect(html).toContain('retryable');
  });
});
