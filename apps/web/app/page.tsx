import Link from 'next/link';
import { getSession } from '../lib/session';

export default async function HomePage() {
  const session = await getSession();
  return (
    <main>
      <h1>Collab Editor</h1>
      <p>Real-time collaborative Markdown editor (CRDT / Yjs).</p>
      {session ? (
        <p>
          Signed in as {session.email} — <Link href="/documents">Documents</Link>
        </p>
      ) : (
        <p>
          <Link href="/login">Sign in</Link> · <Link href="/register">Register</Link>
        </p>
      )}
    </main>
  );
}
