import { type Browser, type BrowserContext, type Page, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';

/**
 * F10 acceptance: offline-first replica + presence.
 * - Reload restores typed content from IndexedDB (local-first boot).
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `f10_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'F10 Room', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  token = await signSession({ id: owner.id, email: owner.email });
});

test.afterAll(async () => {
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

test('reload restores content from local replica', async ({ browser }) => {
  const { ctx, page } = await openEditor(browser);
  await page.locator('.cm-content').click();
  await page.keyboard.type('offline-marker-F10');
  await expect(page.getByTestId('preview')).toContainText('offline-marker-F10');
  await page.waitForTimeout(1000);
  await page.reload();
  await page.waitForSelector('.cm-content');
  await expect(page.locator('.cm-content')).toContainText('offline-marker-F10');
  await ctx.close();
});

test('two clients see each other in presence', async ({ browser }) => {
  const a = await openEditor(browser);
  const b = await openEditor(browser);
  await expect(a.page.getByTestId('presence-count')).toContainText('2 online');
  await expect(b.page.getByTestId('presence-count')).toContainText('2 online');
  await a.ctx.close();
  await b.ctx.close();
});
