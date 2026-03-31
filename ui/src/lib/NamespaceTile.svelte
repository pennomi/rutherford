<script lang="ts">
  import { cluster } from './state.svelte';
  import { podStatus, podAction, failedStates } from './pod';
  import { formatAge, parseCPU, parseMemory } from './format';
  import AppIcon from './AppIcon.svelte';

  const ICON_ANNOTATION = 'rutherford/icon';

  interface Props {
    name: string;
  }

  let { name }: Props = $props();

  let displayName = $derived(
    name.split(/[-_]/).map(w => w[0].toUpperCase() + w.slice(1)).join(' ')
  );

  type Status = 'healthy' | 'warning' | 'error' | 'empty';

  let namespacePods = $derived(
    Object.values(cluster.pods).filter(
      p => p.metadata.namespace === name
        && p.status?.phase !== 'Succeeded'
        && p.status?.phase !== 'Failed'
    )
  );

  let podCount = $derived(namespacePods.length);

  let healthyCount = $derived(
    namespacePods.filter(p => podStatus(p) === 'Running').length
  );

  let worstPodStatus = $derived.by((): { level: Status; label: string } => {
    if (podCount === 0) return { level: 'empty', label: 'No Pods' };
    let worstLabel = '';
    let worstLevel: Status = 'healthy';
    for (const pod of namespacePods) {
      const s = podStatus(pod);
      if (failedStates.has(s)) {
        return { level: 'error', label: s };
      }
      if (s !== 'Running' && worstLevel !== 'warning') {
        worstLevel = 'warning';
        worstLabel = s;
      }
    }
    if (worstLevel === 'healthy') return { level: 'healthy', label: 'Healthy' };
    return { level: worstLevel, label: worstLabel };
  });

  let status = $derived(worstPodStatus.level);
  let statusLabel = $derived(worstPodStatus.label);

  let icon = $derived(
    cluster.namespaces[name]?.metadata.annotations?.[ICON_ANNOTATION] ?? ''
  );

  let uptime = $derived.by(() => {
    let oldest = '';
    for (const pod of namespacePods) {
      const ts = pod.metadata.creationTimestamp;
      if (!oldest || ts < oldest) oldest = ts;
    }
    if (!oldest) return '';
    return formatAge(oldest);
  });

  const statusConfig: Record<Status, { textClass: string; borderClass: string; dotClass: string }> = {
    healthy: { textClass: 'text-emerald-500', borderClass: 'border-l-emerald-500', dotClass: 'bg-emerald-500' },
    warning: { textClass: 'text-amber-500',   borderClass: 'border-l-amber-500',  dotClass: 'bg-amber-500' },
    error:   { textClass: 'text-red-500',     borderClass: 'border-l-red-500',    dotClass: 'bg-red-500' },
    empty:   { textClass: 'text-neutral-700', borderClass: 'border-l-neutral-700', dotClass: 'bg-neutral-700' },
  };

  let config = $derived(statusConfig[status]);

  let ingressHosts = $derived(
    Object.values(cluster.ingresses)
      .filter(ing => ing.metadata.namespace === name)
      .flatMap(ing => ing.spec.rules.map(r => r.host))
  );

  let namespacePvcs = $derived(
    Object.values(cluster.pvcs).filter(p => p.metadata.namespace === name)
  );
  let pvcCount = $derived(namespacePvcs.length);
  let pvcBoundCount = $derived(
    namespacePvcs.filter(p => p.status.phase === 'Bound').length
  );

  let cronJobCount = $derived(
    Object.values(cluster.cronJobs).filter(c => c.metadata.namespace === name).length
  );

  let secretCount = $derived(
    Object.values(cluster.secrets).filter(s => s.metadata.namespace === name).length
  );

  let failedJobCount = $derived(
    Object.values(cluster.jobs).filter(j => j.metadata.namespace === name && j.status.failed > 0).length
  );

  let action = $derived.by(() => {
    if (status === 'healthy' || status === 'empty') return '';
    for (const pod of namespacePods) {
      const s = podStatus(pod);
      if (failedStates.has(s) || s !== 'Running') {
        const a = podAction(pod);
        if (a) return a;
      }
    }
    return '';
  });

  const HOT_THRESHOLD = 0.8;
  const CRITICAL_THRESHOLD = 0.95;

  type HeatLevel = 'warm' | 'critical';

  interface HotResource {
    count: number;
    maxRatio: number;
    level: HeatLevel;
  }

  let hotResources = $derived.by((): { cpu: HotResource; memory: HotResource } => {
    let cpuCount = 0;
    let memCount = 0;
    let cpuMaxRatio = 0;
    let memMaxRatio = 0;

    for (const pod of namespacePods) {
      const key = `${pod.metadata.namespace}/${pod.metadata.name}`;
      const pm = cluster.podMetrics[key];
      if (!pm) continue;

      for (const container of pod.spec.containers) {
        const limits = container.resources.limits;
        if (!limits) continue;

        const metricsContainer = pm.containers.find(c => c.name === container.name);
        if (!metricsContainer) continue;

        if (limits.cpu) {
          const limitCpu = parseCPU(limits.cpu);
          const usageCpu = parseCPU(metricsContainer.usage.cpu);
          if (limitCpu > 0) {
            const ratio = usageCpu / limitCpu;
            if (ratio >= HOT_THRESHOLD) {
              cpuCount++;
              cpuMaxRatio = Math.max(cpuMaxRatio, ratio);
            }
          }
        }

        if (limits.memory) {
          const limitMem = parseMemory(limits.memory);
          const usageMem = parseMemory(metricsContainer.usage.memory);
          if (limitMem > 0) {
            const ratio = usageMem / limitMem;
            if (ratio >= HOT_THRESHOLD) {
              memCount++;
              memMaxRatio = Math.max(memMaxRatio, ratio);
            }
          }
        }
      }
    }

    return {
      cpu: { count: cpuCount, maxRatio: cpuMaxRatio, level: cpuMaxRatio >= CRITICAL_THRESHOLD ? 'critical' : 'warm' },
      memory: { count: memCount, maxRatio: memMaxRatio, level: memMaxRatio >= CRITICAL_THRESHOLD ? 'critical' : 'warm' },
    };
  });

  function heatColor(level: HeatLevel): string {
    return level === 'critical' ? 'text-red-500' : 'text-amber-500';
  }

  function heatBg(level: HeatLevel): string {
    return level === 'critical' ? 'bg-red-500' : 'bg-amber-500';
  }
</script>

{#if podCount > 0}
<a
  href="/namespace/{name}"
  class="w-90 bg-neutral-700 rounded-lg p-4 hover:bg-neutral-600 transition-colors block shadow-lg border-l-4 {config.borderClass}"
>
  <!-- Icon + name -->
  <div class="flex items-start gap-3 mb-3">
    {#if icon}
      <AppIcon src={icon} size="w-24 h-24" />
    {:else}
      <div class="w-24 h-24 shrink-0 rounded-md bg-neutral-800 flex items-center justify-center">
        <span class="text-neutral-400 text-4xl font-bold">{name[0].toUpperCase()}</span>
      </div>
    {/if}
    <div class="flex-1 min-w-0">
      <div class="font-semibold text-neutral-100 text-lg truncate">{displayName}</div>
      {#each ingressHosts as host}
        <span role="link" tabindex="0" class="text-xs text-neutral-400 hover:text-neutral-200 inline-block mt-1 transition-colors cursor-pointer" onclick={(e: MouseEvent) => { e.preventDefault(); e.stopPropagation(); window.open(`https://${host}`, '_blank', 'noopener'); }} onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') { e.preventDefault(); e.stopPropagation(); window.open(`https://${host}`, '_blank', 'noopener'); }}}>{host} ⧉</span>
      {/each}
    </div>
  </div>

  <!-- Health status -->
  <div class="flex items-center gap-2 mb-3">
    {#if status === 'healthy'}
      <div class="w-6 h-6 shrink-0 {config.dotClass} status-icon" style="-webkit-mask-image: url(/icons/healthy.svg); mask-image: url(/icons/healthy.svg);"></div>
    {:else if status === 'warning'}
      <div class="w-6 h-6 shrink-0 {config.dotClass} status-icon animate-spin" style="-webkit-mask-image: url(/icons/working.svg); mask-image: url(/icons/working.svg);"></div>
    {:else if status === 'error'}
      <div class="w-6 h-6 shrink-0 {config.dotClass} status-icon error-flash" style="-webkit-mask-image: url(/icons/error.svg); mask-image: url(/icons/error.svg);"></div>
    {:else}
      <div class="w-6 h-6 shrink-0 rounded-full {config.dotClass}"></div>
    {/if}
    <span class="text-sm font-semibold {config.textClass}">{action || statusLabel}</span>
  </div>

  <!-- Stats -->
  <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs">
    <span><span class="text-neutral-500">Pods</span> <span class="text-neutral-400">{healthyCount}/{podCount}</span></span>
    {#if pvcCount > 0}
      <span><span class="text-neutral-500">PVCs</span> <span class="{pvcBoundCount < pvcCount ? 'text-amber-400' : 'text-neutral-400'}">{pvcBoundCount}/{pvcCount}</span></span>
    {/if}
    {#if cronJobCount > 0}
      <span><span class="text-neutral-500">CronJobs</span> <span class="text-neutral-400">{cronJobCount}</span></span>
    {/if}
    {#if failedJobCount > 0}
      <span><span class="text-neutral-500">Failed Jobs</span> <span class="text-red-400">{failedJobCount}</span></span>
    {/if}
    {#if secretCount > 0}
      <span><span class="text-neutral-500">Secrets</span> <span class="text-neutral-400">{secretCount}</span></span>
    {/if}
    {#if hotResources.cpu.count > 0}
      <span class="inline-flex items-center gap-0.5 {heatColor(hotResources.cpu.level)}">
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.cpu.level)}" style="-webkit-mask-image: url(/icons/flame.svg); mask-image: url(/icons/flame.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.cpu.level)}" style="-webkit-mask-image: url(/icons/cpu.svg); mask-image: url(/icons/cpu.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span>{hotResources.cpu.count}</span>
      </span>
    {/if}
    {#if hotResources.memory.count > 0}
      <span class="inline-flex items-center gap-0.5 {heatColor(hotResources.memory.level)}">
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.memory.level)}" style="-webkit-mask-image: url(/icons/flame.svg); mask-image: url(/icons/flame.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span class="w-3 h-3 shrink-0 {heatBg(hotResources.memory.level)}" style="-webkit-mask-image: url(/icons/memory.svg); mask-image: url(/icons/memory.svg); -webkit-mask-size: contain; mask-size: contain;"></span>
        <span>{hotResources.memory.count}</span>
      </span>
    {/if}
    {#if uptime}
      <span><span class="text-neutral-500">Uptime</span> <span class="text-neutral-400">{uptime}</span></span>
    {/if}
  </div>
</a>
{/if}
