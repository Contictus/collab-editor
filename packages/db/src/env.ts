import { existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { config } from 'dotenv';

/**
 * Monorepo env loader. web (Next, cwd=apps/web) and scripts (cwd varies) all
 * live under the repo root where the single `.env` sits. Walk up from cwd to
 * find it, so both processes and the Prisma CLI share one env file — no per-app
 * duplication of DATABASE_URL / JWT_SECRET.
 */
export function loadRootEnv(): void {
  let dir = process.cwd();
  for (let i = 0; i < 6; i++) {
    const candidate = join(dir, '.env');
    if (existsSync(candidate)) {
      config({ path: candidate });
      return;
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  // No .env found — rely on process env (CI/Docker). Caller should ensure DATABASE_URL/JWT_SECRET are set.
}
