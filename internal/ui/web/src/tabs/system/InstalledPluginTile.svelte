<script lang="ts">
  import ServiceCardShell from '$components/ServiceCardShell.svelte';
  import PluginIcon from '$components/PluginIcon.svelte';
  import StatusDot from '$components/StatusDot.svelte';
  import { marquee } from '$lib/marquee';
  import type { ManagedPlugin } from '$stores/plugins';
  import { m } from '../../paraglide/messages.js';

  // An installed plugin as a card beside the service ones: what it is and
  // whether it loaded, with the rest in its detail.
  interface Props {
    plugin: ManagedPlugin;
    onopen: () => void;
  }
  let { plugin: p, onopen }: Props = $props();

  const dot = $derived(p.error ? 'red' : p.failure ? 'amber' : 'green');
</script>

<ServiceCardShell>
  <button type="button" onclick={onopen} class="flex min-w-0 flex-1 items-center gap-3 text-left">
    <span class="shrink-0 flex items-center justify-center w-10 h-10 rounded-lg bg-gray-100 dark:bg-white/5 text-gray-600 dark:text-gray-300">
      <PluginIcon icon={p.icon} />
    </span>
    <span class="min-w-0 flex-1">
      <span class="flex items-center gap-1.5">
        <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">{p.name}</span>
        {#if p.version}<span class="shrink-0 text-[10px] font-mono tabular-nums text-gray-400">{p.version}</span>{/if}
        {#if p.update}<span class="shrink-0 text-[10px] font-medium font-mono text-emerald-600 dark:text-emerald-400" title={m.system_plugins_updateAvailable()}>↑ {p.update_version ?? ''}</span>{/if}
      </span>
      <span class="flex items-center gap-1.5 text-[10px] text-gray-500 dark:text-gray-400">
        <StatusDot color={dot} />
        <span class="truncate {p.error ? 'text-red-600 dark:text-red-400' : ''}" use:marquee={'button'}><span>{p.error ? m.system_plugins_broken() : p.description}</span></span>
      </span>
    </span>
  </button>
</ServiceCardShell>
