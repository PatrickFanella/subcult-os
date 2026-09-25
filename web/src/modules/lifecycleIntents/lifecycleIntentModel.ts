export type LifecycleIntentRetry = { payload: string; key: string };

// A failed request keeps its key only while every decision-bearing field is
// unchanged. Navigation and authorization failure clear the retry state.
export function nextLifecycleIntentRetry(current: LifecycleIntentRetry | null, payload: string, createKey: () => string): LifecycleIntentRetry {
  if (current?.payload === payload) return current;
  return { payload, key: createKey() };
}
