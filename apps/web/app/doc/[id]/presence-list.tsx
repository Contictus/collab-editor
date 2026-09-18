'use client';

import type { PresenceUser } from '../../../lib/presence';

/**
 * Online collaborators (F10-8). Avatar dots + names from awareness states.
 * Ephemeral by design — never persisted (INVARIANTS uyarısı).
 */
export function PresenceList({ users }: { users: PresenceUser[] }) {
  return (
    <div data-testid="presence" style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: '#555' }}>
      <span data-testid="presence-count" title="Online collaborators">
        ● {users.length + 1} online
      </span>
      <span style={{ display: 'flex', gap: 6 }}>
        {users.map((u) => (
          <span
            key={u.clientId}
            data-testid={`presence-user-${u.clientId}`}
            title={u.name}
            style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}
          >
            <span
              style={{
                width: 12,
                height: 12,
                borderRadius: '50%',
                background: u.color,
                border: '1px solid #0002',
                display: 'inline-block',
              }}
            />
            {u.name}
          </span>
        ))}
      </span>
    </div>
  );
}
