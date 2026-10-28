<script lang="ts">
  import DetailButton from '$components/DetailButton.svelte';
  import Dropdown from '$components/Dropdown.svelte';
  import Modal from '$components/Modal.svelte';
  import Icon from '$components/Icon.svelte';
  import ServiceCardShell from '$components/ServiceCardShell.svelte';
  import ServiceIcon from '$components/ServiceIcon.svelte';
  import { type CategoryKey } from '$lib/presetCategories';
  import { installPresetAndOpen, presetAddLabel, availableVersions, type Preset } from '$stores/presets';
  import { m } from '../../paraglide/messages.js';
  import { serviceLabel } from '$stores/services';

  interface Props {
    preset: Preset;
    category?: CategoryKey;
  }
  let { preset, category }: Props = $props();

  // A preset with versions asks which one first; the tag shown starts on the
  // one the store would pick, so a plain Add keeps the old behaviour.
  let picking = $state(false);
  let version = $state('');
  const versions = $derived(availableVersions(preset));
  const installedTags = $derived(preset.installed_tags ?? []);

  function add() {
    if (preset.installing) return;
    if (!versions.length) return void installPresetAndOpen(preset);
    version = preset.selected_version || versions[0].tag;
    picking = true;
  }

  function installPicked() {
    picking = false;
    void installPresetAndOpen({ ...preset, selected_version: version });
  }
</script>

{#snippet plusIcon()}<Icon name="plus" class="w-4 h-4" />{/snippet}

<ServiceCardShell>
  <ServiceIcon
    name={preset.name}
    category={category ?? preset.category}
    icon={preset.icon}
    color={preset.color}
  />
  <div class="min-w-0 flex-1">
    <div class="text-sm font-semibold text-gray-900 dark:text-white truncate" title={serviceLabel(preset.name)}>{serviceLabel(preset.name)}</div>
    {#if preset.error}
      <p class="text-[11px] leading-snug text-red-500 truncate" title={preset.error}>{preset.error}</p>
    {:else if installedTags.length}
      <p class="text-[11px] leading-snug text-emerald-600 dark:text-emerald-400 truncate">{m.services_preset_installedTag({ tags: installedTags.join(', ') })}</p>
    {:else if preset.description}
      <p class="text-[11px] leading-snug text-gray-500 dark:text-gray-400 truncate" title={preset.description}>{preset.description}</p>
    {/if}
  </div>
  <div class="shrink-0">
    <DetailButton
      tone="secondary"
      onclick={add}
      disabled={Boolean(preset.installing)}
      loading={Boolean(preset.installing)}
      title={preset.installingMessage || presetAddLabel(preset)}
      icon={plusIcon}
    />
  </div>
</ServiceCardShell>

<Modal open={picking} title={m.services_dash_addTitle({ name: serviceLabel(preset.name) })} size="sm" onclose={() => (picking = false)}>
  <div class="px-5 py-4">
    <Dropdown value={version} options={versions.map((v) => ({ value: v.tag, label: v.label || v.tag }))} onchange={(v) => (version = v)} />
  </div>
  {#snippet footer()}
    <DetailButton onclick={() => (picking = false)}>{m.common_cancel()}</DetailButton>
    <DetailButton tone="primary" onclick={installPicked}>{m.common_add()}</DetailButton>
  {/snippet}
</Modal>
