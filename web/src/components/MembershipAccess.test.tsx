import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { MemberDTO, WorkspaceRole } from '../domain';
import { roleLabel } from '../modules/workspace/memberAuthority';
import { MembershipAccess } from './MembershipAccess';

const member: MemberDTO = { id: 'membership-a', email: 'synthetic@example.test', displayName: null, role: 'door' };
describe('membership access roster', () => {
  it.each<[WorkspaceRole, string]>([['owner', 'Owner'], ['organizer', 'Organizer'], ['finance', 'Finance'], ['door', 'Door'], ['crew', 'Crew'], ['member', 'Crew']])('labels configured %s authority as %s', (role, label) => {
    expect(roleLabel(role)).toBe(label);
  });
  it('never presents an unknown role as ordinary crew authority', () => {
    expect(roleLabel('future-role')).toBe('Unknown role');
  });
  it('describes door capabilities only for server-confirmed active membership', () => {
    const html = renderToStaticMarkup(<MembershipAccess member={{ ...member, accessState: 'active' }} />);
    expect(html).toContain('Active access');
    expect(html).toContain('Look up tickets and check in guests.');
  });
  it.each<NonNullable<MemberDTO['accessState']>>(['expired', 'revoked'])('does not imply configured door authority is usable when %s', (accessState) => {
    const html = renderToStaticMarkup(<MembershipAccess member={{ ...member, accessState, expiresAt: '2026-09-30T12:00:00Z' }} />);
    expect(html).toContain('currently grants no workspace access');
    expect(html).not.toContain('Look up tickets');
    expect(html).toContain('dateTime="2026-09-30T12:00:00Z"');
  });
  it('keeps legacy responses unqualified rather than assuming they are active', () => {
    const html = renderToStaticMarkup(<MembershipAccess member={member} />);
    expect(html).toContain('Access status unavailable');
    expect(html).not.toContain('Active access');
    expect(html).not.toContain('Look up tickets');
  });
});
