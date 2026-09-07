<template>
  <section class="card p-4 sm:p-5">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('console.analytics.activity') }}</h3>
      <div class="flex gap-1" :aria-label="t('console.analytics.metric')">
        <button v-for="item in metrics" :key="item.value" type="button" class="btn btn-sm" :class="metric === item.value ? 'btn-secondary' : 'btn-ghost'" :aria-pressed="metric === item.value" @click="metric = item.value">{{ item.label }}</button>
      </div>
    </div>
    <Skeleton v-if="loading" height="224px" :aria-label="t('common.loading')" />
    <div v-else-if="trendData.length" class="h-56"><Line :data="chartData" :options="lineOptions" :plugins="[crosshair]" /></div>
    <div v-else class="flex h-56 items-center justify-center text-xs text-gray-400">{{ t('admin.dashboard.noDataAvailable') }}</div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppearance } from '@/composables/useAppearance'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, type Plugin, type ChartOptions } from 'chart.js'
import { Line } from 'vue-chartjs'
import Skeleton from '@/components/common/Skeleton.vue'
import type { TrendDataPoint } from '@/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)
const props = defineProps<{ trendData: TrendDataPoint[]; loading?: boolean }>()
const { t } = useI18n()
const { isDark } = useAppearance()
const metric = ref<'requests' | 'tokens' | 'cost'>('tokens')
const metrics = computed(() => [
  { value: 'requests' as const, label: t('dashboard.requests') },
  { value: 'tokens' as const, label: t('dashboard.tokens') },
  { value: 'cost' as const, label: t('usage.totalCost') }
])
const colors = computed(() => ({ text: isDark.value ? '#A1A1AA' : '#71717A', grid: isDark.value ? '#27272A' : '#E5E5E7' }))
const number = (value: number) => new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
const series = (label: string, key: keyof TrendDataPoint, color: string, dashed = false) => ({
  label, data: props.trendData.map(point => Number(point[key])), borderColor: color,
  backgroundColor: color, borderWidth: 1.5, pointRadius: 0, pointHoverRadius: 3,
  pointHitRadius: 12, tension: 0.15, fill: false, borderDash: dashed ? [4, 4] : []
})
const chartData = computed(() => ({
  labels: props.trendData.map(point => point.date),
  datasets: metric.value === 'requests'
    ? [series(t('dashboard.requests'), 'requests', '#6366F1')]
    : metric.value === 'cost'
      ? [series(t('dashboard.actual'), 'actual_cost', '#6366F1'), series(t('dashboard.standard'), 'cost', '#A1A1AA', true)]
      : [series(t('dashboard.input'), 'input_tokens', '#6366F1'), series(t('dashboard.output'), 'output_tokens', '#A1A1AA'), series(t('usage.cacheCreationTokensLabel'), 'cache_creation_tokens', '#64748B', true), series(t('usage.cacheReadTokensLabel'), 'cache_read_tokens', '#818CF8', true)]
}))
const crosshair: Plugin<'line'> = {
  id: 'usage-crosshair',
  afterDatasetsDraw(chart) {
    const active = chart.tooltip?.getActiveElements()[0]
    if (!active) return
    const { ctx, chartArea } = chart
    ctx.save()
    ctx.strokeStyle = colors.value.text
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.beginPath()
    ctx.moveTo(active.element.x, chartArea.top)
    ctx.lineTo(active.element.x, chartArea.bottom)
    ctx.stroke()
    ctx.restore()
  }
}
const lineOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true, maintainAspectRatio: false, animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: { position: 'bottom', labels: { color: colors.value.text, usePointStyle: true, pointStyle: 'line', boxWidth: 12, font: { size: 11 } } },
    tooltip: {
      backgroundColor: isDark.value ? '#19191D' : '#FFFFFF', titleColor: isDark.value ? '#FAFAFA' : '#18181B', bodyColor: colors.value.text,
      borderColor: colors.value.grid, borderWidth: 1, padding: 12, cornerRadius: 8,
      callbacks: {
        label: context => `${context.dataset.label}: ${metric.value === 'cost' ? '$' + Number(context.raw).toFixed(4) : Number(context.raw).toLocaleString()}`,
        afterBody: items => {
          const point = props.trendData[items[0]?.dataIndex ?? -1]
          if (!point) return []
          const promptTokens = point.input_tokens + point.cache_creation_tokens + point.cache_read_tokens
          return [
            `${t('dashboard.requests')}: ${point.requests.toLocaleString()}`,
            `${t('dashboard.input')}: ${point.input_tokens.toLocaleString()} · ${t('dashboard.output')}: ${point.output_tokens.toLocaleString()}`,
            `${t('console.analytics.cacheHitRate')}: ${promptTokens > 0 ? (100 * point.cache_read_tokens / promptTokens).toFixed(1) + '%' : '—'}`,
            `${t('dashboard.actual')}: $${point.actual_cost.toFixed(4)} · ${t('dashboard.standard')}: $${point.cost.toFixed(4)}`
          ]
        }
      }
    }
  },
  scales: {
    x: { grid: { display: false }, border: { display: false }, ticks: { color: colors.value.text, maxTicksLimit: 8, maxRotation: 0, font: { size: 10 } } },
    y: { beginAtZero: true, border: { display: false }, grid: { color: colors.value.grid }, ticks: { color: colors.value.text, maxTicksLimit: 5, font: { size: 10 }, callback: value => metric.value === 'cost' ? '$' + number(Number(value)) : number(Number(value)) } }
  }
}))
</script>
