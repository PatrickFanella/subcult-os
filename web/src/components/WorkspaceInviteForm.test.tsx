import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { WorkspaceRole } from '../domain';
import { buildOperatorGuidance } from '../views/WorkspaceView';
import { WorkspaceInviteForm } from './WorkspaceInviteForm';

const props = { email: 'synthetic@example.test', inviteRole: 'participant' as const, onInviteRoleChange: () => {}, busy: false, notice: 'Created invitation receipt', onEmailChange: () => {}, onSubmit: () => {} };

describe('workspace invitation authority', () => {
  it.each<WorkspaceRole | undefined>(['member', 'organizer', 'finance', 'door', 'crew', 'participant', undefined])('omits invitation drafts, receipts and entry points for %s', (role) => {
    expect(renderToStaticMarkup(<WorkspaceInviteForm {...props} role={role} />)).toBe('');
    expect(buildOperatorGuidance([], 'workspace-a', role).actions.some(action => action.href === '#invite-member')).toBe(false);
  });
  it('offers the owner a labeled invitation form and a matching guidance target', () => {
    const html = renderToStaticMarkup(<WorkspaceInviteForm {...props} role="owner" />);
    expect(html).toContain('id="invite-member"');
    expect(html).toContain('type="email"');
    expect(html).toContain('required=""');
    expect(html).toContain('role="status"');
    expect(html).toContain('aria-atomic="true"');
    expect(buildOperatorGuidance([], 'workspace-a', 'owner').actions.some(action => action.href === '#invite-member')).toBe(true);
  });
  it('lets the owner choose the participant or crew tier, defaulting to the selected one', () => {
    const html = renderToStaticMarkup(<WorkspaceInviteForm {...props} role="owner" />);
    expect(html).toContain('checked="" value="participant"');
    expect(html).toContain('value="crew"');
    expect(html).not.toContain('checked="" value="crew"');
    expect(html).toContain('Their own shifts and open places only.');
    const crew = renderToStaticMarkup(<WorkspaceInviteForm {...props} role="owner" inviteRole="crew" />);
    expect(crew).toContain('checked="" value="crew"');
  });
  it('keeps a status region before submission and disables input and action while pending', () => {
    const html = renderToStaticMarkup(<WorkspaceInviteForm {...props} role="owner" busy notice={null} />);
    expect(html).toContain('role="status"');
    expect(html).not.toContain(props.notice);
    expect(html.match(/disabled=""/g)).toHaveLength(3);
  });
});
