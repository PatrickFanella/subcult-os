import type { PropsWithChildren } from 'react';

type NoticeProps = PropsWithChildren<{ tone?: 'success' | 'warning' | 'danger' | 'info' }>;
const tones = { success: 'notice-success', warning: 'notice-warning', danger: 'notice-danger', info: 'notice-info' };
export function Notice({ tone = 'info', children }: NoticeProps) {
  return <p role={tone === 'danger' ? 'alert' : 'status'} className={`${tones[tone]} rounded-control border px-4 py-3 text-sm leading-6`}>{children}</p>;
}
