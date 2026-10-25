<script lang="ts">
  import Icon from '$components/Icon.svelte';
  import Toggle from '$components/Toggle.svelte';
  import { m } from '../paraglide/messages.js';

  // Offered in place of the conflicts warning: uncommitted work in the way is set aside and put back after.
  let { files, on = $bindable() }: { files: string[]; on: boolean } = $props();
</script>

<div class="flex items-center gap-3 rounded-lg border border-gray-200 dark:border-lerd-border px-3 py-2.5">
  <span class="w-8 h-8 shrink-0 rounded-md flex items-center justify-center {on
    ? 'bg-lerd-red/10 text-lerd-red'
    : 'bg-gray-100 dark:bg-white/5 text-gray-400 dark:text-gray-500'}">
    <Icon name="clipboard" class="w-4 h-4" />
  </span>
  <div class="flex-1 min-w-0">
    <div class="text-sm text-gray-800 dark:text-gray-100">{m.pullPlan_stash()}</div>
    <div class="text-xs text-gray-500 dark:text-gray-400 truncate" title={files.join('\n')}>
      {m.pullPlan_stashReason({ files: files.join(', ') })}
    </div>
  </div>
  <Toggle {on} title={m.pullPlan_stash()} onclick={() => (on = !on)} />
</div>
