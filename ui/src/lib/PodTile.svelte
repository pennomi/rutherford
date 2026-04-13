<script lang="ts">
  import { cluster } from './state.svelte';
  import { podStatus, podAction, failedStates } from './pod';
  import { formatAge, parseCPU, parseMemory } from './format';
  import AppIcon from './AppIcon.svelte';
  import type { Pod } from './types';

  const ICON_ANNOTATION = 'rutherford/icon';

  interface Props {
    pod: Pod;
  }

  let { pod }: Props = $props();

  let nsName = $derived(pod.metadata.namespace);
  let podName = $derived(pod.metadata.name);

  let labels = $derived(pod.metadata.labels ?? {});

  let appName = $derived(
    labels['app.kubernetes.io/name']
      ?? labels['app.kubernetes.io/instance']
      ?? labels['app']
      ?? podName
  );

  let displayName = $derived(
    appName.split(/[-_]/).map(w => w[0]?.toUpperCase() + w.slice(1)).join(' ')
  );

  let version = $derived(
    labels['app.kubernetes.io/version'] ?? labels['version'] ?? ''
  );

  let component = $derived(labels['app.kubernetes.io/component'] ?? '');

  let icon = $derived(
    pod.metadata.annotations?.[ICON_ANNOTATION]
      ?? cluster.namespaces[nsName]?.metadata.annotations?.[ICON_ANNOTATION]
      ?? ''
  );

  type Status = 'healthy' | 'warning' | 'error';

  let status = $derived.by((): { level: Status; label: string } => {
    const s = podStatus(pod);
    if (failedStates.has(s)) return { level: 'error', label: s };
    if (s !== 'Running') return { level: 'warning', label: s };
    return { level: 'healthy', label: 'Running' };
  });

  let action = $derived(podAction(pod));

  let statuses = $derived(pod.status?.containerStatuses ?? []);
  let readyCount = $derived(statuses.filter(c => c.ready).length);
  let totalContainers = $derived(pod.spec?.containers?.length ?? 0);
  let restartCount = $derived(statuses.reduce((n, c) => n + c.restartCount, 0));

  let age = $derived(formatAge(pod.metadata.creationTimestamp));

  const statusConfig: Record<Status, { textClass: string; borderClass: string; dotClass: string }> = {
    healthy: { textClass: 'text-emerald-500', borderClass: 'border-l-emerald-500', dotClass: 'bg-emerald-500' },
    warning: { textClass: 'text-amber-500',   borderClass: 'border-l-amber-500',  dotClass: 'bg-amber-500' },
    error:   { textClass: 'text-red-500',     borderClass: 'border-l-red-500',    dotClass: 'bg-red-500' },
  };

  let config = $derived(statusConfig[status.level]);

  // Pod → matching Services (selector ⊆ pod labels) → Ingresses pointing at those services.
  let ingressHosts = $derived.by(() => {
    const matchingServiceNames = new Set<string>();
    for (const svc of Object.values(cluster.services)) {
      if (svc.metadata.namespace !== nsName) continue;
      const sel = svc.spec.selector;
      if (!sel) continue;
      const keys = Object.keys(sel);
      if (keys.length === 0) continue;
      let ok = true;
      for (const k of keys) {
        if (labels[k] !== sel[k]) { ok = false; break; }
      }
      if (ok) matchingServiceNames.add(svc.metadata.name);
    }

    const hosts = new Set<string>();
    for (const ing of Object.values(cluster.ingresses)) {
      if (ing.metadata.namespace !== nsName) continue;
      for (const rule of ing.spec.rules) {
        const paths = rule.http?.paths ?? [];
        for (const p of paths) {
          if (matchingServiceNames.has(p.backend?.service?.name)) {
            if (rule.host) hosts.add(rule.host);
          }
        }
      }
    }
    return [...hosts];
  });

  const HOT_THRESHOLD = 0.8;
  const CRITICAL_THRESHOLD = 0.95;

  type HeatLevel = 'warm' | 'critical';

  interface HotResource {
    hot: boolean;
    maxRatio: number;
    level: HeatLevel;
  }

  let hotResources = $derived.by((): { cpu: HotResource; memory: HotResource } => {
    let cpuMax = 0;
    let memMax = 0;

    const key = `${nsName}/${podName}`;
    const pm = cluster.podMetrics[key];
    if (pm) {
      for (const container of pod.spec.containers) {
        const limits = container.resources?.limits;
        if (!limits) continue;
        const mc = pm.containers.find(c => c.name === container.name);
        if (!mc) continue;

        if (limits.cpu) {
          const lim = parseCPU(limits.cpu);
          const use = parseCPU(mc.usage.cpu);
          if (lim > 0) cpuMax = Math.max(cpuMax, use / lim);
        }
        if (limits.memory) {
          const lim = parseMemory(limits.memory);
          const use = parseMemory(mc.usage.memory);
          if (lim > 0) memMax = Math.max(memMax, use / lim);
        }
      }
    }

    return {
      cpu:    { hot: cpuMax >= HOT_THRESHOLD, maxRatio: cpuMax, level: cpuMax >= CRITICAL_THRESHOLD ? 'critical' : 'warm' },
      memory: { hot: memMax >= HOT_THRESHOLD, maxRatio: memMax, level: memMax >= CRITICAL_THRESHOLD ? 'critical' : 'warm' },
    };
  });

  function heatColor(level: HeatLevel): string {
    return level === 'critical' ? 'text-red-500' : 'text-amber-500';
  }
  function heatBg(level: HeatLevel): string {
    return level === 'critical' ? 'bg-red-500' : 'bg-amber-500';
  }
</script>

<a
  href="/namespace/{nsName}/pod/{podName}"
  class="bg-neutral-700 rounded-lg p-4 hover:bg-neutral-600 transition-colors block shadow-lg border-l-4 {config.borderClass}"
>
  <!-- Icon + name -->
  <div class="flex items-start gap-3 mb-3">
    {#if icon}
      <AppIcon src={icon} size="w-20 h-20" />
    {:else}
      <div class="w-20 h-20 shrink-0 rounded-md bg-neutral-800 flex items-center justify-center">
        <span class="text-neutral-400 text-3xl font-bold">{appName[0]?.toUpperCase()}</span>
      </div>
    {/if}
    <div class="flex-1 min-w-0">
      <div class="font-semibold text-neutral-100 text-base truncate">{displayName}</div>
      <div class="font-mono text-xs text-neutral-500 truncate" title={podName}>{podName}</div>
      {#if component}
        <div class="text-xs text-neutral-400 truncate">{component}</div>
      {/if}
      {#each ingressHosts as host}
        <span role="link" tabindex="0" class="text-xs text-neutral-400 hover:text-neutral-200 inline-block mt-1 transition-colors cursor-pointer truncate" onclick={(e: MouseEvent) => { e.preventDefault(); e.stopPropagation(); window.open(`https://${host}`, '_blank', 'noopener'); }} onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') { e.preventDefault(); e.stopPropagation(); window.open(`https://${host}`, '_blank', 'noopener'); }}}>{host} ⧉</span>
      {/each}
    </div>
  </div>

  <!-- Health status -->
  <div class="flex items-center gap-2 mb-3">
    {#if status.level === 'healthy'}
      <div class="w-5 h-5 shrink-0 {config.dotClass} status-icon" style="-webkit-mask-image: url(/icons/healthy.svg); mask-image: url(/icons/healthy.svg);"></div>
    {:else if status.level === 'warning'}
      <div class="w-5 h-5 shrink-0 {config.dotClass} status-icon animate-spin" style="-webkit-mask-image: url(/icons/working.svg); mask-image: url(/icons/working.svg);"></div>
    {:else}
      <div class="w-5 h-5 shrink-0 {config.dotClass} status-icon error-flash" style="-webkit-mask-image: url(/icons/error.svg); mask-image: url(/icons/error.svg);"></div>
    {/if}
    <span class="text-sm font-semibold {config.textClass} truncate">{action || status.label}</span>
  </div>

  <!-- Stats -->
  <div class="flex flex-wrap gap-x-3 gap-y-1 text-xs">
    <span><span class="text-neutral-500">Ready</span> <span class="{readyCount < totalContainers ? 'text-amber-400' : 'text-neutral-400'}">{readyCount}/{totalContainers}</span></span>
    {#if restartCount > 0}
      <span><span class="text-neutral-500">Restarts</span> <span class="{restartCount > 5 ? 'text-amber-400' : 'text-neutral-400'}">{restartCount}</span></span>
    {/if}
    <span><span class="text-neutral-500">Age</span> <span class="text-neutral-400">{age}</span></span>
    {#if version}
      <span><span class="text-neutral-500">v</span><span class="text-neutral-400">{version}</span></span>
    {/if}
    {#if hotResources.cpu.hot}
      <span class="inline-flex items-center gap-0.5 {heatColor(hotResources.cpu.level)}">
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.cpu.level)}" style="-webkit-mask-image: url(/icons/flame.svg); mask-image: url(/icons/flame.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.cpu.level)}" style="-webkit-mask-image: url(/icons/cpu.svg); mask-image: url(/icons/cpu.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span>{Math.round(hotResources.cpu.maxRatio * 100)}%</span>
      </span>
    {/if}
    {#if hotResources.memory.hot}
      <span class="inline-flex items-center gap-0.5 {heatColor(hotResources.memory.level)}">
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.memory.level)}" style="-webkit-mask-image: url(/icons/flame.svg); mask-image: url(/icons/flame.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.memory.level)}" style="-webkit-mask-image: url(/icons/memory.svg); mask-image: url(/icons/memory.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span>{Math.round(hotResources.memory.maxRatio * 100)}%</span>
      </span>
    {/if}
  </div>
</a>
