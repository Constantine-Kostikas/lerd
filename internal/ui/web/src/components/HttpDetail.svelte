<script lang="ts">
  import { m } from '../paraglide/messages.js';

  type Detail = { headers?: string[]; body?: string; truncated?: boolean };
  let { label, detail }: { label: string; detail?: Detail } = $props();

  const headers = $derived(
    (detail?.headers ?? []).map((h) => {
      const at = h.indexOf(':');
      return at < 0 ? [h, ''] : [h.slice(0, at).trim(), h.slice(at + 1).trim()];
    })
  );
  // A cut body is no longer valid JSON, so it falls through and shows as sent.
  const body = $derived.by(() => {
    const raw = detail?.body ?? '';
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  });
</script>

{#if headers.length || body}
  <div>
    <div class="text-gray-400 mb-0.5">{label}</div>
    {#if headers.length}
      <table class="w-full border-collapse font-mono">
        <tbody>
          {#each headers as [name, value], i (i)}
            <tr class="border-t border-gray-100 dark:border-lerd-border/40 align-top">
              <td class="py-0.5 pr-3 text-gray-500 dark:text-gray-400 whitespace-nowrap w-px">{name}</td>
              <td class="py-0.5 break-all">{value}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
    {#if body}
      <pre class="mt-1 max-h-64 overflow-auto whitespace-pre-wrap break-all rounded-sm bg-gray-50 dark:bg-white/5 p-1.5 text-gray-700 dark:text-gray-300">{body}</pre>
      {#if detail?.truncated}<div class="text-gray-400 mt-0.5">{m.http_truncated()}</div>{/if}
    {/if}
  </div>
{/if}
