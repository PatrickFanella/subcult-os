import { describe, expect, it } from 'vitest';
import { atprotoCallbackNotice } from './ATProtoIdentityPanel';

describe('AT Protocol callback notice', () => {
  it('uses fixed local copy for recognized callback states', () => {
    expect(atprotoCallbackNotice('?atproto=linked')).toContain('proves account control only');
    expect(atprotoCallbackNotice('?atproto=cancelled')).toContain('Nothing was linked');
    expect(atprotoCallbackNotice('?atproto=error')).toContain('safely try again');
  });

  it('does not reflect unknown or provider-controlled text', () => {
    expect(atprotoCallbackNotice('?atproto=%3Cscript%3E')).toBeNull();
    expect(atprotoCallbackNotice('?error_description=provider-secret')).toBeNull();
  });
});
