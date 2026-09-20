import { describe, expect, it } from 'vitest';
import { safeReturnPath } from './returnPath';

describe('authentication return paths', () => {
  it.each([null, '', 'https://evil.example', '//evil.example', '/\\evil.example', '/\t/evil.example', '/\n/evil.example', '/\r/evil.example', '/a/..//evil.example', 'javascript:alert(1)', ' /workspace'])('rejects unsafe return %j', (value) => {
    expect(safeReturnPath(value)).toBe('/');
  });

  it.each(['/workspace?workspaceId=room-1', '/invite/one-use', '/discover#events', '/events/a%2Fb', '/search?q=hello%20world'])('preserves application return %s', (value) => {
    expect(safeReturnPath(value)).toBe(value);
  });

  it('normalizes dot segments without creating a network-path redirect', () => {
    expect(safeReturnPath('/events/../workspace')).toBe('/workspace');
    expect(safeReturnPath('/%2e%2e//evil.example')).toBe('/');
  });
});
