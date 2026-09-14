import { afterEach, describe, expect, it, vi } from 'vitest';
import { rateLimit, resetRateLimits } from './rate-limit';

describe('rateLimit', () => {
  afterEach(() => {
    resetRateLimits();
    vi.useRealTimers();
  });

  it('allows up to the limit then blocks within the window', () => {
    for (let i = 0; i < 3; i++) expect(rateLimit('k', 3, 1000).ok).toBe(true);
    const blocked = rateLimit('k', 3, 1000);
    expect(blocked.ok).toBe(false);
    expect(blocked.retryAfter).toBeGreaterThan(0);
  });

  it('resets after the window elapses', () => {
    vi.useFakeTimers();
    for (let i = 0; i < 3; i++) rateLimit('k', 3, 1000);
    expect(rateLimit('k', 3, 1000).ok).toBe(false);
    vi.advanceTimersByTime(1001);
    expect(rateLimit('k', 3, 1000).ok).toBe(true);
  });

  it('tracks distinct keys independently', () => {
    expect(rateLimit('a', 1, 1000).ok).toBe(true);
    expect(rateLimit('a', 1, 1000).ok).toBe(false);
    expect(rateLimit('b', 1, 1000).ok).toBe(true);
  });
});
