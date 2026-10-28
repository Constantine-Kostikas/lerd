<script lang="ts">
  import type { Snippet } from 'svelte';

  // A catalog's search box and its category chips, the empty key meaning all.
  // The chips only show when there is more than one category to pick between.
  interface Props {
    query: string;
    category: string;
    categories: { key: string; label: string }[];
    placeholder: string;
    groupLabel: string;
    trailing?: Snippet;
  }
  let { query = $bindable(), category = $bindable(), categories, placeholder, groupLabel, trailing }: Props = $props();
</script>

<div class="flex flex-wrap items-center gap-2">
  <input
    type="search"
    bind:value={query}
    {placeholder}
    aria-label={placeholder}
    class="flex-1 min-w-48 h-7 text-xs bg-white dark:bg-lerd-card border border-gray-200 dark:border-lerd-border rounded-md px-2.5 text-gray-700 dark:text-gray-200 placeholder:text-gray-400 focus:outline-hidden focus:border-lerd-red/50"
  />
  {#if trailing}{@render trailing()}{/if}
</div>
{#if categories.length > 2}
  <div class="flex flex-wrap gap-1.5" role="group" aria-label={groupLabel}>
    {#each categories as c (c.key)}
      <button
        type="button"
        aria-pressed={category === c.key}
        onclick={() => (category = c.key)}
        class="h-6 px-2.5 rounded-md border text-[11px] font-medium transition-colors {category === c.key
          ? 'border-lerd-red/50 text-lerd-red bg-lerd-red/5'
          : 'border-gray-200 dark:border-lerd-border text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'}"
      >{c.label}</button>
    {/each}
  </div>
{/if}
