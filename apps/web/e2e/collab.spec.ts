import { type Browser, type BrowserContext, type Page, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';

/**
 * Faz 5 acceptance: two browser clients edit the same document live, see each
 * other's remote cursors, and an offline edit merges back on reconnect with no
 * data loss.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `e2e_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'E2E Room', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  token = await signSession({ id: owner.id, email: owner.email });
});

test.afterAll(async () => {
  // Let the ws-server finish its final-snapshot write for the closed room before
  // deleting, so we don't race its insert. onDelete: Cascade removes op log/snapshots.
  await new Promise((r) => setTimeout(r, 1500));
  await prisma.document.deleteMany({ where: { id: docId } });
  await prisma.user.deleteMany({ where: { id: ownerId } });
  await prisma.$disconnect();
});

async function openEditor(browser: Browser): Promise<{ ctx: BrowserContext; page: Page }> {
  const ctx = await browser.newContext();
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);
  const page = await ctx.newPage();
  await page.goto(`/doc/${docId}`);
  await page.waitForSelector('.cm-content');
  await expect(page.getByTestId('conn-status')).toContainText('connected');
  return { ctx, page };
}

test('two clients co-edit with remote cursors and offline merge', async ({ browser }) => {
  const a = await openEditor(browser);
  const b = await openEditor(browser);

  // 1) Live edit: A types, B sees it.
  await a.page.locator('.cm-content').click();
  await a.page.keyboard.type('Hello from A. ');
  await expect(b.page.locator('.cm-content')).toContainText('Hello from A.');

  // 2) B types, A sees it — and A now shows B's remote caret.
  await b.page.locator('.cm-content').click();
  await b.page.keyboard.type('And hi from B.');
  await expect(a.page.locator('.cm-content')).toContainText('And hi from B.');
  await expect(a.page.locator('.cm-ySelectionCaret')).not.toHaveCount(0);

  // 3) Offline edit on B merges back on reconnect (no data loss).
  await b.ctx.setOffline(true);
  await b.page.keyboard.type(' [OFFLINE]');
  await a.page.waitForTimeout(800);
  await expect(a.page.locator('.cm-content')).not.toContainText('[OFFLINE]');

  await b.ctx.setOffline(false);
  await expect(a.page.locator('.cm-content')).toContainText('[OFFLINE]');
  // Earlier content survived the merge.
  await expect(a.page.locator('.cm-content')).toContainText('Hello from A.');
  await expect(a.page.locator('.cm-content')).toContainText('And hi from B.');

  await a.ctx.close();
  await b.ctx.close();
});
