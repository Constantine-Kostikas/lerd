import type { GitChangedFile } from './gitDiff';

export type FileTreeRow =
  | { kind: 'dir'; path: string; label: string; depth: number }
  | { kind: 'file'; path: string; file: GitChangedFile; label: string; depth: number };

type Node = { dirs: Map<string, Node>; files: GitChangedFile[] };

// Flattened so the list stays one #each; a folder holding only one folder shares
// its row ("app/Filament/Resources"), the way editors show it.
export function fileTreeRows(files: GitChangedFile[], isCollapsed: (dir: string) => boolean): FileTreeRow[] {
  const root: Node = { dirs: new Map(), files: [] };
  for (const f of files) {
    const parts = f.path.split('/');
    let node = root;
    for (const part of parts.slice(0, -1)) {
      if (!node.dirs.has(part)) node.dirs.set(part, { dirs: new Map(), files: [] });
      node = node.dirs.get(part)!;
    }
    node.files.push(f);
  }
  const rows: FileTreeRow[] = [];
  walk(root, '', 0, isCollapsed, rows);
  return rows;
}

function walk(node: Node, prefix: string, depth: number, isCollapsed: (dir: string) => boolean, rows: FileTreeRow[]) {
  for (const name of [...node.dirs.keys()].sort()) {
    let child = node.dirs.get(name)!;
    let label = name;
    while (child.files.length === 0 && child.dirs.size === 1) {
      const [next, only] = [...child.dirs][0];
      label += '/' + next;
      child = only;
    }
    const path = prefix + label;
    rows.push({ kind: 'dir', path, label, depth });
    if (!isCollapsed(path)) walk(child, path + '/', depth + 1, isCollapsed, rows);
  }
  for (const f of [...node.files].sort((a, b) => a.path.localeCompare(b.path))) {
    rows.push({ kind: 'file', path: f.path, file: f, label: f.path.slice(prefix.length), depth });
  }
}
