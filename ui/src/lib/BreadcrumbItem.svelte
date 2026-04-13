<script lang="ts">
  interface Props {
    label: string;
    resourceType: string;
    href: string;
    isLast: boolean;
  }

  let { label, resourceType, href, isLast }: Props = $props();
</script>

{#if isLast}
  <div class="flex flex-col">
    <span class="text-zinc-200 text-sm leading-tight">{label}</span>
    <span class="text-zinc-500 text-[10px] uppercase tracking-wider leading-tight">{resourceType}</span>
  </div>
{:else}
  <a {href} class="flex flex-col group">
    <span class="text-zinc-400 text-sm leading-tight group-hover:text-zinc-200">{label}</span>
    <span class="text-zinc-600 text-[10px] uppercase tracking-wider leading-tight">{resourceType}</span>
  </a>
{/if}
