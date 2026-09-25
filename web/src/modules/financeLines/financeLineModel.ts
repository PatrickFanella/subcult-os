export type FinanceLineLike = {
  id: string;
  entryType: string;
  direction: string;
  currency: string;
  amountCents: number;
  correctsLineId?: string;
};

export type FinanceTotal = {
  key: string;
  entryType: string;
  direction: string;
  currency: string;
  amountCents: number;
};

export type FinanceLineRetry = {
  fingerprint: string;
  key: string;
};

export function nextFinanceLineRetry(
  current: FinanceLineRetry | null,
  fingerprint: string,
  createKey: () => string,
): FinanceLineRetry {
  if (current?.fingerprint === fingerprint) return current;
  return { fingerprint, key: createKey() };
}

export function centsFromDecimal(value: string): number | null {
  const match = value.trim().match(/^(\d+)(?:\.(\d{1,2}))?$/);
  if (!match) return null;

  const whole = Number(match[1]);
  const fraction = Number((match[2] ?? '').padEnd(2, '0'));
  const cents = whole * 100 + fraction;
  return Number.isSafeInteger(cents) && cents <= 1_000_000_000 ? cents : null;
}

export function decimalFromCents(cents: number) {
  return `${Math.floor(cents / 100)}.${String(cents % 100).padStart(2, '0')}`;
}

export function toUTC(value: string): string | null | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date.toISOString();
}

export function toLocalDateTime(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';

  const pad = (part: number) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function preserveHydratedUTC(value: string, hydratedUTC?: string) {
  if (!value) return undefined;
  if (hydratedUTC && toLocalDateTime(hydratedUTC) === value) return hydratedUTC;
  return toUTC(value);
}

export function currentFinanceLines<T extends FinanceLineLike>(lines: T[]) {
  const correctedIDs = new Set(lines.flatMap((line) => line.correctsLineId ? [line.correctsLineId] : []));
  return lines.filter((line) => !correctedIDs.has(line.id));
}

export function financeTotals(lines: FinanceLineLike[]): FinanceTotal[] {
  const totals = new Map<string, FinanceTotal>();
  for (const line of currentFinanceLines(lines)) {
    const key = `${line.currency}|${line.entryType}|${line.direction}`;
    const previous = totals.get(key);
    totals.set(key, {
      key,
      currency: line.currency,
      entryType: line.entryType,
      direction: line.direction,
      amountCents: (previous?.amountCents ?? 0) + line.amountCents,
    });
  }
  return [...totals.values()].sort((left, right) => left.key.localeCompare(right.key));
}
