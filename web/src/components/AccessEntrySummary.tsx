import type { AccessInformationRevisionDTO } from '../domain';
import { accessExpiry, accessSourceLabels, accessValueLabels } from '../modules/eventAccess/eventAccessModel';

export function AccessEntrySummary({ entry, evaluatedAt }: { entry: AccessInformationRevisionDTO; evaluatedAt: string }) {
  const expired = accessExpiry(entry, Date.parse(entry.evaluatedAt || evaluatedAt));
  return <div className="space-y-2 text-sm">
    <p className="font-bold">{accessValueLabels[entry.effectiveValue]}{expired && ' · review expired'}</p>
    {entry.value === 'unknown' ? <p className="text-fg-secondary">No current assertion. Unknown does not mean no.</p> : <>
      {expired && <p className="text-status-warning">The recorded assertion needs confirmation. Its previous value was {accessValueLabels[entry.value].toLowerCase()}.</p>}
      {entry.details && <p className="whitespace-pre-wrap break-words">{entry.details}</p>}
      <p className="text-fg-secondary">{accessSourceLabels[entry.sourceKind]} · {entry.scope === 'venue' ? 'venue assertion' : 'event-specific'}</p>
      <p className="whitespace-pre-wrap break-words text-fg-secondary">Source: {entry.sourceReference}</p>
      <p className="break-words text-xs text-fg-muted">Reviewed: {entry.reviewedAt} {entry.expiresAt && `· Review by: ${entry.expiresAt}`}</p>
    </>}
  </div>;
}

