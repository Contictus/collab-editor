import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E (Faz 5–7). Boots BOTH processes itself — the Next web app (:3000)
 * and the sync server — via the webServer array, polling each health endpoint
 * before running. `reuseExistingServer` means a dev stack you already have up
 * is reused instead of double-booting. Serial, single worker: the collab tests
 * coordinate multiple browser contexts against shared server state.
 *
 * Requires Postgres up (docker compose up -d) and migrations applied.
 *
 * WS_NODE=1 runs the suite against the legacy Node ws-server (:1234) instead
 * of Go (:8080, the default). Both servers read the root .env; exported
 * DATABASE_URL + JWT_SECRET take precedence.
 */
const useGo = process.env.WS_NODE !== '1';
const wsPort = useGo ? 8080 : 1234;
export default defineConfig({
  testDir: './e2e',
  timeout: 40_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'off',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    useGo
      ? {
          command: 'go -C ../../services/api-go run ./cmd/server serve-ws',
          url: 'http://localhost:8080/health',
          reuseExistingServer: true,
          timeout: 120_000,
          stdout: 'ignore',
          stderr: 'pipe',
          env: { ...process.env, API_PORT: '8080' },
        }
      : {
          command: 'pnpm --filter ws-server dev',
          url: 'http://localhost:1234/health',
          reuseExistingServer: true,
          timeout: 60_000,
          stdout: 'ignore',
          stderr: 'pipe',
        },
    {
      command: 'pnpm --filter web dev',
      url: 'http://localhost:3000/api/health',
      reuseExistingServer: true,
      timeout: 120_000,
      stdout: 'ignore',
      stderr: 'pipe',
      env: {
        ...process.env,
        NEXT_PUBLIC_WS_URL: `ws://localhost:${wsPort}`,
      },
    },
  ],
});
