import { type Browser, type BrowserContext, type Page, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';

/**
 * F12 acceptance (part 1): export downloads the live Markdown source.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `f12_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'F12 Room', ownerId: owner.id },
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
  const ctx = await browser.newContext({ acceptDownloads: true });
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);
  const page = await ctx.newPage();
  await page.goto(`/doc/${docId}`);
  await page.waitForSelector('.cm-content');
  await expect(page.getByTestId('conn-status')).toContainText('connected');
  return { ctx, page };
}

test('markdown export downloads live source', async ({ browser }) => {
  const { ctx, page } = await openEditor(browser);
  await page.locator('.cm-content').click();
  await page.keyboard.type('# export me');
  await page.waitForTimeout(500);
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByTestId('export-md').click(),
  ]);
  const path = await download.path();
  expect(path).toBeTruthy();
  expect(download.suggestedFilename()).toMatch(/\.md$/);
  await ctx.close();
});
