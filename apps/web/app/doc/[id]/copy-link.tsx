'use client';

import { useState } from 'react';

export function CopyLink() {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      onClick={async () => {
        await navigator.clipboard.writeText(window.location.href);
        setCopied(true);
        setTimeout(() => setCopied(false), 1400);
      }}
      style={{
        padding: '4px 10px',
        borderRadius: 999,
        border: '1px solid #ddd',
        background: copied ? '#111' : '#fff',
        color: copied ? '#fff' : '#111',
        fontSize: 12,
        cursor: 'pointer',
      }}
    >
      {copied ? 'Copied!' : 'Copy link'}
    </button>
  );
}
