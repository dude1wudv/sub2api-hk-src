<template>
  <div
    class="token-usage-trend chart-panel card p-4 sm:p-5"
    :class="{ 'token-usage-trend--fill': fill }"
  >
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.tokenUsageTrend') }}</h3>
      <div class="token-trend-modes" role="group" :aria-label="locale.startsWith('zh') ? '图表视图' : 'Chart view'">
        <button type="button" :aria-pressed="mode === 'breakdown'" @click="mode = 'breakdown'">{{ locale.startsWith('zh') ? '分项' : 'Breakdown' }}</button>
        <button type="button" :aria-pressed="mode === 'total'" @click="mode = 'total'">{{ locale.startsWith('zh') ? '总量' : 'Total' }}</button>
      </div>
    </div>
    <div class="token-usage-trend__body">
      <div v-if="loading" class="flex h-full items-center justify-center">
        <LoadingSpinner />
      </div>
      <Line v-else-if="trendData.length > 0 && chartData" :data="chartData" :options="lineOptions" />
      <div
        v-else
        class="flex h-full items-center justify-center text-xs text-gray-400 dark:text-dark-400"
      >
        {{ t('admin.dashboard.noDataAvailable') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { computed, ref } from 'vue'
import type { ScriptableContext } from 'chart.js'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { TrendDataPoint } from '@/types'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

const chartTheme = useChartTheme()
const { t, locale } = useI18n()
const mode = ref<'breakdown' | 'total'>('breakdown')

const props = withDefaults(defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
  fill?: boolean
}>(), {
  fill: false
})

const palette = useChartPalette()
const chartColors = computed(() => ({
  text: chartTheme.value.text,
  grid: chartTheme.value.grid,
  input: palette.value[0],
  output: palette.value[1],
  cacheCreation: palette.value[2],
  cacheRead: palette.value[5],
  cacheHitRate: palette.value[3]
}))

function areaGradient(color: string) {
  const gradients = new WeakMap<object, CanvasGradient>()
  return (context: ScriptableContext<'line'>) => {
    const { chartArea, ctx } = context.chart
    if (!chartArea || !ctx) return `${color}18`
    let gradient = gradients.get(chartArea)
    if (!gradient) {
      gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
      gradient.addColorStop(0, `${color}48`)
      gradient.addColorStop(1, `${color}00`)
      gradients.set(chartArea, gradient)
    }
    return gradient
  }
}
const chartData = computed(() => {
  if (!props.trendData?.length) return null

  return {
    labels: props.trendData.map((d) => d.date),
    datasets: mode.value === 'total' ? [{
      label: locale.value?.startsWith('zh') ? '总 Token' : 'Total tokens',
      data: props.trendData.map(d => d.input_tokens + d.output_tokens + d.cache_creation_tokens + d.cache_read_tokens),
      borderColor: chartColors.value.input,
      backgroundColor: areaGradient(chartColors.value.input),
      fill: true,
      tension: 0.25,
      yAxisID: 'y',
    }] : [
      {
        label: 'Input',
        data: props.trendData.map((d) => d.input_tokens),
        borderColor: chartColors.value.input,
        backgroundColor: `${chartColors.value.input}20`,
        fill: true,
        tension: 0.3
      },
      {
        label: 'Output',
        data: props.trendData.map((d) => d.output_tokens),
        borderColor: chartColors.value.output,
        backgroundColor: `${chartColors.value.output}20`,
        fill: true,
        tension: 0.3
      },
      {
        label: 'Cache Creation',
        data: props.trendData.map((d) => d.cache_creation_tokens),
        borderColor: chartColors.value.cacheCreation,
        backgroundColor: `${chartColors.value.cacheCreation}20`,
        fill: true,
        tension: 0.3
      },
      {
        label: 'Cache Read',
        data: props.trendData.map((d) => d.cache_read_tokens),
        borderColor: chartColors.value.cacheRead,
        backgroundColor: `${chartColors.value.cacheRead}20`,
        fill: true,
        tension: 0.3
      },
      {
        label: 'Cache Hit Rate',
        data: props.trendData.map((d) => {
          const totalPromptTokens = d.input_tokens + d.cache_read_tokens + d.cache_creation_tokens
          return totalPromptTokens > 0 ? (d.cache_read_tokens / totalPromptTokens) * 100 : 0
        }),
        borderColor: chartColors.value.cacheHitRate,
        backgroundColor: `${chartColors.value.cacheHitRate}20`,
        borderDash: [5, 5],
        fill: false,
        tension: 0.3,
        yAxisID: 'yPercent'
      }
    ]
  }
})

const lineOptions = computed(() => ({
  responsive: true,
  animation: false as const,
  maintainAspectRatio: false,
  elements: { point: { radius: 1, hoverRadius: 5 }, line: { borderWidth: 2 } },
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 13
        }
      }
    },
    tooltip: {
      ...chartTheme.value.tooltip,
      callbacks: {
        label: (context: any) => {
          if (context.dataset.yAxisID === 'yPercent') {
            return `${context.dataset.label}: ${context.raw.toFixed(1)}%`
          }
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        },
        footer: (tooltipItems: any) => {
          const dataIndex = tooltipItems[0]?.dataIndex
          if (dataIndex !== undefined && props.trendData[dataIndex]) {
            const data = props.trendData[dataIndex]
            return `Actual: $${formatCost(data.actual_cost)} | Standard: $${formatCost(data.cost)}`
          }
          return ''
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 13
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 13
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    },
    yPercent: {
      display: mode.value !== 'total',
      position: 'right' as const,
      min: 0,
      max: 100,
      grid: {
        drawOnChartArea: false
      },
      ticks: {
        color: chartColors.value.cacheHitRate,
        font: {
          size: 13
        },
        callback: (value: string | number) => `${value}%`
      }
    }
  }
}))

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatCost = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}
</script>

<style scoped>
.token-trend-modes { display:flex; gap:3px; padding:3px; border:1px solid rgb(var(--line)); border-radius:8px; background:rgb(var(--surface-2)); }
.token-trend-modes button { padding:4px 10px; border:0; border-radius:5px; color:rgb(var(--ink-3)); background:transparent; font:inherit; font-size:12px; cursor:pointer; transition:background-color 150ms ease,color 150ms ease; }
.token-trend-modes button[aria-pressed=true] { color:rgb(var(--ink)); background:rgb(var(--surface)); }
@media(prefers-reduced-motion:reduce) { .token-trend-modes button { transition:none; } }
.token-usage-trend__body {
  position: relative;
  min-width: 0;
  height: 12rem;
}

.token-usage-trend--fill {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  align-self: stretch;
  min-height: 0;
}

.token-usage-trend--fill > div:first-child {
  flex: none;
}

.token-usage-trend--fill .token-usage-trend__body {
  flex: 1 1 12rem;
  height: auto;
  min-height: 12rem;
}

/* Let the available panel space size Chart.js, not the canvas's intrinsic height. */
.token-usage-trend--fill .token-usage-trend__body :deep(canvas) {
  position: absolute;
  inset: 0;
}
</style>
