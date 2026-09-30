import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { Button } from './Button';
import { Notice } from './Notice';

describe('shared controls', () => {
  it('prevents accidental form submission and blocks busy actions', () => {
    const idle = renderToStaticMarkup(<Button>Save</Button>);
    expect(idle).toContain('type="button"');
    expect(idle).not.toContain('disabled');
    const busy = renderToStaticMarkup(<Button type="submit" busy disabled={false}>Saving…</Button>);
    expect(busy).toContain('type="submit"');
    expect(busy).toContain('disabled=""');
    expect(busy).toContain('aria-busy="true"');
    expect(busy).toContain('Saving…');
  });
  it('announces errors as alerts and other feedback as status', () => {
    expect(renderToStaticMarkup(<Notice tone="danger">Unavailable</Notice>)).toContain('role="alert"');
    expect(renderToStaticMarkup(<Notice tone="success">Saved</Notice>)).toContain('role="status"');
  });
});
