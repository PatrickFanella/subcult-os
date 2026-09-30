import type { MemberDTO } from '../domain';
import { roleHint } from '../modules/workspace/memberAuthority';

function formatInstant(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 'Unavailable' : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'long' }).format(date);
}

export function MembershipAccess({ member }: { member: MemberDTO }) {
  const state = member.accessState;
  const label = state === 'active' ? 'Active access' : state === 'expired' ? 'Access expired' : state === 'revoked' ? 'Access revoked' : 'Access status unavailable';
  const hint = state === 'active' ? roleHint(member.role) : state === 'expired' || state === 'revoked' ? 'This membership currently grants no workspace access.' : 'Reload the workspace to check access.';
  return <div className="mt-2 text-xs leading-5 text-fg-muted">
    <p className={state === 'expired' || state === 'revoked' ? 'text-status-warning' : ''}>{label}</p>
    <p>{hint}</p>
    {member.expiresAt ? <p>Expiry: <time dateTime={member.expiresAt}>{formatInstant(member.expiresAt)}</time></p> : null}
    {member.revokedAt ? <p>Revoked: <time dateTime={member.revokedAt}>{formatInstant(member.revokedAt)}</time></p> : null}
  </div>;
}
