<script lang="ts">
  import IconButton from './IconButton.svelte';
  import Icon, { type IconName } from './Icon.svelte';

  // A page's views as icon buttons at its header's far end, the current one
  // marked, so the header keeps only the title and its stats.
  interface View {
    title: string;
    icon: IconName;
    active: boolean;
    onclick: () => void;
  }
  let { views }: { views: View[] } = $props();
</script>

<div class="flex items-center gap-1">
  {#each views as v (v.title)}
    <IconButton size="sm" placement="bottom" title={v.title} active={v.active} onclick={v.onclick}>
      <Icon name={v.icon} class="w-4 h-4" />
    </IconButton>
  {/each}
</div>
