<script lang="ts">
  import StripeControl from './StripeControl.svelte';

  interface Props {
    running?: boolean;
    loading?: boolean;
    webhookPath?: string;
    secretEnvKey?: string;
    projectName?: string;
    onToggle?: () => void;
    onSaveConfig?: (
      path: string,
      secretEnvKey: string,
      projectName: string
    ) => Promise<{ ok: boolean; error?: string }>;
  }
  let {
    running = false,
    loading = false,
    webhookPath = '',
    secretEnvKey = '',
    projectName = '',
    onToggle = () => {},
    onSaveConfig = async () => ({ ok: true })
  }: Props = $props();
</script>

<StripeControl {running} {loading} {webhookPath} {secretEnvKey} {projectName} {onToggle} {onSaveConfig} />
