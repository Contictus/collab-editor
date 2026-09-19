'use client';

import { useState, useTransition } from 'react';
import { restoreDocumentAction } from '../../../actions/documents';
import type { UpdateEntry } from '../../../../lib/audit-service';

/**
 * History restore button (F11-13): pick a replay point and restore the live
 * document to it. The restored text lands as new CRDT updates — history is
 * never rewritten. Confirm first: restore overwrites current live text.
 */
export function RestoreButton({
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
  const [at, setAt] = useState(points.at(-1)?.id ?? baseUpdateId);
  const [confirming, setConfirming] = useState(false);
  const [result, setResult] = useState<{ error?: string; ok?: string } | null>(null);
  const [pending, startTransition] = useTransition();

  if (points.length <= 1) return null;

  function restore() {
    startTransition(async () => {
      setResult(await restoreDocumentAction(docId, at));
      setConfirming(false);
    });
  }

  return (
    <div style={{ display: 'grid', gap: 8, maxWidth: 680 }}>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <select
          data-testid="restore-point"
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
        {!confirming ? (
          <button
            type="button"
            data-testid="restore-ask"
            onClick={() => setConfirming(true)}
            style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid #b45309', background: '#fff', color: '#b45309', cursor: 'pointer' }}
          >
            Restore…
          </button>
        ) : (
          <>
            <button
              type="button"
              data-testid="restore-confirm"
              onClick={restore}
              disabled={pending}
              style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid #b45309', background: '#b45309', color: '#fff', cursor: pending ? 'wait' : 'pointer' }}
            >
              {pending ? 'Restoring…' : 'Confirm restore'}
            </button>
            <button
              type="button"
              onClick={() => setConfirming(false)}
              style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid #ddd', background: '#fff', cursor: 'pointer' }}
            >
              Cancel
            </button>
          </>
        )}
      </div>
      {result?.error && <p style={{ color: 'crimson', margin: 0, fontSize: 13 }}>{result.error}</p>}
      {result?.ok && <p data-testid="restore-ok" style={{ color: '#22a565', margin: 0, fontSize: 13 }}>{result.ok}</p>}
    </div>
  );
}
