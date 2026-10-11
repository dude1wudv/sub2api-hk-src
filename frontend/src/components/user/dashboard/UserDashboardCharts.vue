<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler, type ChartOptions } from 'chart.js'
import type { TrendDataPoint, ModelStat } from '@/types'
import type { UsageDrilldown } from '@/views/portal/usageRoute'
import { validUsageDate } from '@/views/portal/usageRoute'
import { rankChartRows } from '@/utils/chartDistribution'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)
const props = defineProps<{ loading: boolean; startDate: string; endDate: string; granularity: string; trend: TrendDataPoint[]; models: ModelStat[] }>()
const emit = defineEmits<{
  'update:startDate': [date: string]; 'update:endDate': [date: string]; 'update:granularity': [value: string];
  dateRangeChange: [value: unknown]; granularityChange: []; refresh: []; drilldown: [value: UsageDrilldown]
}>()
const { t, locale } = useI18n()
const tx = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const palette = useChartPalette()
const theme = useChartTheme()
const number = (value: number) => value.toLocaleString(undefined, { notation: value >= 1e6 ? 'compact' : 'standard', maximumFractionDigits: 1 })
const modelTotal = computed(() => props.models.reduce((sum, item) => sum + item.requests, 0))
const displayModels = computed(() => rankChartRows(props.models, model => model.requests, palette.value))
const trendData = computed(() => ({ labels: props.trend.map(item => item.date), datasets: [
  { label: tx('输入 Token', 'Input tokens'), data: props.trend.map(item => item.input_tokens), borderColor: palette.value[0], backgroundColor: `${palette.value[0]}12`, fill: true },
  { label: tx('输出 Token', 'Output tokens'), data: props.trend.map(item => item.output_tokens), borderColor: palette.value[1], backgroundColor: `${palette.value[1]}0A`, fill: true },
  { label: tx('缓存 Token', 'Cache tokens'), data: props.trend.map(item => item.cache_read_tokens + item.cache_creation_tokens), borderColor: palette.value[2], backgroundColor: 'transparent', fill: false },
] }))
function openDate(date: string) {
  const day = date.slice(0, 10)
  if (validUsageDate(day)) emit('drilldown', { start: day, end: day })
}
function openModel(model: string) { emit('drilldown', { start: props.startDate, end: props.endDate, model }) }
const lineOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true, maintainAspectRatio: false, animation: false,
  interaction: { intersect: false, mode: 'index' },
  onClick: (_event, items) => { const point = props.trend[items[0]?.index ?? -1]; if (point) openDate(point.date) },
  elements: { point: { radius: props.trend.length > 48 ? 0 : 2, hoverRadius: 4 }, line: { borderWidth: 2, tension: .28 } },
  plugins: { legend: { position: 'bottom', labels: { color: theme.value.text, usePointStyle: true, boxWidth: 8, font: { size: 13 } } }, tooltip: { ...theme.value.tooltip } },
  scales: { x: { grid: { display: false }, ticks: { color: theme.value.text, maxTicksLimit: 6, maxRotation: 0 } }, y: { beginAtZero: true, grid: { color: theme.value.grid }, ticks: { color: theme.value.text, callback: value => number(Number(value)) } } },
}))
</script>

<template>
  <div class="workspace-chart-controls"><DateRangePicker :start-date="startDate" :end-date="endDate" @update:startDate="emit('update:startDate', $event)" @update:endDate="emit('update:endDate', $event)" @change="emit('dateRangeChange', $event)" /><label>{{ t('dashboard.granularity') }}<select :value="granularity" @change="emit('update:granularity', ($event.target as HTMLSelectElement).value); emit('granularityChange')"><option value="day">{{ t('dashboard.day') }}</option><option value="hour">{{ t('dashboard.hour') }}</option></select></label></div>
  <div class="workspace-charts" :aria-busy="loading">
    <article class="workspace-trend"><header><h3>{{ tx('用量趋势', 'Usage trend') }}</h3><span>{{ startDate }} — {{ endDate }}</span></header><div v-if="loading && !trend.length" class="workspace-chart-empty" role="status"><LoadingSpinner /></div><div v-else-if="!trend.length" class="workspace-chart-empty">{{ t('dashboard.noDataAvailable') }}</div><div v-else class="workspace-trend-chart"><Line :data="trendData" :options="lineOptions" role="img" :aria-label="tx('所选时间内的 Token 用量趋势，选择数据点可打开明细', 'Token usage in the selected period. Select a point to view details.')" /></div><div v-if="trend.length" class="workspace-date-links"><span>{{ tx('按日期查看', 'Explore a date') }}</span><button v-for="day in [...new Set(trend.map(item => item.date.slice(0, 10)))].slice(-7)" :key="day" type="button" @click="openDate(day)">{{ day.slice(5) }}</button></div></article>
    <article class="workspace-models"><header><div><h3>{{ t('dashboard.modelDistribution') }}</h3><span>{{ tx('按请求次数 · 点击查看明细', 'By request count · Open for details') }}</span></div><Icon name="cube" size="sm" /></header><div v-if="!models.length" class="workspace-chart-empty">{{ t('dashboard.noDataAvailable') }}</div><ol v-else><li v-for="model in displayModels" :key="model.model"><button type="button" @click="openModel(model.model)"><span class="workspace-model-name"><i :style="{ background: model.chartColor }"></i><strong>{{ model.model }}</strong><Icon name="arrowRight" size="sm" /></span><span class="workspace-model-values"><span>{{ number(model.requests) }} {{ t('dashboard.requests') }}</span><strong>${{ model.actual_cost.toFixed(4) }}</strong></span><span class="workspace-model-track" aria-hidden="true"><span :style="{ width: `${modelTotal > 0 ? model.requests / modelTotal * 100 : 0}%`, background: model.chartColor }"></span></span></button></li></ol></article>
  </div>
</template>
