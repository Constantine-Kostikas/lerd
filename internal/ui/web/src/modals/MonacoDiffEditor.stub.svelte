<script lang="ts">
  let {
    original,
    modified,
    language,
    ignoreSpace,
    sideBySide
  }: {
    original: string;
    modified: string;
    language: string;
    ignoreSpace: boolean;
    sideBySide: boolean;
  } = $props();
</script>

<div
  data-testid="diff"
  data-original={original}
  data-modified={modified}
  data-language={language}
  data-ignore-space={String(ignoreSpace)}
  data-side-by-side={String(sideBySide)}
></div>
