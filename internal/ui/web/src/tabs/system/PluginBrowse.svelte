<script lang="ts">
  import { onMount, tick } from 'svelte';
  import Dropdown from '$components/Dropdown.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import ConfirmModal from '$components/ConfirmModal.svelte';
  import Modal from '$components/Modal.svelte';
  import EmptyState from '$components/EmptyState.svelte';
  import Pager from '$components/Pager.svelte';
  import CatalogCard from './CatalogCard.svelte';
  import BuildPluginCard from './BuildPluginCard.svelte';
  import CatalogEmpty from './CatalogEmpty.svelte';
  import PluginDetailModal from './PluginDetailModal.svelte';
  import { loadCatalog, installListedPlugin, type CatalogEntry } from '$stores/plugins';
  import { notifyLocalFailure, notifyLocalInfo } from '$lib/notify';
  import { m } from '../../paraglide/messages.js';

  // Product names, not the manifest's ids.
  const PLATFORM_NAMES: Record<string, string> = { linux: 'Linux', macos: 'macOS' };
  const platformNames = (list: string[] = []) => list.map((p) => PLATFORM_NAMES[p] ?? p).join(', ');

  interface Props {
    // A plugin to bring into view, from a plugins.lerd.sh "Install in Lerd" link.
    focus?: string;
  }
  let { focus = '' }: Props = $props();

  let entries = $state<CatalogEntry[]>([]);
  let loadError = $state('');
  let loaded = $state(false);
  let query = $state('');
  let sort = $state('updated');
  let category = $state('');
  let page = $state(0);
  let showMore = $state(false);
  let open = $state<CatalogEntry | null>(null);
  let confirm = $state<CatalogEntry | null>(null);
  let installing = $state('');

  // A page fills a few rows of the grid at any width; search, category and sort
  // start again from the first.
  const PAGE_SIZE = 12;
  const categories = $derived([...new Set(entries.map((e) => e.category).filter((c): c is string => !!c))].sort());

  // The same orders plugins.lerd.sh offers, newest or most first, by name on a tie.
  const byName = (a: CatalogEntry, b: CatalogEntry) => a.name.localeCompare(b.name);
  const SORTS: Record<string, (a: CatalogEntry, b: CatalogEntry) => number> = {
    updated: (a, b) => (b.updated_at ?? '').localeCompare(a.updated_at ?? '') || byName(a, b),
    stars: (a, b) => (b.stars ?? 0) - (a.stars ?? 0) || byName(a, b),
    added: (a, b) => (b.added_at ?? '').localeCompare(a.added_at ?? '') || byName(a, b),
    name: byName
  };
  const sortOptions = $derived([
    { value: 'updated', label: m.system_plugins_sortUpdated() },
    { value: 'stars', label: m.system_plugins_sortStars() },
    { value: 'added', label: m.system_plugins_sortNewest() },
    { value: 'name', label: m.system_plugins_sortName() }
  ]);

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return entries
      .filter((e) => !category || e.category === category)
      .filter((e) => !q || [e.name, e.description, e.author, ...(e.tags ?? [])].join(' ').toLowerCase().includes(q))
      .sort(SORTS[sort] ?? SORTS.updated);
  });
  const current = $derived(Math.min(page, Math.max(0, Math.ceil(shown.length / PAGE_SIZE) - 1)));
  const paged = $derived(shown.slice(current * PAGE_SIZE, (current + 1) * PAGE_SIZE));
  $effect(() => {
    void query;
    void category;
    void sort;
    page = 0;
  });

  async function load(fresh = false) {
    try {
      entries = await loadCatalog(fresh);
      loadError = '';
    } catch (err) {
      loadError = err instanceof Error ? err.message : String(err);
    } finally {
      loaded = true;
    }
  }

  onMount(async () => {
    await load(true);
    // A plugins.lerd.sh "Install in Lerd" link lands on the plugin's own page.
    const i = shown.findIndex((e) => e.name === focus);
    if (i < 0) return;
    page = Math.floor(i / PAGE_SIZE);
    await tick();
    document.getElementById('plugin-card-' + focus)?.scrollIntoView({ block: 'center' });
    open = shown[i];
  });

  async function install() {
    const entry = confirm;
    if (!entry) return;
    installing = entry.name;
    try {
      const err = await installListedPlugin(entry.name);
      if (err) {
        notifyLocalFailure('plugin_install', m.system_plugins_installFailed(), err);
        return;
      }
      notifyLocalInfo('plugin_install', m.system_plugins_title(), m.system_plugins_installed());
      await load();
      open = entries.find((x) => x.name === entry.name) ?? null;
    } finally {
      installing = '';
      confirm = null;
    }
  }
</script>

<div class="flex-1 flex flex-col gap-3">
  {#if loaded && !loadError && entries.length > 0}
    <div class="flex flex-wrap items-center gap-2">
      <input
        type="search"
        bind:value={query}
        placeholder={m.system_plugins_search()}
        aria-label={m.system_plugins_search()}
        class="flex-1 min-w-48 h-7 text-xs bg-white dark:bg-lerd-card border border-gray-200 dark:border-lerd-border rounded-md px-2.5 text-gray-700 dark:text-gray-200 placeholder:text-gray-400 focus:outline-hidden focus:border-lerd-red/50"
      />
      <Dropdown value={sort} options={sortOptions} onchange={(v) => (sort = v)} align="right" />
    </div>
    {#if categories.length > 1}
      <div class="flex flex-wrap gap-1.5" role="group" aria-label={m.system_plugins_category()}>
        {#each ['', ...categories] as c (c)}
          <button
            type="button"
            aria-pressed={category === c}
            onclick={() => (category = c)}
            class="h-6 px-2.5 rounded-md border text-[11px] font-medium transition-colors {category === c
              ? 'border-lerd-red/50 text-lerd-red bg-lerd-red/5'
              : 'border-gray-200 dark:border-lerd-border text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'}"
          >{c || m.system_plugins_allCategories()}</button>
        {/each}
      </div>
    {/if}
  {/if}

  {#if loadError}
    <EmptyState title={m.system_plugins_catalogFailed()} size="sm">
      {#snippet hint()}<span class="break-words">{loadError}</span>{/snippet}
    </EmptyState>
  {:else if loaded && entries.length === 0}
    <div class="flex-1 flex items-center justify-center"><CatalogEmpty /></div>
  {:else if loaded && shown.length === 0}
    <EmptyState title={m.system_plugins_noMatch()} size="sm" />
  {:else}
    <div class="grid gap-3 grid-cols-[repeat(auto-fill,minmax(250px,1fr))]">
      {#each paged as e (e.name)}
        <CatalogCard entry={e} focused={e.name === focus} onopen={() => (open = e)} />
      {/each}
      <BuildPluginCard onopen={() => (showMore = true)} />
    </div>
    <Pager page={current} total={shown.length} size={PAGE_SIZE} onchange={(p) => (page = p)} />
  {/if}
  <Modal open={showMore} title={m.system_plugins_buildTitle()} size="xl" onclose={() => (showMore = false)}>
    <div class="px-5"><CatalogEmpty more /></div>
  </Modal>
</div>

{#if open}
  {@const e = open}
  <PluginDetailModal
    name={e.name}
    description={e.description}
    version={e.version}
    author={e.author}
    images={e.images}
    repo={e.repo}
    commit={e.commit}
    requires={e.requires}
    missing={e.missing}
    platforms={e.platforms}
    lerd={e.lerd}
    tags={e.tags}
    onclose={() => (open = null)}
  >
    {#snippet actions()}
      {#if e.installed}
        <span class="text-xs font-medium text-emerald-600 dark:text-emerald-400">{m.system_plugins_installedBadge()}</span>
      {:else if e.runs_here === false}
        <span class="text-xs text-gray-500 dark:text-gray-400">{m.system_plugins_platformOnly({ list: platformNames(e.platforms) })}</span>
      {:else if !e.compatible}
        <span class="text-xs text-gray-500 dark:text-gray-400">{m.system_plugins_needsLerd({ constraint: e.lerd ?? '' })}</span>
      {:else if e.missing?.length}
        <span class="text-xs text-red-600 dark:text-red-400">{m.system_plugins_missing({ list: e.missing.join(', ') })}</span>
      {:else}
        <DetailButton tone="primary" onclick={() => (confirm = e)} loading={installing === e.name} disabled={installing !== ''}>
          {m.common_install()}
        </DetailButton>
      {/if}
    {/snippet}
  </PluginDetailModal>
{/if}

<ConfirmModal
  open={confirm !== null}
  title={m.system_plugins_installConfirmTitle({ name: confirm?.name ?? '' })}
  body={m.system_plugins_installConfirmBody({ commit: confirm?.commit.slice(0, 7) ?? '' })}
  confirmLabel={m.common_install()}
  loading={installing !== ''}
  onconfirm={install}
  onclose={() => (confirm = null)}
/>
