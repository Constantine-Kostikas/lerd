<script lang="ts">
  import PluginArt from './PluginArt.svelte';
  import { m } from '../../paraglide/messages.js';

  // The catalog's last card, shaped like a listing with the illustration as its
  // screenshot and a dashed edge: the plugin that isn't written yet.
  let { onopen }: { onopen: () => void } = $props();
</script>

<button
  type="button"
  onclick={onopen}
  class="group flex flex-col text-left rounded-xl border border-dashed overflow-hidden border-gray-300 dark:border-white/15 transition duration-150 hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/5 hover:border-gray-400 dark:hover:border-white/25"
>
  <div class="relative aspect-[16/10] flex items-center justify-center overflow-hidden border-b border-dashed border-gray-300 dark:border-white/15 bg-gray-50 dark:bg-lerd-bg p-3 *:w-full *:h-full">
    <PluginArt kind="catalog" />
  </div>
  <div class="flex-1 flex flex-col gap-1 p-3">
    <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">{m.system_plugins_buildTitle()}</span>
    <p class="text-xs text-gray-500 dark:text-gray-400 line-clamp-2">{m.system_plugins_buildBody()}</p>
    <div class="mt-auto pt-2 flex items-center justify-end text-[10px] font-medium text-lerd-red">{m.system_plugins_step3Title()} →</div>
  </div>
</button>
