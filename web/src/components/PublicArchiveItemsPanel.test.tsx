import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { PublicArchiveItemsPanel } from './PublicArchiveItemsPanel';

describe('private archive initial authority state', () => {
  it('withholds approval controls and ledger until the owner read succeeds', () => {
    const html = renderToStaticMarkup(<PublicArchiveItemsPanel eventId="synthetic-event" />);
    expect(html).toContain('role="status"');
    expect(html).toContain('Loading approvals');
    expect(html).not.toContain('<form');
    expect(html).not.toContain('Approval ledger');
    expect(html).not.toContain('No archive items are approved yet');
    expect(html).toContain('Every approval remains unpublished');
  });
});
