<script lang="ts">
  import { page } from '$app/state';
  import { cluster } from '$lib/state.svelte';
  import { containerAction } from '$lib/pod';
  import { parseCPU, parseMemory, formatCPU, formatMemory, formatAge } from '$lib/format';
  import LogViewer from '$lib/LogViewer.svelte';

  let nsName = $derived(page.params.name);
  let podName = $derived(page.params.pod);
  let podKey = $derived(`${nsName}/${podName}`);

  let pod = $derived(cluster.pods[podKey]);
  let metrics = $derived(cluster.podMetrics[podKey]);

  let activeLog = $state('');

  let containers = $derived.by(() => {
    if (!pod) return [];
    const statuses = pod.status?.containerStatuses ?? [];
    return (pod.spec?.containers ?? []).map(c => {
      const cs = statuses.find(s => s.name === c.name);
      const pm = metrics?.containers.find(m => m.name === c.name);
      let status = 'Unknown';
      if (cs?.state?.running) status = 'Running';
      else if (cs?.state?.waiting) status = cs.state.waiting.reason;
      else if (cs?.state?.terminated) status = cs.state.terminated.reason;

      const cpuUsage = pm ? parseCPU(pm.usage.cpu) : 0;
      const memUsage = pm ? parseMemory(pm.usage.memory) : 0;
      const cpuLimit = c.resources?.limits?.cpu ? parseCPU(c.resources.limits.cpu) : 0;
      const memLimit = c.resources?.limits?.memory ? parseMemory(c.resources.limits.memory) : 0;

      return {
        name: c.name,
        image: c.image,
        status,
        action: containerAction(c.name, pod),
        ready: cs?.ready ?? false,
        restarts: cs?.restartCount ?? 0,
        cpu: pm ? formatCPU(cpuUsage) : '-',
        cpuLimit: cpuLimit ? formatCPU(cpuLimit) : '-',
        cpuPct: cpuLimit ? Math.round((cpuUsage / cpuLimit) * 100) : 0,
        memory: pm ? formatMemory(memUsage) : '-',
        memLimit: memLimit ? formatMemory(memLimit) : '-',
        memPct: memLimit ? Math.round((memUsage / memLimit) * 100) : 0,
      };
    });
  });

  function toggleLog(containerName: string) {
    activeLog = activeLog === containerName ? '' : containerName;
  }
</script>

<div class="text-zinc-300 p-6 font-sans">
  <div class="flex items-center gap-4 mb-6">
    <a href="/namespace/{nsName}" class="text-zinc-500 hover:text-zinc-300 text-sm">&larr; {nsName}</a>
    <h1 class="text-xl font-bold text-zinc-100">{podName}</h1>
  </div>

  {#if !pod}
    <div class="text-sm text-zinc-500 italic">Pod not found.</div>
  {:else}
    <div class="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-4 mb-6">
      <div class="bg-neutral-700 rounded-lg p-4">
        <div class="text-xs text-zinc-500 mb-1">Status</div>
        <div class="text-sm text-zinc-100">{pod.status?.phase ?? 'Unknown'}</div>
      </div>
      <div class="bg-neutral-700 rounded-lg p-4">
        <div class="text-xs text-zinc-500 mb-1">Node</div>
        <div class="text-sm text-zinc-100">{pod.spec?.nodeName ?? '-'}</div>
      </div>
      <div class="bg-neutral-700 rounded-lg p-4">
        <div class="text-xs text-zinc-500 mb-1">Age</div>
        <div class="text-sm text-zinc-100">{formatAge(pod.metadata.creationTimestamp)}</div>
      </div>
    </div>

    <h2 class="text-xs text-zinc-500 uppercase tracking-wider mb-3">Containers</h2>
    <div class="space-y-3">
      {#each containers as c}
        <div class="bg-neutral-700 rounded-lg p-4">
          <div class="flex items-center justify-between mb-2">
            <span class="font-mono text-sm text-zinc-100">{c.name}</span>
            <div class="flex items-center gap-2">
              <button
                onclick={() => toggleLog(c.name)}
                class="text-xs px-2 py-0.5 rounded cursor-pointer {activeLog === c.name ? 'bg-zinc-600 text-zinc-200' : 'bg-neutral-800 text-zinc-400 hover:text-zinc-200'}"
              >
                Logs
              </button>
              <span class="text-xs px-2 py-0.5 rounded {c.status === 'Running' ? 'bg-emerald-950 text-emerald-400' : 'bg-red-950 text-red-400'}">{c.action || c.status}</span>
            </div>
          </div>
          <div class="text-xs text-zinc-500 mb-2 break-all">{c.image}</div>
          <div class="flex gap-4 text-sm text-zinc-500">
            <span class={c.cpuPct >= 90 ? 'text-red-400' : c.cpuPct >= 70 ? 'text-amber-400' : ''}>CPU: {c.cpu} / {c.cpuLimit}{#if c.cpuPct > 0} ({c.cpuPct}%){/if}</span>
            <span class={c.memPct >= 90 ? 'text-red-400' : c.memPct >= 70 ? 'text-amber-400' : ''}>Mem: {c.memory} / {c.memLimit}{#if c.memPct > 0} ({c.memPct}%){/if}</span>
            {#if c.restarts > 0}
              <span>{c.restarts} restarts</span>
            {/if}
          </div>
          {#if activeLog === c.name}
            <div class="mt-3 h-96">
              {#key c.name}
                <LogViewer namespace={nsName!} pod={podName!} container={c.name} />
              {/key}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
