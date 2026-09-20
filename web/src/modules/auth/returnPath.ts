// Return paths stay relative to this application, never to an arbitrary origin.
export function safeReturnPath(value: string | null): string {
  if (!value?.startsWith('/') || value.startsWith('//') || value.includes('\\')) return '/';
  if ([...value].some((character) => character.charCodeAt(0) < 32 || character.charCodeAt(0) === 127)) return '/';
  try {
    const base = 'https://subcult.invalid';
    const url = new URL(value, base);
    if (url.origin !== base || url.pathname.startsWith('//')) return '/';
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return '/';
  }
}
