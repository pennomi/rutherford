<script lang="ts">
  import { page } from '$app/state';
  import { cluster } from '$lib/state.svelte';
  import BreadcrumbItem from './BreadcrumbItem.svelte';

  interface Crumb {
    label: string;
    resourceType: string;
    href: string;
  }

  function displayName(name: string): string {
    return name.split(/[-_]/).map(w => w[0].toUpperCase() + w.slice(1)).join(' ');
  }

  let breadcrumbs = $derived.by((): Crumb[] => {
    const path = page.url.pathname;
    const crumbs: Crumb[] = [{ label: 'Cluster', resourceType: 'Cluster', href: '/' }];

    const nsMatch = path.match(/^\/namespace\/([^/]+)/);
    if (nsMatch) {
      const ns = decodeURIComponent(nsMatch[1]);
      crumbs.push({ label: displayName(ns), resourceType: 'Namespace', href: `/namespace/${nsMatch[1]}` });

      const podMatch = path.match(/^\/namespace\/[^/]+\/pod\/([^/]+)/);
      if (podMatch) {
        const podName = decodeURIComponent(podMatch[1]);
        crumbs.push({ label: podName, resourceType: 'Pod', href: path });
      }
    }

    return crumbs;
  });
</script>

<nav class="flex items-center gap-1.5 text-md">
  {#each breadcrumbs as crumb, i}
    {#if i > 0}
      <span class="text-zinc-600">/</span>
    {/if}
    <BreadcrumbItem
      label={crumb.label}
      resourceType={crumb.resourceType}
      href={crumb.href}
      isLast={i === breadcrumbs.length - 1}
    />
  {/each}
</nav>
