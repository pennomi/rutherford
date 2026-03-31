import { cluster } from './state.svelte';
import type { WatchEvent } from './types';

const MIN_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30000;

export function connectWebSocket(token: string = '') {
  let backoffMs = MIN_BACKOFF_MS;

  function connect() {
    cluster.reconnectStatus = '';
    const wsBase = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}`;
    const ws = new WebSocket(`${wsBase}/ws`);

    ws.onopen = () => {
      ws.send(token);
      cluster.purge();
      cluster.connected = true;
      cluster.reconnectStatus = '';
      backoffMs = MIN_BACKOFF_MS;
    };

    ws.onmessage = (msg) => {
      const event: WatchEvent = JSON.parse(msg.data);
      cluster.handleEvent(event);
    };

    ws.onclose = () => {
      cluster.connected = false;
      scheduleReconnect();
    };
  }

  function scheduleReconnect() {
    const delaySec = Math.round(backoffMs / 1000);
    cluster.reconnectStatus = `Reconnecting in ${delaySec}s...`;

    let remaining = delaySec;
    const countdown = setInterval(() => {
      remaining--;
      if (remaining > 0) {
        cluster.reconnectStatus = `Reconnecting in ${remaining}s...`;
      } else {
        clearInterval(countdown);
      }
    }, 1000);

    setTimeout(() => {
      clearInterval(countdown);
      cluster.reconnectStatus = 'Connecting...';
      backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS);
      connect();
    }, backoffMs);
  }

  connect();
}
