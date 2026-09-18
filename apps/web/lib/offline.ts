/**
 * Offline persistence naming (F10-2). The IndexedDB room key namespaces the
 * local replica per document so reloads restore instantly and reconnects merge
 * through the CRDT — no server round-trip on boot.
 */
export function offlineRoomKey(docId: string): string {
  return `collab-doc-${docId}`;
}
