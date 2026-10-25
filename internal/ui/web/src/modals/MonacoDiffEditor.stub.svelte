<script lang="ts">
  import type { Hunk } from '$lib/gitDiff';
  let {
    original,
    modified,
    language,
    ignoreSpace,
    sideBySide,
    hunkAction
  }: {
    original: string;
    modified: string;
    language: string;
    ignoreSpace: boolean;
    sideBySide: boolean;
    hunkAction?: { label: string; unstage: boolean; run: (h: Hunk) => void };
  } = $props();
  const hunk: Hunk = { orig_start: 1, orig_end: 1, mod_start: 1, mod_end: 1, removed: ['a'], added: ['b'] };
</script>

<div
  data-testid="diff"
  data-original={original}
  data-modified={modified}
  data-language={language}
  data-ignore-space={String(ignoreSpace)}
  data-side-by-side={String(sideBySide)}
></div>
{#if hunkAction}<button type="button" onclick={() => hunkAction.run(hunk)}>{hunkAction.label}</button>{/if}
