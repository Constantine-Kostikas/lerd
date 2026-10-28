<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    stats?: Snippet;
    trailing?: Snippet;
  }
  let { title, stats, trailing }: Props = $props();
</script>

<div class="shrink-0 flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-1.5 page-header">
  <h1 class="text-base font-semibold text-gray-900 dark:text-white">{title}</h1>
  <div class="flex items-center gap-4">
    {#if stats}
      <div class="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        {@render stats()}
      </div>
    {/if}
    {#if trailing}{@render trailing()}{/if}
  </div>
</div>
