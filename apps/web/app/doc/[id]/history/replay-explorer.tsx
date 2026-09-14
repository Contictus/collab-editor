'use client';

import { useState } from 'react';
import type { UpdateEntry } from '../../../../lib/audit-service';

/**
 * Replay explorer (Faz 6): pick a point in the surviving op log and fetch the
 * reconstructed text from the replay endpoint. Demonstrates event-sourcing —
 * "state at update K" — over the live history without touching the editor.
 */
export function ReplayExplorer({
  docId,
  baseUpdateId,
  updates,
}: {
  docId: string;
  baseUpdateId: string;
  updates: UpdateEntry[];
}) {
  // Replayable points: the base snapshot (0/base) plus each surviving update id.
  const points = [
    { id: baseUpdateId, label: `base (snapshot @ ${baseUpdateId})` },
    ...updates.map((u) => ({ id: u.id, label: `update #${u.id} (clock ${u.clock})` })),
  ];
  const [at, setAt] = useState(points.at(-1)?.id ?? baseUpdateId);
  const [text, setText] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function replay(target: string) {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/documents/${docId}/replay?at=${target}`);
      if (!res.ok) {
        setText(null);
        setError(`replay failed (${res.status})`);
        return;
      }
      const data = (await res.json()) as { text: string };
      setText(data.text);
    } catch {
      setText(null);
      setError('network error');
    } finally {
      setLoading(false);
    }
  }

  if (points.length <= 1) {
    return <p style={{ color: '#777' }}>No surviving updates to replay yet.</p>;
  }

  return (
    <div style={{ display: 'grid', gap: 10, maxWidth: 640 }}>
      <label style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        Replay at{' '}
        <select
          data-testid="replay-point"
          value={at}
          onChange={(e) => setAt(e.target.value)}
        >
          {points.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
        <button type="button" onClick={() => replay(at)} disabled={loading}>
          {loading ? 'Replaying…' : 'Replay'}
        </button>
      </label>
      {error && <p style={{ color: 'crimson', margin: 0 }}>{error}</p>}
      {text !== null && (
        <pre
          data-testid="replay-text"
          style={{
            border: '1px solid #ddd',
            borderRadius: 6,
            padding: 12,
            background: '#fafafa',
            whiteSpace: 'pre-wrap',
            margin: 0,
          }}
        >
          {text === '' ? '(empty)' : text}
        </pre>
      )}
    </div>
  );
}
