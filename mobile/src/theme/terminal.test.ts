import { describe, expect, it } from 'vitest';
import { terminalStyles } from './terminal';

describe('terminalStyles', () => {
  it('squares every corner except avatars', () => {
    const styles = terminalStyles({
      card: { borderRadius: 24 },
      corner: { borderTopLeftRadius: 40, borderBottomEndRadius: 12 },
      avatar: { borderRadius: 38 },
    }, 'android');
    expect(styles.card.borderRadius).toBe(0);
    expect(styles.corner).toEqual({ borderTopLeftRadius: 0, borderBottomEndRadius: 0 });
    expect(styles.avatar.borderRadius).toBe(38);
  });

  it('applies monospace to text styles and caps weight at bold', () => {
    const styles = terminalStyles({
      title: { fontSize: 28, fontWeight: '900' as const },
      caption: { color: '#fff' } as { color: string; fontFamily?: string },
      frame: { padding: 8 },
    }, 'ios');
    expect(styles.title).toEqual({ fontSize: 28, fontWeight: '700', fontFamily: 'Courier' });
    expect(styles.caption.fontFamily).toBe('Courier');
    expect(styles.frame).toEqual({ padding: 8 });
  });

  it('leaves arrays and primitives alone', () => {
    const list = [{ borderRadius: 4 }];
    expect(terminalStyles({ list }, 'android').list).toBe(list);
    expect(terminalStyles(null, 'android')).toBeNull();
  });
});
