import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    include: ['lib/**/*.test.ts', 'test/**/*.test.ts'],
    // JWT_SECRET needed by protocol's sign/verify; DB is never touched by these tests.
    env: { JWT_SECRET: 'test-secret-for-vitest' },
  },
});
