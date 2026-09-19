'use client';

import { useState } from 'react';
import { diffLines, type DiffLine } from '../../../../lib/diff';
import type { UpdateEntry } from '../../../../lib/audit-service';

/**
 * History compare (F11-8): pick two replay points, fetch both texts, render a
 * line diff. Read-only — uses the existing replay endpoint twice.
 */
export function CompareView({
  docId,
  baseUpdateId,
  updates,
}: {
  docId: string;
  baseUpdateId: string;
  updates: UpdateEntry[];
}) {
  const points = [
    { id: baseUpdateId, label: `base (snapshot @ ${baseUpdateId})` },
    ...updates.map((u) => ({ id: u.id, label: `update #${u.id} (clock ${u.clock})` })),
  ];
  const [from, setFrom] = useState(points[0]?.id ?? baseUpdateId);
  const [to, setTo] = useState(points.at(-1)?.id ?? baseUpdateId);
  const [diff, setDiff] = useState<DiffLine[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function fetchText(target: string): Promise<string> {
    const res = await fetch(`/api/documents/${docId}/replay?at=${target}`);
    if (!res.ok) throw new Error(`replay ${target} failed (${res.status})`);
    const data = (await res.json()) as { text: string };
    return data.text;
  }

  async function compare() {
    setLoading(true);
    setError(null);
    try {
      const [a, b] = await Promise.all([fetchText(from), fetchText(to)]);
      setDiff(diffLines(a, b));
    } catch {
      setDiff(null);
      setError('compare failed');
    } finally {
      setLoading(false);
    }
  }

  if (points.length <= 1) {
    return <p style={{ color: '#777' }}>Need at least two points to compare.</p>;
  }

  const sel = { flex: 1, padding: '6px 8px', border: '1px solid #ddd', borderRadius: 6, background: '#fff' };

  return (
    <div style={{ display: 'grid', gap: 12, maxWidth: 680 }}>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <select data-testid="compare-from" value={from} onChange={(e) => setFrom(e.target.value)} style={sel}>
          {points.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
        <span style={{ color: '#999' }}>→</span>
        <select data-testid="compare-to" value={to} onChange={(e) => setTo(e.target.value)} style={sel}>
          {points.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
        <button
          type="button"
          data-testid="compare-run"
          onClick={compare}
          disabled={loading}
          style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid #111', background: '#111', color: '#fff', cursor: loading ? 'wait' : 'pointer' }}
        >
          {loading ? 'Comparing…' : 'Compare'}
        </button>
      </div>
      {error && <p style={{ color: 'crimson', margin: 0, fontSize: 13 }}>{error}</p>}
      {diff !== null && (
        <pre
          data-testid="compare-diff"
          style={{ border: '1px solid #ddd', borderRadius: 8, padding: 12, background: '#fff', whiteSpace: 'pre-wrap', margin: 0, fontSize: 13 }}
        >
          {diff.map((l, i) => (
            <div
              key={i}
              data-kind={l.kind}
              style={{
                background: l.kind === 'add' ? '#e6ffec' : l.kind === 'del' ? '#ffebe9' : 'transparent',
                padding: '1px 6px',
              }}
            >
              {l.kind === 'add' ? '+ ' : l.kind === 'del' ? '− ' : '  '}
              {l.text}
            </div>
          ))}
        </pre>
      )}
    </div>
  );
}
