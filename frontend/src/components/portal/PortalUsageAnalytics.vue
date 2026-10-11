<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler, type ChartOptions } from 'chart.js'
import { Doughnut, Line } from 'vue-chartjs'
import { usageAPI } from '@/api'
import type { UsageStatsResponse, ModelStat, GroupStat, TrendDataPoint } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import type { UsageDrilldown } from '@/views/portal/usageRoute'
import { validUsageDate } from '@/views/portal/usageRoute'
import { distributionValue, rankChartRows } from '@/utils/chartDistribution'

ChartJS.register(ArcElement, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)
const { locale, t } = useI18n()
const emit = defineEmits<{ drilldown: [value: UsageDrilldown] }>()
const tx = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const dateText = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
const now = new Date()
const start = ref(dateText(new Date(now.getFullYear(), now.getMonth(), 1)))
const end = ref(dateText(now))
const preset = ref('month')
const granularity = ref<'day' | 'hour'>('day')
const applied = ref({ start: start.value, end: end.value })
const rangeError = ref('')
const loading = ref(false)
const stats = ref<UsageStatsResponse | null>(null)
const models = ref<ModelStat[]>([])
const groups = ref<GroupStat[]>([])
const trend = ref<TrendDataPoint[]>([])
const errors = ref({ stats: false, models: false, snapshot: false })
const metrics = ref<Record<string, 'tokens' | 'actual_cost'>>({ models: 'tokens', groups: 'tokens', endpoints: 'tokens' })
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
let requestSequence = 0
const colors = useChartPalette()
const chartTheme = useChartTheme()
const number = (value: number) => !Number.isFinite(value) ? '—' : value >= 1e9 ? `${(value / 1e9).toFixed(2)}B` : value >= 1e6 ? `${(value / 1e6).toFixed(2)}M` : value >= 1e3 ? `${(value / 1e3).toFixed(2)}K` : value.toLocaleString()
const money = (value: number) => Number.isFinite(value) ? `$${value.toFixed(value < 0.01 && value > 0 ? 6 : 4)}` : '—'
const duration = (value: number) => Number.isFinite(value) ? `${(value / 1000).toFixed(2)}s` : '—'
const metricNumberStyle = (value: string) => ({ '--metric-width': Math.max(value.length * 0.64, 1) })
const selectedDay = ref('')
const availableDays = computed(() => [...new Set(trend.value.map(item => item.date.slice(0, 10)).filter(validUsageDate))])
function openModel(model: string) { emit('drilldown', { ...applied.value, model }) }
function openDate(date: string) {
  const day = date.slice(0, 10)
  if (validUsageDate(day)) emit('drilldown', { start: day, end: day })
}

async function load() {
  if (!validUsageDate(start.value) || !validUsageDate(end.value) || start.value > end.value) {
    rangeError.value = tx('请选择有效的起止日期。', 'Choose a valid date range.')
    return
  }
  rangeError.value = ''
  applied.value = { start: start.value, end: end.value }
  const sequence = ++requestSequence
  loading.value = true
  errors.value = { stats: false, models: false, snapshot: false }
  stats.value = null
  models.value = []; groups.value = []; trend.value = []
  const params = { start_date: start.value, end_date: end.value, timezone }
  const results = await Promise.allSettled([
    usageAPI.getStats(params),
    usageAPI.getDashboardModels({ ...params, model_source: 'requested' }),
    usageAPI.getDashboardSnapshotV2({ ...params, granularity: granularity.value, include_trend: true, include_model_stats: false, include_group_stats: true }),
  ])
  if (sequence !== requestSequence) return
  const [summary, modelResult, snapshot] = results
  if (summary.status === 'fulfilled') stats.value = summary.value
  else errors.value.stats = true
  if (modelResult.status === 'fulfilled') models.value = modelResult.value.models ?? []
  else errors.value.models = true
  if (snapshot.status === 'fulfilled') {
    groups.value = snapshot.value.groups ?? []
    trend.value = snapshot.value.trend ?? []
  } else errors.value.snapshot = true
  loading.value = false
  if (!availableDays.value.includes(selectedDay.value)) selectedDay.value = availableDays.value[availableDays.value.length - 1] || ''
}

function changePreset() {
  if (preset.value === 'custom') return
  const today = new Date()
  const first = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  if (preset.value === 'month') first.setDate(1)
  else if (preset.value === '7' || preset.value === '30') first.setDate(first.getDate() - Number(preset.value) + 1)
  start.value = dateText(first); end.value = dateText(today)
  granularity.value = preset.value === 'today' ? 'hour' : 'day'
  void load()
}

type DistributionItem = { id: string; name: string; requests: number; total_tokens: number; actual_cost: number; cost: number }
const sections = computed(() => [
  { id: 'models', title: tx('模型分布', 'Model distribution'), column: tx('请求模型', 'Requested model'), icon: 'cube' as const, error: errors.value.models, rows: models.value.map(item => ({ ...item, name: item.model, id: item.model })) },
  { id: 'groups', title: tx('分组使用分布', 'Group distribution'), column: tx('分组', 'Group'), icon: 'grid' as const, error: errors.value.snapshot, rows: groups.value.map(item => ({ ...item, name: item.group_name || `#${item.group_id}`, id: String(item.group_id) })) },
  { id: 'endpoints', title: tx('端点分布', 'Endpoint distribution'), column: tx('请求端点', 'Request endpoint'), icon: 'link' as const, error: errors.value.stats, rows: (stats.value?.endpoints ?? []).map(item => ({ ...item, name: item.endpoint || t('common.unknown'), id: item.endpoint || '' })) },
].map(section => ({
  ...section,
  rows: rankChartRows<DistributionItem>(section.rows, row => distributionValue(row, metrics.value[section.id]), colors.value),
})))
type DistributionRow = DistributionItem & { chartColor: string }
function distributionData(id: string, rows: DistributionRow[]) {
  return { labels: rows.map(row => row.name), datasets: [{ data: rows.map(row => distributionValue(row, metrics.value[id])), backgroundColor: rows.map(row => row.chartColor), borderWidth: 0, hoverOffset: 5 }] }
}
function hasDistribution(id: string, rows: DistributionRow[]) {
  return rows.some(row => distributionValue(row, metrics.value[id]) > 0)
}
const doughnutOptions = computed<ChartOptions<'doughnut'>>(() => ({ responsive: true, maintainAspectRatio: false, animation: false, cutout: '72%', plugins: { legend: { display: false }, tooltip: { ...chartTheme.value.tooltip, callbacks: { label: context => `${context.label}: ${Number(context.raw).toLocaleString(undefined, { maximumFractionDigits: 6 })}` } } } }))
function distributionOptions(id: string, rows: DistributionRow[]): ChartOptions<'doughnut'> {
  return { ...doughnutOptions.value, onClick: (_event, items) => {
    const row = rows[items[0]?.index ?? -1]
    if (id === 'models' && row) openModel(row.name)
  } }
}
const trendData = computed(() => ({
  labels: trend.value.map(item => item.date),
  datasets: [
    { label: tx('输入', 'Input'), data: trend.value.map(item => item.input_tokens), borderColor: colors.value[0], backgroundColor: `${colors.value[0]}14`, fill: true },
    { label: tx('输出', 'Output'), data: trend.value.map(item => item.output_tokens), borderColor: colors.value[1], backgroundColor: `${colors.value[1]}14`, fill: true },
    { label: tx('缓存写入', 'Cache creation'), data: trend.value.map(item => item.cache_creation_tokens), borderColor: colors.value[2], backgroundColor: `${colors.value[2]}14`, fill: true },
    { label: tx('缓存读取', 'Cache read'), data: trend.value.map(item => item.cache_read_tokens), borderColor: colors.value[5], backgroundColor: `${colors.value[5]}14`, fill: true },
    { label: tx('缓存命中率', 'Cache hit rate'), data: trend.value.map(item => { const prompt = item.input_tokens + item.cache_creation_tokens + item.cache_read_tokens; return prompt > 0 ? item.cache_read_tokens / prompt * 100 : 0 }), borderColor: colors.value[3], backgroundColor: colors.value[3], borderDash: [5, 5], yAxisID: 'percentage', fill: false },
  ],
}))
const lineOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true, maintainAspectRatio: false, animation: false,
  onClick: (_event, items) => { const item = trend.value[items[0]?.index ?? -1]; if (item) openDate(item.date) },
  interaction: { intersect: false, mode: 'index' },
  elements: { point: { radius: trend.value.length > 48 ? 0 : 2, hoverRadius: 5 }, line: { borderWidth: 2, tension: 0.2 } },
  plugins: { legend: { position: 'bottom', labels: { color: chartTheme.value.text, usePointStyle: true, boxWidth: 8, font: { size: 13 } } }, tooltip: { ...chartTheme.value.tooltip, callbacks: {
    label: context => `${context.dataset.label}: ${context.dataset.yAxisID === 'percentage' ? `${Number(context.raw).toFixed(1)}%` : number(Number(context.raw))}`,
    footer: items => { const item = trend.value[items[0]?.dataIndex ?? -1]; return item ? `${tx('实际', 'Actual')} ${money(item.actual_cost)} · ${tx('标准', 'Standard')} ${money(item.cost)}` : '' },
  } } },
  scales: {
    x: { grid: { display: false }, ticks: { color: chartTheme.value.text, maxTicksLimit: 8, font: { size: 13 } } },
    y: { beginAtZero: true, grid: { color: chartTheme.value.grid }, ticks: { color: chartTheme.value.text, callback: value => number(Number(value)), font: { size: 13 } } },
    percentage: { position: 'right', min: 0, max: 100, grid: { drawOnChartArea: false }, ticks: { color: colors.value[3], callback: value => `${value}%`, font: { size: 13 } } },
  },
}))
onMounted(() => { void load() })
onBeforeUnmount(() => { requestSequence++ })
</script>

<template>
  <section class="portal-usage-analytics" :aria-label="tx('用量分析', 'Usage analytics')" :aria-busy="loading">
    <div class="analytics-heading">
      <div><h2>{{ tx('用量分析', 'Usage analytics') }}</h2><p>{{ tx('统计所选时间段内的全部请求，与下方明细筛选独立。', 'All requests in the analysis period. Independent of the record filters below.') }}</p></div>
      <button class="analytics-refresh" type="button" :disabled="loading" @click="load"><Icon name="refresh" size="sm" />{{ tx('刷新', 'Refresh') }}</button>
    </div>
    <form class="analytics-controls" @submit.prevent="load">
      <label>{{ tx('时间范围', 'Period') }}<select v-model="preset" data-testid="analytics-preset" @change="changePreset"><option value="month">{{ tx('本月', 'This month') }}</option><option value="today">{{ tx('今天', 'Today') }}</option><option value="7">{{ tx('近 7 天', 'Last 7 days') }}</option><option value="30">{{ tx('近 30 天', 'Last 30 days') }}</option><option value="custom">{{ tx('自定义', 'Custom') }}</option></select></label>
      <template v-if="preset === 'custom'"><label>{{ tx('开始日期', 'Start date') }}<input v-model="start" data-testid="analytics-start" type="date" required /></label><label>{{ tx('结束日期', 'End date') }}<input v-model="end" data-testid="analytics-end" type="date" required /></label><button class="analytics-apply" type="submit">{{ tx('应用', 'Apply') }}</button></template>
      <span class="analytics-range">{{ applied.start }} — {{ applied.end }}</span>
      <label class="analytics-granularity">{{ tx('粒度', 'Granularity') }}<select v-model="granularity" data-testid="analytics-granularity" @change="load"><option value="day">{{ tx('按天', 'Daily') }}</option><option value="hour">{{ tx('按小时', 'Hourly') }}</option></select></label>
    </form>
    <p v-if="rangeError" role="alert" class="analytics-error">{{ rangeError }}</p>
    <p class="analytics-timezone">{{ tx('按自然日统计，时区', 'Calendar days, time zone') }} {{ timezone }}</p>
    <p v-if="errors.stats" role="alert" class="analytics-error">{{ tx('汇总与端点统计暂时无法加载，请重试。', 'Summary and endpoint statistics could not be loaded. Please retry.') }}</p>
    <div class="analytics-summary">
      <article><span class="analytics-metric-icon blue"><Icon name="document" /></span><div><span>{{ tx('总请求数', 'Total requests') }}</span><strong data-testid="analytics-requests" :style="metricNumberStyle(stats ? number(stats.total_requests) : '—')">{{ stats ? number(stats.total_requests) : '—' }}</strong><small>{{ tx('所选分析时间段内', 'In the analysis period') }}</small></div></article>
      <article><span class="analytics-metric-icon amber"><Icon name="cube" /></span><div><span>{{ tx('总 Token', 'Total tokens') }}</span><strong :style="metricNumberStyle(stats ? number(stats.total_tokens) : '—')">{{ stats ? number(stats.total_tokens) : '—' }}</strong><small v-if="stats">{{ tx('输入', 'Input') }} {{ number(stats.total_input_tokens) }} · {{ tx('输出', 'Output') }} {{ number(stats.total_output_tokens) }}</small><small v-if="stats">{{ tx('缓存写入', 'Cache creation') }} {{ number(stats.total_cache_creation_tokens) }} · {{ tx('读取', 'Read') }} {{ number(stats.total_cache_read_tokens) }}</small></div></article>
      <article :class="{ 'analytics-summary-wide': stats && money(stats.total_actual_cost).length > 14 }"><span class="analytics-metric-icon green"><Icon name="dollar" /></span><div><span>{{ tx('实际消费', 'Actual cost') }}</span><strong data-testid="analytics-actual-cost" :style="metricNumberStyle(stats ? money(stats.total_actual_cost) : '—')">{{ stats ? money(stats.total_actual_cost) : '—' }}</strong><small v-if="stats">{{ tx('标准费用', 'Standard cost') }} <s>{{ money(stats.total_cost) }}</s></small></div></article>
      <article><span class="analytics-metric-icon purple"><Icon name="clock" /></span><div><span>{{ tx('平均耗时', 'Average duration') }}</span><strong :style="metricNumberStyle(stats ? duration(stats.average_duration_ms) : '—')">{{ stats ? duration(stats.average_duration_ms) : '—' }}</strong><small>{{ tx('单次请求平均总耗时', 'Average request duration') }}</small></div></article>
    </div>
    <div class="analytics-grid">
      <article v-for="section in sections" :key="section.id" class="analytics-panel" :data-distribution="section.id">
        <header><h3><Icon :name="section.icon" size="sm" />{{ section.title }}</h3><div class="analytics-segment" :aria-label="section.title"><button type="button" :aria-pressed="metrics[section.id] === 'tokens'" @click="metrics[section.id] = 'tokens'">Token</button><button type="button" :aria-pressed="metrics[section.id] === 'actual_cost'" @click="metrics[section.id] = 'actual_cost'">{{ tx('实际消费', 'Actual cost') }}</button></div></header>
        <p v-if="loading" class="analytics-state" role="status">{{ tx('正在加载统计…', 'Loading statistics…') }}</p>
        <p v-else-if="section.error" class="analytics-state analytics-error" role="alert">{{ tx('统计加载失败，点击刷新重试。', 'Could not load statistics. Refresh to retry.') }}</p>
        <p v-else-if="!section.rows.length" class="analytics-state">{{ tx('所选时间段暂无记录', 'No requests in this period') }}</p>
        <div v-else class="analytics-distribution">
          <div class="analytics-doughnut"><Doughnut v-if="hasDistribution(section.id, section.rows)" :data="distributionData(section.id, section.rows)" :options="distributionOptions(section.id, section.rows)" :aria-label="section.title" role="img" /><span v-else class="analytics-zero">{{ tx('当前指标为 0', 'Metric total is 0') }}</span></div>
          <div class="analytics-table-scroll"><table><thead><tr><th>{{ section.column }}</th><th>{{ tx('请求', 'Requests') }}</th><th>Token</th><th>{{ tx('实际', 'Actual') }}</th><th>{{ tx('标准', 'Standard') }}</th></tr></thead><tbody><tr v-for="row in section.rows" :key="row.id"><td :title="row.name"><span class="analytics-row-label"><i :style="{ background: row.chartColor }" /><ModelIcon v-if="section.id === 'models'" :model="row.name" size="16px" /><button v-if="section.id === 'models'" type="button" class="analytics-model-link" :aria-label="tx(`查看 ${row.name} 的请求明细`, `View requests for ${row.name}`)" @click="openModel(row.name)">{{ row.name }} <span aria-hidden="true">↗</span></button><span v-else>{{ row.name }}</span></span></td><td>{{ number(row.requests) }}</td><td>{{ number(row.total_tokens) }}</td><td class="analytics-actual">{{ money(row.actual_cost) }}</td><td>{{ money(row.cost) }}</td></tr></tbody></table></div>
        </div>
      </article>
      <article class="analytics-panel analytics-trend"><header><div><h3><Icon name="chart" size="sm" />{{ tx('Token 使用趋势', 'Token usage trend') }}</h3></div><form v-if="availableDays.length" class="analytics-drilldown" @submit.prevent="openDate(selectedDay)"><select v-model="selectedDay" :aria-label="tx('选择明细日期', 'Choose a date for details')"><option v-for="day in availableDays" :key="day" :value="day">{{ day }}</option></select><button type="submit">{{ tx('查看明细', 'View details') }} ↗</button></form></header><p v-if="loading" class="analytics-state" role="status">{{ tx('正在加载统计…', 'Loading statistics…') }}</p><p v-else-if="errors.snapshot" class="analytics-state analytics-error" role="alert">{{ tx('趋势加载失败，点击刷新重试。', 'Could not load trend. Refresh to retry.') }}</p><p v-else-if="!trend.length" class="analytics-state">{{ tx('所选时间段暂无记录', 'No requests in this period') }}</p><div v-else class="analytics-line"><Line :data="trendData" :options="lineOptions" :aria-label="tx('输入、输出、缓存写入、缓存读取和缓存命中率趋势', 'Input, output, cache creation, cache read and cache hit rate trend')" role="img" /></div><p class="analytics-chart-note">{{ tx('缓存命中率 = 缓存读取 /（输入 + 缓存写入 + 缓存读取）', 'Cache hit rate = cache read / (input + cache creation + cache read)') }}</p></article>
    </div>
  </section>
</template>

<style scoped>
.portal-usage-analytics { margin: 34px 0 42px; color: rgb(var(--ink)); font-size: 14px; }
.analytics-heading { display: flex; justify-content: space-between; gap: 16px; align-items: center; margin-bottom: 22px; }
.analytics-heading h2 { margin: 0; font-size: 24px; font-weight: 600; }
.analytics-heading p, .analytics-timezone, .analytics-chart-note { color: rgb(var(--ink-2)); font-size: 13px; margin: 8px 0 0; line-height: 1.6; }
.analytics-refresh { display: flex; min-height: 44px; align-items: center; gap: 7px; border: 1px solid rgb(var(--line-strong)); border-radius: 10px; padding: 9px 12px; background: rgb(var(--surface)); color: rgb(var(--ink)); white-space: nowrap; }
.analytics-refresh:disabled { opacity: .5; }
.analytics-controls { display: flex; align-items: end; gap: 14px; flex-wrap: wrap; }
.analytics-controls label { display: flex; align-items: center; gap: 8px; color: rgb(var(--ink-2)); font-size: 13px; }
.analytics-controls select, .analytics-controls input { min-width: 0; min-height: 44px; border: 1px solid rgb(var(--line-strong)); border-radius: 10px; background: rgb(var(--surface)); color: rgb(var(--ink)); padding: 9px 10px; font: inherit; }
.analytics-range { font-size: 13px; color: rgb(var(--ink-2)); padding: 10px 0; }
.analytics-granularity { margin-left: auto; }
.analytics-apply { min-height: 44px; background: rgb(var(--action-bg)); color: rgb(var(--action-text)); border: 0; border-radius: 10px; padding: 10px 16px; }
.analytics-timezone { margin: 12px 0 18px; }
.analytics-summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr)); gap: 1px; border: 1px solid rgb(var(--line)); border-radius: 16px; margin-bottom: 20px; overflow: hidden; background: rgb(var(--line)); }
.analytics-summary article { display: flex; gap: 12px; padding: 23px 18px; min-width: 0; background: rgb(var(--surface)); }
.analytics-summary-wide { grid-column: 1 / -1; }
.analytics-summary article > div { flex: 1; min-width: 0; container-type: inline-size; }
.analytics-summary article > div > span { color: rgb(var(--ink-2)); font-size: 13px; }
.analytics-summary strong { display: block; font-size: clamp(14px, calc(100cqi / var(--metric-width, 6)), 30px); line-height: 1.4; font-weight: 600; font-variant-numeric: tabular-nums; white-space: nowrap; }
.analytics-summary small { display: block; color: rgb(var(--ink-2)); font-size: 13px; line-height: 1.7; }
.analytics-metric-icon { width: 34px; height: 34px; padding: 8px; display: flex; flex-shrink: 0; border-radius: 10px; }
.blue { background: color-mix(in srgb, var(--chart-1) 12%, transparent); color: var(--chart-1); }
.amber { background: color-mix(in srgb, var(--chart-3) 12%, transparent); color: var(--chart-3); }
.green { background: color-mix(in srgb, var(--chart-2) 12%, transparent); color: var(--chart-2); }
.purple { background: color-mix(in srgb, var(--chart-4) 12%, transparent); color: var(--chart-4); }
.analytics-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px; }
.analytics-trend { order: -1; grid-column: 1 / -1; }
.analytics-trend .analytics-line { height: 300px; }
.analytics-model-link { display: inline-flex; align-items: center; gap: 8px; min-height: 44px; padding: 0; text-align: left; color: rgb(var(--ink)); text-decoration: underline; text-decoration-color: rgb(var(--line-strong)); text-underline-offset: 4px; }
.analytics-model-link:hover { color: rgb(var(--text-link)); }
.analytics-model-link > span { color: rgb(var(--ink-2)); flex-shrink: 0; }
.analytics-drilldown { display: flex; align-items: center; gap: 8px; max-width: 100%; }
.analytics-drilldown select { min-width: 0; min-height: 44px; padding: 8px 10px; border: 1px solid rgb(var(--line)); border-radius: 12px; background: rgb(var(--surface)); color: rgb(var(--ink)); font-size: 14px; }
.analytics-drilldown button { min-height: 44px; padding: 8px 12px; border: 1px solid rgb(var(--line)); border-radius: 999px; color: rgb(var(--ink)); white-space: nowrap; }
.analytics-panel { container-type: inline-size; border: 1px solid rgb(var(--line)); border-radius: 16px; padding: 20px; min-width: 0; background: rgb(var(--surface)); }
.analytics-panel header { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; margin-bottom: 20px; }
.analytics-panel h3 { display: flex; align-items: center; gap: 8px; font-size: 15px; font-weight: 600; margin: 0; }
.analytics-panel h3 svg { color: rgb(var(--ink-2)); }
.analytics-segment { display: flex; background: rgb(var(--surface-2)); border-radius: 10px; padding: 3px; }
.analytics-segment button { min-height: 44px; padding: 7px 10px; border: 0; border-radius: 7px; font-size: 13px; background: transparent; color: rgb(var(--ink-2)); white-space: nowrap; }
.analytics-segment button[aria-pressed='true'] { background: rgb(var(--surface)); color: rgb(var(--ink)); box-shadow: var(--shadow-panel); }
.analytics-distribution { display: flex; align-items: center; gap: 18px; min-height: 210px; }
.analytics-doughnut { width: 136px; height: 136px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; }
.analytics-zero { color: rgb(var(--ink-2)); font-size: 13px; text-align: center; border: 14px solid rgb(var(--surface-2)); width: 136px; height: 136px; border-radius: 50%; display: flex; align-items: center; justify-content: center; }
.analytics-table-scroll { min-width: 0; flex: 1; max-height: 264px; overflow: auto; }
.analytics-table-scroll table { border-collapse: collapse; width: 100%; font-size: 14px; white-space: nowrap; font-variant-numeric: tabular-nums; }
.analytics-table-scroll th { font-weight: 500; color: rgb(var(--ink-2)); text-align: right; padding: 0 10px 10px; }
.analytics-table-scroll td { padding: 12px 10px; border-top: 1px solid rgb(var(--line)); text-align: right; color: rgb(var(--ink-2)); }
.analytics-table-scroll td:first-child, .analytics-table-scroll th:first-child { text-align: left; padding-left: 0; }
.analytics-row-label { display: flex; gap: 6px; align-items: center; color: rgb(var(--ink)); max-width: 180px; }
.analytics-row-label > span { overflow: hidden; text-overflow: ellipsis; }
.analytics-row-label i { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.analytics-row-label :deep(svg) { flex-shrink: 0; }
.analytics-table-scroll .analytics-actual { color: var(--chart-1); }
.analytics-state { min-height: 210px; display: flex; align-items: center; justify-content: center; text-align: center; color: rgb(var(--ink-2)); }
.analytics-error { color: rgb(var(--text-err)); }
.analytics-line { height: 260px; min-width: 0; }
.analytics-chart-note { font-size: 13px; line-height: 1.6; }
.portal-usage-analytics :is(button, input, select):focus-visible { outline: 2px solid rgb(var(--focus-ring)); outline-offset: 3px; }
.portal-usage-analytics button { transition: background-color var(--motion-fast) var(--motion-ease), color var(--motion-fast) var(--motion-ease); }
.analytics-refresh:hover:not(:disabled) { background: rgb(var(--surface-2)); }
.analytics-apply:hover { background: rgb(var(--action-hover)); }
@container (max-width: 550px) { .analytics-distribution { flex-direction: column; align-items: stretch; } .analytics-doughnut { align-self: center; } .analytics-table-scroll { width: 100%; } }
@media (min-width: 768px) and (max-width: 1199px) { .analytics-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 767px) { .analytics-summary { grid-template-columns: repeat(auto-fit, minmax(min(100%, 152px), 1fr)); } .analytics-heading { align-items: flex-start; } .analytics-heading h2 { font-size: 22px; } .analytics-grid { grid-template-columns: 1fr; } .analytics-panel { padding: 16px; } .analytics-summary article { gap: 8px; padding: 18px 12px; } .analytics-summary .analytics-metric-icon { display: none; } .analytics-controls { display: grid; grid-template-columns: minmax(0, 1fr); } .analytics-controls label { display: grid; grid-template-columns: 76px minmax(0, 1fr); } .analytics-controls select, .analytics-controls input { width: 100%; font-size: 16px; } .analytics-granularity { margin-left: 0; } .analytics-apply { width: 100%; } .analytics-line { height: 260px; } }
@media (prefers-reduced-motion: reduce) { .portal-usage-analytics button { transition: none; } }
</style>
