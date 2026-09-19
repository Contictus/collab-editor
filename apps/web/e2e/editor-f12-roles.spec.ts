import { type Browser, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';
import { TEXT_KEY } from 'shared/crdt';
import * as Y from 'yjs';

/**
 * F12 acceptance (part 2): viewer role is read-only (banner, no toolbar,
 * non-editable, but live reads stream in) and public links serve text
 * without a session.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let guestId: string;
let docId: string;
let ownerToken: string;
let guestToken: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `f12o_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const guest = await prisma.user.create({
    data: { email: `f12g_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  guestId = guest.id;
  const doc = await prisma.document.create({
    data: { title: 'F12 Room', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  ownerToken = await signSession({ id: owner.id, email: owner.email });
  guestToken = await signSession({ id: guest.id, email: guest.email });

  await prisma.documentCollaborator.create({ data: { documentId: docId, userId: guest.id, role: 'viewer' } });

  const ydoc = new Y.Doc();
  const text = ydoc.getText(TEXT_KEY);
  const captured: Uint8Array[] = [];
  ydoc.on('update', (u: Uint8Array) => captured.push(u));
  text.insert(0, 'shared text');
  ydoc.destroy();
  for (let i = 0; i < captured.length; i++) {
    await prisma.documentUpdate.create({
      data: { documentId: docId, update: Buffer.from(captured[i]!), clock: i },
    });
  }
});

test.afterAll(async () => {
  await new Promise((r) => setTimeout(r, 1500));
  await prisma.documentUpdate.deleteMany({ where: { documentId: docId } });
  await prisma.documentCollaborator.deleteMany({ where: { documentId: docId } });
  await prisma.document.deleteMany({ where: { id: docId } });
  await prisma.user.deleteMany({ where: { id: { in: [ownerId, guestId] } } });
  await prisma.$disconnect();
});

async function ctxWith(browser: Browser, token: string) {
  const ctx = await browser.newContext();
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);
  return ctx;
}

test('viewer sees banner, no toolbar, but live text', async ({ browser }) => {
  const ctx = await ctxWith(browser, guestToken);
  const page = await ctx.newPage();
  await page.goto(`/doc/${docId}`);
  await page.waitForSelector('.cm-content');
  await expect(page.getByTestId('viewer-banner')).toBeVisible();
  await expect(page.getByTestId('toolbar')).toHaveCount(0);
  await expect(page.locator('.cm-content')).toHaveAttribute('contenteditable', 'false');
  await expect(page.locator('.cm-content')).toContainText('shared text');
  await ctx.close();
});

test('public link serves text without session', async ({ browser }) => {
  const ownerCtx = await ctxWith(browser, ownerToken);
  const ownerPage = await ownerCtx.newPage();
  await ownerPage.goto(`/doc/${docId}`);
  await ownerPage.getByTestId('public-link-enable').click();
  const urlInput = ownerPage.getByTestId('public-link-url');
  await expect(urlInput).toBeVisible();
  const path = (await urlInput.inputValue()).trim();
  await ownerCtx.close();

  const anon = await browser.newContext();
  const page = await anon.newPage();
  await page.goto(path);
  await expect(page.getByTestId('public-badge')).toBeVisible();
  await expect(page.getByTestId('public-text')).toContainText('shared text');
  await anon.close();
});
