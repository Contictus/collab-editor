import { prisma } from 'db';
import { registerUser } from '../lib/auth-service';
import {
  createDocument,
  getOwnedDocument,
  listDocuments,
  loadInitialText,
} from '../lib/document-service';

/**
 * Faz 2 integration smoke vs real Postgres: owner-scoped CRUD + SSR bootstrap.
 *   pnpm --filter web exec tsx scripts/faz2-smoke.ts
 */
let failures = 0;
function check(name: string, cond: boolean) {
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}`);
  if (!cond) failures++;
}

async function main() {
  const owner = await registerUser({ email: `doc_${Date.now()}@ex.com`, password: 'password123' });
  const other = await registerUser({ email: `other_${Date.now()}@ex.com`, password: 'password123' });

  const a = await createDocument(owner.id, 'First doc');
  const b = await createDocument(owner.id, 'Second doc');

  const list = await listDocuments(owner.id);
  check('owner lists both documents', list.length === 2);
  check('other user lists none', (await listDocuments(other.id)).length === 0);

  check('owner can fetch own document', (await getOwnedDocument(a.id, owner.id))?.id === a.id);
  check('non-owner cannot fetch document', (await getOwnedDocument(a.id, other.id)) === null);

  check('SSR bootstrap text is empty for new doc', (await loadInitialText(b.id)) === '');

  // cleanup
  await prisma.document.deleteMany({ where: { ownerId: { in: [owner.id, other.id] } } });
  await prisma.user.deleteMany({ where: { id: { in: [owner.id, other.id] } } });
  await prisma.$disconnect();

  console.log(failures === 0 ? '\nALL PASS' : `\n${failures} FAILED`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
