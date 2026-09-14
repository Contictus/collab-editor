import { PrismaClient } from '@prisma/client';
import { loadRootEnv } from './env';

// Ensure DATABASE_URL (root .env) is present before the client initializes,
// regardless of which app process imported us.
loadRootEnv();

/**
 * Single shared Prisma client. Both apps import this; the schema lives here so
 * web and ws-server use the same model definitions (see CLAUDE.md layout rule).
 * Guard against multiple instances during Next.js dev hot-reload — without this,
 * every HMR reload would leak a DB connection pool.
 */
const globalForPrisma = globalThis as unknown as { prisma?: PrismaClient };

export const prisma =
  globalForPrisma.prisma ??
  new PrismaClient({
    log: process.env.NODE_ENV === 'development' ? ['warn', 'error'] : ['error'],
  });

if (process.env.NODE_ENV !== 'production') {
  globalForPrisma.prisma = prisma;
}

export * from '@prisma/client';
