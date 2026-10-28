import { render, fireEvent, within } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PresetCard from './PresetCard.svelte';
import type { Preset } from '$stores/presets';

const install = vi.fn();
vi.mock('$stores/presets', async (importOriginal) => {
  const actual = await importOriginal<typeof import('$stores/presets')>();
  return { ...actual, installPresetAndOpen: (...args: unknown[]) => install(...args) };
});

function preset(over: Partial<Preset> = {}): Preset {
  return { name: 'postgres', ...over } as Preset;
}

describe('PresetCard', () => {
  beforeEach(() => install.mockReset());

  it('installs a preset without versions straight from its add button', async () => {
    const { getByRole, queryByRole } = render(PresetCard, { props: { preset: preset({ name: 'valkey' }) } });
    await fireEvent.click(getByRole('button'));
    expect(queryByRole('dialog')).toBeNull();
    expect(install).toHaveBeenCalledWith(expect.objectContaining({ name: 'valkey' }));
  });

  // A preset with versions asks which one first, offering only those not
  // installed yet, and installs the one picked.
  it('asks for the version of a multi-version preset before installing it', async () => {
    const p = preset({
      name: 'mysql',
      installed_tags: ['8.4'],
      versions: [{ tag: '8.4' }, { tag: '8.0' }, { tag: '5.7', label: '5.7 (legacy)' }],
      selected_version: '8.0'
    });
    const { getByRole, getByText, queryByRole } = render(PresetCard, { props: { preset: p } });
    expect(getByText('8.4 installed')).toBeTruthy();
    await fireEvent.click(getByRole('button'));
    const dialog = within(getByRole('dialog'));
    expect(install).not.toHaveBeenCalled();
    await fireEvent.click(dialog.getByRole('button', { name: /8\.0/ }));
    expect(queryByRole('option', { name: '8.4' })).toBeNull();
    await fireEvent.click(getByRole('option', { name: '5.7 (legacy)' }));
    await fireEvent.click(dialog.getByRole('button', { name: 'Add' }));
    expect(install).toHaveBeenCalledWith(expect.objectContaining({ name: 'mysql', selected_version: '5.7' }));
  });

  it('renders the human label and the description blurb', () => {
    const { getByText } = render(PresetCard, {
      props: { preset: preset({ description: 'Relational database with extensions' }) }
    });
    expect(getByText('PostgreSQL')).toBeTruthy();
    expect(getByText('Relational database with extensions')).toBeTruthy();
  });

  it('renders a service glyph icon', () => {
    const { container } = render(PresetCard, { props: { preset: preset() } });
    expect(container.querySelector('svg')).toBeTruthy();
  });

  it('disables the Add button and shows a spinner while installing', () => {
    const { container } = render(PresetCard, { props: { preset: preset({ installing: true }) } });
    const btn = container.querySelector('button')!;
    expect(btn.disabled).toBe(true);
    expect(container.querySelector('.animate-spin')).toBeTruthy();
  });

  it('shows the install error when present', () => {
    const { getByText } = render(PresetCard, { props: { preset: preset({ error: 'boom' }) } });
    expect(getByText('boom')).toBeTruthy();
  });

  it('tints the icon by the given category', () => {
    const { container } = render(PresetCard, {
      props: { preset: preset(), category: 'databases' }
    });
    expect(container.innerHTML).toMatch(/text-indigo-600/);
  });

  it('uses the category the preset declares when none is passed', () => {
    const { container } = render(PresetCard, {
      props: { preset: preset({ name: 'redis', category: 'cache' }) }
    });
    expect(container.innerHTML).toMatch(/text-amber-600/);
  });
});
