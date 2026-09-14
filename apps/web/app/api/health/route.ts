import { NextResponse } from 'next/server';
import { prisma } from 'db';

/**
 * Health check (Faz 0 acceptance). Verifies the web process is up and can reach
 * Postgres via the shared Prisma client.
 */
export async function GET() {
  try {
    await prisma.$queryRaw`SELECT 1`;
    return NextResponse.json({ status: 'ok', service: 'web', db: 'up' });
  } catch (err) {
    return NextResponse.json(
      { status: 'degraded', service: 'web', db: 'down', error: String(err) },
      { status: 503 },
    );
  }
}
