import type { Pod } from './types';

export const failedStates = new Set([
  'CrashLoopBackOff', 'ImagePullBackOff', 'ErrImagePull', 'Error',
  'OOMKilled', 'CreateContainerError', 'InvalidImageName',
  'CreateContainerConfigError', 'RunContainerError',
]);

const actionLabels: Record<string, string> = {
  ContainerCreating: 'Creating container',
  PodInitializing: 'Initializing',
  CrashLoopBackOff: 'Crash looping',
  ImagePullBackOff: 'Image pull failing',
  ErrImagePull: 'Image pull failed',
  CreateContainerConfigError: 'Bad container config',
  CreateContainerError: 'Container creation failed',
  RunContainerError: 'Container failed to start',
  InvalidImageName: 'Invalid image',
};

export function containerAction(containerName: string, pod: Pod): string {
  for (const cs of pod.status?.initContainerStatuses ?? []) {
    if (cs.state?.waiting) return `Init: ${actionLabels[cs.state.waiting.reason] ?? cs.state.waiting.reason}`;
  }

  const cs = pod.status?.containerStatuses?.find(s => s.name === containerName);
  if (!cs) return '';
  if (cs.ready) return '';
  if (cs.state?.waiting) return actionLabels[cs.state.waiting.reason] ?? cs.state.waiting.reason;
  if (cs.state?.running && !cs.ready) return 'Waiting for health check';
  if (cs.state?.terminated) return `Terminated: ${cs.state.terminated.reason}`;
  return '';
}

export function podAction(pod: Pod): string {
  const initStatuses = pod.status?.initContainerStatuses ?? [];
  const statuses = pod.status?.containerStatuses ?? [];

  if (pod.status?.phase === 'Pending' && statuses.length === 0 && initStatuses.length === 0) {
    return 'Scheduling';
  }

  for (const cs of initStatuses) {
    if (cs.state?.waiting) return `Init: ${actionLabels[cs.state.waiting.reason] ?? cs.state.waiting.reason}`;
    if (cs.state?.running) return 'Running init container';
  }

  for (const cs of statuses) {
    if (cs.state?.waiting) return actionLabels[cs.state.waiting.reason] ?? cs.state.waiting.reason;
    if (cs.state?.running && !cs.ready) return 'Waiting for health check';
  }

  return '';
}

export function podStatus(pod: Pod): string {
  const initStatuses = pod.status?.initContainerStatuses ?? [];
  const statuses = pod.status?.containerStatuses ?? [];

  for (const cs of initStatuses) {
    if (cs.state?.waiting) return `Init:${cs.state.waiting.reason}`;
  }
  for (const cs of statuses) {
    if (cs.state?.waiting) return cs.state.waiting.reason;
    if (cs.state?.terminated && pod.status.phase !== 'Succeeded') {
      return cs.state.terminated.reason;
    }
  }

  if (pod.status?.phase === 'Running') {
    for (const cs of statuses) {
      if (!cs.ready) return 'NotReady';
    }
  }

  return pod.status?.phase ?? 'Unknown';
}
