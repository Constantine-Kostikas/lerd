import { render, fireEvent, waitFor, within } from '@testing-library/svelte';
import { it, expect, vi, beforeEach } from 'vitest';
import PluginBrowse from './PluginBrowse.svelte';

const A = 'a'.repeat(40);
let catalog: object[];
let posts: unknown[];
const calls: string[] = [];

beforeEach(() => {
  catalog = [
    { name: 'old', repo: 'r', commit: A, version: '1.2.0', stars: 50, updated_at: '2026-01-01', installed: false, compatible: true, category: 'Tools' },
    { name: 'fresh', repo: 'r', commit: 'b'.repeat(40), stars: 2, updated_at: '2026-10-01', installed: false, images: ['https://plugins.lerd.sh/images/fresh/1.png', 'https://plugins.lerd.sh/images/fresh/2.png'], compatible: true, category: 'Debugging' },
    { name: 'mine', repo: 'r', commit: A, installed: true, compatible: true, category: 'Tools' }
  ];
  posts = [];
  globalThis.fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (init?.method === 'POST') {
      posts.push(JSON.parse(init.body as string));
      return new Response('{}');
    }
    if (String(url).includes('/api/plugin-manager/catalog')) { calls.push(String(url)); return new Response(JSON.stringify(catalog)); }
    return new Response('[]');
  }) as unknown as typeof fetch;
});

const names = (c: HTMLElement) => [...c.querySelectorAll('[id^="plugin-card-"]')].map((a) => a.id.replace('plugin-card-', ''));
const many = (n: number) =>
  Array.from({ length: n }, (_, i) => ({ name: `p${String(i).padStart(2, '0')}`, repo: 'r', commit: A, installed: false, compatible: true }));

it('lists the catalog newest first as cards with a thumbnail, and marks what is installed', async () => {
  const { container, findByText, getByText } = render(PluginBrowse);
  await findByText('fresh');
  expect(names(container)).toEqual(['fresh', 'old', 'mine']);
  expect(container.querySelector('#plugin-card-fresh img')?.getAttribute('src')).toBe('https://plugins.lerd.sh/images/fresh/1.png');
  expect(within(container.querySelector('#plugin-card-mine') as HTMLElement).getByText('Installed')).toBeTruthy();
  expect(getByText('★ 50')).toBeTruthy();
});

it('opens a card in a detail with every screenshot', async () => {
  const { findByText, getByRole } = render(PluginBrowse);
  await fireEvent.click(await findByText('fresh'));
  const dialog = getByRole('dialog');
  const shots = [...dialog.querySelectorAll('img')].map((i) => i.getAttribute('src'));
  expect(shots).toContain('https://plugins.lerd.sh/images/fresh/2.png');
  await fireEvent.click(within(dialog).getByRole('button', { name: '2 / 2' }));
  expect(dialog.querySelector('figure > div > img')?.getAttribute('src')).toBe('https://plugins.lerd.sh/images/fresh/2.png');
  await fireEvent.click(within(dialog).getByRole('button', { name: 'Next page' }));
  expect(dialog.querySelector('figure > div > img')?.getAttribute('src')).toBe('https://plugins.lerd.sh/images/fresh/1.png');
  await fireEvent.keyDown(window, { key: 'ArrowLeft' });
  expect(dialog.querySelector('figure > div > img')?.getAttribute('src')).toBe('https://plugins.lerd.sh/images/fresh/2.png');
  expect(dialog.textContent).toContain('2 / 2');
});

// A gallery steps on when the active thumbnail's bar finishes filling, and the
// fill holds while the pointer is on the gallery.
it('moves the screenshots on when the bar fills, and holds it while hovered', async () => {
  const { findByText, getByRole } = render(PluginBrowse);
  await fireEvent.click(await findByText('fresh'));
  const dialog = getByRole('dialog');
  const src = () => dialog.querySelector('figure > div > img')?.getAttribute('src');
  const bar = () => dialog.querySelector('[data-slide-bar]') as HTMLElement;
  expect(src()).toBe('https://plugins.lerd.sh/images/fresh/1.png');
  await fireEvent.animationEnd(bar());
  expect(src()).toBe('https://plugins.lerd.sh/images/fresh/2.png');
  await fireEvent.mouseEnter(dialog.querySelector('figure')!);
  expect(bar().classList.contains('paused')).toBe(true);
});

it('narrows the cards to one category', async () => {
  const { container, findByText, getByRole } = render(PluginBrowse);
  await findByText('fresh');
  await fireEvent.click(getByRole('button', { name: 'Tools' }));
  expect(names(container)).toEqual(['old', 'mine']);
});

// A catalog that has grown keeps the page short; a search starts again from the first page.
it('pages a long catalog and goes back to the first page on a search', async () => {
  catalog = many(30);
  const { container, findByText, getByRole, getByLabelText } = render(PluginBrowse);
  await findByText('p00');
  expect(names(container)).toHaveLength(12);
  expect(container.textContent).toContain('1–12 of 30');
  await fireEvent.click(getByRole('button', { name: 'Next page' }));
  expect(names(container)[0]).toBe('p12');
  await fireEvent.input(getByLabelText('Search plugins'), { target: { value: 'p2' } });
  expect(names(container)).toEqual(['p20', 'p21', 'p22', 'p23', 'p24', 'p25', 'p26', 'p27', 'p28', 'p29']);
  expect(container.textContent).not.toContain('of 30');
});

// The commit is in the question because it is the one thing the user is
// trusting: lerd installs that and nothing newer.
it('installs by name from the detail after confirming the approved commit', async () => {
  const { findByText, getByRole, getAllByRole } = render(PluginBrowse);
  await fireEvent.click(await findByText('old'));
  await fireEvent.click(within(getByRole('dialog')).getByRole('button', { name: 'Install' }));
  const confirm = getAllByRole('dialog').at(-1)!;
  expect(confirm.textContent).toContain('aaaaaaa');
  await fireEvent.click(within(confirm).getByRole('button', { name: 'Install' }));
  await waitFor(() => expect(posts).toEqual([{ name: 'old' }]));
});

// Offering the install would only end in a refusal from the binary.
it('names the lerd a plugin needs instead of offering to install it', async () => {
  catalog = [{ name: 'later', repo: 'r', commit: A, installed: false, compatible: false, lerd: '>=1.40' }];
  const { findByText, getByRole, queryByRole } = render(PluginBrowse);
  await fireEvent.click(await findByText('later'));
  expect(within(getByRole('dialog')).getByText('Needs Lerd >=1.40')).toBeTruthy();
  expect(queryByRole('button', { name: 'Install' })).toBeNull();
});

// A plugin for the other platform would be refused at install, so say where it runs.
it('names the platform a plugin runs on instead of offering to install it', async () => {
  catalog = [{ name: 'linuxy', repo: 'r', commit: A, installed: false, compatible: true, runs_here: false, platforms: ['linux'] }];
  const { findByText, getByRole, queryByRole } = render(PluginBrowse);
  await fireEvent.click(await findByText('linuxy'));
  expect(within(getByRole('dialog')).getByText('Linux only')).toBeTruthy();
  expect(queryByRole('button', { name: 'Install' })).toBeNull();
});

// Installing would only end in lerd refusing it, so say what is missing first.
it('lists what a plugin requires and offers no install while something is missing', async () => {
  catalog = [{ name: 'needy', repo: 'r', commit: A, installed: false, compatible: true, requires: ['python3', 'jq'], missing: ['jq'] }];
  const { findByText, getByRole, queryByRole } = render(PluginBrowse);
  await fireEvent.click(await findByText('needy'));
  const dialog = within(getByRole('dialog'));
  expect(dialog.getByText('python3')).toBeTruthy();
  expect(dialog.getByText('jq').className).toContain('text-red');
  expect(queryByRole('button', { name: 'Install' })).toBeNull();
});

// An empty catalog asks for the first plugins: how to build one and where to send it.
it('says the catalog is empty and how to get a plugin into it', async () => {
  catalog = [];
  const { findByText, queryByLabelText, getByRole, getByText } = render(PluginBrowse);
  await findByText('No plugins listed yet');
  expect(queryByLabelText('Search plugins')).toBeNull();
  expect(getByText('Submit it')).toBeTruthy();
  expect(getByRole('link', { name: 'Read the submission guide' }).getAttribute('href')).toBe('https://github.com/lerd-env/plugins/blob/main/SUBMITTING.md');
  expect(getByRole('link', { name: 'See an example submission' }).getAttribute('href')).toBe('https://github.com/lerd-env/plugins/pull/1');
});

it('says why when the catalog cannot be reached', async () => {
  globalThis.fetch = vi.fn(async () => new Response('fetching the plugin catalog: no route', { status: 502 })) as unknown as typeof fetch;
  const { findByText } = render(PluginBrowse);
  await findByText('The plugin catalog could not be loaded');
  await findByText(/no route/);
});

// Once plugins are listed the catalog stays clear: one more card ends the grid
// and opens the build and submit steps on demand.
it('invites building and submitting a plugin from a card after the listed ones', async () => {
  const { findByText, getByRole, queryByText } = render(PluginBrowse);
  await findByText('fresh');
  expect(queryByText('Build the next one')).toBeNull();
  const card = getByRole('button', { name: /^Build your own/ });
  expect(within(card).getByText(/Start from a working plugin/)).toBeTruthy();
  expect(card.querySelector('svg')).toBeTruthy();
  await fireEvent.click(card);
  const dialog = within(getByRole('dialog'));
  expect(dialog.getByText('Build the next one')).toBeTruthy();
  expect(dialog.getByText('lerd plugin new my-plugin')).toBeTruthy();
  expect(dialog.getByRole('link', { name: /submission guide/i }).getAttribute('href')).toContain('lerd-env/plugins/blob/main/SUBMITTING.md');
});

// The approved commit's version, on its card and in its detail.
it('shows each listed plugin its version', async () => {
  const { container, findByText, getByRole } = render(PluginBrowse);
  await findByText('old');
  expect(within(container.querySelector('#plugin-card-old') as HTMLElement).getByText('1.2.0')).toBeTruthy();
  await fireEvent.click(await findByText('old'));
  expect(within(getByRole('dialog')).getByText(/^v?1\.2\.0$/)).toBeTruthy();
});

// Opening Browse shows the catalog as it is now, not as of the last hourly check.
it('asks for a fresh catalog when it opens', async () => {
  calls.length = 0;
  render(PluginBrowse);
  await waitFor(() => expect(calls.some((u) => u.endsWith('/api/plugin-manager/catalog?fresh=1'))).toBe(true));
});
