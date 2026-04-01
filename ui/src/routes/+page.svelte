<script lang="ts">
  import { cluster } from '$lib/state.svelte';
  import { parseCPU, parseMemory, formatCPU, formatMemory, formatDisk, pct } from '$lib/format';
  import { getUserManager } from '$lib/auth';
  import type { NodeDiskStats } from '$lib/types';
  import { formatAge } from '$lib/format';
  import { Chart, DoughnutController, ArcElement, Tooltip, Legend } from 'chart.js';
  import ResourceCard from '$lib/ResourceCard.svelte';
  import NamespaceTile from '$lib/NamespaceTile.svelte';

  Chart.register(DoughnutController, ArcElement, Tooltip, Legend);


  const twColorNames = [
    '--color-blue-500', '--color-violet-500', '--color-amber-500', '--color-emerald-500', '--color-red-500',
    '--color-cyan-500', '--color-pink-500', '--color-lime-500', '--color-orange-500', '--color-indigo-500',
    '--color-teal-500', '--color-rose-600', '--color-lime-400', '--color-sky-500', '--color-fuchsia-500',
  ];

  function getTwColors(): string[] {
    const style = getComputedStyle(document.documentElement);
    return twColorNames.map(name => style.getPropertyValue(name).trim());
  }

  function getTwColor(name: string): string {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  }


  // --- Resource overview ---

  let totalCPUCap = $derived(Object.values(cluster.nodes).reduce((s, n) => s + parseCPU(n.status.allocatable.cpu), 0));
  let totalCPUUsage = $derived(Object.values(cluster.nodeMetrics).reduce((s, m) => s + parseCPU(m.usage.cpu), 0));
  let totalMemCap = $derived(Object.values(cluster.nodes).reduce((s, n) => s + parseMemory(n.status.allocatable.memory), 0));
  let totalMemUsage = $derived(Object.values(cluster.nodeMetrics).reduce((s, m) => s + parseMemory(m.usage.memory), 0));

  let totalDiskCap = $derived(Object.values(cluster.nodeDiskStats).reduce((s, d: NodeDiskStats) => s + d.nodeFs.capacityBytes, 0));
  let totalDiskUsed = $derived(Object.values(cluster.nodeDiskStats).reduce((s, d: NodeDiskStats) => s + d.nodeFs.usedBytes, 0));

  let hasNodes = $derived(Object.keys(cluster.nodes).length > 0);
  let hasMetrics = $derived(Object.keys(cluster.nodeMetrics).length > 0);
  let hasDiskStats = $derived(Object.keys(cluster.nodeDiskStats).length > 0);

  let nodeArch = $derived(() => {
    const nodes = Object.values(cluster.nodes);
    if (nodes.length === 0) return '';
    const archs = new Set(nodes.map(n => n.status.nodeInfo.architecture));
    return [...archs].join(', ');
  });

  let cpuHistory = $derived(cluster.metricsHistory.map(snap => {
    const cap = Object.values(cluster.nodes).reduce((s, n) => s + parseCPU(n.status.allocatable.cpu), 0);
    const used = snap.nodes.reduce((s, m) => s + parseCPU(m.usage.cpu), 0);
    return pct(used, cap);
  }));

  let memHistory = $derived(cluster.metricsHistory.map(snap => {
    const cap = Object.values(cluster.nodes).reduce((s, n) => s + parseMemory(n.status.allocatable.memory), 0);
    const used = snap.nodes.reduce((s, m) => s + parseMemory(m.usage.memory), 0);
    return pct(used, cap);
  }));

  // --- Dialog state ---

  interface Segment {
    label: string;
    value: number;
    formatted: string;
  }

  let dialogEl = $state<HTMLDialogElement>() as HTMLDialogElement;
  let canvasEl = $state<HTMLCanvasElement>() as HTMLCanvasElement;
  let chartInstance: Chart | null = null;
  let dialogTitle = $state('');
  let dialogSegments: Segment[] = $state([]);
  let dialogTotal = $state(0);
  let dialogTotalLabel = $state('');
  let dialogError = $state('');
  let dialogLoading = $state(false);

  function openDialog(title: string, segments: Segment[], total: number, totalLabel: string) {
    dialogTitle = title;
    dialogSegments = segments.toSorted((a, b) => b.value - a.value);
    dialogTotal = total;
    dialogTotalLabel = totalLabel;
    dialogEl.showModal();

    if (chartInstance) {
      chartInstance.destroy();
      chartInstance = null;
    }

    const palette = getTwColors();
    const usedTotal = dialogSegments.reduce((s, seg) => s + seg.value, 0);
    const free = Math.max(0, total - usedTotal);

    const labels = [...dialogSegments.map(s => s.label)];
    const data = [...dialogSegments.map(s => s.value)];
    const colors = [...dialogSegments.map((_, i) => palette[i % palette.length])];

    if (free > 0) {
      labels.push('Free');
      data.push(free);
      colors.push(getTwColor('--color-neutral-800'));
    }

    requestAnimationFrame(() => {
      chartInstance = new Chart(canvasEl, {
        type: 'doughnut',
        data: {
          labels,
          datasets: [{
            data,
            backgroundColor: colors,
            borderWidth: 0
          }]
        },
        options: {
          responsive: true,
          cutout: '60%',
          plugins: {
            legend: { display: false },
            tooltip: {
              callbacks: {
                label: (ctx) => {
                  const seg = dialogSegments[ctx.dataIndex];
                  if (seg) return ` ${seg.formatted} (${pct(seg.value, total)}%)`;
                  return ' Free';
                }
              }
            }
          }
        }
      });
    });
  }

  // --- Pod resources helper ---

  interface PodResource {
    name: string;
    namespace: string;
    cpu: number;
    memory: number;
  }

  function allPodResources(): PodResource[] {
    const result: PodResource[] = [];
    for (const pod of Object.values(cluster.pods)) {
      if (pod.status?.phase === 'Succeeded' || pod.status?.phase === 'Failed') continue;
      const key = `${pod.metadata.namespace}/${pod.metadata.name}`;
      const pm = cluster.podMetrics[key];
      if (!pm) continue;
      let cpu = 0;
      let memory = 0;
      for (const c of pm.containers) {
        cpu += parseCPU(c.usage.cpu);
        memory += parseMemory(c.usage.memory);
      }
      result.push({ name: pod.metadata.name, namespace: pod.metadata.namespace, cpu, memory });
    }
    return result;
  }

  function openCPU() {
    const byNs: Record<string, number> = {};
    for (const pr of allPodResources()) {
      byNs[pr.namespace] = (byNs[pr.namespace] ?? 0) + pr.cpu;
    }
    const segments = Object.entries(byNs).map(([ns, cpu]) => ({
      label: ns,
      value: cpu,
      formatted: formatCPU(cpu)
    })).filter(s => s.value > 0);
    openDialog('CPU Breakdown', segments, totalCPUCap, formatCPU(totalCPUCap));
  }

  function openMem() {
    const byNs: Record<string, number> = {};
    for (const pr of allPodResources()) {
      byNs[pr.namespace] = (byNs[pr.namespace] ?? 0) + pr.memory;
    }
    const segments = Object.entries(byNs).map(([ns, mem]) => ({
      label: ns,
      value: mem,
      formatted: formatMemory(mem)
    })).filter(s => s.value > 0);
    openDialog('Memory Breakdown', segments, totalMemCap, formatMemory(totalMemCap));
  }

  function openDisk() {
    const diskStats = Object.values(cluster.nodeDiskStats)[0];
    if (!diskStats) return;
    dialogError = '';
    dialogLoading = true;
    openDialog('Disk Breakdown', [], totalDiskCap, formatDisk(totalDiskCap));
    dialogLoading = false;
  }

  function noop() {}


</script>

<div class="bg-grid text-zinc-300 p-6 font-sans min-h-screen">
  <!-- Cluster Metrics -->
  <div class="mb-8">
    <h2 class="text-xs text-zinc-500 uppercase tracking-wider mb-3">Cluster Metrics</h2>
    <div class="grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-4">
      <ResourceCard
        title="CPU"
        icon="/icons/cpu.svg"
        used={totalCPUUsage}
        capacity={totalCPUCap}
        unit="cpu"
        color="--color-emerald-500"
        chartType="line"
        history={cpuHistory.length > 1 ? cpuHistory : [0, 0]}
        onclick={openCPU}
        loading={!hasNodes || !hasMetrics}
        unavailable={false}
        detail={nodeArch()}
      />
      <ResourceCard
        title="Memory"
        icon="/icons/memory.svg"
        used={totalMemUsage}
        capacity={totalMemCap}
        unit="memory"
        color="--color-violet-500"
        chartType="line"
        history={memHistory.length > 1 ? memHistory : [0, 0]}
        onclick={openMem}
        loading={!hasNodes || !hasMetrics}
        unavailable={false}
        detail=""
      />
      <ResourceCard
        title="GPU"
        icon="/icons/gpu.svg"
        used={0}
        capacity={0}
        unit="none"
        color="--color-zinc-600"
        chartType="line"
        history={[0, 0]}
        onclick={noop}
        loading={!hasNodes || !hasMetrics}
        unavailable={hasNodes && hasMetrics}
        detail=""
      />
      <ResourceCard
        title="Disk"
        icon="/icons/disk.svg"
        used={totalDiskUsed}
        capacity={totalDiskCap}
        unit="disk"
        color="--color-amber-500"
        chartType="bar"
        history={[]}
        onclick={openDisk}
        loading={!hasDiskStats}
        unavailable={false}
        detail=""
      />
    </div>
  </div>

  <!-- Namespace Status -->
  <div>
    <h2 class="text-xs text-zinc-500 uppercase tracking-wider mb-3">Namespaces</h2>
    <div class="flex flex-wrap gap-4">
      {#each Object.keys(cluster.namespaces).sort() as ns (ns)}
        <NamespaceTile name={ns} />
      {/each}
    </div>
  </div>

  <!-- Cluster Resources -->
  <div class="mt-8">
    <h2 class="text-xs text-zinc-500 uppercase tracking-wider mb-3">Cluster Resources</h2>
    <div class="grid grid-cols-[repeat(auto-fill,minmax(400px,1fr))] gap-4">

      <!-- Nodes -->
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Nodes</h3>
        <div class="space-y-2">
          {#each Object.values(cluster.nodes) as node (node.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300">{node.metadata.name}</span>
              <div class="flex gap-4 text-xs text-zinc-500">
                <span>{formatCPU(parseCPU(node.status.allocatable.cpu))} CPU</span>
                <span>{formatMemory(parseMemory(node.status.allocatable.memory))} RAM</span>
                <span>{formatAge(node.metadata.creationTimestamp)}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- Persistent Volumes -->
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Persistent Volumes</h3>
        <div class="space-y-2">
          {#each Object.values(cluster.pvs).toSorted((a, b) => a.metadata.name.localeCompare(b.metadata.name)) as pv (pv.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{pv.metadata.name}</span>
              <div class="flex items-center gap-3 shrink-0 text-xs text-zinc-500">
                <span>{pv.spec.capacity.storage}</span>
                <span class="{pv.status.phase === 'Bound' ? 'text-emerald-400' : 'text-amber-400'}">{pv.status.phase}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>

    </div>
  </div>
</div>

<!-- Resource breakdown dialog -->
<dialog bind:this={dialogEl} class="bg-neutral-700 border border-zinc-600 rounded-xl p-0 max-w-lg w-full backdrop:bg-black/60 text-zinc-300">
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h3 class="text-lg font-semibold text-zinc-100">{dialogTitle}</h3>
      <button onclick={() => dialogEl.close()} class="text-zinc-500 hover:text-zinc-300 text-xl cursor-pointer leading-none">&times;</button>
    </div>

    {#if dialogLoading}
      <div class="flex flex-col items-center justify-center py-12">
        <div class="w-8 h-8 border-2 border-zinc-700 border-t-zinc-400 rounded-full animate-spin"></div>
        <div class="text-sm text-zinc-500 mt-3">Scanning storage...</div>
      </div>
    {:else if dialogError}
      <div class="text-sm text-red-400 text-center py-8 font-mono break-all">{dialogError}</div>
    {:else if dialogSegments.length > 0}
      <div class="flex justify-center mb-6">
        <div class="w-48 h-48">
          <canvas bind:this={canvasEl}></canvas>
        </div>
      </div>

      <div class="text-center text-sm text-zinc-500 mb-4">Total capacity: {dialogTotalLabel}</div>

      <div class="space-y-2 max-h-64 overflow-y-auto">
        {#each dialogSegments as seg, i}
          {@const palette = getTwColors()}
          <div class="flex items-center gap-3">
            <span class="w-3 h-3 rounded-sm shrink-0" style="background-color: {palette[i % palette.length]}"></span>
            <span class="text-sm text-zinc-400 truncate flex-1">{seg.label}</span>
            <span class="text-sm text-zinc-500 whitespace-nowrap">{seg.formatted}</span>
            <span class="text-xs text-zinc-600 whitespace-nowrap w-10 text-right">{pct(seg.value, dialogTotal)}%</span>
          </div>
        {/each}
      </div>
    {:else}
      <div class="text-sm text-zinc-500 italic text-center py-8">No usage data available</div>
    {/if}
  </div>
</dialog>
