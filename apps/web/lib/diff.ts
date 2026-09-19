/**
 * Line diff for history compare (F11-6). Pure LCS over lines — small texts
 * (Markdown docs), no dependency. Returns hunks with unchanged context so the
 * compare view reads like a review.
 */

export type DiffLine =
  | { kind: 'same'; text: string }
  | { kind: 'add'; text: string }
  | { kind: 'del'; text: string };

export function diffLines(before: string, after: string): DiffLine[] {
  const a = before.split('\n');
  const b = after.split('\n');
  const m = a.length;
  const n = b.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0));
  for (let i = m - 1; i >= 0; i--) {
    for (let j = n - 1; j >= 0; j--) {
      const same = (a[i] as string) === (b[j] as string);
      dp[i]![j] = same ? dp[i + 1]![j + 1]! + 1 : Math.max(dp[i + 1]![j]!, dp[i]![j + 1]!);
    }
  }
  const out: DiffLine[] = [];
  let i = 0;
  let j = 0;
  while (i < m && j < n) {
    const ai = a[i] as string;
    const bj = b[j] as string;
    if (ai === bj) {
      out.push({ kind: 'same', text: ai });
      i++;
      j++;
    } else if (dp[i + 1]![j]! >= dp[i]![j + 1]!) {
      out.push({ kind: 'del', text: ai });
      i++;
    } else {
      out.push({ kind: 'add', text: bj });
      j++;
    }
  }
  while (i < m) out.push({ kind: 'del', text: a[i++] as string });
  while (j < n) out.push({ kind: 'add', text: b[j++] as string });
  return out;
}
