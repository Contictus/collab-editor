/**
 * Presence list derivation (F10-6). Pure: awareness states in, renderable
 * users out. The local client is excluded by id — presence is about *others*.
 */

export interface AwarenessUser {
  name: string;
  color: string;
}

export interface PresenceUser extends AwarenessUser {
  clientId: number;
}

export function presenceList(
  states: Map<number, { user?: AwarenessUser }>,
  localClientId: number,
): PresenceUser[] {
  const out: PresenceUser[] = [];
  for (const [clientId, state] of states) {
    if (clientId === localClientId) continue;
    if (!state.user) continue;
    out.push({ clientId, name: state.user.name, color: state.user.color });
  }
  return out.sort((a, b) => a.clientId - b.clientId);
}
