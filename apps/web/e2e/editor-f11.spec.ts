import { expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';
import { TEXT_KEY } from 'shared/crdt';
import * as Y from 'yjs';

/**
 * F11 acceptance: history compare renders a line diff between two points,
 * and restore rolls the live document back to a past point.
 * Seeds a deterministic op log (replay.spec pattern) — no WS timing.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;
const updateIds: string[] = [];

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `f11e2e_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'F11 Room', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  token = await signSession({ id: owner.id, email: owner.email });

  const ydoc = new Y.Doc();
  const text = ydoc.getText(TEXT_KEY);
  const captured: Uint8Array[] = [];
  ydoc.on('update', (u: Uint8Array) => captured.push(u));
  text.insert(0, 'version one');
  text.insert(text.length, ' version two');
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
  await new Promise((r) => setTimeout(r, 1500));
  await prisma.documentUpdate.deleteMany({ where: { documentId: docId } });
  await prisma.documentSnapshot.deleteMany({ where: { documentId: docId } });
  await prisma.document.deleteMany({ where: { id: docId } });
  await prisma.user.deleteMany({ where: { id: ownerId } });
  await prisma.$disconnect();
});

test('compare shows line diff between base and latest', async ({ browser }) => {
  const ctx = await browser.newContext();
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);
  const page = await ctx.newPage();
  await page.goto(`/doc/${docId}/history`);
  await page.getByTestId('compare-run').click();
  const diff = page.getByTestId('compare-diff');
  await expect(diff).toContainText('version one version two');
  await ctx.close();
});

test('restore rolls live text back to first point', async ({ browser }) => {
  const ctx = await browser.newContext();
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);
  const page = await ctx.newPage();
  await page.goto(`/doc/${docId}/history`);
  await page.getByTestId('restore-point').selectOption(updateIds[0]!);
  await page.getByTestId('restore-ask').click();
  await page.getByTestId('restore-confirm').click();
  await expect(page.getByTestId('restore-ok')).toContainText(`update #${updateIds[0]}`);

  await page.goto(`/doc/${docId}`);
  await page.waitForSelector('.cm-content');
  await expect(page.locator('.cm-content')).toContainText('version one');
  await ctx.close();
});
