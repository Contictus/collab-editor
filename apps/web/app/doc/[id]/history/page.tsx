import Link from 'next/link';
import { notFound } from 'next/navigation';
import { getAuditLog } from '../../../../lib/audit-service';
import { getAccessibleDocument } from '../../../../lib/document-service';
import { requireSession } from '../../../../lib/session';
import { displayName } from '../../../../lib/user-color';
import { ReplayExplorer } from './replay-explorer';
import { CompareView } from './compare-view';

/**
 * Audit view (Faz 6) — the persisted event history of a document: op-log update
 * count, snapshot checkpoints, and an interactive replay of "state at update K".
 * Owner-scoped (only the sole authorized editor); unknown/non-owned id → 404.
 */
export default async function HistoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const user = await requireSession();

  const doc = await getAccessibleDocument(id, user.id);
  if (!doc) notFound();

  const audit = await getAuditLog(doc.id, doc.title);
  const editor = displayName(user.email);

  const cell: React.CSSProperties = { padding: '4px 10px', borderBottom: '1px solid #eee', textAlign: 'left' };

  return (
    <main style={{ display: 'grid', gap: 20 }}>
      <p style={{ margin: 0 }}>
        <Link href={`/doc/${doc.id}`}>← Editor</Link>
      </p>
      <h1 style={{ margin: 0 }}>History · {audit.title}</h1>

      <section data-testid="audit-summary" style={{ color: '#444' }}>
        <p style={{ margin: 0 }}>
          <strong>{audit.counts.liveUpdates}</strong> live update(s),{' '}
          <strong>{audit.counts.snapshots}</strong> snapshot checkpoint(s). Viewing as{' '}
          <strong>{editor}</strong> — the owner and invited collaborators share one op log.
        </p>
        <p style={{ margin: '4px 0 0', fontSize: 13, color: '#777' }}>
          The op log is compacted into snapshots and pruned (invariant #3), so granular replay
          survives only after the latest snapshot boundary (update #{audit.baseUpdateId}).
        </p>
      </section>

      <section style={{ display: 'grid', gap: 8 }}>
        <h2 style={{ margin: 0, fontSize: 16 }}>Replay</h2>
        <ReplayExplorer
          docId={doc.id}
          baseUpdateId={audit.baseUpdateId}
          updates={audit.updates}
        />
      </section>

      <section style={{ display: 'grid', gap: 8 }}>
        <h2 style={{ margin: 0, fontSize: 16 }}>Compare</h2>
        <CompareView
          docId={doc.id}
          baseUpdateId={audit.baseUpdateId}
          updates={audit.updates}
        />
      </section>

      <section style={{ display: 'grid', gap: 8 }}>
        <h2 style={{ margin: 0, fontSize: 16 }}>Snapshots</h2>
        {audit.snapshots.length === 0 ? (
          <p style={{ color: '#777', margin: 0 }}>None yet.</p>
        ) : (
          <table style={{ borderCollapse: 'collapse', fontSize: 14 }}>
            <thead>
              <tr>
                <th style={cell}>id</th>
                <th style={cell}>covers up to</th>
                <th style={cell}>bytes</th>
                <th style={cell}>at</th>
              </tr>
            </thead>
            <tbody>
              {audit.snapshots.map((s) => (
                <tr key={s.id}>
                  <td style={cell}>{s.id}</td>
                  <td style={cell}>#{s.upToUpdateId}</td>
                  <td style={cell}>{s.bytes}</td>
                  <td style={cell}>{new Date(s.createdAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section style={{ display: 'grid', gap: 8 }}>
        <h2 style={{ margin: 0, fontSize: 16 }}>Live op log</h2>
        {audit.updates.length === 0 ? (
          <p style={{ color: '#777', margin: 0 }}>Empty (all compacted into snapshots).</p>
        ) : (
          <table style={{ borderCollapse: 'collapse', fontSize: 14 }}>
            <thead>
              <tr>
                <th style={cell}>id</th>
                <th style={cell}>clock</th>
                <th style={cell}>bytes</th>
                <th style={cell}>at</th>
              </tr>
            </thead>
            <tbody>
              {audit.updates.map((u) => (
                <tr key={u.id}>
                  <td style={cell}>{u.id}</td>
                  <td style={cell}>{u.clock}</td>
                  <td style={cell}>{u.bytes}</td>
                  <td style={cell}>{new Date(u.createdAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </main>
  );
}
