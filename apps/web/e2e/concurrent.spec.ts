import { type Browser, type BrowserContext, type Page, expect, test } from '@playwright/test';
import { prisma } from 'db';
import { signSession } from 'protocol';

/**
 * Faz 7 hardening: many clients editing one room converge with no lost writes,
 * including simultaneous inserts at the same caret position (the case naive
 * last-write-wins would corrupt). CRDT guarantees every client ends identical and
 * every character survives.
 */
test.describe.configure({ mode: 'serial' });

let ownerId: string;
let docId: string;
let token: string;

test.beforeAll(async () => {
  const stamp = Date.now();
  const owner = await prisma.user.create({
    data: { email: `p7_${stamp}@ex.com`, password: 'x' },
    select: { id: true, email: true },
  });
  ownerId = owner.id;
  const doc = await prisma.document.create({
    data: { title: 'Concurrency Room', ownerId: owner.id },
    select: { id: true },
  });
  docId = doc.id;
  token = await signSession({ id: owner.id, email: owner.email });
});

test.afterAll(async () => {
  // Let the ws-server flush its final snapshot for the closed room (cascade deletes op log).
  await new Promise((r) => setTimeout(r, 1500));
  await prisma.document.deleteMany({ where: { id: docId } });
  await prisma.user.deleteMany({ where: { id: ownerId } });
  await prisma.$disconnect();
});

interface Client {
  ctx: BrowserContext;
  page: Page;
}

async function open(browser: Browser): Promise<Client> {
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

/**
 * Poll until every client's editor contains every expected fragment. We assert
 * containment (the no-lost-write property) rather than DOM-string equality across
 * pages: y-codemirror injects each viewer's *remote cursor* decorations into
 * `.cm-content`, so the rendered DOM text legitimately differs per client even
 * when the underlying CRDT document is identical. Convergence of the document
 * itself is proven at the Yjs level by scripts/faz7-load.ts.
 */
async function waitAllContain(clients: Client[], fragments: string[]): Promise<void> {
  await expect
    .poll(
      async () => {
        const texts = await Promise.all(clients.map((c) => c.page.locator('.cm-content').textContent()));
        return texts.every((t) => fragments.every((f) => (t ?? '').includes(f)));
      },
      { timeout: 15_000 },
    )
    .toBe(true);
}

test('four clients converge; simultaneous same-position inserts lose nothing', async ({ browser }) => {
  const markers = ['[Aaa]', '[Bbb]', '[Ccc]', '[Ddd]'];
  const clients: Client[] = [];
  for (let i = 0; i < markers.length; i++) clients.push(await open(browser));

  // 1) Concurrent appends: each client types its unique marker at the same time.
  await Promise.all(
    clients.map(async (c, i) => {
      await c.page.locator('.cm-content').click();
      await c.page.keyboard.press('ControlOrMeta+End');
      await c.page.keyboard.type(markers[i]!);
    }),
  );

  // Every client sees every marker — no write was lost to a concurrent one.
  await waitAllContain(clients, markers);

  // 2) Simultaneous inserts at position 0 — the classic conflict. All at caret 0.
  await Promise.all(
    clients.map(async (c, i) => {
      await c.page.locator('.cm-content').click();
      await c.page.keyboard.press('ControlOrMeta+Home');
      await c.page.keyboard.type(`${i}#`);
    }),
  );

  // Both the conflicting round-2 inserts AND the round-1 content survive on all clients.
  await waitAllContain(clients, [...markers, ...clients.map((_c, i) => `${i}#`)]);

  await Promise.all(clients.map((c) => c.ctx.close()));
});
