import type { Pod, Node, NodeMetrics, PodMetrics, NodeDiskStats, Namespace, PVC, Deployment, K8sEvent, WatchEvent, MetricsSnapshot, PersistentVolume, StorageClass, IngressClass, ClusterRole, ClusterRoleBinding, Service, Ingress, ConfigMap, Secret, ServiceAccount, ReplicaSet, StatefulSet, DaemonSet, Job, CronJob, Role, RoleBinding, NetworkPolicy, EndpointSlice } from './types';

class ClusterState {
  namespaces: Record<string, Namespace> = $state({});
  pods: Record<string, Pod> = $state({});
  nodes: Record<string, Node> = $state({});
  nodeMetrics: Record<string, NodeMetrics> = $state({});
  podMetrics: Record<string, PodMetrics> = $state({});
  nodeDiskStats: Record<string, NodeDiskStats> = $state({});
  pvcs: Record<string, PVC> = $state({});
  pvs: Record<string, PersistentVolume> = $state({});
  deployments: Record<string, Deployment> = $state({});
  replicaSets: Record<string, ReplicaSet> = $state({});
  statefulSets: Record<string, StatefulSet> = $state({});
  daemonSets: Record<string, DaemonSet> = $state({});
  jobs: Record<string, Job> = $state({});
  cronJobs: Record<string, CronJob> = $state({});
  services: Record<string, Service> = $state({});
  ingresses: Record<string, Ingress> = $state({});
  ingressClasses: Record<string, IngressClass> = $state({});
  configMaps: Record<string, ConfigMap> = $state({});
  secrets: Record<string, Secret> = $state({});
  serviceAccounts: Record<string, ServiceAccount> = $state({});
  endpoints: Record<string, EndpointSlice> = $state({});
  storageClasses: Record<string, StorageClass> = $state({});
  clusterRoles: Record<string, ClusterRole> = $state({});
  clusterRoleBindings: Record<string, ClusterRoleBinding> = $state({});
  roles: Record<string, Role> = $state({});
  roleBindings: Record<string, RoleBinding> = $state({});
  networkPolicies: Record<string, NetworkPolicy> = $state({});
  events: Record<string, K8sEvent> = $state({});
  metricsHistory: MetricsSnapshot[] = $state([]);
  connected = $state(false);
  reconnectStatus = $state('');
  authError = $state('');

  purge() {
    this.namespaces = {};
    this.pods = {};
    this.nodes = {};
    this.nodeMetrics = {};
    this.podMetrics = {};
    this.nodeDiskStats = {};
    this.pvcs = {};
    this.pvs = {};
    this.deployments = {};
    this.replicaSets = {};
    this.statefulSets = {};
    this.daemonSets = {};
    this.jobs = {};
    this.cronJobs = {};
    this.services = {};
    this.ingresses = {};
    this.ingressClasses = {};
    this.configMaps = {};
    this.secrets = {};
    this.serviceAccounts = {};
    this.endpoints = {};
    this.storageClasses = {};
    this.clusterRoles = {};
    this.clusterRoleBindings = {};
    this.roles = {};
    this.roleBindings = {};
    this.networkPolicies = {};
    this.events = {};
    this.metricsHistory = [];
  }

  handleEvent(event: WatchEvent) {
    const obj = event.object;
    switch (obj.kind) {
      case 'MetricsSnapshot': {
        const snap = obj as unknown as MetricsSnapshot;
        const newNodeMetrics: Record<string, NodeMetrics> = {};
        for (const nm of snap.nodes) {
          newNodeMetrics[nm.metadata.name] = nm;
        }
        this.nodeMetrics = newNodeMetrics;
        const newPodMetrics: Record<string, PodMetrics> = {};
        for (const pm of snap.pods) {
          newPodMetrics[`${pm.metadata.namespace}/${pm.metadata.name}`] = pm;
        }
        this.podMetrics = newPodMetrics;
        const newDiskStats: Record<string, NodeDiskStats> = {};
        for (const ds of snap.diskStats) {
          newDiskStats[ds.metadata.name] = ds;
        }
        this.nodeDiskStats = newDiskStats;
        const existing = this.metricsHistory.find(h => h.timestamp === snap.timestamp);
        if (!existing) {
          this.metricsHistory = [...this.metricsHistory, snap].slice(-30);
        }
        break;
      }
      case 'Namespace': {
        const ns = obj as unknown as Namespace;
        this.upsertOrDelete('namespaces', ns.metadata.name, ns, event.type);
        break;
      }
      case 'Pod': {
        const pod = obj as unknown as Pod;
        this.upsertOrDelete('pods', `${pod.metadata.namespace}/${pod.metadata.name}`, pod, event.type);
        break;
      }
      case 'Node': {
        const node = obj as unknown as Node;
        this.upsertOrDelete('nodes', node.metadata.name, node, event.type);
        break;
      }
      case 'PersistentVolumeClaim': {
        const pvc = obj as unknown as PVC;
        this.upsertOrDelete('pvcs', `${pvc.metadata.namespace}/${pvc.metadata.name}`, pvc, event.type);
        break;
      }
      case 'Deployment': {
        const dep = obj as unknown as Deployment;
        this.upsertOrDelete('deployments', `${dep.metadata.namespace}/${dep.metadata.name}`, dep, event.type);
        break;
      }
      case 'PersistentVolume': {
        const pv = obj as unknown as PersistentVolume;
        this.upsertOrDelete('pvs', pv.metadata.name, pv, event.type);
        break;
      }
      case 'ReplicaSet': {
        const rs = obj as unknown as ReplicaSet;
        this.upsertOrDelete('replicaSets', `${rs.metadata.namespace}/${rs.metadata.name}`, rs, event.type);
        break;
      }
      case 'StatefulSet': {
        const ss = obj as unknown as StatefulSet;
        this.upsertOrDelete('statefulSets', `${ss.metadata.namespace}/${ss.metadata.name}`, ss, event.type);
        break;
      }
      case 'DaemonSet': {
        const ds = obj as unknown as DaemonSet;
        this.upsertOrDelete('daemonSets', `${ds.metadata.namespace}/${ds.metadata.name}`, ds, event.type);
        break;
      }
      case 'Job': {
        const job = obj as unknown as Job;
        this.upsertOrDelete('jobs', `${job.metadata.namespace}/${job.metadata.name}`, job, event.type);
        break;
      }
      case 'CronJob': {
        const cj = obj as unknown as CronJob;
        this.upsertOrDelete('cronJobs', `${cj.metadata.namespace}/${cj.metadata.name}`, cj, event.type);
        break;
      }
      case 'Service': {
        const svc = obj as unknown as Service;
        this.upsertOrDelete('services', `${svc.metadata.namespace}/${svc.metadata.name}`, svc, event.type);
        break;
      }
      case 'Ingress': {
        const ing = obj as unknown as Ingress;
        this.upsertOrDelete('ingresses', `${ing.metadata.namespace}/${ing.metadata.name}`, ing, event.type);
        break;
      }
      case 'IngressClass': {
        const ic = obj as unknown as IngressClass;
        this.upsertOrDelete('ingressClasses', ic.metadata.name, ic, event.type);
        break;
      }
      case 'ConfigMap': {
        const cm = obj as unknown as ConfigMap;
        this.upsertOrDelete('configMaps', `${cm.metadata.namespace}/${cm.metadata.name}`, cm, event.type);
        break;
      }
      case 'Secret': {
        const sec = obj as unknown as Secret;
        this.upsertOrDelete('secrets', `${sec.metadata.namespace}/${sec.metadata.name}`, sec, event.type);
        break;
      }
      case 'ServiceAccount': {
        const sa = obj as unknown as ServiceAccount;
        this.upsertOrDelete('serviceAccounts', `${sa.metadata.namespace}/${sa.metadata.name}`, sa, event.type);
        break;
      }
      case 'EndpointSlice': {
        const ep = obj as unknown as EndpointSlice;
        this.upsertOrDelete('endpoints', `${ep.metadata.namespace}/${ep.metadata.name}`, ep, event.type);
        break;
      }
      case 'StorageClass': {
        const sc = obj as unknown as StorageClass;
        this.upsertOrDelete('storageClasses', sc.metadata.name, sc, event.type);
        break;
      }
      case 'ClusterRole': {
        const cr = obj as unknown as ClusterRole;
        this.upsertOrDelete('clusterRoles', cr.metadata.name, cr, event.type);
        break;
      }
      case 'ClusterRoleBinding': {
        const crb = obj as unknown as ClusterRoleBinding;
        this.upsertOrDelete('clusterRoleBindings', crb.metadata.name, crb, event.type);
        break;
      }
      case 'Role': {
        const role = obj as unknown as Role;
        this.upsertOrDelete('roles', `${role.metadata.namespace}/${role.metadata.name}`, role, event.type);
        break;
      }
      case 'RoleBinding': {
        const rb = obj as unknown as RoleBinding;
        this.upsertOrDelete('roleBindings', `${rb.metadata.namespace}/${rb.metadata.name}`, rb, event.type);
        break;
      }
      case 'NetworkPolicy': {
        const np = obj as unknown as NetworkPolicy;
        this.upsertOrDelete('networkPolicies', `${np.metadata.namespace}/${np.metadata.name}`, np, event.type);
        break;
      }
      case 'Event': {
        const evt = obj as unknown as K8sEvent;
        this.upsertOrDelete('events', `${evt.metadata.namespace}/${evt.metadata.name}`, evt, event.type);
        break;
      }
    }
  }

  private upsertOrDelete<K extends 'namespaces' | 'pods' | 'nodes' | 'pvcs' | 'pvs' | 'deployments' | 'replicaSets' | 'statefulSets' | 'daemonSets' | 'jobs' | 'cronJobs' | 'services' | 'ingresses' | 'ingressClasses' | 'configMaps' | 'secrets' | 'serviceAccounts' | 'endpoints' | 'storageClasses' | 'clusterRoles' | 'clusterRoleBindings' | 'roles' | 'roleBindings' | 'networkPolicies' | 'events'>(
    field: K, key: string, value: ClusterState[K][string], type: string
  ) {
    if (type === 'DELETED') {
      delete this[field][key];
    } else {
      this[field][key] = value;
    }
  }
}

export const cluster = new ClusterState();
