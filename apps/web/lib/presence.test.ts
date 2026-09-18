import { describe, expect, it } from 'vitest';
import { presenceList } from './presence';

describe('presenceList (F10-6)', () => {
  it('excludes the local client', () => {
    const states = new Map([
      [1, { user: { name: 'me', color: 'red' } }],
      [2, { user: { name: 'peer', color: 'blue' } }],
    ]);
    const list = presenceList(states, 1);
    expect(list).toEqual([{ clientId: 2, name: 'peer', color: 'blue' }]);
  });

  it('skips states without user info', () => {
    const states = new Map([[7, {}]]);
    expect(presenceList(states, 1)).toEqual([]);
  });
});
