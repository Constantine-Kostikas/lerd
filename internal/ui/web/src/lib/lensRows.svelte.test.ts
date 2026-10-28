import { describe, it, expect, vi, beforeEach } from 'vitest';

const { fetchEvent } = vi.hoisted(() => ({ fetchEvent: vi.fn() }));
vi.mock('$lib/lens', () => ({ fetchEvent }));

import { LensRows, localTime, revealOrLoad } from './lensRows.svelte';

const group = (...ids: string[]) => ({ rows: ids.map((id) => ({ event: { id } })) }) as never;

describe('LensRows', () => {
  beforeEach(() => fetchEvent.mockReset().mockResolvedValue({ id: 'a', data: { trace: [] } }));

  it('reveals a row the page holds by opening it, and leaves one it does not', async () => {
    const rows = new LensRows();
    expect(rows.reveal([group('a')], 'b')).toBe(false);
    expect(rows.reveal([group('a')], 'a')).toBe(true);
    expect(rows.expanded.a).toBe(true);
    expect(rows.reveal([group('a')], 'a')).toBe(true);
    expect(rows.expanded.a).toBe(true);
  });

  it('reads the whole event when a row opens and forgets it when it closes', async () => {
    const rows = new LensRows();
    rows.toggle('a');
    await vi.waitFor(() => expect(rows.details.a).toBeTruthy());
    rows.toggle('a');
    expect(rows.expanded.a).toBe(false);
    expect(rows.details.a).toBeUndefined();
  });

  it('drops an event that lands after its row closed', async () => {
    let resolve!: (e: unknown) => void;
    fetchEvent.mockReturnValue(new Promise((r) => (resolve = r)));
    const rows = new LensRows();
    rows.toggle('a');
    rows.toggle('a');
    resolve({ id: 'a' });
    await Promise.resolve();
    expect(rows.details.a).toBeUndefined();
  });

  it('closes rows that left the page', async () => {
    const rows = new LensRows();
    rows.toggle('a');
    await vi.waitFor(() => expect(rows.details.a).toBeTruthy());
    rows.keepOnly([group('b')]);
    expect(rows.expanded.a).toBeUndefined();
    expect(rows.details.a).toBeUndefined();
  });
});

describe('localTime', () => {
  it('passes an unparseable timestamp through', () => {
    expect(localTime('soon')).toBe('soon');
  });
});

describe('revealOrLoad', () => {
  const lens = () => ({ loadRows: vi.fn(), loadMore: vi.fn() });
  const g = (key: string, count: number, ...ids: string[]) => ({ key, count, rows: ids.map((id) => ({ event: { id } })) }) as never;

  it('opens a row the page already holds', () => {
    const l = lens();
    expect(revealOrLoad(new LensRows(), l, [g('r1', 1, 'a')], false, 'a')).toBe('shown');
    expect(l.loadRows).not.toHaveBeenCalled();
  });

  it('reads the rest of a cut group before the next page', () => {
    const l = lens();
    expect(revealOrLoad(new LensRows(), l, [g('r1', 150, 'a')], true, 'z')).toBe('loading');
    expect(l.loadRows).toHaveBeenCalledWith('r1');
    expect(l.loadMore).not.toHaveBeenCalled();
  });

  it('reads the next page once every group is whole', () => {
    const l = lens();
    expect(revealOrLoad(new LensRows(), l, [g('r1', 1, 'a')], true, 'z')).toBe('loading');
    expect(l.loadMore).toHaveBeenCalled();
  });

  it('waits for a lens that has not drawn yet, and gives up once nothing is left', () => {
    const l = lens();
    expect(revealOrLoad(new LensRows(), l, [], false, 'z')).toBe('loading');
    expect(revealOrLoad(new LensRows(), l, [g('r1', 1, 'a')], false, 'z')).toBe('missing');
    expect(l.loadRows).not.toHaveBeenCalled();
    expect(l.loadMore).not.toHaveBeenCalled();
  });
});
