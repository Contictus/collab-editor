import { describe, expect, it } from 'vitest';
import { diffLines } from './diff';

describe('diffLines (F11-6)', () => {
  it('marks changed lines add/del with context', () => {
    const d = diffLines('a\nb\nc', 'a\nB\nc');
    expect(d).toEqual([
      { kind: 'same', text: 'a' },
      { kind: 'del', text: 'b' },
      { kind: 'add', text: 'B' },
      { kind: 'same', text: 'c' },
    ]);
  });

  it('returns empty diff for identical texts', () => {
    expect(diffLines('x', 'x')).toEqual([{ kind: 'same', text: 'x' }]);
  });
});
