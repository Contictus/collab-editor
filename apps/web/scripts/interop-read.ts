import { loadInitialText } from '../lib/document-service';

/**
 * Go↔Node interop reader (F5-11). Prints the SSR text for a document whose
 * history was written by the Go service, proving the op log + snapshots are
 * byte-compatible across implementations.
 *
 * Usage: pnpm --filter web exec tsx ./scripts/interop-read.ts <documentId>
 */
const docId = process.argv[2];
if (!docId) {
  console.error('usage: interop-read.ts <documentId>');
  process.exit(1);
}
const text = await loadInitialText(docId);
console.log(JSON.stringify({ docId, text }));
