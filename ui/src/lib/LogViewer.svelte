<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getUserManager } from './auth';

  interface Props {
    namespace: string;
    pod: string;
    container: string;
  }

  let { namespace, pod, container }: Props = $props();

  let lines = $state<string[]>([]);
  let connected = $state(false);
  let error = $state('');
  let logEl = $state<HTMLDivElement>() as HTMLDivElement;
  let ws: WebSocket | null = null;
  let autoScroll = $state(true);

  function scrollToBottom() {
    if (autoScroll && logEl) {
      logEl.scrollTop = logEl.scrollHeight;
    }
  }

  function handleScroll() {
    if (!logEl) return;
    const atBottom = logEl.scrollHeight - logEl.scrollTop - logEl.clientHeight < 30;
    autoScroll = atBottom;
  }

  async function connect() {
    const user = await getUserManager().getUser();
    if (!user) {
      error = 'Not authenticated';
      return;
    }

    const wsBase = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}`;
    const url = `${wsBase}/ws/logs?namespace=${encodeURIComponent(namespace)}&pod=${encodeURIComponent(pod)}&container=${encodeURIComponent(container)}`;

    ws = new WebSocket(url);

    ws.onopen = () => {
      ws!.send(user.access_token);
      connected = true;
      error = '';
    };

    ws.onmessage = (msg) => {
      lines = [...lines, msg.data];
      requestAnimationFrame(scrollToBottom);
    };

    ws.onclose = (e) => {
      connected = false;
      if (e.reason) {
        error = e.reason;
      }
    };
  }

  onMount(() => {
    connect();
  });

  onDestroy(() => {
    if (ws) {
      ws.close();
      ws = null;
    }
  });
</script>

<div class="flex flex-col h-full">
  <div class="flex items-center justify-between mb-2">
    <div class="flex items-center gap-2">
      <span class="w-2 h-2 rounded-full {connected ? 'bg-emerald-500' : 'bg-red-500'}"></span>
      <span class="text-xs text-zinc-500">{connected ? 'Streaming' : error ? error : 'Disconnected'}</span>
    </div>
    <span class="text-xs text-zinc-600">{lines.length} lines</span>
  </div>
  <div
    bind:this={logEl}
    onscroll={handleScroll}
    class="flex-1 bg-neutral-800 rounded-lg p-3 overflow-auto font-mono text-xs text-zinc-400 leading-5 min-h-0"
  >
    {#if lines.length === 0 && connected}
      <div class="text-zinc-600 italic">Waiting for logs...</div>
    {:else}
      {#each lines as line}
        <div class="whitespace-pre-wrap break-all">{line}</div>
      {/each}
    {/if}
  </div>
</div>
