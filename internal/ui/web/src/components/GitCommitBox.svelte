<script lang="ts">
  import DetailButton from '$components/DetailButton.svelte';
  import { commitChanges, loadLastMessage } from '$lib/gitDiff';
  import { m } from '../paraglide/messages.js';

  interface Props {
    domain: string;
    path: string;
    staged: number;
    oncommitted: () => void;
  }
  let { domain, path, staged, oncommitted }: Props = $props();

  let message = $state('');
  let amend = $state(false);
  let busy = $state<'commit' | 'push' | ''>('');
  let error = $state('');

  // An amend may only reword the last commit, so it needs nothing staged.
  const ready = $derived(!busy && message.trim() !== '' && (staged > 0 || amend));

  async function toggleAmend() {
    amend = !amend;
    if (amend && message.trim() === '') {
      try {
        message = await loadLastMessage(domain, path);
      } catch {
        /* the box stays empty to type into */
      }
    }
  }

  async function commit(push: boolean) {
    busy = push ? 'push' : 'commit';
    error = '';
    try {
      const res = await commitChanges(domain, path, message, amend, push);
      if (res.error) throw new Error(res.error);
      message = '';
      amend = false;
      oncommitted();
    } catch (e) {
      error = e instanceof Error ? e.message : m.common_failed();
    } finally {
      busy = '';
    }
  }
</script>

<div class="shrink-0 border-t border-gray-100 dark:border-lerd-border p-3 space-y-2">
  <label class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
    <input type="checkbox" checked={amend} onchange={toggleAmend} class="rounded-sm border-gray-300 dark:border-lerd-border" />
    {m.gitDiff_amend()}
  </label>
  <textarea
    bind:value={message}
    rows="4"
    spellcheck="false"
    placeholder={m.gitDiff_commitMessage()}
    class="w-full resize-y min-h-16 text-xs font-mono px-2.5 py-1.5 rounded-sm border border-gray-200 dark:border-lerd-border bg-white dark:bg-lerd-card text-gray-800 dark:text-gray-100 focus:outline-none focus:ring-1 focus:ring-lerd-red"
  ></textarea>
  {#if error}<p class="text-xs text-red-500 whitespace-pre-line">{error}</p>{/if}
  <div class="flex gap-2">
    <DetailButton tone="primary" disabled={!ready} loading={busy === 'commit'} onclick={() => commit(false)}>{m.gitDiff_commit()}</DetailButton>
    <DetailButton disabled={!ready} loading={busy === 'push'} onclick={() => commit(true)}>{m.gitDiff_commitPush()}</DetailButton>
  </div>
</div>
