import { describe, expect, it } from 'vitest';
import { nextLifecycleIntentRetry } from './lifecycleIntentModel';

describe('lifecycle intent retry identity', () => {
  it('reuses the decision key after a failed response when the payload is unchanged', () => {
    const first = nextLifecycleIntentRetry(null, 'same-decision', () => 'key-1');
    expect(nextLifecycleIntentRetry(first, 'same-decision', () => 'key-2')).toBe(first);
  });

  it('replaces the decision key after an edited payload', () => {
    const first = nextLifecycleIntentRetry(null, 'cancellation:weather', () => 'key-1');
    expect(nextLifecycleIntentRetry(first, 'reschedule:weather', () => 'key-2')).toEqual({ payload: 'reschedule:weather', key: 'key-2' });
  });
});
