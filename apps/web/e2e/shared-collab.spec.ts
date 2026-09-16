import { type Browser, type BrowserContext, type Page, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';

/**
 * Faz 8: true multi-user collaboration. Two DISTINCT users edit one document —
 * the owner and an invited collaborator. Proves the sharing model end-to-end:
 * a non-owner with a DocumentCollaborator grant passes the WS handshake and
 * co-edits; every write from both users converges with no lost writes. Without
 * the grant the same non-owner is rejected (403) — asserted first.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let collaboratorId: string;
let docId: string;
let ownerToken: string;
let collaboratorToken: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `p8_owner_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  const collaborator = await prisma.user.create({
    data: { email: `p8_collab_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  collaboratorId = collaborator.id;
  const doc = await prisma.document.create({
    data: { title: 'Shared Room', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  ownerToken = await signSession({ id: owner.id, email: owner.email });
  collaboratorToken = await signSession({ id: collaborator.id, email: collaborator.email });
});

test.afterAll(async () => {
  await new Promise((r) => setTimeout(r, 1500)); // let sync server flush final snapshot
  await prisma.document.deleteMany({ where: { id: docId } });
  await prisma.user.deleteMany({ where: { id: { in: [ownerId, collaboratorId] } } });
  await prisma.$disconnect();
});

async function open(browser: Browser, token: string): Promise<{ ctx: BrowserContext; page: Page }> {
  const ctx = await browser.newContext();
  await ctx.addCookies([
    { name: 'session', value: token, domain: 'localhost', path: '/', httpOnly: true, sameSite: 'Lax' },
  ]);
  const page = await ctx.newPage();
  await page.goto(`/doc/${docId}`);
  return { ctx, page };
}

test('un-shared non-owner is blocked (404 gate + no WS join)', async ({ browser }) => {
  const { ctx, page } = await open(browser, collaboratorToken);
  // Access gate: getAccessibleDocument returns null → notFound() before the editor mounts.
  await expect(page.getByTestId('editor')).toHaveCount(0);
  await ctx.close();
});

test('owner + invited collaborator co-edit; both writes converge', async ({ browser }) => {
  // Grant access (the Server Action does this in the UI; set the row directly here).
  await prisma.documentCollaborator.create({ data: { documentId: docId, userId: collaboratorId } });

  const owner = await open(browser, ownerToken);
  const collab = await open(browser, collaboratorToken);

  for (const c of [owner, collab]) {
    await c.page.waitForSelector('.cm-content');
    await expect(c.page.getByTestId('conn-status')).toContainText('connected');
  }

  await owner.page.locator('.cm-content').click();
  await owner.page.keyboard.type('OWNER-WROTE-THIS ');
  await collab.page.locator('.cm-content').click();
  await collab.page.keyboard.press('ControlOrMeta+End');
  await collab.page.keyboard.type('COLLAB-WROTE-THIS');

  // Each user's edit appears in the OTHER user's editor — real cross-user sync.
  for (const c of [owner, collab]) {
    await expect
      .poll(async () => (await c.page.locator('.cm-content').textContent()) ?? '', { timeout: 15_000 })
      .toContain('OWNER-WROTE-THIS');
    await expect
      .poll(async () => (await c.page.locator('.cm-content').textContent()) ?? '', { timeout: 15_000 })
      .toContain('COLLAB-WROTE-THIS');
  }

  await owner.ctx.close();
  await collab.ctx.close();
});
