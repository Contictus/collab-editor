import { type Browser, type BrowserContext, type Page, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';

/**
 * F9 acceptance: typed Markdown renders live in the preview pane, the toolbar
 * formats through the shared doc, and view modes toggle without killing sync.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `f9_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'F9 Room', ownerId: owner.id },
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

test('typed markdown renders live in preview', async ({ browser }) => {
  const { ctx, page } = await openEditor(browser);
  await page.locator('.cm-content').click();
  await page.keyboard.type('# Hello F9');
  await expect(page.getByTestId('preview').locator('h1')).toContainText('Hello F9');
  await ctx.close();
});

test('toolbar bold wraps selection and preview shows it', async ({ browser }) => {
  const { ctx, page } = await openEditor(browser);
  await page.locator('.cm-content').click();
  await page.keyboard.type('boldme');
  await page.keyboard.press('ControlOrMeta+a');
  await page.getByTestId('toolbar-bold').click();
  await expect(page.getByTestId('preview').locator('strong')).toContainText('boldme');
  await ctx.close();
});

test('view modes toggle without losing content', async ({ browser }) => {
  const { ctx, page } = await openEditor(browser);
  await page.locator('.cm-content').click();
  await page.keyboard.type('mode check');
  await page.getByTestId('view-preview').click();
  await expect(page.getByTestId('preview')).toContainText('mode check');
  await page.getByTestId('view-edit').click();
  await expect(page.locator('.cm-content')).toContainText('mode check');
  await page.getByTestId('view-split').click();
  await expect(page.getByTestId('preview')).toContainText('mode check');
  await ctx.close();
});
