import { describe, expect, it } from 'vitest';
import { offlineRoomKey } from './offline';

describe('offlineRoomKey (F10-2)', () => {
  it('namespaces keys per document', () => {
    expect(offlineRoomKey('abc')).toBe('collab-doc-abc');
    expect(offlineRoomKey('abc')).not.toBe(offlineRoomKey('xyz'));
  });
});
