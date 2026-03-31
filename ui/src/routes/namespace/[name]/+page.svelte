<script lang="ts">
  import { page } from '$app/state';
  import { cluster } from '$lib/state.svelte';
  import { formatAge } from '$lib/format';
  import { podStatus, podAction, failedStates } from '$lib/pod';
  import type { Pod } from '$lib/types';

  function statusBorderColor(status: string): string {
    if (status === 'Running') return 'border-l-emerald-500';
    if (failedStates.has(status)) return 'border-l-red-500';
    return 'border-l-amber-500';
  }

  function statusBadgeColor(status: string): string {
    if (failedStates.has(status)) return 'bg-red-950 text-red-400';
    return 'bg-amber-950 text-amber-400';
  }

  interface PodInfo {
    key: string;
    name: string;
    status: string;
    action: string;
    ready: number;
    total: number;
    restarts: number;
    age: string;
  }

  let nsName = $derived(page.params.name);

  let pods = $derived.by((): PodInfo[] => {
    const result: PodInfo[] = [];

    for (const [key, pod] of Object.entries(cluster.pods)) {
      if (pod.metadata.namespace !== nsName) continue;
      if (pod.status?.phase === 'Succeeded' || pod.status?.phase === 'Failed') continue;

      const status = podStatus(pod);
      const statuses = pod.status?.containerStatuses ?? [];
      let readyCount = 0;
      let totalRestarts = 0;
      for (const cs of statuses) {
        if (cs.ready) readyCount++;
        totalRestarts += cs.restartCount;
      }

      result.push({
        key,
        name: pod.metadata.name,
        status,
        action: podAction(pod),
        ready: readyCount,
        total: pod.spec?.containers?.length ?? 0,
        restarts: totalRestarts,
        age: formatAge(pod.metadata.creationTimestamp),
      });
    }

    return result.sort((a, b) => a.name.localeCompare(b.name));
  });

  function byNs<T extends { metadata: { namespace: string; name: string } }>(records: Record<string, T>): T[] {
    return Object.values(records)
      .filter(r => r.metadata.namespace === nsName)
      .sort((a, b) => a.metadata.name.localeCompare(b.metadata.name));
  }

  let nsDeployments = $derived(byNs(cluster.deployments));
let nsStatefulSets = $derived(byNs(cluster.statefulSets));
  let nsDaemonSets = $derived(byNs(cluster.daemonSets));
  let nsJobs = $derived(byNs(cluster.jobs));
  let nsCronJobs = $derived(byNs(cluster.cronJobs));
  let nsServices = $derived(byNs(cluster.services));
  let nsIngresses = $derived(byNs(cluster.ingresses));
  let nsPVCs = $derived(byNs(cluster.pvcs));
  let nsConfigMaps = $derived(byNs(cluster.configMaps));
  let nsSecrets = $derived(byNs(cluster.secrets));
  let nsServiceAccounts = $derived(byNs(cluster.serviceAccounts));
  let nsRoles = $derived(byNs(cluster.roles));
  let nsRoleBindings = $derived(byNs(cluster.roleBindings));
  let nsNetworkPolicies = $derived(byNs(cluster.networkPolicies));
  let nsEndpoints = $derived(byNs(cluster.endpoints));
  let nsEvents = $derived(
    byNs(cluster.events).toSorted((a, b) => new Date(b.lastTimestamp).getTime() - new Date(a.lastTimestamp).getTime())
  );
</script>

<div class="text-zinc-300 p-6 font-sans">
  <!-- Pods -->
  <h2 class="text-xs text-zinc-500 uppercase tracking-wider mb-3">Pods</h2>
  {#if pods.length === 0}
    <div class="text-sm text-zinc-500 italic mb-8">No running pods in this namespace.</div>
  {:else}
    <div class="grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-4 mb-8">
      {#each pods as pod (pod.key)}
        <a href="/namespace/{nsName}/pod/{pod.name}" class="bg-neutral-700 rounded-lg p-4 border-l-4 {statusBorderColor(pod.status)} block hover:bg-neutral-600 transition-colors">
          <div class="flex items-center gap-3 mb-3">
            <span class="font-mono text-sm text-zinc-100 break-all">{pod.name}</span>
            {#if pod.status !== 'Running'}
              <span class="shrink-0 ml-auto text-xs px-2 py-0.5 rounded {statusBadgeColor(pod.status)}">{pod.action || pod.status}</span>
            {/if}
          </div>
          <div class="flex gap-4 text-sm text-zinc-500">
            <span>{pod.ready}/{pod.total} ready</span>
            <span>{pod.age}</span>
            {#if pod.restarts > 0}
              <span>{pod.restarts} restarts</span>
            {/if}
          </div>
        </a>
      {/each}
    </div>
  {/if}

  <!-- Namespace Resources -->
  <h2 class="text-xs text-zinc-500 uppercase tracking-wider mb-3">Resources</h2>
  <div class="grid grid-cols-[repeat(auto-fill,minmax(400px,1fr))] gap-4">

    <!-- Deployments -->
    {#if nsDeployments.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Deployments</h3>
        <div class="space-y-2">
          {#each nsDeployments as dep (dep.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{dep.metadata.name}</span>
              <span class="text-xs {dep.status.readyReplicas === dep.spec.replicas ? 'text-emerald-400' : 'text-amber-400'}">{dep.status.readyReplicas}/{dep.spec.replicas} ready</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- StatefulSets -->
    {#if nsStatefulSets.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Stateful Sets</h3>
        <div class="space-y-2">
          {#each nsStatefulSets as ss (ss.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{ss.metadata.name}</span>
              <span class="text-xs {ss.status.readyReplicas === ss.spec.replicas ? 'text-emerald-400' : 'text-amber-400'}">{ss.status.readyReplicas}/{ss.spec.replicas} ready</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- DaemonSets -->
    {#if nsDaemonSets.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Daemon Sets</h3>
        <div class="space-y-2">
          {#each nsDaemonSets as ds (ds.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{ds.metadata.name}</span>
              <span class="text-xs {ds.status.numberReady === ds.status.desiredNumberScheduled ? 'text-emerald-400' : 'text-amber-400'}">{ds.status.numberReady}/{ds.status.desiredNumberScheduled} ready</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Jobs -->
    {#if nsJobs.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Jobs</h3>
        <div class="space-y-2">
          {#each nsJobs as job (job.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{job.metadata.name}</span>
              <span class="text-xs text-zinc-500">{job.status.succeeded} succeeded, {job.status.active} active</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- CronJobs -->
    {#if nsCronJobs.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Cron Jobs</h3>
        <div class="space-y-2">
          {#each nsCronJobs as cj (cj.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{cj.metadata.name}</span>
              <span class="text-xs text-zinc-500 font-mono">{cj.spec.schedule}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Services -->
    {#if nsServices.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Services</h3>
        <div class="space-y-2">
          {#each nsServices as svc (svc.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{svc.metadata.name}</span>
              <div class="flex items-center gap-2 shrink-0 text-xs text-zinc-500">
                <span>{svc.spec.type}</span>
                <span>{svc.spec.clusterIP}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Ingresses -->
    {#if nsIngresses.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Ingresses</h3>
        <div class="space-y-2">
          {#each nsIngresses as ing (ing.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{ing.metadata.name}</span>
              <span class="text-xs text-zinc-500">{ing.spec.rules.map(r => r.host).join(', ')}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- PVCs -->
    {#if nsPVCs.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Persistent Volume Claims</h3>
        <div class="space-y-2">
          {#each nsPVCs as pvc (pvc.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{pvc.metadata.name}</span>
              <div class="flex items-center gap-3 shrink-0 text-xs text-zinc-500">
                <span>{pvc.status.capacity.storage}</span>
                <span class="{pvc.status.phase === 'Bound' ? 'text-emerald-400' : 'text-amber-400'}">{pvc.status.phase}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- ConfigMaps -->
    {#if nsConfigMaps.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Config Maps</h3>
        <div class="space-y-2">
          {#each nsConfigMaps as cm (cm.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate">{cm.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{formatAge(cm.metadata.creationTimestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Secrets -->
    {#if nsSecrets.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Secrets</h3>
        <div class="space-y-2">
          {#each nsSecrets as sec (sec.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate">{sec.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{formatAge(sec.metadata.creationTimestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Service Accounts -->
    {#if nsServiceAccounts.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Service Accounts</h3>
        <div class="space-y-2">
          {#each nsServiceAccounts as sa (sa.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate">{sa.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{formatAge(sa.metadata.creationTimestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Roles -->
    {#if nsRoles.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Roles</h3>
        <div class="space-y-2">
          {#each nsRoles as role (role.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate">{role.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{formatAge(role.metadata.creationTimestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Role Bindings -->
    {#if nsRoleBindings.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Role Bindings</h3>
        <div class="space-y-2">
          {#each nsRoleBindings as rb (rb.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{rb.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{rb.roleRef.kind}/{rb.roleRef.name}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Network Policies -->
    {#if nsNetworkPolicies.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Network Policies</h3>
        <div class="space-y-2">
          {#each nsNetworkPolicies as np (np.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate">{np.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{formatAge(np.metadata.creationTimestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Endpoint Slices -->
    {#if nsEndpoints.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Endpoint Slices</h3>
        <div class="space-y-2">
          {#each nsEndpoints as ep (ep.metadata.name)}
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-300 truncate mr-4">{ep.metadata.name}</span>
              <span class="text-xs text-zinc-500 shrink-0">{ep.endpoints?.reduce((n, e) => n + (e.addresses?.length ?? 0), 0) ?? 0} addresses</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Events -->
    {#if nsEvents.length > 0}
      <div class="bg-neutral-700 rounded-lg p-5">
        <h3 class="text-sm font-semibold text-zinc-100 mb-3">Events</h3>
        <div class="space-y-2">
          {#each nsEvents.slice(0, 20) as evt (evt.metadata.name)}
            <div class="text-sm">
              <div class="flex items-center gap-2">
                <span class="text-xs shrink-0 {evt.type === 'Normal' ? 'text-zinc-500' : 'text-amber-400'}">{evt.type}</span>
                <span class="text-zinc-300 truncate">{evt.involvedObject.kind}/{evt.involvedObject.name}</span>
                <span class="text-xs text-zinc-600 shrink-0 ml-auto">{formatAge(evt.lastTimestamp)}</span>
              </div>
              <div class="text-xs text-zinc-500 truncate mt-0.5">{evt.message}</div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

  </div>
</div>
