'use client';

import { useState, useTransition } from 'react';
import {
  disablePublicLinkAction,
  enablePublicLinkAction,
  type PublicLinkState,
} from '../../actions/documents';

/**
 * Public read-only link toggle (F12-22). Owner-only. The token is the
 * capability — anyone with the link reads title+text, no session, no sync.
 */
export function PublicLinkControls({
  docId,
  initialPublicId,
}: {
  docId: string;
  initialPublicId: string | null;
}) {
  const [publicId, setPublicId] = useState(initialPublicId);
  const [state, setState] = useState<PublicLinkState | null>(null);
  const [pending, startTransition] = useTransition();

  function run(action: (docId: string) => Promise<PublicLinkState>) {
    startTransition(async () => {
      const res = await action(docId);
      setState(res);
      if (res.publicId !== undefined) setPublicId(res.publicId);
    });
  }

  return (
    <section
      style={{ display: 'grid', gap: 8, maxWidth: 480, margin: '10px 0', padding: 12, border: '1px solid #e8e8e8', borderRadius: 8, background: '#fafaf7' }}
    >
      <div style={{ fontSize: 12, letterSpacing: 0.3, textTransform: 'uppercase', color: '#888' }}>
        Public link — owner only
      </div>
      {publicId ? (
        <>
          <input
            readOnly
            data-testid="public-link-url"
            value={`/p/${publicId}`}
            onFocus={(e) => e.target.select()}
            style={{ padding: '6px 8px', border: '1px solid #ddd', borderRadius: 6, background: '#fff', fontSize: 13 }}
          />
          <div>
            <button
              type="button"
              data-testid="public-link-disable"
              onClick={() => run(disablePublicLinkAction)}
              disabled={pending}
              style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid #ddd', background: '#fff', cursor: pending ? 'wait' : 'pointer', fontSize: 13 }}
            >
              Disable link
            </button>
          </div>
        </>
      ) : (
        <div>
          <button
            type="button"
            data-testid="public-link-enable"
            onClick={() => run(enablePublicLinkAction)}
            disabled={pending}
            style={{ padding: '6px 12px', borderRadius: 6, border: '1px solid #111', background: '#111', color: '#fff', cursor: pending ? 'wait' : 'pointer', fontSize: 13 }}
          >
            {pending ? 'Enabling…' : 'Enable public link'}
          </button>
        </div>
      )}
      {state?.error && <p style={{ color: 'crimson', margin: 0, fontSize: 13 }}>{state.error}</p>}
      {state?.ok && <p style={{ color: '#1a7a4a', margin: 0, fontSize: 13 }}>{state.ok}</p>}
    </section>
  );
}
