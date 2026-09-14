'use client';

import { markdown } from '@codemirror/lang-markdown';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { useEffect, useRef, useState } from 'react';
import { yCollab } from 'y-codemirror.next';
import { WebsocketProvider } from 'y-websocket';
import * as Y from 'yjs';
import { TEXT_KEY } from 'shared/crdt';

type ConnState = 'connecting' | 'connected' | 'offline';

/**
 * Live collaborative editor (Faz 5). CodeMirror 6 + markdown, bound to a Yjs
 * Y.Text via y-codemirror.next (remote cursors/selections + awareness). The
 * WebsocketProvider connects to the standalone ws-server; the httpOnly session
 * cookie rides the upgrade request (same site), so the server authenticates the
 * handshake without a token in the URL.
 */
export function Editor({
  docId,
  userName,
  userColor,
}: {
  docId: string;
  userName: string;
  userColor: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [conn, setConn] = useState<ConnState>('connecting');
  const [synced, setSynced] = useState(false);

  useEffect(() => {
    const container = ref.current;
    if (!container) return;

    const wsUrl = process.env.NEXT_PUBLIC_WS_URL ?? 'ws://localhost:1234';
    const ydoc = new Y.Doc();
    const provider = new WebsocketProvider(wsUrl, docId, ydoc);
    const ytext = ydoc.getText(TEXT_KEY);

    provider.awareness.setLocalStateField('user', {
      name: userName,
      color: userColor,
      colorLight: userColor,
    });

    const onStatus = (e: { status: string }) =>
      setConn(e.status === 'connected' ? 'connected' : 'offline');
    const onSync = (isSynced: boolean) => setSynced(isSynced);
    provider.on('status', onStatus);
    provider.on('sync', onSync);

    const view = new EditorView({
      parent: container,
      state: EditorState.create({
        doc: ytext.toString(),
        extensions: [
          markdown(),
          yCollab(ytext, provider.awareness),
          EditorView.lineWrapping,
        ],
      }),
    });

    return () => {
      provider.off('status', onStatus);
      provider.off('sync', onSync);
      view.destroy();
      provider.destroy();
      ydoc.destroy();
    };
  }, [docId, userName, userColor]);

  const label = conn === 'connected' ? (synced ? 'connected · synced' : 'connected · syncing') : conn;
  const dot = conn === 'connected' ? '#22a565' : conn === 'connecting' ? '#e0a800' : '#c0392b';

  return (
    <div>
      <div
        data-testid="conn-status"
        style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '8px 0', color: '#555' }}
      >
        <span style={{ width: 10, height: 10, borderRadius: '50%', background: dot }} />
        {label}
      </div>
      <div
        ref={ref}
        data-testid="editor"
        style={{ border: '1px solid #ddd', borderRadius: 6, minHeight: 240 }}
      />
    </div>
  );
}
