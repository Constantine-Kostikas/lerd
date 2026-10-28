<script lang="ts">
  import StatusDot from '$components/StatusDot.svelte';
  import MoonMark from '$components/MoonMark.svelte';
  import type { Service } from '$stores/services';
  import { m } from '../paraglide/messages.js';

  interface Props {
    svc: Pick<Service, 'status' | 'idle_suspended'> | undefined;
    size?: 'xs' | 'sm';
  }
  let { svc, size = 'sm' }: Props = $props();
  // The moon is wider than the dot, so both sit in its box and a sleeping
  // row's name starts where every other row's does.
  const box = $derived(`inline-flex shrink-0 items-center justify-center ${size === 'xs' ? 'w-2.5 h-2.5' : 'w-3 h-3'}`);
</script>

{#if svc?.idle_suspended && svc.status !== 'active'}
  <span class="{box} text-sky-500 dark:text-sky-400" title={m.services_sleeping()} data-testid="service-moon">
    <MoonMark class="w-full h-full" />
  </span>
{:else}
  <span class={box}><StatusDot color={svc?.status === 'active' ? 'green' : 'gray'} {size} /></span>
{/if}
