<script lang="ts">
  import { Chart, LineController, BarController, LineElement, BarElement, PointElement, CategoryScale, LinearScale, Filler } from 'chart.js';
  import { onMount, onDestroy } from 'svelte';
  import { formatPair, pct, type ResourceUnit } from './format';

  Chart.register(LineController, BarController, LineElement, BarElement, PointElement, CategoryScale, LinearScale, Filler);

  interface Props {
    title: string;
    icon: string;
    used: number;
    capacity: number;
    unit: ResourceUnit;
    color: string;
    chartType: 'line' | 'bar';
    history: number[];
    onclick: () => void;
    loading: boolean;
    unavailable: boolean;
    detail: string;
  }

  let { title, icon, used, capacity, unit, color, chartType, history, onclick, loading, unavailable, detail }: Props = $props();

  let inactive = $derived(loading || unavailable);

  let percent = $derived(pct(used, capacity));
  let totalLabel = $derived(formatPair(used, capacity, unit));

  let canvasEl = $state<HTMLCanvasElement>() as HTMLCanvasElement;
  let chart: Chart | null = null;
  let mounted = false;

  function getTwColor(name: string): string {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  }



  function createChart() {
    if (!canvasEl) return;
    if (chart) chart.destroy();

    const resolvedColor = getTwColor(color);

    if (chartType === 'line') {
      chart = new Chart(canvasEl, {
        type: 'line',
        data: {
          labels: history.map(() => ''),
          datasets: [{
            data: [...history],
            borderColor: resolvedColor,
            backgroundColor: `color-mix(in oklch, ${resolvedColor} 20%, transparent)`,
            fill: true,
            tension: 0.3,
            pointRadius: 0,
            borderWidth: 2,
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          layout: { padding: 0 },
          scales: {
            x: { display: false },
            y: {
              min: 0,
              max: 100,
              border: { display: false },
              grid: {
                display: true,
                drawTicks: false,
                color: 'rgba(255, 255, 255, 0.06)',
              },
              ticks: { stepSize: 25, display: false },
            },
          },
          plugins: { legend: { display: false }, tooltip: { enabled: false } },
          animation: false,
          events: [],
        }
      });
    } else {
      const darkColor = `color-mix(in oklch, ${resolvedColor} 25%, transparent)`;
      chart = new Chart(canvasEl, {
        type: 'bar',
        data: {
          labels: [''],
          datasets: [{
            data: [used],
            backgroundColor: [resolvedColor],
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          indexAxis: 'y',
          layout: { padding: 0 },
          scales: {
            x: { display: false, min: 0, max: capacity },
            y: {
              display: false,
              afterFit(scale) { scale.paddingTop = 0; scale.paddingBottom = 0; },
            },
          },
          datasets: { bar: { barPercentage: 1, categoryPercentage: 1 } },
          plugins: {
            legend: { display: false },
            tooltip: { enabled: false },
          },
          animation: false,
          events: [],
        },
        plugins: [{
          id: 'bgBar',
          beforeDraw(c) {
            const { ctx, chartArea } = c;
            ctx.save();
            ctx.fillStyle = darkColor;
            ctx.fillRect(chartArea.left, chartArea.top, chartArea.right - chartArea.left, chartArea.bottom - chartArea.top);
            ctx.restore();
          }
        }],
      });
    }
  }

  function updateChart() {
    if (!chart) return;

    if (chartType === 'line') {
      chart.data.labels = history.map(() => '');
      chart.data.datasets[0].data = [...history];
    } else {
      chart.data.datasets[0].data = [used];
      (chart.options.scales!.x as any).max = capacity;
    }
    chart.update('none');
  }

  onMount(() => {
    mounted = true;
    createChart();
  });

  onDestroy(() => {
    if (chart) chart.destroy();
  });

  $effect(() => {
    history;
    percent;
    used;
    capacity;
    color;
    chartType;
    loading;
    unavailable;
    if (!mounted || loading || unavailable) return;
    if (chart) {
      updateChart();
    } else {
      createChart();
    }
  });
</script>

{#if inactive}
  <div class="bg-neutral-700 rounded-xl p-4 text-left w-full flex gap-4 opacity-50 shadow-lg">
    <div class="flex-1 flex flex-col min-w-0 {loading ? 'animate-pulse' : ''}">
      <div class="text-lg font-semibold text-neutral-500 mb-2">{title}</div>
      {#if unavailable}
        <div class="mt-auto {chartType === 'bar' ? 'h-7' : 'h-16'} rounded-lg overflow-hidden bg-neutral-600 flex items-center justify-center">
          <span class="text-xs text-neutral-500">No data available</span>
        </div>
      {:else}
        <div class="mt-auto {chartType === 'bar' ? 'h-7' : 'h-16'} rounded-lg overflow-hidden bg-neutral-600"></div>
      {/if}
    </div>
    <div class="flex flex-col items-end shrink-0 {loading ? 'animate-pulse' : ''}">
      <div
        class="w-10 h-10 shrink-0 mask-contain mask-no-repeat mask-center"
        style="mask-image: url({icon}); background-color: var(--color-neutral-600);"
      ></div>
      <div class="mt-auto">
        <div class="text-xl font-semibold text-neutral-500 text-right">--%</div>
        <div class="text-sm text-neutral-600 text-right">n/a</div>
      </div>
    </div>
  </div>
{:else}
  <button class="bg-neutral-700 rounded-xl p-4 cursor-pointer hover:bg-neutral-600 transition-colors text-left w-full flex gap-4 shadow-lg" onclick={onclick}>
    <div class="flex-1 flex flex-col min-w-0">
      <div class="flex items-baseline gap-2 mb-2">
        <div class="text-lg font-semibold" style="color: var({color});">{title}</div>
        {#if detail}
          <div class="text-xs text-neutral-500">{detail}</div>
        {/if}
      </div>
      <div class="mt-auto {chartType === 'bar' ? 'h-7' : 'h-16'} rounded-lg overflow-hidden">
        <canvas bind:this={canvasEl}></canvas>
      </div>
    </div>
    <div class="flex flex-col items-end shrink-0">
      <div
        class="w-10 h-10 shrink-0 mask-contain mask-no-repeat mask-center"
        style="mask-image: url({icon}); background-color: var({color});"
      ></div>
      <div class="mt-auto">
        <div class="text-xl font-semibold text-neutral-100 text-right">{percent}%</div>
        <div class="text-sm text-neutral-300 text-right">{totalLabel}</div>
      </div>
    </div>
  </button>
{/if}
