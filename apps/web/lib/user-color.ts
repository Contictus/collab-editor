/**
 * Deterministic display name + color for awareness presence, derived from email.
 * Same user → stable color; good enough for remote-cursor identity (Faz 5).
 */
export function displayName(email: string): string {
  return email.split('@')[0] || email;
}

export function userColor(seed: string): string {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = (hash << 5) - hash + seed.charCodeAt(i);
    hash |= 0;
  }
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue} 70% 45%)`;
}
