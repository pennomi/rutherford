export function parseCPU(s: string): number {
  if (s.endsWith('n')) return parseInt(s) / 1_000_000;
  if (s.endsWith('m')) return parseInt(s);
  return parseFloat(s) * 1000;
}

export function parseMemory(s: string): number {
  if (s.endsWith('Ki')) return parseInt(s) * 1024;
  if (s.endsWith('Mi')) return parseInt(s) * 1024 * 1024;
  if (s.endsWith('Gi')) return parseInt(s) * 1024 * 1024 * 1024;
  return parseInt(s);
}

export function formatCPU(millis: number): string {
  if (millis >= 1000) return `${(millis / 1000).toFixed(1)} cores`;
  return `${Math.round(millis)}m`;
}

export function formatMemory(bytes: number): string {
  const gib = bytes / (1024 * 1024 * 1024);
  if (gib >= 1.0) return `${gib.toFixed(1)} GiB`;
  const mib = bytes / (1024 * 1024);
  return `${Math.round(mib)} MiB`;
}

export function formatDisk(bytes: number): string {
  const gib = bytes / (1024 * 1024 * 1024);
  if (gib >= 100) return `${Math.round(gib)} GiB`;
  if (gib >= 1.0) return `${gib.toFixed(1)} GiB`;
  const mib = bytes / (1024 * 1024);
  return `${Math.round(mib)} MiB`;
}

export type ResourceUnit = 'cpu' | 'memory' | 'disk' | 'none';

export function formatPair(used: number, capacity: number, unit: ResourceUnit): string {
  if (unit === 'cpu') {
    if (capacity >= 1000) {
      return `${(used / 1000).toFixed(1)} / ${(capacity / 1000).toFixed(1)} cores`;
    }
    return `${Math.round(used)} / ${Math.round(capacity)}m`;
  }
  if (unit === 'memory' || unit === 'disk') {
    const capGib = capacity / (1024 * 1024 * 1024);
    if (capGib >= 1.0) {
      const usedGib = used / (1024 * 1024 * 1024);
      return `${usedGib.toFixed(1)} / ${capGib.toFixed(1)} GiB`;
    }
    const capMib = capacity / (1024 * 1024);
    const usedMib = used / (1024 * 1024);
    return `${Math.round(usedMib)} / ${Math.round(capMib)} MiB`;
  }
  return 'N/A';
}

export function pct(used: number, total: number): number {
  if (total === 0) return 0;
  return Math.round((used * 100) / total);
}

export function barColor(used: number, total: number): string {
  if (total === 0) return 'bg-green-500';
  const p = (used * 100) / total;
  if (p > 90) return 'bg-red-500';
  if (p > 70) return 'bg-yellow-500';
  return 'bg-green-500';
}

export function formatAge(timestamp: string): string {
  const ms = Date.now() - new Date(timestamp).getTime();
  const days = Math.floor(ms / 86_400_000);
  if (days > 0) return `${days}d`;
  const hours = Math.floor(ms / 3_600_000);
  if (hours > 0) return `${hours}h`;
  return `${Math.floor(ms / 60_000)}m`;
}
