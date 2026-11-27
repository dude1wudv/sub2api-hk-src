<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler, type ChartOptions } from 'chart.js'
import { Doughnut, Line } from 'vue-chartjs'
import { usageAPI } from '@/api'
import type { TrendParams } from '@/api/usage'
import type { UsageStatsResponse, ModelStat, GroupStat, TrendDataPoint } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import type { UsageDrilldown } from '@/views/portal/usageRoute'
import { validUsageDate } from '@/views/portal/usageRoute'
import { distributionValue, rankChartRows } from '@/utils/chartDistribution'
import { formatUsageMoney } from '@/utils/portalUsage'
import { BrandModelIcon, UiButton, UiSegmented, UiSelect, type UiSelectOption } from '@/ui'

ChartJS.register(ArcElement, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)
const { locale, t } = useI18n()
const props = defineProps<{ filters?: TrendParams }>()
const emit = defineEmits<{ drilldown: [value: UsageDrilldown]; rangeChange: [value: { start: string; end: string }] }>()
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
const money = (value: number) => formatUsageMoney(value, 4)
const duration = (value: number) => Number.isFinite(value) ? `${(value / 1000).toFixed(2)}s` : '—'
const selectedDay = ref('')
const availableDays = computed(() => [...new Set(trend.value.map(item => item.date.slice(0, 10)).filter(validUsageDate))])

const presetOptions = computed<UiSelectOption[]>(() => [
  { value: 'month', label: tx('本月', 'This month') },
  { value: 'today', label: tx('今天', 'Today') },
  { value: '7', label: tx('近 7 天', 'Last 7 days') },
  { value: '30', label: tx('近 30 天', 'Last 30 days') },
  { value: 'custom', label: tx('自定义', 'Custom') },
])
const granularityOptions = computed<UiSelectOption[]>(() => [
  { value: 'day', label: tx('按天', 'Daily') },
  { value: 'hour', label: tx('按小时', 'Hourly') },
])
const daySelectOptions = computed<UiSelectOption[]>(() =>
  availableDays.value.map(day => ({ value: day, label: day }))
)
const metricSegmentedOptions = computed(() => [
  { value: 'tokens' as const, label: 'Token' },
  { value: 'actual_cost' as const, label: tx('实际消费', 'Actual cost') },
])

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
  if (props.filters?.start_date && props.filters.end_date
      && (start.value !== props.filters.start_date || end.value !== props.filters.end_date)) {
    emit('rangeChange', { start: start.value, end: end.value })
    return
  }
  applied.value = { start: start.value, end: end.value }
  const sequence = ++requestSequence
  loading.value = true
  errors.value = { stats: false, models: false, snapshot: false }
  stats.value = null
  models.value = []; groups.value = []; trend.value = []
  const params = { ...props.filters, start_date: start.value, end_date: end.value, timezone }
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

function changePreset(value: string | number | boolean | null) {
  preset.value = String(value ?? 'month')
  if (preset.value === 'custom') return
  const today = new Date()
  const first = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  if (preset.value === 'month') first.setDate(1)
  else if (preset.value === '7' || preset.value === '30') first.setDate(first.getDate() - Number(preset.value) + 1)
  start.value = dateText(first); end.value = dateText(today)
  granularity.value = preset.value === 'today' ? 'hour' : 'day'
  void load()
}

function handleGranularityChange(value: string | number | boolean | null) {
  granularity.value = value === 'hour' ? 'hour' : 'day'
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
  return {
    labels: rows.map(row => row.name),
    datasets: [{
      data: rows.map(row => distributionValue(row, metrics.value[id])),
      backgroundColor: rows.map(row => row.chartColor),
      borderColor: chartTheme.value.pointBorder,
      borderWidth: rows.length > 1 ? 2 : 0,
      hoverOffset: 5,
    }],
  }
}
function hasDistribution(id: string, rows: DistributionRow[]) {
  return rows.some(row => distributionValue(row, metrics.value[id]) > 0)
}
const doughnutOptions = computed<ChartOptions<'doughnut'>>(() => ({ responsive: true, maintainAspectRatio: false, animation: false, cutout: '74%', plugins: { legend: { display: false }, tooltip: { ...chartTheme.value.tooltip, callbacks: { label: context => `${context.label}: ${Number(context.raw).toLocaleString(undefined, { maximumFractionDigits: 6 })}` } } } }))
function distributionOptions(id: string, rows: DistributionRow[]): ChartOptions<'doughnut'> {
  return { ...doughnutOptions.value, plugins: { ...doughnutOptions.value.plugins, tooltip: { ...chartTheme.value.tooltip, callbacks: {
    label: context => `${context.label}: ${metrics.value[id] === 'actual_cost' ? money(Number(context.raw)) : number(Number(context.raw))}`,
  } } }, onClick: (_event, items) => {
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
  plugins: { legend: { position: 'bottom', labels: { color: chartTheme.value.text, usePointStyle: true, boxWidth: 8, font: { size: 12 } } }, tooltip: { ...chartTheme.value.tooltip, callbacks: {
    label: context => `${context.dataset.label}: ${context.dataset.yAxisID === 'percentage' ? `${Number(context.raw).toFixed(1)}%` : number(Number(context.raw))}`,
    footer: items => { const item = trend.value[items[0]?.dataIndex ?? -1]; return item ? `${tx('实际', 'Actual')} ${money(item.actual_cost)} · ${tx('标准', 'Standard')} ${money(item.cost)}` : '' },
  } } },
  scales: {
    x: { grid: { display: false }, ticks: { color: chartTheme.value.text, maxTicksLimit: 8, font: { size: 12 } } },
    y: { beginAtZero: true, grid: { color: chartTheme.value.grid }, ticks: { color: chartTheme.value.text, callback: value => number(Number(value)), font: { size: 12 } } },
    percentage: { position: 'right', min: 0, max: 100, grid: { drawOnChartArea: false }, ticks: { color: colors.value[3], callback: value => `${value}%`, font: { size: 12 } } },
  },
}))
watch(() => props.filters, filters => {
  if (filters?.start_date && filters.end_date
      && (start.value !== filters.start_date || end.value !== filters.end_date)) {
    start.value = filters.start_date
    end.value = filters.end_date
    preset.value = 'custom'
  }
  void load()
}, { deep: true, immediate: true })
onBeforeUnmount(() => { requestSequence++ })
</script>

<template>
  <section class="space-y-5" :aria-label="tx('用量分析', 'Usage analytics')" :aria-busy="loading">
    <p v-if="errors.stats" role="alert" class="rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]">
      {{ tx('汇总与端点统计暂时无法加载，请重试。', 'Summary and endpoint statistics could not be loaded. Please retry.') }}
    </p>

    <!-- Borderless Horizontal Metric Ribbon -->
    <div class="border-b border-[var(--atelier-border)] pb-5">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="text-xs font-medium text-[var(--atelier-ink-muted)]">{{ tx('实际消费', 'Actual cost') }}</p>
          <div class="mt-1 flex flex-wrap items-baseline gap-2.5">
            <strong
              data-testid="analytics-actual-cost"
              class="font-mono text-3xl font-semibold tracking-tight tabular-nums text-[var(--atelier-ink)]"
            >
              {{ stats ? money(stats.total_actual_cost) : '—' }}
            </strong>
            <span v-if="stats" class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
              {{ tx('标准费用', 'Standard cost') }} <s>{{ money(stats.total_cost) }}</s>
            </span>
          </div>
        </div>

        <div class="flex flex-wrap items-start gap-x-6 gap-y-3 lg:flex-nowrap">
          <div class="min-w-[115px]">
            <p class="text-xs text-[var(--atelier-ink-muted)]">{{ tx('总请求数', 'Total requests') }}</p>
            <strong
              data-testid="analytics-requests"
              class="mt-1 block font-mono text-lg font-semibold tabular-nums text-[var(--atelier-ink)]"
            >
              {{ stats ? number(stats.total_requests) : '—' }}
            </strong>
            <p class="mt-0.5 text-[11px] text-[var(--atelier-ink-muted)]">{{ tx('所选分析时间段内', 'In the analysis period') }}</p>
          </div>

          <div class="hidden h-10 w-px bg-[var(--atelier-border)] sm:block" aria-hidden="true" />

          <div class="min-w-[170px]">
            <p class="text-xs text-[var(--atelier-ink-muted)]">{{ tx('总 Token', 'Total tokens') }}</p>
            <strong class="mt-1 block font-mono text-lg font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ stats ? number(stats.total_tokens) : '—' }}
            </strong>
            <p v-if="stats" class="mt-0.5 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
              {{ tx('输入', 'In') }} {{ number(stats.total_input_tokens) }} · {{ tx('输出', 'Out') }} {{ number(stats.total_output_tokens) }}
            </p>
            <p v-if="stats" class="font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
              {{ tx('缓存写', 'Cache W') }} {{ number(stats.total_cache_creation_tokens) }} · {{ tx('读', 'R') }} {{ number(stats.total_cache_read_tokens) }}
            </p>
          </div>

          <div class="hidden h-10 w-px bg-[var(--atelier-border)] sm:block" aria-hidden="true" />

          <div class="min-w-[110px]">
            <p class="text-xs text-[var(--atelier-ink-muted)]">{{ tx('平均耗时', 'Average duration') }}</p>
            <strong class="mt-1 block font-mono text-lg font-semibold tabular-nums text-[var(--atelier-ink)]">
              {{ stats ? duration(stats.average_duration_ms) : '—' }}
            </strong>
            <p class="mt-0.5 text-[11px] text-[var(--atelier-ink-muted)]">{{ tx('单次请求平均耗时', 'Avg request latency') }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Period & Granularity Controls -->
    <form class="atelier-surface flex flex-wrap items-center gap-3 px-4 py-3" @submit.prevent="load">
      <div class="flex items-center gap-2 text-xs font-medium text-[var(--atelier-ink-secondary)]">
        <Icon name="calendar" size="sm" />
        <span>{{ tx('时间范围', 'Period') }}</span>
        <div class="w-32" data-testid="analytics-preset">
          <UiSelect
            :model-value="preset"
            :options="presetOptions"
            size="sm"
            :aria-label="tx('时间范围', 'Period')"
            @update:model-value="changePreset"
          />
        </div>
      </div>

      <template v-if="preset === 'custom'">
        <label class="flex items-center gap-1.5 text-xs text-[var(--atelier-ink-secondary)]">
          <span>{{ tx('开始', 'Start') }}</span>
          <input v-model="start" data-testid="analytics-start" type="date" required class="atelier-input !min-h-8 !w-auto !py-1 !text-xs" />
        </label>
        <label class="flex items-center gap-1.5 text-xs text-[var(--atelier-ink-secondary)]">
          <span>{{ tx('结束', 'End') }}</span>
          <input v-model="end" data-testid="analytics-end" type="date" required class="atelier-input !min-h-8 !w-auto !py-1 !text-xs" />
        </label>
        <UiButton variant="primary" size="sm" type="submit">{{ tx('应用', 'Apply') }}</UiButton>
      </template>

      <span class="rounded-md bg-[var(--atelier-subtle)] px-2.5 py-1 font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">
        {{ applied.start }} — {{ applied.end }}
      </span>

      <div class="ml-auto flex items-center gap-2">
        <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ tx('粒度', 'Granularity') }}</span>
        <div class="w-28" data-testid="analytics-granularity">
          <UiSelect
            :model-value="granularity"
            :options="granularityOptions"
            size="sm"
            :aria-label="tx('粒度', 'Granularity')"
            @update:model-value="handleGranularityChange"
          />
        </div>
        <UiButton
          variant="secondary"
          size="sm"
          type="button"
          :aria-label="tx('刷新用量分析', 'Refresh usage analytics')"
          :disabled="loading"
          @click="load"
        >
          <Icon name="refresh" size="sm" />
        </UiButton>
      </div>
    </form>

    <p v-if="rangeError" role="alert" class="text-xs text-[var(--atelier-err)]">{{ rangeError }}</p>
    <p class="text-xs text-[var(--atelier-ink-muted)]">
      {{ tx('统计使用下方已应用的明细筛选条件。时区：', 'Statistics use the applied record filters. Time zone: ') }}{{ timezone }}
    </p>

    <!-- Asymmetric Analytics Grid -->
    <div class="grid grid-cols-1 gap-4 xl:grid-cols-12">
      <!-- Trend Panel (wide hero chart on top/right) -->
      <article class="atelier-surface flex flex-col xl:col-span-12">
        <header class="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--atelier-border)] px-5 py-3.5">
          <h3 class="flex items-center gap-2 text-sm font-semibold text-[var(--atelier-ink)]">
            <Icon name="chart" size="sm" class="text-[var(--atelier-ink-secondary)]" />
            {{ tx('Token 使用趋势', 'Token usage trend') }}
          </h3>
          <form v-if="availableDays.length" class="flex items-center gap-2" @submit.prevent="openDate(selectedDay)">
            <div class="w-36">
              <UiSelect
                :model-value="selectedDay"
                :options="daySelectOptions"
                size="sm"
                :aria-label="tx('选择明细日期', 'Choose a date for details')"
                @update:model-value="(v) => { selectedDay = String(v ?? '') }"
              />
            </div>
            <UiButton variant="secondary" size="sm" type="submit">
              {{ tx('查看明细', 'View details') }} ↗
            </UiButton>
          </form>
        </header>

        <div class="p-5">
          <p v-if="loading" class="flex h-[240px] items-center justify-center text-sm text-[var(--atelier-ink-muted)]" role="status">
            {{ tx('正在加载统计…', 'Loading statistics…') }}
          </p>
          <p v-else-if="errors.snapshot" class="flex h-[240px] items-center justify-center text-sm text-[var(--atelier-err)]" role="alert">
            {{ tx('趋势加载失败，点击刷新重试。', 'Could not load trend. Refresh to retry.') }}
          </p>
          <p v-else-if="!trend.length" class="flex h-[240px] items-center justify-center text-sm text-[var(--atelier-ink-muted)]">
            {{ tx('所选时间段暂无记录', 'No requests in this period') }}
          </p>
          <div v-else class="h-[260px] min-w-0">
            <Line
              :data="trendData"
              :options="lineOptions"
              :aria-label="tx('输入、输出、缓存写入、缓存读取和缓存命中率趋势', 'Input, output, cache creation, cache read and cache hit rate trend')"
              role="img"
            />
          </div>
          <p class="mt-3 text-xs text-[var(--atelier-ink-muted)]">
            {{ tx('缓存命中率 = 缓存读取 /（输入 + 缓存写入 + 缓存读取）', 'Cache hit rate = cache read / (input + cache creation + cache read)') }}
          </p>
        </div>
      </article>

      <!-- Distribution Panels (Models 6 cols, Groups 6 cols, Endpoints 12 cols or 4/4/4) -->
      <article
        v-for="(section, idx) in sections"
        :key="section.id"
        :data-distribution="section.id"
        :class="[
          'atelier-surface flex min-w-0 flex-col',
          idx === 0 ? 'xl:col-span-6' : idx === 1 ? 'xl:col-span-6' : 'xl:col-span-12'
        ]"
      >
        <header class="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--atelier-border)] px-5 py-3.5">
          <h3 class="flex items-center gap-2 text-sm font-semibold text-[var(--atelier-ink)]">
            <Icon :name="section.icon" size="sm" class="text-[var(--atelier-ink-secondary)]" />
            {{ section.title }}
          </h3>
          <UiSegmented
            :model-value="metrics[section.id]"
            :options="metricSegmentedOptions"
            size="sm"
            :aria-label="section.title"
            @update:model-value="(val) => { metrics[section.id] = val }"
          />
        </header>

        <div class="p-5">
          <p v-if="loading" class="flex min-h-[200px] items-center justify-center text-sm text-[var(--atelier-ink-muted)]" role="status">
            {{ tx('正在加载统计…', 'Loading statistics…') }}
          </p>
          <p v-else-if="section.error" class="flex min-h-[200px] items-center justify-center text-sm text-[var(--atelier-err)]" role="alert">
            {{ tx('统计加载失败，点击刷新重试。', 'Could not load statistics. Refresh to retry.') }}
          </p>
          <p v-else-if="!section.rows.length" class="flex min-h-[200px] items-center justify-center text-sm text-[var(--atelier-ink-muted)]">
            {{ tx('所选时间段暂无记录', 'No requests in this period') }}
          </p>
          <div v-else class="flex flex-col items-center gap-5 sm:flex-row sm:items-center">
            <div class="flex h-36 w-36 shrink-0 items-center justify-center">
              <Doughnut
                v-if="hasDistribution(section.id, section.rows)"
                :data="distributionData(section.id, section.rows)"
                :options="distributionOptions(section.id, section.rows)"
                :aria-label="section.title"
                role="img"
              />
              <span
                v-else
                class="flex h-32 w-32 items-center justify-center rounded-full border-[10px] border-[var(--atelier-subtle)] p-2 text-center text-xs text-[var(--atelier-ink-muted)]"
              >
                {{ tx('当前指标为 0', 'Metric total is 0') }}
              </span>
            </div>

            <div class="max-h-64 w-full min-w-0 flex-1 overflow-auto">
              <table class="atelier-table text-xs">
                <thead>
                  <tr>
                    <th class="!pl-0">{{ section.column }}</th>
                    <th class="!text-right">{{ tx('请求', 'Requests') }}</th>
                    <th class="!text-right">Token</th>
                    <th class="!text-right">{{ tx('实际', 'Actual') }}</th>
                    <th class="!text-right">{{ tx('标准', 'Standard') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in section.rows" :key="row.id">
                    <td :title="row.name" class="!pl-0">
                      <span class="flex max-w-[200px] items-center gap-2 break-all text-[var(--atelier-ink)]">
                        <span class="h-2 w-2 shrink-0 rounded-full" :style="{ background: row.chartColor }" />
                        <BrandModelIcon v-if="section.id === 'models'" :model="row.name" size="sm" />
                        <button
                          v-if="section.id === 'models'"
                          type="button"
                          class="inline-flex items-center gap-1 text-left font-mono font-medium text-[var(--atelier-ink)] underline decoration-[var(--atelier-border-strong)] underline-offset-4 hover:decoration-[var(--atelier-ink)]"
                          :aria-label="tx(`查看 ${row.name} 的请求明细`, `View requests for ${row.name}`)"
                          @click="openModel(row.name)"
                        >
                          {{ row.name }} <span class="text-[var(--atelier-ink-muted)]" aria-hidden="true">↗</span>
                        </button>
                        <span v-else class="font-mono">{{ row.name }}</span>
                      </span>
                    </td>
                    <td class="text-right font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ number(row.requests) }}</td>
                    <td class="text-right font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ number(row.total_tokens) }}</td>
                    <td class="text-right font-mono font-semibold tabular-nums text-[var(--atelier-ok)]">{{ money(row.actual_cost) }}</td>
                    <td class="text-right font-mono tabular-nums text-[var(--atelier-ink-muted)]">{{ money(row.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
