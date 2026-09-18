import { describe, expect, it } from 'vitest';
import { prefixLines, wrapSelection } from './format';

describe('toolbar transforms (F9-3)', () => {
  it('wraps a selection with bold markers', () => {
    const r = wrapSelection('hello world', 6, 11, '**', '**');
    expect(r.text).toBe('hello **world**');
    expect(r.start).toBe(8);
  });

  it('inserts placeholder when selection is empty', () => {
    const r = wrapSelection('hi ', 3, 3, '**', '**');
    expect(r.text).toBe('hi **text**');
  });

  it('prefixes heading lines idempotently', () => {
    const once = prefixLines('a\nb', 0, 3, '# ');
    expect(once.text).toBe('# a\n# b');
    const twice = prefixLines(once.text, 0, once.end, '# ');
    expect(twice.text).toBe('# a\n# b');
  });
});
