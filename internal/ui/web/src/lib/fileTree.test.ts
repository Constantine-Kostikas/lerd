import { describe, it, expect } from 'vitest';
import { fileTreeRows } from './fileTree';

const files = [
  { path: 'app/Filament/Resources/A.php', status: 'M' },
  { path: '.env.example', status: 'M' },
  { path: 'app/Enums/Type.php', status: 'A' },
  { path: 'app/Filament/Resources/B.php', status: 'M' },
  { path: 'routes/web.php', status: 'M' }
];
const show = (rows: ReturnType<typeof fileTreeRows>) => rows.map((r) => `${'  '.repeat(r.depth)}${r.kind === 'dir' ? r.label + '/' : r.label}`);

describe('fileTreeRows', () => {
  it('nests by folder, folders first, and folds single-child folder chains', () => {
    expect(show(fileTreeRows(files, () => false))).toEqual([
      'app/',
      '  Enums/',
      '    Type.php',
      '  Filament/Resources/',
      '    A.php',
      '    B.php',
      'routes/',
      '  web.php',
      '.env.example'
    ]);
  });

  it('hides what a collapsed folder holds, keyed by its full path', () => {
    const rows = fileTreeRows(files, (p) => p === 'app/Filament/Resources');
    expect(show(rows)).toContain('  Filament/Resources/');
    expect(show(rows)).not.toContain('    A.php');
    expect(show(rows)).toContain('    Type.php');
  });
});
