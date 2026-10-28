import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { get } from 'svelte/store';

function readerFrom(chunks: string[]): ReadableStream<Uint8Array> {
  const enc = new TextEncoder();
  let i = 0;
  return new ReadableStream<Uint8Array>({
    pull(ctrl) {
      if (i >= chunks.length) {
        ctrl.close();
        return;
      }
      ctrl.enqueue(enc.encode(chunks[i++]));
    }
  });
}

describe('presets store', () => {
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  it('loadPresets populates with default selected_version', async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify([
          {
            name: 'mysql',
            versions: [{ tag: '5.7' }, { tag: '8.0' }],
            default_version: '8.0',
            installed_tags: []
          }
        ]),
        { status: 200 }
      )
    ) as unknown as typeof fetch;
    const { presets, loadPresets } = await import('./presets');
    await loadPresets();
    const list = get(presets);
    expect(list[0].selected_version).toBe('8.0');
  });

  it('installablePresets hides presets with missing deps', async () => {
    const { presets, installablePresets } = await import('./presets');
    presets.set([
      { name: 'a', site_count: 0 } as never,
      { name: 'b', missing_deps: ['x'], site_count: 0 } as never
    ]);
    expect(get(installablePresets).map((p) => p.name)).toEqual(['a']);
  });

  it('installablePresets only shows versioned when new versions exist', async () => {
    const { presets, installablePresets } = await import('./presets');
    presets.set([
      { name: 'full', versions: [{ tag: '1' }], installed_tags: ['1'] } as never,
      { name: 'part', versions: [{ tag: '1' }, { tag: '2' }], installed_tags: ['1'] } as never
    ]);
    expect(get(installablePresets).map((p) => p.name)).toEqual(['part']);
  });

  it('matchesPresetQuery matches name, description, and image, ANDing terms', async () => {
    const { matchesPresetQuery } = await import('./presets');
    const redis = { name: 'redis', description: 'In-memory cache', image: 'docker.io/redis:7' } as never;
    expect(matchesPresetQuery(redis, '')).toBe(true);
    expect(matchesPresetQuery(redis, 'red')).toBe(true);
    expect(matchesPresetQuery(redis, 'cache')).toBe(true);
    expect(matchesPresetQuery(redis, 'redis:7')).toBe(true);
    expect(matchesPresetQuery(redis, 'redis cache')).toBe(true);
    expect(matchesPresetQuery(redis, 'redis postgres')).toBe(false);
    expect(matchesPresetQuery(redis, 'postgres')).toBe(false);
  });

  it('presetAddLabel maps installing phases', async () => {
    const { presetAddLabel } = await import('./presets');
    expect(presetAddLabel({ name: 'x' } as never)).toBe('Add');
    expect(presetAddLabel({ name: 'x', installing: true } as never)).toBe('Adding...');
    expect(
      presetAddLabel({ name: 'x', installing: true, installingPhase: 'pulling_image' } as never)
    ).toBe('Pulling image...');
  });

  it('installPreset streams NDJSON events and sets phases', async () => {
    const body = readerFrom([
      '{"phase":"pulling_image","image":"mysql:8"}\n',
      '{"phase":"starting_unit"}\n',
      '{"phase":"done","name":"mysql"}\n'
    ]);
    globalThis.fetch = vi.fn(async (url: unknown) =>
      String(url).includes('/api/image-estimate')
        ? new Response(JSON.stringify({ image: 'mysql:8', bytes: 0, local: true }), { status: 200 })
        : new Response(body, { status: 200 })
    ) as unknown as typeof fetch;
    const { installPreset } = await import('./presets');
    const r = await installPreset({ name: 'mysql', site_count: 0 } as never);
    expect(r.ok).toBe(true);
    expect(r.name).toBe('mysql');
  });

  it('installPreset surfaces error from final event', async () => {
    const body = readerFrom(['{"phase":"error","error":"boom"}\n']);
    globalThis.fetch = vi.fn(async (url: unknown) =>
      String(url).includes('/api/image-estimate')
        ? new Response(JSON.stringify({ image: 'bad:1', bytes: 0, local: true }), { status: 200 })
        : new Response(body, { status: 200 })
    ) as unknown as typeof fetch;
    const { installPreset } = await import('./presets');
    const r = await installPreset({ name: 'bad' } as never);
    expect(r.ok).toBe(false);
    expect(r.error).toBe('boom');
  });

  // Declining the download must abort before the install POST, and must not
  // leave the card stuck on its spinner.
  it('installPreset does not install when the download is declined', async () => {
    const urls: string[] = [];
    globalThis.fetch = vi.fn(async (url: unknown) => {
      urls.push(String(url));
      return new Response(JSON.stringify({ image: 'mysql:8', bytes: 1024, local: false }), {
        status: 200
      });
    }) as unknown as typeof fetch;
    const { installPreset, presets } = await import('./presets');
    const { downloadConfirm, answerDownloadConfirm } = await import('./downloadConfirm');
    presets.set([{ name: 'mysql' } as never]);

    const pending = installPreset({ name: 'mysql' } as never);
    await vi.waitFor(() => expect(get(downloadConfirm).open).toBe(true));
    answerDownloadConfirm(false);

    expect((await pending).ok).toBe(false);
    expect(urls.every((u) => u.includes('/api/image-estimate'))).toBe(true);
    expect(get(presets)[0].installing).toBe(false);
  });

  it('installPreset short-circuits when missing_deps is non-empty', async () => {
    globalThis.fetch = vi.fn(async () => {
      throw new Error('should not be called');
    }) as unknown as typeof fetch;
    const { installPreset } = await import('./presets');
    const r = await installPreset({ name: 'x', missing_deps: ['mysql'] } as never);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/mysql first/);
  });

  it('installPresetAndOpen installs, refreshes services + presets, runs onSuccess, and navigates', async () => {
    location.hash = '';
    const calls: string[] = [];
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const u = String(url);
      calls.push(u);
      if (u.includes('/api/services/presets/')) {
        return new Response(readerFrom(['{"phase":"done","name":"mysql"}\n']), { status: 200 });
      }
      return new Response('[]', { status: 200 });
    }) as unknown as typeof fetch;
    const { installPresetAndOpen } = await import('./presets');
    let opened = '';
    const r = await installPresetAndOpen({ name: 'mysql', site_count: 0 } as never, {
      onSuccess: (n) => (opened = n)
    });
    expect(r.ok).toBe(true);
    expect(opened).toBe('mysql');
    expect(location.hash).toBe('#services/mysql');
    expect(calls.some((c) => c.endsWith('/api/services'))).toBe(true);
    expect(calls.some((c) => c.endsWith('/api/services/presets'))).toBe(true);
  });

  it('installPresetAndOpen does not navigate when the install fails', async () => {
    location.hash = '';
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const u = String(url);
      if (u.includes('/api/services/presets/')) {
        return new Response(readerFrom(['{"phase":"error","error":"boom"}\n']), { status: 200 });
      }
      return new Response('[]', { status: 200 });
    }) as unknown as typeof fetch;
    const { installPresetAndOpen } = await import('./presets');
    const r = await installPresetAndOpen({ name: 'bad', site_count: 0 } as never);
    expect(r.ok).toBe(false);
    expect(location.hash).toBe('');
  });
});
