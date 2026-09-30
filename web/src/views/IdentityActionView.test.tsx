import { Children, isValidElement } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { renderToString } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { postJSON } from '../api';
import { IdentityActionView } from './IdentityActionView';

const stateUpdates = vi.hoisted(() => [] as unknown[]);

vi.mock('../api', () => ({ postJSON: vi.fn() }));
// Handler-level tests; actual Strict Mode/browser behavior is qualified separately.
vi.mock('react', async () => ({
  ...await vi.importActual<typeof import('react')>('react'),
  useState: (initial: unknown) => [initial, vi.fn((value: unknown) => stateUpdates.push(value))],
  useRef: (initial: unknown) => ({ current: initial }),
}));

afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); stateUpdates.length = 0; });

function formSubmit(node: ReactNode): (event: FormEvent<HTMLFormElement>) => Promise<void> {
  for (const child of Children.toArray(node)) {
    if (!isValidElement<{ children?: ReactNode; onSubmit?: (event: FormEvent<HTMLFormElement>) => Promise<void> }>(child)) continue;
    if (child.type === 'form' && child.props.onSubmit) return child.props.onSubmit;
    try { return formSubmit(child.props.children); } catch { /* search siblings */ }
  }
  throw new Error('form missing');
}

describe('identity challenge interaction', () => {
  it('renders an explicit verification action without consuming a challenge', () => {
    const html = renderToString(<IdentityActionView action="verify" />);
    expect(html).toContain('Verify email and sign in');
    expect(html).not.toContain('Verified.');
    expect(postJSON).not.toHaveBeenCalled();
  });

  it('submits verification once while pending and replaces the token URL on success', async () => {
    const replace = vi.fn();
    vi.stubGlobal('window', { location: { search: '?token=test-one-use', replace } });
    let complete!: () => void;
    vi.mocked(postJSON).mockImplementationOnce(() => new Promise<void>((resolve) => { complete = resolve; }));
    const submit = formSubmit(IdentityActionView({ action: 'verify' }));
    const event = { preventDefault: vi.fn() } as unknown as FormEvent<HTMLFormElement>;
    const pending = submit(event);
    await submit(event);
    expect(postJSON).toHaveBeenCalledExactlyOnceWith('/api/auth/verify-email', { token: 'test-one-use' });
    expect(replace).not.toHaveBeenCalled();
    complete();
    await pending;
    expect(replace).toHaveBeenCalledExactlyOnceWith('/');
  });

  it('does not send a missing verification token', async () => {
    vi.stubGlobal('window', { location: { search: '' } });
    await formSubmit(IdentityActionView({ action: 'verify' }))({ preventDefault: vi.fn() } as unknown as FormEvent<HTMLFormElement>);
    expect(postJSON).not.toHaveBeenCalled();
  });

  it('labels recovery fields', () => {
    expect(renderToString(<IdentityActionView action="request-recovery" />)).toContain('aria-label="Email address"');
    expect(renderToString(<IdentityActionView action="complete-recovery" />)).toContain('aria-label="New password"');
  });
});


describe('recovery request receipt', () => {
  it('keeps account existence conditional and does not claim delivery after request acceptance', async () => {
    vi.mocked(postJSON).mockResolvedValueOnce({ ok: true });
    await formSubmit(IdentityActionView({ action: 'request-recovery' }))({ preventDefault: vi.fn() } as unknown as FormEvent<HTMLFormElement>);
    expect(postJSON).toHaveBeenCalledExactlyOnceWith('/api/auth/recovery/request', { email: '' });
    expect(stateUpdates).toContain('If that address belongs to a verified account, check its email for a recovery link. Email delivery is not confirmed.');
    expect(stateUpdates.filter(value => typeof value === 'string').join(' ')).not.toMatch(/has been sent|on its way/);
  });
  it('shows a rejected request without a success receipt', async () => {
    vi.mocked(postJSON).mockRejectedValueOnce(new Error('Unable to request recovery'));
    await formSubmit(IdentityActionView({ action: 'request-recovery' }))({ preventDefault: vi.fn() } as unknown as FormEvent<HTMLFormElement>);
    expect(stateUpdates).toContain('Unable to request recovery');
    expect(stateUpdates.filter(value => typeof value === 'string').join(' ')).not.toContain('check its email');
  });
});
