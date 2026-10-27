<script lang="ts">
  import BuildLog from '$components/BuildLog.svelte';
  import type { StashKept } from '$stores/worktree';
  import { m } from '../paraglide/messages.js';

  // The running or finished state of a switch or pull: git's refusal, the local
  // changes the stash kept, what the snapshot taken first means now, and the output.
  let { error, note, logs, stash = null }: { error: string; note: string; logs: string[]; stash?: StashKept | null } = $props();
</script>

<div class="px-5 py-3 space-y-2">
  {#if error}
    <div class="rounded-lg border border-red-200 dark:border-red-500/30 bg-red-50 dark:bg-red-500/10 px-3 py-2 text-xs text-red-700 dark:text-red-300 whitespace-pre-line">
      {error}
    </div>
  {/if}
  {#if stash}
    <div class="rounded-lg border border-amber-200 dark:border-amber-500/30 bg-amber-50 dark:bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-300">
      {stash.files?.length ? m.pullPlan_stashKept({ files: stash.files.join(', '), label: stash.label }) : m.pullPlan_stashKeptOther({ label: stash.label })}
    </div>
  {/if}
  {#if note}
    <div class="rounded-lg border border-gray-200 dark:border-lerd-border px-3 py-2.5 text-xs text-gray-700 dark:text-gray-300">
      {note}
    </div>
  {/if}
  {#if logs.length > 0}<BuildLog {logs} />{/if}
</div>
