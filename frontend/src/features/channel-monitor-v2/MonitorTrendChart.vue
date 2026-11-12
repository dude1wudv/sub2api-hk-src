<template>
  <section
    class="card telemetry-chart-shell flex min-h-[360px] flex-col overflow-hidden rounded-3xl p-5 ring-1 ring-gray-900/5 dark:ring-white/5 sm:p-6"
  >
    <div class="card-header telemetry-chart-header mb-4 flex shrink-0 flex-wrap items-start justify-between gap-3 !border-0 !p-0">
      <div class="min-w-0">
        <h2 class="telemetry-panel-title flex items-center gap-2 text-[15px] font-semibold tracking-tight">
          <span class="telemetry-panel-icon inline-flex h-6 w-6 items-center justify-center rounded-md" aria-hidden="true">
            <Icon name="chart" size="sm" />
          </span>
          {{ t('channelMonitorV2.chart.title') }}
        </h2>
        <p class="telemetry-panel-subtitle mt-0.5 text-xs">
          {{ t('channelMonitorV2.chart.description') }}
        </p>
      </div>
      <div class="telemetry-chart-meta flex w-full min-w-0 flex-wrap items-center justify-end gap-2.5 text-xs sm:w-auto">
        <span class="inline-flex shrink-0 items-center gap-1.5">
          <span class="h-2 w-2 rounded-full" :style="{ backgroundColor: seriesColors.error }"></span>{{ t('channelMonitorV2.chart.errorLegend') }}
        </span>
        <span class="inline-flex shrink-0 items-center gap-1.5">
          <span class="h-2 w-2 rounded-full" :style="{ backgroundColor: seriesColors.cache }"></span>{{ t('channelMonitorV2.chart.cacheLegend') }}
        </span>
        <span class="inline-flex shrink-0 items-center gap-1.5">
          <span class="h-2 w-2 rounded-full" :style="{ backgroundColor: seriesColors.ttft }"></span>{{ t('channelMonitorV2.chart.ttftLegend') }}
        </span>
        <span class="telemetry-bucket-pill badge badge-gray shrink-0 tabular-nums">{{ bucketLabel }}</span>
        <button
          type="button"
          class="telemetry-reset-btn inline-flex shrink-0 items-center rounded-lg px-2.5 py-1 text-[11px] font-medium focus:outline-none focus:ring-2 focus:ring-primary-500/20 disabled:cursor-not-allowed disabled:opacity-45"
          :disabled="!zoomed"
          @click="resetChartZoom"
        >
          {{ t('channelMonitorV2.chart.resetZoom') }}
        </button>
      </div>
    </div>
    <div class="card-body min-h-0 flex-1 !p-0">
      <div v-if="loading" class="flex h-[280px] items-center justify-center sm:h-[300px]">
        <div class="telemetry-muted animate-pulse text-sm">{{ t('common.loading') }}</div>
      </div>
      <div
        v-else-if="chartData"
        ref="chartRef"
        class="h-[280px] sm:h-[300px]"
        @wheel="onChartWheel"
      >
        <Line :data="chartData" :options="chartOptions" />
      </div>
      <div v-else class="flex h-[280px] items-center justify-center sm:h-[300px]">
        <EmptyState
          :title="t('channelMonitorV2.chart.emptyTitle')"
          :description="t('channelMonitorV2.empty.description')"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import type { MonitorCoverage, MonitorMetric, MonitorHealth } from '@/api/channelMonitorV2'
import { formatMonitorMs, formatMonitorPercent } from '@/features/channel-monitor-v2/monitorFormat'
import {
  applyWheelZoom,
  clientXRatio,
  isZoomed,
  resetZoom,
  sliceByZoom,
  type ZoomState,
} from '@/features/channel-monitor-v2/monitorZoom'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)
const chartTheme = useChartTheme()
const seriesPalette = useChartPalette()
const { t, locale } = useI18n()

const props = defineProps<{
  trend: Array<{ bucket_start: string; metrics: MonitorMetric; health: MonitorHealth }>
  coverage: MonitorCoverage | null
  loading?: boolean
}>()

const chartRef = ref<HTMLElement | null>(null)
const zoom = ref<ZoomState>(resetZoom())
const zoomed = computed(() => isZoomed(zoom.value))

const seriesColors = computed(() => ({
  ttft: seriesPalette.value[0],
  cache: seriesPalette.value[2],
  error: seriesPalette.value[7],
}))


const bucketLabel = computed(() => {
  const seconds = props.coverage?.bucket_seconds || 60
  const minutes = seconds / 60
  if (minutes < 60) return t('channelMonitorV2.bucket.minutes', { count: minutes })
  const hours = minutes / 60
  if (hours < 24) return t('channelMonitorV2.bucket.hours', { count: hours })
  return t('channelMonitorV2.bucket.days', { count: hours / 24 })
})

const chartData = computed(() => {
  const points = visibleTrend.value
  if (!points.length) return null
  const pointRadius = points.length > 48 ? 0 : 2.5
  const pointSurface = chartTheme.value.pointBorder
  const labels = points.map((p) =>
    new Intl.DateTimeFormat(locale.value || undefined, {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(p.bucket_start))
  )
  const errorRates = smoothTrend(points.map((p) => (p.metrics.error_rate || 0) * 100))
  const cacheRates = smoothTrend(points.map((p) => (p.metrics.cache_rate || 0) * 100))
  const ttftP50 = smoothTrend(points.map((p) => p.metrics.ttft?.p50_ms ?? null))
  return {
    labels,
    datasets: [
      {
        label: t('channelMonitorV2.chart.errorDataset'),
        data: errorRates,
        borderColor: seriesColors.value.error,
        backgroundColor: `${seriesColors.value.error}16`,
        pointBackgroundColor: pointSurface,
        pointBorderColor: seriesColors.value.error,
        pointBorderWidth: 2,
        yAxisID: 'yPct',
        tension: 0.24,
        cubicInterpolationMode: 'monotone' as const,
        fill: 'origin' as const,
        pointRadius,
        pointHoverRadius: 5,
        pointHitRadius: 12,
        borderWidth: 2,
      },
      {
        label: t('channelMonitorV2.chart.cacheDataset'),
        data: cacheRates,
        borderColor: seriesColors.value.cache,
        backgroundColor: `${seriesColors.value.cache}12`,
        pointBackgroundColor: pointSurface,
        pointBorderColor: seriesColors.value.cache,
        pointBorderWidth: 2,
        yAxisID: 'yPct',
        tension: 0.24,
        cubicInterpolationMode: 'monotone' as const,
        fill: false,
        pointRadius,
        pointHoverRadius: 5,
        pointHitRadius: 12,
        borderWidth: 2,
      },
      {
        label: t('channelMonitorV2.chart.ttftDataset'),
        data: ttftP50,
        borderColor: seriesColors.value.ttft,
        backgroundColor: `${seriesColors.value.ttft}12`,
        pointBackgroundColor: pointSurface,
        pointBorderColor: seriesColors.value.ttft,
        pointBorderWidth: 2,
        yAxisID: 'yTtft',
        tension: 0.24,
        cubicInterpolationMode: 'monotone' as const,
        fill: false,
        pointRadius,
        pointHoverRadius: 5,
        pointHitRadius: 12,
        borderWidth: 2,
        spanGaps: true,
      },
    ],
  }
})

/** Window the series by zoom state around the cursor — not always the last N points. */
const visibleTrend = computed(() => sliceByZoom(props.trend || [], zoom.value))

function onChartWheel(event: WheelEvent) {
  // Plain vertical wheel zooms X (narrower time range); shift/horizontal pans.
  event.preventDefault()
  const ratio = clientXRatio(event.clientX, chartRef.value)
  zoom.value = applyWheelZoom(zoom.value, event, ratio)
}

function resetChartZoom() {
  zoom.value = resetZoom()
}

watch(() => props.trend, () => {
  zoom.value = resetZoom()
})

function smoothTrend(values: Array<number | null>): Array<number | null> {
  if (values.length <= 2) return values
  return values.map((value, index) => {
    if (value == null) return null
    const neighbors = values.slice(Math.max(0, index - 1), Math.min(values.length, index + 2))
      .filter((item): item is number => item != null)
    if (!neighbors.length) return value
    return neighbors.reduce((sum, item) => sum + item, 0) / neighbors.length
  })
}

const chartOptions = computed<ChartOptions<'line'>>(() => {
  const text = chartTheme.value.text
  const grid = chartTheme.value.grid
  return {
    responsive: true,
    animation: false as const,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    elements: {
      point: {
        radius: visibleTrend.value.length > 48 ? 0 : 2.5,
        hoverRadius: 5,
        hitRadius: 12,
        borderWidth: 2,
        backgroundColor: chartTheme.value.pointBorder,
      },
      line: {
        borderWidth: 2,
        tension: 0.24,
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        ...chartTheme.value.tooltip,
        boxPadding: 4,
        usePointStyle: true,
        displayColors: true,
        callbacks: {
          label(ctx: { dataset: { label?: string }; parsed: { y: number | null } }) {
            const label = ctx.dataset.label || ''
            const y = ctx.parsed.y
            if (y == null) return `${label}: -`
            if (label === t('channelMonitorV2.chart.errorDataset') || label === t('channelMonitorV2.chart.cacheDataset')) {
              return `${label}: ${formatMonitorPercent(y / 100)}`
            }
            return `${label}: ${formatMonitorMs(y)}`
          },
        },
      },
    },
    scales: {
      x: {
        border: { display: false },
        grid: { display: false },
        ticks: { color: text, maxRotation: 0, autoSkip: true, maxTicksLimit: 8, autoSkipPadding: 12, font: { size: 12 } },
      },
      yPct: {
        type: 'linear' as const,
        position: 'left' as const,
        min: 0,
        suggestedMax: 100,
        border: { display: false },
        grid: { color: grid },
        ticks: {
          color: text,
          padding: 8,
          font: { size: 12 },
          callback: (v: string | number) => `${v}%`,
        },
        title: { display: true, text: t('channelMonitorV2.chart.percentAxis'), color: text, font: { size: 12 } },
      },
      yTtft: {
        type: 'linear' as const,
        position: 'right' as const,
        min: 0,
        border: { display: false },
        grid: { display: false },
        ticks: {
          color: text,
          padding: 8,
          font: { size: 12 },
          callback: (v: string | number) => formatMonitorMs(Number(v)),
        },
        title: { display: true, text: t('channelMonitorV2.metrics.ttftP50'), color: text, font: { size: 12 } },
      },
    },
  }
})
</script>

<style scoped>
.telemetry-chart-shell {
  background: rgb(var(--surface));
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-panel);
  --tw-ring-shadow: 0 0 #0000;
}
.telemetry-panel-title {
  color: rgb(var(--ink));
}
.telemetry-panel-icon {
  color: rgb(var(--brand-moss));
  background: rgb(var(--brand-moss) / 0.1);
  border: 1px solid rgb(var(--brand-moss) / 0.22);
}
.telemetry-panel-subtitle,
.telemetry-muted {
  color: rgb(var(--ink-3));
}
.telemetry-chart-meta {
  color: rgb(var(--ink-2));
  font-variant-numeric: tabular-nums;
}
.telemetry-bucket-pill {
  background: rgb(var(--surface-2));
  color: rgb(var(--ink-2));
  border: 1px solid rgb(var(--line));
}
.telemetry-reset-btn {
  background: rgb(var(--surface));
  color: rgb(var(--ink-2));
  border: 1px solid rgb(var(--line));
}
.telemetry-reset-btn:hover:not(:disabled) {
  background: rgb(var(--surface-2));
  color: rgb(var(--ink));
  border-color: rgb(var(--line-strong));
}
</style>
