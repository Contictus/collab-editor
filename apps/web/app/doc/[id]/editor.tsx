'use client';

import { markdown } from '@codemirror/lang-markdown';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { useEffect, useRef, useState } from 'react';
import { yCollab } from 'y-codemirror.next';
import { IndexeddbPersistence } from 'y-indexeddb';
import { WebsocketProvider } from 'y-websocket';
import * as Y from 'yjs';
import { TEXT_KEY } from 'shared/crdt';
import { offlineRoomKey } from '../../../lib/offline';
import { Preview, type PreviewTheme } from './preview';
import { Toolbar } from './toolbar';

type ViewMode = 'split' | 'edit' | 'preview';

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
  const viewRef = useRef<EditorView | null>(null);
  const [conn, setConn] = useState<ConnState>('connecting');
  const [synced, setSynced] = useState(false);
  const [text, setText] = useState('');
  const [mode, setMode] = useState<ViewMode>('split');
  const [theme, setTheme] = useState<PreviewTheme>('light');

  useEffect(() => {
    const saved = window.localStorage.getItem('editor-theme');
    if (saved === 'dark' || saved === 'light') setTheme(saved);
  }, []);

  useEffect(() => {
    const container = ref.current;
    if (!container) return;

    const wsUrl = process.env.NEXT_PUBLIC_WS_URL ?? 'ws://localhost:8080';
    const ydoc = new Y.Doc();
    // Local-first replica (F10): IndexedDB persists every update locally and
    // merges on reconnect through the CRDT — reloads restore instantly.
    const persistence = new IndexeddbPersistence(offlineRoomKey(docId), ydoc);
    const provider = new WebsocketProvider(wsUrl, docId, ydoc);
    const ytext = ydoc.getText(TEXT_KEY);
    setText(ytext.toString());
    const observer = () => setText(ytext.toString());
    ytext.observe(observer);

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
    viewRef.current = view;

    return () => {
      ytext.unobserve(observer);
      viewRef.current = null;
      persistence.destroy();
      provider.off('status', onStatus);
      provider.off('sync', onSync);
      view.destroy();
      provider.destroy();
      ydoc.destroy();
    };
  }, [docId, userName, userColor]);

  // Connection label: synced means Yjs syncStep2 received, not just socket open.
  const label = conn === 'connected' ? (synced ? 'connected · synced' : 'connected · syncing') : conn;
  const dot = conn === 'connected' ? '#22a565' : conn === 'connecting' ? '#e0a800' : '#c0392b';
  const dark = theme === 'dark';

  function toggleTheme() {
    setTheme((t) => {
      const next = t === 'dark' ? 'light' : 'dark';
      window.localStorage.setItem('editor-theme', next);
      return next;
    });
  }

  const modeBtn = (m: ViewMode, testId: string, title: string) => (
    <button
      data-testid={testId}
      title={title}
      onClick={() => setMode(m)}
      style={{
        border: '1px solid #ddd',
        borderRadius: 6,
        background: mode === m ? '#111' : '#fff',
        color: mode === m ? '#fff' : '#111',
        padding: '4px 10px',
        fontSize: 13,
        cursor: 'pointer',
      }}
    >
      {title}
    </button>
  );

  const showSource = mode !== 'preview';
  const showPreview = mode !== 'edit';

  return (
    <div data-theme={theme}>
      <div
        data-testid="conn-status"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          margin: '8px 0',
          color: '#555',
          fontSize: 13,
          letterSpacing: 0.2,
        }}
      >
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: '50%',
            background: dot,
            boxShadow: conn === 'connected' ? '0 0 0 4px #22a56522' : 'none',
          }}
        />
        {label}
        <span style={{ marginLeft: 8, color: '#999', fontSize: 12 }}>
          {userName}
        </span>
        <span
          style={{
            width: 12,
            height: 12,
            borderRadius: 3,
            background: userColor,
            border: '1px solid #0001',
          }}
          title={userColor}
        />
      </div>
      <Toolbar getView={() => viewRef.current} />
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', margin: '0 0 8px' }}>
        {modeBtn('split', 'view-split', 'Split')}
        {modeBtn('edit', 'view-edit', 'Edit')}
        {modeBtn('preview', 'view-preview', 'Preview')}
        <button
          data-testid="theme-toggle"
          title="Toggle dark mode"
          onClick={toggleTheme}
          style={{
            border: '1px solid #ddd',
            borderRadius: 6,
            background: dark ? '#222' : '#fff',
            color: dark ? '#fff' : '#111',
            padding: '4px 10px',
            fontSize: 13,
            cursor: 'pointer',
          }}
        >
          {dark ? 'Light' : 'Dark'}
        </button>
      </div>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'stretch' }}>
        {/* Source stays mounted when hidden — unmount would kill the Yjs binding. */}
        <div style={{ flex: '1 1 320px', minWidth: 0, display: showSource ? undefined : 'none' }}>
          <p style={{ color: '#999', fontSize: 12, margin: '6px 0' }}>Source</p>
          <div
            ref={ref}
            data-testid="editor"
            style={{
              border: `1px solid ${dark ? '#444' : '#ddd'}`,
              borderRadius: 8,
              minHeight: 320,
              background: '#fff',
              boxShadow: '0 1px 6px #0000a08, 0 1px 2px #00000014',
              overflow: 'hidden',
            }}
          />
        </div>
        {showPreview && (
        <div style={{ flex: '1 1 320px', minWidth: 0 }}>
          <p style={{ color: '#999', fontSize: 12, margin: '6px 0' }}>Preview</p>
          <Preview text={text} theme={theme} />
        </div>
        )}
      </div>
      <p style={{ color: '#999', fontSize: 12, margin: '6px 0 0' }}>
        Markdown · Yjs CRDT · edits merge live — open this doc in another window to see cursors.
      </p>
    </div>
  );
}
