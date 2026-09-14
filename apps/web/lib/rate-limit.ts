/**
 * Minimal in-memory fixed-window rate limiter (Faz 8 hardening). Single-node
 * scope — matches the app's single-process authoritative model (invariant #4);
 * a horizontally-scaled deployment would swap this for Redis. Used to throttle
 * auth attempts (brute-force / credential-stuffing defense) keyed by IP and email.
 */
interface Window {
  count: number;
  resetAt: number;
}

const windows = new Map<string, Window>();

const SWEEP_THRESHOLD = 5000; // start pruning only when map is large — sweep itself is O(n)

/** Prune expired windows opportunistically so the map can't grow unbounded. */
function sweep(now: number): void {
  if (windows.size < SWEEP_THRESHOLD) return;
  for (const [key, w] of windows) if (now >= w.resetAt) windows.delete(key);
}

export interface RateResult {
  ok: boolean;
  /** Seconds until the window resets (only meaningful when !ok). */
  retryAfter: number;
}

/**
 * Consume one unit against `key`. Returns ok=false once `limit` is reached within
 * `windowMs`; the window resets `windowMs` after the first hit.
 */
export function rateLimit(key: string, limit: number, windowMs: number): RateResult {
  const now = Date.now();
  sweep(now);
  const w = windows.get(key);
  if (!w || now >= w.resetAt) {
    windows.set(key, { count: 1, resetAt: now + windowMs });
    return { ok: true, retryAfter: 0 };
  }
  if (w.count >= limit) {
    return { ok: false, retryAfter: Math.ceil((w.resetAt - now) / 1000) };
  }
  w.count += 1;
  return { ok: true, retryAfter: 0 };
}

/** Test/util: clear all windows. */
export function resetRateLimits(): void {
  windows.clear();
}
