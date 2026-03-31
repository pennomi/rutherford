export interface KubeMeta {
  name: string;
  namespace: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  creationTimestamp: string;
}

export interface ContainerStatus {
  name: string;
  ready: boolean;
  restartCount: number;
  state: {
    running: { startedAt: string } | null;
    waiting: { reason: string } | null;
    terminated: { reason: string } | null;
  };
}

export interface Pod {
  kind: 'Pod';
  metadata: KubeMeta;
  spec: {
    containers: { name: string; image: string; resources: { limits: { cpu: string; memory: string }; requests: { cpu: string; memory: string } } }[];
    initContainers: { name: string; image: string }[];
    nodeName: string;
  };
  status: {
    phase: string;
    containerStatuses: ContainerStatus[];
    initContainerStatuses: ContainerStatus[];
  };
}

export interface NodeInfo {
  architecture: string;
  containerRuntimeVersion: string;
  kernelVersion: string;
  kubeletVersion: string;
  operatingSystem: string;
  osImage: string;
}

export interface Node {
  kind: 'Node';
  metadata: KubeMeta;
  status: {
    allocatable: {
      cpu: string;
      memory: string;
    };
    nodeInfo: NodeInfo;
  };
}

export interface NodeMetrics {
  kind: 'NodeMetrics';
  metadata: { name: string };
  usage: {
    cpu: string;
    memory: string;
  };
}

export interface Namespace {
  kind: 'Namespace';
  metadata: KubeMeta;
}

export interface PVC {
  kind: 'PersistentVolumeClaim';
  metadata: KubeMeta;
  spec: {
    accessModes: string[];
    storageClassName: string;
    resources: {
      requests: {
        storage: string;
      };
    };
  };
  status: {
    phase: string;
    capacity: {
      storage: string;
    };
  };
}

export interface Deployment {
  kind: 'Deployment';
  metadata: KubeMeta;
  spec: {
    replicas: number;
  };
  status: {
    readyReplicas: number;
    availableReplicas: number;
    replicas: number;
  };
}

export interface K8sEvent {
  kind: 'Event';
  metadata: KubeMeta;
  involvedObject: {
    kind: string;
    name: string;
    namespace: string;
  };
  reason: string;
  message: string;
  type: string;
  lastTimestamp: string;
  firstTimestamp: string;
}

export interface PodMetrics {
  kind: 'PodMetrics';
  metadata: KubeMeta;
  containers: {
    name: string;
    usage: {
      cpu: string;
      memory: string;
    };
  }[];
}

export interface DiskUsage {
  capacityBytes: number;
  usedBytes: number;
}

export interface VolumeUsage {
  podName: string;
  podNamespace: string;
  volumeName: string;
  pvcName: string;
  capacityBytes: number;
  usedBytes: number;
}

export interface NodeDiskStats {
  kind: 'NodeDiskStats';
  metadata: { name: string };
  nodeFs: DiskUsage;
  imageFs: DiskUsage;
  volumes: VolumeUsage[];
}

export interface PersistentVolume {
  kind: 'PersistentVolume';
  metadata: KubeMeta;
  spec: {
    capacity: { storage: string };
    storageClassName: string;
    claimRef: {
      name: string;
      namespace: string;
    };
  };
  status: {
    phase: string;
  };
}

export interface StorageClass {
  kind: 'StorageClass';
  metadata: KubeMeta;
  provisioner: string;
  reclaimPolicy: string;
}

export interface IngressClass {
  kind: 'IngressClass';
  metadata: KubeMeta;
  spec: {
    controller: string;
  };
}

export interface ClusterRole {
  kind: 'ClusterRole';
  metadata: KubeMeta;
}

export interface ClusterRoleBinding {
  kind: 'ClusterRoleBinding';
  metadata: KubeMeta;
  roleRef: {
    kind: string;
    name: string;
  };
}

export interface Service {
  kind: 'Service';
  metadata: KubeMeta;
  spec: {
    type: string;
    clusterIP: string;
    ports: { port: number; targetPort: number; protocol: string; name: string }[];
  };
}

export interface Ingress {
  kind: 'Ingress';
  metadata: KubeMeta;
  spec: {
    rules: { host: string }[];
    tls: { hosts: string[]; secretName: string }[];
  };
}

export interface ConfigMap {
  kind: 'ConfigMap';
  metadata: KubeMeta;
}

export interface Secret {
  kind: 'Secret';
  metadata: KubeMeta;
}

export interface ServiceAccount {
  kind: 'ServiceAccount';
  metadata: KubeMeta;
}

export interface ReplicaSet {
  kind: 'ReplicaSet';
  metadata: KubeMeta;
  spec: { replicas: number };
  status: { readyReplicas: number; replicas: number };
}

export interface StatefulSet {
  kind: 'StatefulSet';
  metadata: KubeMeta;
  spec: { replicas: number };
  status: { readyReplicas: number; replicas: number };
}

export interface DaemonSet {
  kind: 'DaemonSet';
  metadata: KubeMeta;
  status: { desiredNumberScheduled: number; numberReady: number };
}

export interface Job {
  kind: 'Job';
  metadata: KubeMeta;
  status: {
    succeeded: number;
    failed: number;
    active: number;
    completionTime: string;
  };
}

export interface CronJob {
  kind: 'CronJob';
  metadata: KubeMeta;
  spec: { schedule: string };
  status: { lastScheduleTime: string };
}

export interface Role {
  kind: 'Role';
  metadata: KubeMeta;
}

export interface RoleBinding {
  kind: 'RoleBinding';
  metadata: KubeMeta;
  roleRef: {
    kind: string;
    name: string;
  };
}

export interface NetworkPolicy {
  kind: 'NetworkPolicy';
  metadata: KubeMeta;
}

export interface EndpointSlice {
  kind: 'EndpointSlice';
  metadata: KubeMeta;
  endpoints: {
    addresses: string[];
    conditions: { ready: boolean };
  }[];
  ports: { port: number; protocol: string; name: string }[];
}

export interface WatchEvent {
  type: 'ADDED' | 'MODIFIED' | 'DELETED';
  object: { kind: string } & Record<string, unknown>;
}

export interface MetricsSnapshot {
  kind: 'MetricsSnapshot';
  timestamp: string;
  nodes: NodeMetrics[];
  pods: PodMetrics[];
  diskStats: NodeDiskStats[];
}
