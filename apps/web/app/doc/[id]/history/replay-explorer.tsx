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
    <div style={{ display: 'grid', gap: 12, maxWidth: 680 }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '10px 12px',
          border: '1px solid #e8e8e8',
          borderRadius: 8,
          background: '#fafaf7',
        }}
      >
        <span style={{ fontSize: 12, letterSpacing: 0.3, textTransform: 'uppercase', color: '#888' }}>
          Replay at
        </span>
        <select
          data-testid="replay-point"
          value={at}
          onChange={(e) => setAt(e.target.value)}
          style={{ flex: 1, padding: '6px 8px', border: '1px solid #ddd', borderRadius: 6, background: '#fff' }}
        >
          {points.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
        <button
          type="button"
          onClick={() => replay(at)}
          disabled={loading}
          style={{
            padding: '6px 12px',
            borderRadius: 6,
            border: '1px solid #111',
            background: '#111',
            color: '#fff',
            cursor: loading ? 'wait' : 'pointer',
          }}
        >
          {loading ? 'Replaying…' : 'Replay'}
        </button>
      </div>
      <p style={{ margin: 0, color: '#999', fontSize: 12 }}>
        Picks update #id in the surviving op-log and reconstructs text at that point (snapshot + updates up to id).
      </p>
      {error && <p style={{ color: 'crimson', margin: 0, fontSize: 13 }}>{error}</p>}
      {text !== null && (
        <pre
          data-testid="replay-text"
          style={{
            border: '1px solid #ddd',
            borderRadius: 8,
            padding: 12,
            background: '#fff',
            whiteSpace: 'pre-wrap',
            margin: 0,
            minHeight: 80,
            boxShadow: '0 1px 4px #0000000a',
          }}
        >
          {text === '' ? '(empty document)' : text}
        </pre>
      )}
    </div>
  );
}
