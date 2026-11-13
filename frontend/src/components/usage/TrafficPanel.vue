<!-- Adapted from ranxi2001/sub2api @ 9620d315, licensed under LGPL-3.0-or-later. -->
<template>
  <section class="traffic-panel" :aria-label="t('traffic.title')" :aria-busy="loading">
    <div class="traffic-toolbar">
      <form class="traffic-filter-form" @submit.prevent="applyFilters">
        <div><label for="traffic-period" class="input-label">{{ t('traffic.period') }}</label><select id="traffic-period" v-model="period" class="input"><option v-for="value in periods" :key="value" :value="value">{{ t(`traffic.periods.${value}`) }}</option></select></div>
        <div><label for="traffic-date" class="input-label">{{ t('traffic.date') }}</label><input id="traffic-date" :value="date || currentDate" type="date" class="input" @change="setDate" /></div>
        <UsageFilters v-if="admin" :key="searchResetKey" v-model="searchFilters" :api-key-search="searchAdminKeys" @change="applyFilters" />
        <div v-else class="traffic-key-filter">
          <label for="traffic-key" class="input-label">{{ t('usage.apiKeyFilter') }}</label>
          <Select :key="searchResetKey" id="traffic-key" v-model="searchFilters.api_key_id" :options="ownKeyOptions" searchable remote :loading="ownKeyLoading" @click="onOwnKeyOpen" @keydown.down="onOwnKeyOpen" @keydown.up="onOwnKeyOpen" @search="searchOwnKeys" @change="selectOwnKey"><template #selected="{ option }">{{ searchFilters.api_key_id ? selectedKeyLabel : option?.label || t('usage.allApiKeys') }}</template></Select>
          <p v-if="ownKeyError" role="alert" class="traffic-error">{{ t('traffic.keysFailed') }} <button type="button" class="underline" @click="searchOwnKeys(ownKeyQuery)">{{ t('common.refresh') }}</button></p>
        </div>
        <div class="traffic-filter-actions">
          <button class="btn btn-primary" type="submit" :disabled="loading"><Icon name="search" size="sm" />{{ t('traffic.apply') }}</button>
          <button class="btn btn-secondary" type="button" :disabled="loading" @click="load"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button>
          <button class="traffic-reset" type="button" @click="reset">{{ t('common.reset') }}</button>
        </div>
      </form>
      <div class="traffic-toolbar-footer"><p>{{ t('traffic.keyHistoryHint') }}</p><span class="traffic-scope-badge"><Icon name="shield" size="xs" />{{ t('traffic.noCharge') }}</span></div>
    </div>

    <div class="traffic-range-row">
      <div><span class="traffic-section-label">{{ t('traffic.overview') }}</span><p v-if="report">{{ rangeLabel }}<span class="traffic-timezone">{{ report.timezone }}</span></p></div>
      <div class="traffic-period-navigation"><button class="btn btn-secondary btn-sm" type="button" :disabled="loading || !report" @click="shift(-1)"><Icon name="chevronLeft" size="sm" /><span>{{ t('traffic.previousPeriod') }}</span></button><button class="btn btn-secondary btn-sm" type="button" :disabled="loading || !report" @click="shift(1)"><span>{{ t('traffic.nextPeriod') }}</span><Icon name="chevronRight" size="sm" /></button></div>
    </div>
    <p v-if="error" role="alert" class="traffic-error">{{ error }}</p>
    <p v-if="summaryLoading" role="status" class="traffic-loading">{{ t('traffic.loading') }}</p>

    <template v-if="summary">
      <div class="traffic-metrics">
        <article v-for="card in cards" :key="card.label" class="traffic-metric" :class="card.tone">
          <div class="traffic-metric-top"><span>{{ card.label }}</span><span class="traffic-metric-icon"><Icon :name="card.icon" size="sm" /></span></div>
          <strong :title="card.raw">{{ card.value }}</strong><span class="traffic-metric-hint">{{ card.hint }}</span>
        </article>
      </div>
      <div v-if="summary.requests" class="traffic-analysis">
        <article class="traffic-chart-card">
          <header class="traffic-card-heading"><div><h3>{{ t('traffic.trend') }}</h3><p>{{ t(`traffic.trendScopes.${trendBucket}`) }}</p></div><span class="traffic-card-tag">{{ t(`traffic.periods.${summary.period}`) }}</span></header>
          <div class="traffic-trend-chart" role="img" :aria-label="t('traffic.trend')"><Bar :data="chartData" :options="chartOptions" /></div>
        </article>
        <article class="traffic-chart-card traffic-composition">
          <header class="traffic-card-heading"><h3>{{ t('traffic.composition') }}</h3></header>
          <div class="traffic-donut" role="img" :aria-label="t('traffic.composition')"><Doughnut :data="directionData" :options="directionOptions" /><div aria-hidden="true"><strong>{{ bytes(summary.total_bytes) }}</strong><span>{{ t('traffic.total') }}</span></div></div>
          <div class="traffic-direction-legend"><span><i :style="{ background: palette[0] }" />{{ t('traffic.upload') }}</span><strong>{{ share(summary.upload_bytes) }}</strong><span><i :style="{ background: palette[2] }" />{{ t('traffic.download') }}</span><strong>{{ share(summary.download_bytes) }}</strong></div>
          <dl class="traffic-insights"><div><dt>{{ t('traffic.average') }}</dt><dd>{{ bytes(summary.requests ? Math.round(summary.total_bytes / summary.requests) : 0) }}</dd></div><div><dt>{{ t('traffic.peakBucket') }}</dt><dd>{{ peakBucket ? bytes(peakBucket.total_bytes) : '—' }}<small v-if="peakBucket">{{ peakBucket.bucket }}</small></dd></div></dl>
        </article>
      </div>
      <div v-else class="traffic-empty"><Icon name="chart" size="lg" /><h3>{{ t('traffic.empty') }}</h3><p>{{ t('traffic.emptyHint') }}</p></div>

      <details v-if="summary.requests" class="traffic-trend-details">
        <summary><Icon name="chart" size="sm" />{{ t('traffic.trendDetails') }}</summary>
        <div class="traffic-table-scroll"><table class="traffic-table"><caption class="sr-only">{{ t('traffic.trend') }} · {{ t('traffic.upload') }} / {{ t('traffic.download') }}</caption>
          <thead><tr><th scope="col">{{ t('traffic.bucket') }}</th><th scope="col">{{ t('traffic.upload') }}</th><th scope="col">{{ t('traffic.download') }}</th><th scope="col">{{ t('traffic.total') }}</th></tr></thead>
          <tbody><tr v-for="point in summary.trend" :key="point.bucket"><td>{{ point.bucket }}</td><td v-for="direction in directions" :key="direction" :title="`${point[direction]} B`">{{ bytes(point[direction]) }}</td><td>{{ bytes(point.total_bytes) }}</td></tr></tbody>
        </table></div>
      </details>
    </template>

    <section class="traffic-records-card" :aria-label="t('traffic.records')">
      <header class="traffic-card-heading"><div><h3>{{ t('traffic.records') }}</h3><p>{{ t('traffic.recordsHint') }}</p></div><span v-if="details" class="traffic-card-tag">{{ t('traffic.page', { page }) }}</span></header>
      <p v-if="detailsError" role="alert" class="traffic-error">{{ detailsError }} <button type="button" class="underline" @click="loadDetails(requestedPage)">{{ t('common.refresh') }}</button></p>
      <p v-if="detailsLoading" role="status" class="traffic-loading">{{ t('traffic.loadingRecords') }}</p>
      <template v-if="details && !detailsLoading">
        <div class="traffic-table-scroll"><table class="traffic-table traffic-records-table">
          <caption class="sr-only">{{ t('traffic.records') }}</caption>
          <thead><tr><th scope="col">{{ t('usage.time') }}</th><th v-if="admin" scope="col">{{ t('traffic.user') }}</th><th scope="col">API Key</th><th scope="col">{{ t('traffic.endpoint') }}</th><th scope="col">{{ t('traffic.upload') }}</th><th scope="col">{{ t('traffic.download') }}</th><th scope="col">{{ t('traffic.result') }}</th></tr></thead>
          <tbody>
            <tr v-for="item in details.items" :key="item.event_id">
              <td class="traffic-timestamp">{{ displayTime(item.started_at) }}</td><td v-if="admin">#{{ item.user_id }}</td><td><span class="traffic-key-id"><Icon name="key" size="xs" />#{{ item.api_key_id }}</span></td>
              <td><div class="traffic-endpoint"><span class="traffic-method">{{ item.method }}</span><code>{{ item.endpoint }}</code></div><span class="traffic-request-id" :title="item.request_id || item.event_id">{{ item.request_id || item.event_id }}</span></td>
              <td :title="`${item.upload_bytes} B`">{{ bytes(item.upload_bytes) }}<small v-if="!item.request_complete" class="traffic-partial">{{ t('traffic.partialRead') }}</small></td><td :title="`${item.download_bytes} B`">{{ bytes(item.download_bytes) }}</td>
              <td><span class="traffic-status" :class="item.outcome === 'completed' ? 'is-complete' : item.outcome === 'canceled' ? 'is-canceled' : 'is-error'">{{ item.status_code ? 'HTTP ' + item.status_code : '—' }}</span><small>{{ t(`traffic.outcomes.${item.outcome}`) }}</small></td>
            </tr>
            <tr v-if="!details.items.length"><td :colspan="admin ? 7 : 6" class="traffic-table-empty">{{ t('traffic.empty') }}</td></tr>
          </tbody>
        </table></div>
        <footer class="traffic-pagination"><span>{{ t('traffic.page', { page }) }}</span><div><button type="button" class="btn btn-secondary btn-sm" :disabled="page <= 1 || detailsLoading" @click="previousPage">{{ t('traffic.previous') }}</button><button type="button" class="btn btn-secondary btn-sm" :disabled="!details.next_cursor || detailsLoading" @click="nextPage">{{ t('traffic.next') }}</button></div></footer>
      </template>
    </section>

    <details class="traffic-measurement-notes"><summary><Icon name="infoCircle" size="sm" />{{ t('traffic.scopeTitle') }}<span>{{ t('traffic.scopeShort') }}</span></summary><div><p>{{ t('traffic.scope') }}</p><p v-if="summary">{{ summary.recorded_since ? t('traffic.since', { time: displayTime(summary.recorded_since) }) : t('traffic.notRecorded') }} {{ t('traffic.history') }}</p></div></details>
    <details v-if="admin" class="traffic-node-health" :open="healthError || health?.status === 'degraded' || health?.status === 'delayed'" :class="{ 'needs-attention': healthError || (health && health.status !== 'ok') }" data-testid="traffic-health">
      <summary>{{ t('traffic.nodeHealth') }} · {{ health ? t(`traffic.healthStatuses.${health.status}`) : t('common.loading') }}</summary>
      <div v-if="health"><p>{{ t('traffic.queueSummary', { queued: health.queued_events ?? 0, capacity: health.queue_capacity ?? 0, processing: health.processing_events ?? 0, dropped: health.dropped_events ?? 0 }) }}</p><p>{{ t('traffic.healthSummary', { pending: health.pending_events, age: health.oldest_pending_age_seconds, quarantined: health.quarantined_events, lost: health.persistence_failures }) }}</p><p v-if="health.storage_available === false" role="alert">{{ t('traffic.storageUnavailable') }}</p><p>{{ t('traffic.nodeHealthScope') }} · {{ displayTime(health.checked_at) }}</p></div>
      <p v-if="healthError" role="alert">{{ t('traffic.healthFailed') }}</p>
    </details>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, CategoryScale, LinearScale, BarElement, ArcElement, Tooltip, Legend, type ChartOptions } from 'chart.js'
import { Bar, Doughnut } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import UsageFilters from '@/components/usage/TrafficIdentityFilters.vue'
import trafficZh from '@/i18n/locales/zh/traffic'
import trafficEn from '@/i18n/locales/en/traffic'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { getTraffic, getTrafficKeys, getTrafficHealth, formatTrafficBytes as bytes, type TrafficPeriod, type TrafficReport, type TrafficKeyOption, type TrafficDeliveryHealth } from '@/api/traffic'

const props = withDefaults(defineProps<{ admin?: boolean; apiKeyOptions?: SelectOption[] }>(), { admin: false, apiKeyOptions: () => [] })
const { t } = useI18n({ useScope: 'local', messages: { zh: { traffic: trafficZh }, en: { traffic: trafficEn } } })
ChartJS.register(CategoryScale, LinearScale, BarElement, ArcElement, Tooltip, Legend)
const palette = useChartPalette()
const chartTheme = useChartTheme()
const periods: TrafficPeriod[] = ['day', 'week', 'month', 'year']
const directions = ['upload_bytes', 'download_bytes'] as const
const period = ref<TrafficPeriod>('month'), date = ref(''), page = ref(1)
const searchFilters = ref<{ user_id?: number; api_key_id?: number | null }>({ api_key_id: null })
const searchResetKey = ref(0)
const filters = ref<{ user_id?: number; api_key_id?: number }>({})
const summary = ref<TrafficReport | null>(null), details = ref<TrafficReport | null>(null)
const report = computed(() => details.value || summary.value)
const summaryLoading = ref(false), detailsLoading = ref(false), error = ref(''), detailsError = ref('')
const loading = computed(() => summaryLoading.value || detailsLoading.value)
let cursors: Array<string | undefined> = [undefined]
let queryDate: string | undefined
const requestedPage = ref(1)
const health = ref<TrafficDeliveryHealth | null>(null), healthError = ref(false)
const ownKeyResults = ref<TrafficKeyOption[]>([]), ownKeyQuery = ref(''), ownKeyLoading = ref(false), ownKeyError = ref(false), ownKeyLabel = ref('')
let keyController: AbortController | undefined
let controller: AbortController | undefined
let detailsController: AbortController | undefined
const ownKeyOptions = computed<SelectOption[]>(() => {
  const options = new Map<string | number | boolean | null, SelectOption>()
  if (!ownKeyQuery.value) for (const option of props.apiKeyOptions) options.set(option.value, option)
  for (const key of ownKeyResults.value) options.set(key.id, { value: key.id, label: `${key.name || `#${key.id}`}${key.deleted ? ` (${t('traffic.deletedKey')})` : ''}` })
  options.delete(null)
  return [{ value: null, label: t('usage.allApiKeys') }, ...options.values()]
})
const selectedKeyLabel = computed(() => ownKeyLabel.value || ownKeyOptions.value.find(option => option.value === searchFilters.value.api_key_id)?.label || `#${searchFilters.value.api_key_id}`)
const searchAdminKeys = (userId: number | undefined, query: string) => getTrafficKeys(true, userId, query)
async function searchOwnKeys(query = '') {
  keyController?.abort()
  const current = new AbortController(); keyController = current
  ownKeyQuery.value = query; ownKeyLoading.value = true; ownKeyError.value = false; ownKeyResults.value = []
  try {
    const result = await getTrafficKeys(false, undefined, query, current.signal)
    if (!current.signal.aborted) ownKeyResults.value = result
  } catch {
    if (!current.signal.aborted) ownKeyError.value = true
  } finally { if (!current.signal.aborted) ownKeyLoading.value = false }
}
function selectOwnKey(_value: unknown, option: SelectOption | null) { ownKeyLabel.value = option?.label || ''; applyFilters() }
function onOwnKeyOpen(event: Event) {
  if (!(event.target instanceof Element) || !event.target.closest('#traffic-key')) return
  if (ownKeyQuery.value) void searchOwnKeys()
}
const siteDate = (value: string | Date) => {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: report.value?.timezone || 'UTC', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date(value))
  const get = (type: string) => parts.find(p => p.type === type)?.value || ''
  return `${get('year')}-${get('month')}-${get('day')}`
}
const currentDate = computed(() => report.value ? siteDate(new Date()) : '')
const displayTime = (value: string) => new Intl.DateTimeFormat(undefined, { timeZone: report.value?.timezone || 'UTC', dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value))
const rangeLabel = computed(() => report.value ? `${siteDate(report.value.start)} — ${siteDate(new Date(new Date(report.value.end).getTime() - 1))}` : '')
const cards = computed(() => summary.value ? [
  { label: t('traffic.total'), value: bytes(summary.value.total_bytes), raw: `${summary.value.total_bytes} B`, icon: 'chart' as const, tone: 'total', hint: t('traffic.totalHint') },
  { label: t('traffic.upload'), value: bytes(summary.value.upload_bytes), raw: `${summary.value.upload_bytes} B`, icon: 'arrowUp' as const, tone: 'upload', hint: t('traffic.uploadHint') },
  { label: t('traffic.download'), value: bytes(summary.value.download_bytes), raw: `${summary.value.download_bytes} B`, icon: 'arrowDown' as const, tone: 'download', hint: t('traffic.downloadHint') },
  { label: t('traffic.requests'), value: summary.value.requests.toLocaleString(), raw: String(summary.value.requests), icon: 'bolt' as const, tone: 'requests', hint: t('traffic.requestsHint') }
] : [])
const peakBucket = computed(() => summary.value?.trend.reduce<(TrafficReport['trend'][number]) | null>((peak, point) => !peak || point.total_bytes > peak.total_bytes ? point : peak, null))
const share = (value: number) => summary.value?.total_bytes ? `${(value / summary.value.total_bytes * 100).toFixed(1)}%` : '0%'
const directionData = computed(() => ({ labels: [t('traffic.upload'), t('traffic.download')], datasets: [{ data: [summary.value?.upload_bytes ?? 0, summary.value?.download_bytes ?? 0], backgroundColor: [palette.value[0], palette.value[2]], borderWidth: 0, hoverOffset: 3 }] }))
const directionOptions = computed<ChartOptions<'doughnut'>>(() => ({ responsive: true, maintainAspectRatio: false, cutout: '80%', animation: false, plugins: { legend: { display: false }, tooltip: { ...chartTheme.value.tooltip, callbacks: { label: context => `${context.label}: ${bytes(Number(context.raw))}` } } } }))
const trendBucket = computed(() => summary.value?.period === 'day' ? 'hour' : summary.value?.period === 'year' ? 'month' : 'day')
const chartData = computed(() => ({
  labels: summary.value?.trend.map(point => point.bucket) || [],
  datasets: [
    { label: t('traffic.upload'), data: summary.value?.trend.map(point => point.upload_bytes) || [], backgroundColor: palette.value[0], borderColor: chartTheme.value.pointBorder, borderWidth: 1.5, borderRadius: 4 },
    { label: t('traffic.download'), data: summary.value?.trend.map(point => point.download_bytes) || [], backgroundColor: palette.value[2], borderColor: chartTheme.value.pointBorder, borderWidth: 1.5, borderRadius: 4 }
  ]
}))
const chartOptions = computed<ChartOptions<'bar'>>(() => ({
  responsive: true, maintainAspectRatio: false, animation: false,
  plugins: {
    legend: { position: 'bottom', labels: { color: chartTheme.value.text, usePointStyle: true, pointStyle: 'circle', boxWidth: 8, padding: 14, font: { size: 12.5 } } },
    tooltip: { ...chartTheme.value.tooltip, callbacks: { label: context => `${context.dataset.label}: ${bytes(Number(context.raw))}` } },
  },
  scales: {
    x: { stacked: true, border: { display: false }, grid: { display: false }, ticks: { color: chartTheme.value.text, maxRotation: 0, maxTicksLimit: 7, font: { size: 12 } } },
    y: { stacked: true, beginAtZero: true, border: { display: false }, grid: { color: chartTheme.value.grid }, ticks: { color: chartTheme.value.text, font: { size: 12 }, callback: value => bytes(Number(value)) } },
  }
}))
function setDate(event: Event) { date.value = (event.target as HTMLInputElement).value }
function applyFilters() {
  const user = searchFilters.value.user_id, key = searchFilters.value.api_key_id
  filters.value = { ...(props.admin && user ? { user_id: user } : {}), ...(key ? { api_key_id: key } : {}) }
}
function reset() {
  searchFilters.value = { api_key_id: null }; searchResetKey.value++; date.value = ''; period.value = 'month'; filters.value = {}; ownKeyLabel.value = ''
  if (!props.admin) void searchOwnKeys()
}
function shift(direction: number) {
  if (!report.value) return
  const anchor = new Date(`${siteDate(report.value.start)}T00:00:00Z`)
  if (period.value === 'year') anchor.setUTCFullYear(anchor.getUTCFullYear() + direction)
  else if (period.value === 'month') anchor.setUTCMonth(anchor.getUTCMonth() + direction)
  else anchor.setUTCDate(anchor.getUTCDate() + direction * (period.value === 'week' ? 7 : 1))
  date.value = anchor.toISOString().slice(0, 10)
}
function nextPage() {
  if (!details.value?.next_cursor || detailsLoading.value) return
  cursors[page.value] = details.value.next_cursor
  cursors.length = page.value + 1
  void loadDetails(page.value + 1)
}
function previousPage() { if (page.value > 1 && !detailsLoading.value) void loadDetails(page.value - 1) }
async function loadDetails(targetPage: number) {
  detailsController?.abort()
  const current = new AbortController(); detailsController = current
  requestedPage.value = targetPage
  detailsLoading.value = true; detailsError.value = ''; details.value = null
  try {
    const result = await getTraffic(props.admin, { period: period.value, date: queryDate, view: 'details', cursor: cursors[targetPage - 1], page_size: 20, ...filters.value }, current.signal)
    if (!current.signal.aborted) {
      details.value = result; page.value = targetPage
      // Keep subsequent pages in the same calendar period across midnight.
      queryDate = siteDate(result.start)
      if (summary.value && summary.value.start !== result.start && controller) void loadSummary(controller)
    }
  } catch {
    if (!current.signal.aborted) detailsError.value = t('traffic.recordsFailed')
  } finally { if (!current.signal.aborted) detailsLoading.value = false }
}
async function load() {
  controller?.abort(); detailsController?.abort()
  const current = new AbortController(); controller = current
  cursors = [undefined]; page.value = 1; queryDate = date.value || undefined
  summaryLoading.value = true; error.value = ''; summary.value = null
  if (props.admin) void loadHealth(current)
  // Details remain usable while the more expensive summary is loading/failing.
  void loadDetails(1)
  void loadSummary(current)
}
async function loadSummary(current: AbortController) {
  summaryLoading.value = true; summary.value = null; error.value = ''
  try {
    let result = await getTraffic(props.admin, { period: period.value, date: queryDate, view: 'summary', page_size: 20, ...filters.value }, current.signal)
    if (current.signal.aborted) return
    // Initial requests may straddle midnight. Align the summary with the
    // details' resolved site-calendar period rather than mixing two periods.
    if (details.value && details.value.start !== result.start) {
      result = await getTraffic(props.admin, { period: period.value, date: queryDate, view: 'summary', page_size: 20, ...filters.value }, current.signal)
    }
    if (!current.signal.aborted) summary.value = result
  } catch {
    if (!current.signal.aborted) error.value = t('traffic.failed')
  } finally { if (!current.signal.aborted) summaryLoading.value = false }
}
async function loadHealth(current: AbortController) {
  try {
    const result = await getTrafficHealth(current.signal)
    if (!current.signal.aborted) { health.value = result; healthError.value = false }
  } catch {
    if (!current.signal.aborted) { health.value = null; healthError.value = true }
  }
}
watch([period, date, filters], () => { void load() }, { deep: true })
void load()
if (!props.admin) void searchOwnKeys()
onUnmounted(() => { controller?.abort(); detailsController?.abort(); keyController?.abort() })
</script>

<style scoped src="./TrafficPanel.css"></style>
