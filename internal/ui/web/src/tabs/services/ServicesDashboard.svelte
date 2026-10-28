<script lang="ts">
  import { onMount } from 'svelte';
  import StatusDot from '$components/StatusDot.svelte';
  import Icon from '$components/Icon.svelte';
  import DashboardHeader from '$components/DashboardHeader.svelte';
  import DashboardSection from '$components/DashboardSection.svelte';
  import ServiceCardShell from '$components/ServiceCardShell.svelte';
  import ViewSwitch from '$components/ViewSwitch.svelte';
  import CatalogFilter from '$components/CatalogFilter.svelte';
  import InstalledServiceTile from './InstalledServiceTile.svelte';
  import PresetCard from './PresetCard.svelte';
  import UpdateAllModal from './UpdateAllModal.svelte';
  import {
    coreServices,
    servicesLoaded,
    serviceLabel,
    type Service,
    type ServiceUpdateResult
  } from '$stores/services';
  import { installablePresets, presetsLoaded, loadPresets } from '$stores/presets';
  import { accessMode } from '$stores/accessMode';
  import { routeRest, goToTab } from '$stores/route';
  import { groupByCategory, categoryOf, CATEGORY_LABELS, CATEGORY_ORDER, asCategory } from '$lib/presetCategories';
  import { m } from '../../paraglide/messages.js';

  onMount(() => {
    loadPresets();
  });

  const running = $derived($coreServices.filter((s) => s.status === 'active').length);
  const total = $derived($coreServices.length);
  const outdated = $derived($coreServices.filter((s) => s.update_available));
  const updates = $derived(outdated.length);
  // Grouped by type like the rail, and within a type the services waiting on an
  // update lead so they are not lost among the rest.
  const installedGroups = $derived(
    groupByCategory($coreServices).map((g) => ({ ...g, items: [...g.items.filter((s) => s.update_available), ...g.items.filter((s) => !s.update_available)] }))
  );

  // #services is the installed list, #services/discover the presets to add.
  const discovering = $derived($routeRest === 'discover');
  let query = $state('');
  let category = $state('');
  const views = $derived([
    { title: m.services_dash_installed(), icon: 'grid' as const, active: !discovering, onclick: () => goToTab('services') },
    { title: m.services_dash_discover(), icon: 'download' as const, active: discovering, onclick: () => goToTab('services', 'discover') }
  ]);

  // A snapshot of what was outdated when the dialog opened, null while it is closed.
  let updateBatch = $state<Service[] | null>(null);
  let results = $state<ServiceUpdateResult[]>([]);
  const updated = $derived(results.filter((r) => r.ok).length);
  const failed = $derived(results.filter((r) => !r.ok));

  function openUpdateAll() {
    results = [];
    updateBatch = [...outdated];
  }
  const sitesServed = $derived(
    new Set($coreServices.flatMap((s) => s.site_domains ?? [])).size
  );

  const shownPresets = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return $installablePresets.filter(
      (p) =>
        (!category || categoryOf(p) === category) &&
        (!q || [p.name, serviceLabel(p.name), p.description ?? ''].some((t) => t.toLowerCase().includes(q)))
    );
  });
  const groups = $derived(groupByCategory(shownPresets));
  const categories = $derived([
    { key: '', label: m.services_cat_all() },
    ...CATEGORY_ORDER.filter((k) => $installablePresets.some((p) => asCategory(p.category) === k)).map((k) => ({ key: k, label: CATEGORY_LABELS[k]() }))
  ]);
</script>

<div class="flex-1 flex flex-col min-h-0">
  <DashboardHeader title={m.nav_services()} stats={$servicesLoaded && total > 0 ? summary : undefined} trailing={$accessMode.localControl ? viewSwitch : undefined} />

  <!-- Keyed so each view opens at its top, not where the other was scrolled to. -->
  {#key discovering}
  <div class="flex-1 min-h-0 overflow-y-auto p-4 space-y-6">
    {#if !discovering}
      <section class="space-y-4">
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.services_dash_installed()}</h2>
          {#if $accessMode.localControl && updates > 0}
            <button
              onclick={openUpdateAll}
              disabled={updateBatch !== null}
              class="inline-flex items-center gap-1 text-xs font-medium text-emerald-600 hover:text-emerald-700 disabled:opacity-50 dark:text-emerald-400 dark:hover:text-emerald-300"
            >
              <span>↑</span>
              {m.services_updateAll()}
            </button>
          {/if}
        </div>
        {#if results.length > 0}
          <div class="text-xs space-y-0.5">
            {#if updated > 0}
              <p class="text-emerald-600 dark:text-emerald-400">{m.services_updateAll_done({ count: updated })}</p>
            {/if}
            {#each failed as r (r.name)}
              <p class="text-red-600 dark:text-red-400">{serviceLabel(r.name)}: {r.error}</p>
            {/each}
          </div>
        {/if}
        {#if $servicesLoaded && total === 0}
          <p class="text-sm text-gray-500 dark:text-gray-400">{m.services_dash_noInstalled()}</p>
        {/if}
        {#each installedGroups as group (group.key)}
          <DashboardSection label={CATEGORY_LABELS[group.key]()} sub>
            {#each group.items as svc (svc.name)}
              <InstalledServiceTile {svc} />
            {/each}
          </DashboardSection>
        {/each}
        {#if $accessMode.localControl}
          <DashboardSection>
            <ServiceCardShell suggested>
              <button type="button" onclick={() => goToTab('services', 'discover')} class="flex min-w-0 flex-1 items-center gap-3 text-left text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-lerd-red">
                <span class="shrink-0 flex items-center justify-center w-10 h-10 rounded-lg bg-lerd-red/10 text-lerd-red"><Icon name="plus" class="w-4 h-4" /></span>
                {m.services_dash_addService()}
              </button>
            </ServiceCardShell>
          </DashboardSection>
        {/if}
      </section>
    {:else if $accessMode.localControl}
      <section class="space-y-3">
        <CatalogFilter bind:query bind:category {categories} placeholder={m.services_preset_searchPlaceholder()} groupLabel={m.services_dash_discover()} />
        {#if !$presetsLoaded}
          <p class="text-xs text-gray-400">{m.common_loading()}</p>
        {:else if $installablePresets.length === 0}
          <p class="text-sm text-gray-500 dark:text-gray-400">{m.services_preset_allInstalled()}</p>
        {:else if groups.length === 0}
          <p class="text-sm text-gray-500 dark:text-gray-400">{m.services_preset_noMatches()}</p>
        {:else}
          <div class="space-y-4">
            {#each groups as group (group.key)}
              <DashboardSection label={CATEGORY_LABELS[group.key]()} sub>
                {#each group.items as preset (preset.name)}
                  <PresetCard {preset} category={group.key} />
                {/each}
              </DashboardSection>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  </div>
  {/key}
</div>

{#if updateBatch}
  <UpdateAllModal
    services={updateBatch}
    onclose={() => (updateBatch = null)}
    ondone={(r) => (results = r)}
  />
{/if}

{#snippet viewSwitch()}<ViewSwitch {views} />{/snippet}

{#snippet summary()}
  <span class="inline-flex items-center gap-1.5">
    <StatusDot color={running > 0 ? 'green' : 'gray'} />
    {m.dashboard_services_summary({ running, total })}
  </span>
  {#if updates > 0}
    <span class="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
      <span>↑</span>
      {m.dashboard_services_updates({ count: updates })}
    </span>
  {/if}
  {#if sitesServed > 0}
    <span class="inline-flex items-center gap-1.5">
      <Icon name="sites" class="w-3.5 h-3.5" />
      {m.services_dash_sitesServed({ count: sitesServed })}
    </span>
  {/if}
{/snippet}
