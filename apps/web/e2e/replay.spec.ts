import { expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';
import { TEXT_KEY } from 'shared/crdt';
import * as Y from 'yjs';

/**
 * Faz 6 acceptance: the replay endpoint reconstructs the correct text at a known
 * point. Seeds a deterministic op log directly, then drives the HTTP endpoints
 * (/api/documents/[id]/audit and .../replay?at=K) with the owner's session cookie.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;
let updateIds: string[] = [];
const steps = ['# Doc\n', 'alpha ', 'beta ', 'gamma'];
const expected: string[] = [];

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `p6e2e_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'Replay E2E', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  token = await signSession({ id: owner.id, email: owner.email });

  // Build one incremental Yjs update per step and persist as op-log rows.
  const ydoc = new Y.Doc();
  const text = ydoc.getText(TEXT_KEY);
  const captured: Uint8Array[] = [];
  ydoc.on('update', (u: Uint8Array) => captured.push(u));
  for (const s of steps) {
    text.insert(text.length, s);
    expected.push(text.toString());
  }
  ydoc.destroy();

  for (let i = 0; i < captured.length; i++) {
    const row = await prisma.documentUpdate.create({
      data: { documentId: docId, update: Buffer.from(captured[i]!), clock: i },
      select: { id: true },
    });
    updateIds.push(row.id.toString());
  }
});

test.afterAll(async () => {
  await prisma.documentUpdate.deleteMany({ where: { documentId: docId } });
  await prisma.documentSnapshot.deleteMany({ where: { documentId: docId } });
  await prisma.document.deleteMany({ where: { id: docId } });
  await prisma.user.deleteMany({ where: { id: ownerId } });
  await prisma.$disconnect();
});

test('replay endpoint reconstructs correct text at a known point', async ({ browser }) => {
  const ctx = await browser.newContext();
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);

  // Audit reflects the seeded op log.
  const auditRes = await ctx.request.get(`/api/documents/${docId}/audit`);
  expect(auditRes.ok()).toBeTruthy();
  const audit = await auditRes.json();
  expect(audit.counts.liveUpdates).toBe(steps.length);
  expect(audit.latestUpdateId).toBe(updateIds.at(-1));

  // Replay at each known point → exact cumulative text.
  for (let i = 0; i < updateIds.length; i++) {
    const res = await ctx.request.get(`/api/documents/${docId}/replay?at=${updateIds[i]}`);
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.text).toBe(expected[i]);
  }

  // at=0 → empty base state.
  const zero = await ctx.request.get(`/api/documents/${docId}/replay?at=0`);
  expect((await zero.json()).text).toBe('');

  // Unauthorized (no cookie) → 401.
  const anon = await browser.newContext();
  const denied = await anon.request.get(`/api/documents/${docId}/replay?at=1`);
  expect(denied.status()).toBe(401);
  await anon.close();

  await ctx.close();
});
