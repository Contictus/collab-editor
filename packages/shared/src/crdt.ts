import * as Y from 'yjs';

/**
 * CRDT reconstruction shared by web (SSR read-only bootstrap, Faz 2) and
 * ws-server (authoritative load-on-open, Faz 4). Keeping the Y.Text field name
 * and the rebuild algorithm in one place guarantees server SSR, ws-server, and
 * the client editor all agree.
 *
 * INVARIANT #1: state is always applied as binary Yjs updates — never text diffs.
 */

/** The Y.Text key holding the document body. Client CodeMirror binding uses the same. */
export const TEXT_KEY = 'content';

/**
 * Rebuild a Y.Doc from a snapshot + the ordered updates recorded after it
 * (data-model.md load-on-open). Pass snapshot=null to replay from the start.
 */
export function buildDoc(snapshot: Uint8Array | null, updates: Uint8Array[]): Y.Doc {
  const doc = new Y.Doc();
  if (snapshot) Y.applyUpdate(doc, snapshot);
  for (const update of updates) Y.applyUpdate(doc, update);
  return doc;
}

/** Read the current plain text from a live Y.Doc (without cloning). */
export function docText(doc: Y.Doc): string {
  return doc.getText(TEXT_KEY).toString();
}

/**
 * Convenience: reconstruct and return the plain text (used by SSR bootstrap).
 * Builds a temporary doc, extracts text, then destroys it — callers never
 * hold the ephemeral doc. Snapshot may be null for brand-new documents.
 */
export function loadText(snapshot: Uint8Array | null, updates: Uint8Array[]): string {
  const doc = buildDoc(snapshot, updates);
  const text = docText(doc);
  doc.destroy();
  return text;
}
