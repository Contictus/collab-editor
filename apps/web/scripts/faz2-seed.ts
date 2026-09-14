import * as Y from 'yjs';
import { prisma } from 'db';
import { TEXT_KEY } from 'shared/crdt';
import { registerUser } from '../lib/auth-service';
import { createDocument } from '../lib/document-service';

/**
 * Seeds a user, a second user, and one document whose op log holds a single real
 * Yjs update ("Hello from SSR") so the /doc/[id] SSR bootstrap renders non-empty
 * persisted content. Prints ids as JSON for the HTTP e2e step. Cleaned up after.
 */
async function main() {
  const stamp = Date.now();
  const owner = await registerUser({ email: `seed_${stamp}@ex.com`, password: 'password123' });
  const other = await registerUser({ email: `seedother_${stamp}@ex.com`, password: 'password123' });
  const doc = await createDocument(owner.id, 'Seeded Document');

  const ydoc = new Y.Doc();
  ydoc.getText(TEXT_KEY).insert(0, 'Hello from SSR');
  const update = Y.encodeStateAsUpdate(ydoc);
  await prisma.documentUpdate.create({
    data: { documentId: doc.id, update: Buffer.from(update), clock: 0 },
  });

  await prisma.$disconnect();
  console.log(JSON.stringify({ ownerId: owner.id, otherId: other.id, docId: doc.id }));
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
