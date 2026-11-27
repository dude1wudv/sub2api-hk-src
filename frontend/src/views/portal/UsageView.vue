<script setup lang="ts">
import { timezoneLabel } from '@/utils/timezoneLabel'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { keysAPI, usageAPI, userGroupsAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isUsageRequestType, requestTypeToLegacyStream } from '@/utils/usageRequestType'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'
import {
  usageMonthWindow, aggregateUsageMonths, formatUsageMoney, formatUsageNumber, isUsageAbort,
  portalUsageColumns, portalErrorColumns, usageLogsCsv, usageErrorsCsv, usageErrorRequestLabel, usageColumnLabel,
  type PortalUsageColumnKey, type PortalErrorColumnKey, type PortalUsageMonth,
} from '@/utils/portalUsage'
import type { ApiKey, Group, UsageLog, UsageQueryParams, UsageStatsResponse, UserErrorRequest, UserErrorRequestDetail, UserErrorListParams } from '@/types'
import PortalUsageChart from '@/components/portal/PortalUsageChart.vue'
import PortalUsageAnalytics from '@/components/portal/PortalUsageAnalytics.vue'
import PortalUsageTable from '@/components/portal/PortalUsageTable.vue'
import PortalUsageErrorTable from '@/components/portal/PortalUsageErrorTable.vue'
import PortalUsageDetail from '@/components/portal/PortalUsageDetail.vue'
import PortalDialog from '@/components/portal/PortalDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import InflightRequests from '@/components/usage/InflightRequests.vue'
import TrafficPanel from '@/components/usage/TrafficPanel.vue'
import { formatDateLocalInput } from '@/utils/format'
import { parseUsageRoute, usageFiltersQuery, usageDrilldownQuery, validUsageDate, type UsageDrilldown } from './usageRoute'
import { UiButton, UiCheckbox, UiSegmented, UiSelect, UiStatusDot, UiTabs, type UiSelectOption } from '@/ui'

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const auth = useAuthStore()
const app = useAppStore()
const route = useRoute()
const router = useRouter()
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
const dates = usageMonthWindow()
const sectionTab = ref<string>(route.query.tab === 'traffic' ? 'traffic' : 'usage')
const sectionTabs = computed(() => [
  { value: 'usage', label: copy('Token 用量与明细', 'Token usage & logs') },
  { value: 'traffic', label: copy('网络流量监控', 'Network traffic') },
])
const recordTypeOptions = computed(() => [
  { value: 'usage' as const, label: copy('用量记录', 'Usage logs') },
  { value: 'errors' as const, label: copy('错误请求', 'Error requests') },
])
const quickRangeOptions = computed(() => [
  { value: 'today', label: copy('今日', 'Today') },
  { value: '7d', label: copy('近 7 天', '7 days') },
  { value: 'month', label: copy('本月', 'This month') },
])
const monthStats = ref<UsageStatsResponse | null>(null)
const months = ref<PortalUsageMonth[]>([])
const overviewLoading = ref(true)
const monthError = ref('')
const trendError = ref('')
const activeTab = ref<'usage' | 'errors'>('usage')
const errorsEnabled = computed(() => app.cachedPublicSettings?.allow_user_view_error_requests === true)
const subscriptionEnabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription))
const usageRows = ref<UsageLog[]>([])
const errorRows = ref<UserErrorRequest[]>([])
const rowsLoading = ref(false)
const rowsError = ref('')
const showFilters = ref(true)
const showColumns = ref(false)
const filterError = ref('')
const optionsError = ref('')
const keys = ref<ApiKey[]>([])
const groups = ref<Array<Pick<Group, 'id' | 'name'>>>([])
const modelNames = ref<string[]>([])
const pagination = reactive({ usage: { page: 1, size: 10, total: 0 }, errors: { page: 1, size: 10, total: 0 } })
const sorting = reactive({ usage: { by: 'created_at', order: 'desc' as 'asc' | 'desc' }, errors: { by: 'created_at', order: 'desc' as 'asc' | 'desc' } })
const pageState = computed(() => pagination[activeTab.value])
const sortState = computed(() => sorting[activeTab.value])
const totalPages = computed(() => Math.max(1, Math.ceil(pageState.value.total / pageState.value.size)))
const categoryOptions = computed<Record<string, string>>(() => zh.value ? { auth: '认证', rate_limit: '速率限制', quota: '额度', invalid_request: '请求参数', service_unavailable: '服务不可用', upstream: '上游', internal: '内部错误', cyber: '安全策略' } : { auth: 'Authentication', rate_limit: 'Rate limit', quota: 'Quota', invalid_request: 'Invalid request', service_unavailable: 'Service unavailable', upstream: 'Upstream', internal: 'Internal', cyber: 'Security policy' })

const keySelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部 Key', 'All keys') },
  ...keys.value.map(key => ({ value: String(key.id), label: key.name })),
])
const groupSelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部分组', 'All groups') },
  ...groups.value.map(group => ({ value: String(group.id), label: group.name })),
])
const requestTypeSelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部类型', 'All types') },
  { value: 'sync', label: copy('同步', 'Sync') },
  { value: 'stream', label: copy('流式', 'Stream') },
  { value: 'ws_v2', label: 'WS' },
  { value: 'live', label: 'Live' },
  { value: 'cyber', label: 'Cyber' },
])
const billingTypeSelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部方式', 'All sources') },
  { value: '0', label: copy('余额', 'Balance') },
  { value: '1', label: copy('订阅', 'Subscription') },
])
const billingModeSelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部模式', 'All modes') },
  { value: 'token', label: 'Token' },
  { value: 'per_request', label: copy('按次', 'Per request') },
  { value: 'image', label: copy('图片', 'Image') },
  { value: 'video', label: copy('视频', 'Video') },
])
const compactionSelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部请求', 'All requests') },
  { value: 'true', label: copy('仅原生压缩', 'Compaction only') },
  { value: 'false', label: copy('非原生压缩', 'No compaction') },
])
const categorySelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部分类', 'All categories') },
  ...Object.entries(categoryOptions.value).map(([key, label]) => ({ value: key, label })),
])
const statusSelectOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: copy('全部状态', 'All status codes') },
  ...COMMON_ERROR_STATUS_CODES.map(code => ({ value: String(code), label: String(code) })),
])
const pageSizeOptions: UiSelectOption[] = [
  { value: 10, label: '10' },
  { value: 20, label: '20' },
  { value: 50, label: '50' },
]

function defaultFilters() {
  return { start: dates.currentStart, end: dates.end, key: '', group: '', model: '', requestType: '', billingType: '', billingMode: '', compaction: '', status: '', category: '' }
}
const draft = reactive(parseUsageRoute(route.query, defaultFilters()))
const applied = ref({ ...draft })
const filtersActive = computed(() => Object.entries(applied.value).some(([key, value]) => value !== defaultFilters()[key as keyof typeof draft]))
const usageColumns = ref<PortalUsageColumnKey[]>(portalUsageColumns.filter(column => 'default' in column && column.default).map(column => column.key))
const errorColumns = ref<PortalErrorColumnKey[]>(portalErrorColumns.filter(column => 'default' in column && column.default).map(column => column.key))
const columnsAvailable = computed(() => activeTab.value === 'usage' ? portalUsageColumns : portalErrorColumns)
const columnKeys = computed<string[]>(() => activeTab.value === 'usage' ? usageColumns.value : errorColumns.value)
const exporting = ref(false)
const exportProgress = ref('')
const exportError = ref('')
const detailOpen = ref(false)
const detailRow = ref<UsageLog | null>(null)
const detailLoading = ref(false)
const detailError = ref('')
const errorDetailOpen = ref(false)
const errorDetail = ref<UserErrorRequestDetail | null>(null)
const errorDetailLoading = ref(false)
const errorDetailError = ref('')
let disposed = false
let listSequence = 0
let overviewSequence = 0
let detailSequence = 0
let errorDetailSequence = 0
let exportSequence = 0
let listController: AbortController | null = null
let exportController: AbortController | null = null
const optionsController = new AbortController()

function usageParams(): UsageQueryParams {
  const filter = applied.value
  const requestType = isUsageRequestType(filter.requestType) ? filter.requestType : undefined
  const legacyStream = requestTypeToLegacyStream(requestType)
  return {
    start_date: filter.start, end_date: filter.end, timezone,
    api_key_id: filter.key ? Number(filter.key) : undefined,
    group_id: filter.group ? Number(filter.group) : undefined,
    model: filter.model.trim() || undefined,
    request_type: requestType, stream: legacyStream == null ? undefined : legacyStream,
    native_compaction_v2: filter.compaction === '' ? undefined : filter.compaction === 'true',
    billing_type: subscriptionEnabled.value && filter.billingType !== '' ? Number(filter.billingType) : undefined,
    billing_mode: filter.billingMode || undefined,
  }
}
const analyticsFilters = computed(() => usageParams())
function errorParams(): UserErrorListParams {
  const filter = applied.value
  return { start_date: filter.start, end_date: filter.end, timezone, api_key_id: filter.key ? Number(filter.key) : undefined, model: filter.model.trim() || undefined, category: filter.category || undefined, status_code: filter.status ? Number(filter.status) : undefined }
}
async function loadRows() {
  const tab = activeTab.value
  if (tab === 'errors' && !errorsEnabled.value) return
  const sequence = ++listSequence
  listController?.abort()
  const controller = new AbortController()
  listController = controller
  rowsLoading.value = true
  rowsError.value = ''
  const state = pagination[tab]
  const sort = sorting[tab]
  try {
    if (tab === 'usage') {
      const result = await usageAPI.query({ ...usageParams(), page: state.page, page_size: state.size, sort_by: sort.by, sort_order: sort.order }, { signal: controller.signal })
      if (disposed || sequence !== listSequence || controller.signal.aborted) return
      usageRows.value = result.items
      state.total = result.total
    } else {
      const result = await usageAPI.listMyErrorRequests({ ...errorParams(), page: state.page, page_size: state.size, sort_by: sort.by, sort_order: sort.order })
      if (disposed || sequence !== listSequence || !errorsEnabled.value) return
      errorRows.value = result.items
      state.total = result.total
    }
  } catch (cause) {
    if (!disposed && sequence === listSequence && !isUsageAbort(cause)) rowsError.value = extractApiErrorMessage(cause, tab === 'usage' ? copy('用量明细加载失败，请重试。', 'Unable to load usage records. Please retry.') : copy('错误记录加载失败，请重试。', 'Unable to load error records. Please retry.'))
  } finally { if (!disposed && sequence === listSequence) rowsLoading.value = false }
}
async function loadOverview() {
  const sequence = ++overviewSequence
  overviewLoading.value = true
  monthError.value = ''
  trendError.value = ''
  const results = await Promise.allSettled([
    usageAPI.getStats({ start_date: dates.currentStart, end_date: dates.end, timezone }),
    usageAPI.getDashboardTrend({ start_date: dates.start, end_date: dates.end, granularity: 'day', timezone }),
  ])
  if (disposed || sequence !== overviewSequence) return
  if (results[0].status === 'fulfilled') monthStats.value = results[0].value
  else { monthStats.value = null; monthError.value = extractApiErrorMessage(results[0].reason, copy('本月消费暂时无法获取。', 'Monthly spending is currently unavailable.')) }
  if (results[1].status === 'fulfilled') months.value = aggregateUsageMonths(results[1].value.trend || [])
  else { months.value = []; trendError.value = extractApiErrorMessage(results[1].reason, copy('每月用量暂时无法获取。', 'Monthly usage is currently unavailable.')) }
  overviewLoading.value = false
}
async function loadAllKeys() {
  const first = await keysAPI.list(1, 100, undefined, { signal: optionsController.signal })
  const all = [...first.items]
  for (let page = 2; page <= first.pages; page++) {
    if (disposed) return []
    const next = await keysAPI.list(page, 100, undefined, { signal: optionsController.signal })
    all.push(...next.items)
  }
  return all
}
async function loadOptions() {
  const results = await Promise.allSettled([
    loadAllKeys(), userGroupsAPI.getAvailable(),
    usageAPI.getDashboardSnapshotV2({ start_date: dates.start, end_date: dates.end, granularity: 'day', timezone, include_trend: false, include_model_stats: true, include_group_stats: true }),
  ])
  if (disposed) return
  if (results[0].status === 'fulfilled') keys.value = results[0].value
  if (results[1].status === 'fulfilled') groups.value = results[1].value
  if (results[2].status === 'fulfilled') {
    modelNames.value = [...new Set((results[2].value.models || []).map(model => model.model))].sort()
    const found = new Map(groups.value.map(group => [group.id, group]))
    for (const group of results[2].value.groups || []) if (group.group_id && !found.has(group.group_id)) found.set(group.group_id, { id: group.group_id, name: group.group_name })
    groups.value = [...found.values()]
  }
  optionsError.value = results.some(result => result.status === 'rejected') ? copy('部分筛选候选暂未加载完整，可输入模型名，或重试加载候选。', 'Some filter options could not be loaded. Enter a model name or retry.') : ''
}
function applyFilters() {
  if (!validUsageDate(draft.start) || !validUsageDate(draft.end) || draft.start > draft.end) {
    filterError.value = copy('请选择有效的开始日期和结束日期。', 'Choose a valid start and end date.')
    return
  }
  filterError.value = ''
  applied.value = { ...draft }
  pagination.usage.page = 1
  pagination.errors.page = 1
  void loadRows()
  void router.push({ path: '/usage', query: usageFiltersQuery(applied.value) })
}
function resetFilters() { Object.assign(draft, defaultFilters()); applyFilters() }
function changeAnalysisRange(value: { start: string; end: string }) {
  Object.assign(draft, applied.value, value)
  applyFilters()
}
async function drilldown(value: UsageDrilldown) {
  const query = { ...usageFiltersQuery(applied.value), ...usageDrilldownQuery(value) }
  const restored = parseUsageRoute(query as LocationQuery, defaultFilters())
  const alreadyUsage = activeTab.value === 'usage'
  Object.assign(draft, restored)
  applied.value = restored
  pagination.usage.page = 1
  pagination.errors.page = 1
  filterError.value = ''
  activeTab.value = 'usage'
  if (alreadyUsage) void loadRows()
  await router.push({ path: '/usage', query, hash: '#usage-records' })
  await nextTick()
  document.getElementById('usage-records')?.scrollIntoView({ block: 'start', behavior: 'auto' })
}
function changePage(page: number) { pageState.value.page = page; void loadRows() }
function changePageSize(value: string | number | boolean | null) {
  pageState.value.size = Number(value) || 10
  pageState.value.page = 1
  void loadRows()
}
function changeSort(key: string) {
  if (!['created_at', 'model', ...(activeTab.value === 'errors' ? ['status_code'] : [])].includes(key)) return
  const state = sortState.value
  state.order = state.by === key && state.order === 'desc' ? 'asc' : 'desc'
  state.by = key
  pageState.value.page = 1
  void loadRows()
}
function toggleColumn(key: string) {
  const definition = columnsAvailable.value.find(column => column.key === key)
  if (!definition || ('fixed' in definition && definition.fixed)) return
  if (activeTab.value === 'usage') {
    const typed = key as PortalUsageColumnKey
    usageColumns.value = usageColumns.value.includes(typed) ? usageColumns.value.filter(column => column !== typed) : [...usageColumns.value, typed]
  } else {
    const typed = key as PortalErrorColumnKey
    errorColumns.value = errorColumns.value.includes(typed) ? errorColumns.value.filter(column => column !== typed) : [...errorColumns.value, typed]
  }
  try { localStorage.setItem(`portal-usage-columns-${activeTab.value}`, JSON.stringify(columnKeys.value)) } catch { /* Column selection still applies during this visit. */ }
}
function restoreColumns() {
  for (const tab of ['usage', 'errors'] as const) {
    try {
      const saved: unknown = JSON.parse(localStorage.getItem(`portal-usage-columns-${tab}`) || 'null')
      if (!Array.isArray(saved)) continue
      const definitions = tab === 'usage' ? portalUsageColumns : portalErrorColumns
      const values = definitions.filter(column => ('fixed' in column && column.fixed) || saved.includes(column.key)).map(column => column.key)
      if (tab === 'usage') usageColumns.value = values as PortalUsageColumnKey[]
      else errorColumns.value = values as PortalErrorColumnKey[]
    } catch { /* Invalid saved preferences fall back to the default columns. */ }
  }
}
async function openDetail(row: UsageLog) {
  const sequence = ++detailSequence
  detailOpen.value = true
  detailRow.value = row
  detailLoading.value = true
  detailError.value = ''
  try { const detail = await usageAPI.getById(row.id); if (!disposed && sequence === detailSequence) detailRow.value = detail }
  catch (cause) { if (!disposed && sequence === detailSequence) detailError.value = extractApiErrorMessage(cause, copy('完整记录加载失败，当前显示列表中的数据。', 'Unable to load the complete record. Showing the list data.')) }
  finally { if (!disposed && sequence === detailSequence) detailLoading.value = false }
}
function closeDetail() { detailOpen.value = false; detailSequence += 1 }
async function openErrorDetail(row: UserErrorRequest) {
  if (!errorsEnabled.value) return
  const sequence = ++errorDetailSequence
  errorDetailOpen.value = true
  errorDetail.value = null
  errorDetailLoading.value = true
  errorDetailError.value = ''
  try { const result = await usageAPI.getMyErrorDetail(row.id); if (!disposed && sequence === errorDetailSequence && errorsEnabled.value) errorDetail.value = result }
  catch (cause) { if (!disposed && sequence === errorDetailSequence) errorDetailError.value = extractApiErrorMessage(cause, copy('错误详情加载失败。', 'Unable to load error details.')) }
  finally { if (!disposed && sequence === errorDetailSequence) errorDetailLoading.value = false }
}
function closeErrorDetail() { errorDetailOpen.value = false; errorDetailSequence += 1 }
function cancelExport() { exportController?.abort(); exportSequence += 1; exporting.value = false; exportProgress.value = copy('已取消导出。', 'Export canceled.') }
async function exportCsv() {
  if (exporting.value) return
  const tab = activeTab.value
  if (tab === 'errors' && !errorsEnabled.value) return
  const sequence = ++exportSequence
  const controller = new AbortController()
  exportController = controller
  const params = tab === 'usage' ? usageParams() : errorParams()
  const sort = { ...sorting[tab] }
  const collectedUsage: UsageLog[] = []
  const collectedErrors: UserErrorRequest[] = []
  const seen = new Set<number>()
  let pages = 1
  let expected = 0
  exporting.value = true
  exportError.value = ''
  exportProgress.value = copy('正在准备导出…', 'Preparing export…')
  try {
    for (let page = 1; page <= pages; page++) {
      if (controller.signal.aborted || sequence !== exportSequence || disposed) return
      const common = { ...params, page, page_size: 100, sort_by: sort.by, sort_order: sort.order }
      const response = tab === 'usage' ? await usageAPI.query(common as UsageQueryParams, { signal: controller.signal }) : await usageAPI.listMyErrorRequests(common as UserErrorListParams)
      if (controller.signal.aborted || sequence !== exportSequence || disposed || (tab === 'errors' && !errorsEnabled.value)) return
      if (page === 1) { expected = response.total; pages = response.pages || Math.ceil(response.total / (response.page_size || 100)) }
      for (const row of response.items) {
        if (seen.has(row.id)) continue
        seen.add(row.id)
        if (tab === 'usage') collectedUsage.push(row as UsageLog)
        else collectedErrors.push(row as UserErrorRequest)
      }
      exportProgress.value = copy(`正在导出 ${seen.size} / ${expected} 条…`, `Exporting ${seen.size} / ${expected} records…`)
    }
    if (!expected) { exportProgress.value = copy('当前筛选范围没有可导出的记录。', 'No records to export for these filters.'); return }
    if (seen.size !== expected) throw new Error(copy('导出期间记录数量发生变化，为避免不完整文件，请重新导出或选择已经结束的日期范围。', 'Records changed during export. Please retry or choose a completed date range to avoid an incomplete file.'))
    const csv = tab === 'usage' ? usageLogsCsv(collectedUsage, !zh.value) : usageErrorsCsv(collectedErrors, !zh.value)
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8;' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `${tab === 'usage' ? 'usage' : 'errors'}_${params.start_date}_${params.end_date}.csv`
    link.click()
    URL.revokeObjectURL(url)
    exportProgress.value = copy(`已导出 ${seen.size} 条记录。`, `Exported ${seen.size} records.`)
  } catch (cause) {
    if (!disposed && sequence === exportSequence && !isUsageAbort(cause)) { exportError.value = extractApiErrorMessage(cause, copy('导出失败，未生成文件。', 'Export failed. No file was created.')); exportProgress.value = '' }
  } finally { if (!disposed && sequence === exportSequence) exporting.value = false }
}
watch(activeTab, () => { showColumns.value = false; void loadRows() })
watch(errorsEnabled, enabled => { if (!enabled) { closeErrorDetail(); errorDetail.value = null; errorRows.value = []; if (activeTab.value === 'errors') { cancelExport(); activeTab.value = 'usage' } } })
watch(subscriptionEnabled, enabled => { if (!enabled && applied.value.billingType) { draft.billingType = ''; applied.value.billingType = ''; pagination.usage.page = 1; if (activeTab.value === 'usage') void loadRows() } })
watch(() => route.query, query => {
  const restored = parseUsageRoute(query, defaultFilters())
  if (JSON.stringify(restored) === JSON.stringify(applied.value) && pagination.usage.page === 1 && pagination.errors.page === 1) return
  Object.assign(draft, restored)
  applied.value = restored
  pagination.usage.page = 1
  pagination.errors.page = 1
  filterError.value = ''
  void loadRows()
})
onMounted(() => { restoreColumns(); void loadRows(); void loadOverview(); void loadOptions() })
onBeforeUnmount(() => {
  disposed = true
  listController?.abort()
  exportController?.abort()
  optionsController.abort()
  listSequence += 1
  overviewSequence += 1
  detailSequence += 1
  errorDetailSequence += 1
  exportSequence += 1
})
const quickDatePreset = computed(() => {
  const today = formatDateLocalInput(new Date())
  const sevenDaysAgo = formatDateLocalInput(new Date(Date.now() - 6 * 86400000))
  if (applied.value.start === today && applied.value.end === today) return 'today'
  if (applied.value.start === sevenDaysAgo && applied.value.end === today) return '7d'
  if (applied.value.start === dates.currentStart && applied.value.end === dates.end) return 'month'
  return ''
})

function applyQuickRange(preset: string) {
  if (preset !== 'today' && preset !== '7d' && preset !== 'month') return
  const today = formatDateLocalInput(new Date())
  if (preset === 'today') {
    draft.start = today
    draft.end = today
  } else if (preset === '7d') {
    draft.start = formatDateLocalInput(new Date(Date.now() - 6 * 86400000))
    draft.end = today
  } else {
    draft.start = dates.currentStart
    draft.end = dates.end
  }
  applyFilters()
}

function handleSectionTabChange(val: string) {
  sectionTab.value = val
  if (val === 'traffic') {
    void router.replace({ path: '/usage', query: { ...route.query, tab: 'traffic' } })
  } else {
    const nextQuery = { ...route.query }
    delete nextQuery.tab
    void router.replace({ path: '/usage', query: nextQuery })
  }
}

watch(() => route.query.tab, (tab) => {
  if (tab === 'traffic') sectionTab.value = 'traffic'
  else if (tab === 'usage' || tab === undefined) sectionTab.value = 'usage'
})
</script>

<template>
  <div class="mx-auto max-w-[1440px] space-y-6 pb-10">
    <!-- Editorial Page Header -->
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
          {{ copy('用量与流量中心', 'Usage & traffic') }}
        </h1>
        <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
          {{ copy('按时间、模型与 API Key 追踪 Token 消耗、费用构成与应用流量。', 'Track token consumption, billing cost, and application traffic by date, model, and API key.') }}
        </p>
      </div>

      <div class="flex items-center gap-3 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-3.5 py-2 text-sm">
        <span class="text-xs text-[var(--atelier-ink-muted)]">{{ copy('当前余额', 'Balance') }}</span>
        <strong class="font-mono text-sm font-semibold tabular-nums text-[var(--atelier-ink)]">
          {{ auth.user ? formatUsageMoney(auth.user.balance, 2) : '—' }}
        </strong>
        <RouterLink
          v-if="!auth.isSimpleMode"
          to="/purchase"
          class="border-l border-[var(--atelier-border)] pl-3 text-xs font-medium text-[var(--atelier-ink)] hover:underline"
        >
          {{ copy('充值', 'Recharge') }} ↗
        </RouterLink>
      </div>
    </header>

    <!-- Primary Underline Tabs: Usage vs Traffic -->
    <UiTabs
      :model-value="sectionTab"
      :tabs="sectionTabs"
      :aria-label="copy('用量与流量视图', 'Usage and traffic views')"
      @update:model-value="handleSectionTabChange"
    />

    <!-- Traffic Panel -->
    <div v-if="sectionTab === 'traffic'" class="pt-1">
      <TrafficPanel :admin="false" />
    </div>

    <!-- Token Usage & Details -->
    <template v-else>
      <PortalUsageAnalytics
        :filters="analyticsFilters"
        @range-change="changeAnalysisRange"
        @drilldown="drilldown"
      />

      <section
        id="usage-records"
        class="atelier-surface scroll-mt-6 p-5 sm:p-6"
        aria-labelledby="usage-records-title"
      >
        <div class="flex flex-wrap items-center justify-between gap-4 pb-4">
          <div>
            <h2 id="usage-records-title" class="text-base font-semibold tracking-tight text-[var(--atelier-ink)]">
              {{ copy('请求明细', 'Request details') }}
            </h2>
            <p class="mt-0.5 text-xs text-[var(--atelier-ink-secondary)]">
              {{ copy('查看每次请求的 Token、费用与响应表现。', 'Inspect tokens, cost and response performance for each request.') }}
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <UiSegmented
              v-if="errorsEnabled"
              v-model="activeTab"
              :options="recordTypeOptions"
              size="sm"
              :aria-label="copy('明细类别', 'Record type')"
            />
            <UiSegmented
              :model-value="quickDatePreset"
              :options="quickRangeOptions"
              size="sm"
              :aria-label="copy('快捷时间范围', 'Quick date range')"
              @update:model-value="applyQuickRange"
            />
          </div>
        </div>

        <!-- Filters Form -->
        <form
          v-if="showFilters"
          id="usage-filters"
          class="mb-4 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/55 p-4"
          @submit.prevent="applyFilters"
        >
          <div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
            <label class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>{{ copy('开始日期', 'Start date') }}</span>
              <input v-model="draft.start" type="date" :max="draft.end || dates.end" required class="atelier-input" />
            </label>
            <label class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>{{ copy('结束日期', 'End date') }}</span>
              <input v-model="draft.end" type="date" :min="draft.start" :max="dates.end" required class="atelier-input" />
            </label>
            <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>API Key</span>
              <UiSelect
                :model-value="draft.key"
                :options="keySelectOptions"
                :aria-label="copy('筛选 API Key', 'Filter API Key')"
                @update:model-value="(v) => { draft.key = String(v ?? '') }"
              />
            </div>
            <label class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
              <span>{{ copy('模型', 'Model') }}</span>
              <input
                v-model="draft.model"
                list="portal-usage-models"
                :placeholder="copy('全部模型', 'All models')"
                class="atelier-input"
              />
              <datalist id="portal-usage-models">
                <option v-for="model in modelNames" :key="model" :value="model" />
              </datalist>
            </label>

            <template v-if="activeTab === 'usage'">
              <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('分组', 'Group') }}</span>
                <UiSelect
                  :model-value="draft.group"
                  :options="groupSelectOptions"
                  :aria-label="copy('筛选分组', 'Filter group')"
                  @update:model-value="(v) => { draft.group = String(v ?? '') }"
                />
              </div>
              <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('请求类型', 'Request type') }}</span>
                <UiSelect
                  :model-value="draft.requestType"
                  :options="requestTypeSelectOptions"
                  :aria-label="copy('筛选请求类型', 'Filter request type')"
                  @update:model-value="(v) => { draft.requestType = String(v ?? '') }"
                />
              </div>
              <div v-if="subscriptionEnabled" class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('扣费方式', 'Billing source') }}</span>
                <UiSelect
                  :model-value="draft.billingType"
                  :options="billingTypeSelectOptions"
                  :aria-label="copy('筛选扣费方式', 'Filter billing source')"
                  @update:model-value="(v) => { draft.billingType = String(v ?? '') }"
                />
              </div>
              <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('计费模式', 'Billing mode') }}</span>
                <UiSelect
                  :model-value="draft.billingMode"
                  :options="billingModeSelectOptions"
                  :aria-label="copy('筛选计费模式', 'Filter billing mode')"
                  @update:model-value="(v) => { draft.billingMode = String(v ?? '') }"
                />
              </div>
              <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('原生压缩', 'Native compaction') }}</span>
                <UiSelect
                  :model-value="draft.compaction"
                  :options="compactionSelectOptions"
                  :aria-label="copy('筛选原生压缩', 'Filter native compaction')"
                  @update:model-value="(v) => { draft.compaction = String(v ?? '') }"
                />
              </div>
            </template>

            <template v-else>
              <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('错误分类', 'Error category') }}</span>
                <UiSelect
                  :model-value="draft.category"
                  :options="categorySelectOptions"
                  :aria-label="copy('筛选错误分类', 'Filter error category')"
                  @update:model-value="(v) => { draft.category = String(v ?? '') }"
                />
              </div>
              <div class="flex flex-col gap-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ copy('状态码', 'Status code') }}</span>
                <UiSelect
                  :model-value="draft.status"
                  :options="statusSelectOptions"
                  :aria-label="copy('筛选状态码', 'Filter status code')"
                  @update:model-value="(v) => { draft.status = String(v ?? '') }"
                />
              </div>
            </template>
          </div>

          <p v-if="optionsError" class="mt-3 text-xs text-[var(--atelier-ink-muted)]">
            {{ optionsError }}
            <button type="button" class="ml-1.5 underline hover:text-[var(--atelier-ink)]" @click="loadOptions">
              {{ copy('重试', 'Retry') }}
            </button>
          </p>
          <p v-if="filterError" class="mt-3 text-xs text-[var(--atelier-err)]" role="alert">{{ filterError }}</p>

          <div class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--atelier-border)] pt-3.5">
            <span class="text-xs text-[var(--atelier-ink-muted)]">
              {{ activeTab === 'usage' ? copy('筛选同时应用于上方统计和明细', 'Filters apply to statistics and details') : copy('错误分类和状态码仅应用于错误明细', 'Error category and status apply to error details only') }}
              · {{ timezoneLabel(timezone) }}
            </span>
            <div class="flex items-center gap-2">
              <UiButton variant="secondary" size="sm" type="button" @click="resetFilters">
                {{ copy('重置', 'Reset') }}
              </UiButton>
              <UiButton variant="primary" size="sm" type="submit">
                {{ copy('应用筛选', 'Apply filters') }}
              </UiButton>
            </div>
          </div>
        </form>

        <!-- Action Toolbar -->
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2 text-xs text-[var(--atelier-ink-secondary)]">
            <span class="font-mono tabular-nums">{{ applied.start }} — {{ applied.end }}</span>
            <span
              v-if="applied.model"
              class="inline-flex items-center rounded-md border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-xs text-[var(--atelier-ink)]"
            >
              {{ applied.model }}
            </span>
            <span
              v-if="applied.key"
              class="inline-flex items-center rounded-md border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2 py-0.5 text-xs text-[var(--atelier-ink)]"
            >
              {{ keys.find(key => String(key.id) === applied.key)?.name || `Key #${applied.key}` }}
            </span>
            <button
              v-if="filtersActive"
              type="button"
              class="text-xs font-medium text-[var(--atelier-ink-secondary)] underline hover:text-[var(--atelier-ink)]"
              @click="resetFilters"
            >
              {{ copy('清除条件', 'Clear filters') }} ×
            </button>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <span class="mr-1 font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
              {{ zh ? `共 ${pageState.total} 条记录` : `${pageState.total} records` }}
            </span>
            <UiButton
              variant="secondary"
              size="sm"
              :disabled="rowsLoading"
              :aria-label="copy('刷新明细', 'Refresh details')"
              @click="loadRows"
            >
              <Icon name="refresh" size="sm" />
            </UiButton>
            <UiButton
              variant="secondary"
              size="sm"
              :aria-expanded="showFilters"
              aria-controls="usage-filters"
              @click="showFilters = !showFilters"
            >
              <Icon name="filter" size="sm" />
              {{ copy('筛选', 'Filters') }}
              <UiStatusDot v-if="filtersActive" tone="info" />
            </UiButton>
            <UiButton variant="secondary" size="sm" @click="showColumns = true">
              <Icon name="grid" size="sm" />
              {{ copy('显示列', 'Columns') }}
            </UiButton>
            <UiButton variant="primary" size="sm" :disabled="exporting" :loading="exporting" @click="exportCsv">
              <Icon v-if="!exporting" name="download" size="sm" />
              {{ exporting ? copy('导出中…', 'Exporting…') : copy('导出 CSV', 'Export CSV') }}
            </UiButton>
          </div>
        </div>

        <InflightRequests class="mb-4" />

        <p v-if="exportProgress" class="mb-3 text-xs text-[var(--atelier-ink-secondary)]" role="status">
          {{ exportProgress }}
          <button
            v-if="exporting"
            type="button"
            class="ml-2 font-medium text-[var(--atelier-ink)] underline"
            @click="cancelExport"
          >
            {{ copy('取消导出', 'Cancel export') }}
          </button>
        </p>
        <p v-if="exportError" class="mb-3 rounded-lg border border-[var(--atelier-err)]/20 bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]" role="alert">
          {{ exportError }}
        </p>

        <div
          v-if="rowsLoading"
          class="flex min-h-[240px] items-center justify-center rounded-xl border border-[var(--atelier-border)] text-sm text-[var(--atelier-ink-muted)]"
          role="status"
        >
          {{ copy('正在加载明细…', 'Loading details…') }}
        </div>
        <div
          v-else-if="rowsError"
          class="rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]"
          role="alert"
        >
          {{ rowsError }}
          <button type="button" class="ml-2 font-medium underline" @click="loadRows">
            {{ copy('重试', 'Retry') }}
          </button>
        </div>
        <PortalUsageTable
          v-else-if="activeTab === 'usage'"
          :rows="usageRows"
          :columns="usageColumns"
          :sort-by="sorting.usage.by"
          :sort-order="sorting.usage.order"
          @sort="changeSort"
          @detail="openDetail"
        />
        <PortalUsageErrorTable
          v-else-if="errorsEnabled"
          :rows="errorRows"
          :columns="errorColumns"
          :sort-by="sorting.errors.by"
          :sort-order="sorting.errors.order"
          @sort="changeSort"
          @detail="openErrorDetail"
        />

        <!-- Pagination -->
        <div
          v-if="!rowsError && pageState.total > 0"
          class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--atelier-border)] pt-4 text-xs text-[var(--atelier-ink-secondary)]"
        >
          <div class="flex items-center gap-2">
            <span>{{ copy('每页', 'Per page') }}</span>
            <div class="w-20">
              <UiSelect
                :model-value="pageState.size"
                :options="pageSizeOptions"
                :disabled="rowsLoading"
                size="sm"
                :aria-label="copy('每页条数', 'Rows per page')"
                @update:model-value="changePageSize"
              />
            </div>
          </div>

          <div class="flex items-center gap-3">
            <span class="font-mono tabular-nums text-[var(--atelier-ink-muted)]">
              {{ zh ? `共 ${pageState.total} 条` : `${pageState.total} records` }} · {{ pageState.page }} / {{ totalPages }}
            </span>
            <div class="flex items-center gap-1.5">
              <UiButton
                variant="secondary"
                size="sm"
                :disabled="rowsLoading || pageState.page <= 1"
                :aria-label="copy('上一页', 'Previous page')"
                @click="changePage(pageState.page - 1)"
              >
                ←
              </UiButton>
              <UiButton
                variant="secondary"
                size="sm"
                :disabled="rowsLoading || pageState.page >= totalPages"
                :aria-label="copy('下一页', 'Next page')"
                @click="changePage(pageState.page + 1)"
              >
                →
              </UiButton>
            </div>
          </div>
        </div>
      </section>

      <!-- Last 12 months spending -->
      <details class="atelier-surface px-5 py-4 sm:px-6">
        <summary class="cursor-pointer select-none text-sm font-medium text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)]">
          {{ copy('查看过去 12 个月的消费', 'View spending over the last 12 months') }}
        </summary>
        <div class="mt-4 flex flex-wrap items-baseline gap-3 border-t border-[var(--atelier-border)] pt-4">
          <span class="text-xs text-[var(--atelier-ink-muted)]">{{ copy('本月消费', 'This month') }}</span>
          <strong class="font-mono text-2xl font-semibold tabular-nums text-[var(--atelier-ink)]">
            {{ overviewLoading ? '—' : monthStats ? formatUsageMoney(monthStats.total_actual_cost, 2) : '—' }}
          </strong>
          <small v-if="monthStats && !overviewLoading" class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
            {{ formatUsageNumber(monthStats.total_requests) }} {{ copy('次请求', 'requests') }}
          </small>
        </div>
        <p v-if="monthError" class="mt-3 text-xs text-[var(--atelier-err)]" role="alert">
          {{ monthError }}
          <button type="button" class="ml-2 underline" @click="loadOverview">{{ copy('重新加载', 'Reload') }}</button>
        </p>
        <section class="mt-5" aria-labelledby="usage-monthly-title">
          <div class="mb-3 flex items-center justify-between">
            <h2 id="usage-monthly-title" class="text-sm font-semibold text-[var(--atelier-ink)]">
              {{ copy('每月用量', 'Monthly usage') }}
            </h2>
            <UiButton variant="ghost" size="sm" :disabled="overviewLoading" @click="loadOverview">
              {{ overviewLoading ? copy('加载中…', 'Loading…') : copy('刷新', 'Refresh') }}
            </UiButton>
          </div>
          <div v-if="overviewLoading" class="flex h-[240px] items-center justify-center text-sm text-[var(--atelier-ink-muted)]" role="status">
            {{ copy('正在加载每月用量…', 'Loading monthly usage…') }}
          </div>
          <div v-else-if="trendError" class="rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]" role="alert">
            {{ trendError }}
            <button type="button" class="ml-2 underline" @click="loadOverview">{{ copy('重新加载', 'Reload') }}</button>
          </div>
          <PortalUsageChart v-else :months="months" />
        </section>
      </details>
    </template>

    <!-- Columns Dialog -->
    <PortalDialog :open="showColumns" :title="copy('显示列', 'Visible columns')" @close="showColumns = false">
      <div class="grid grid-cols-2 gap-3">
        <UiCheckbox
          v-for="column in columnsAvailable"
          :key="column.key"
          :model-value="columnKeys.includes(column.key)"
          :disabled="'fixed' in column && column.fixed"
          :label="usageColumnLabel(column.key, !zh)"
          @update:model-value="toggleColumn(column.key)"
        />
      </div>
    </PortalDialog>

    <PortalUsageDetail :open="detailOpen" :row="detailRow" :loading="detailLoading" :error="detailError" @close="closeDetail" />

    <PortalDialog :open="errorDetailOpen && errorsEnabled" :title="copy('错误请求详情', 'Error request details')" wide @close="closeErrorDetail">
      <p v-if="errorDetailLoading" class="text-sm text-[var(--atelier-ink-muted)]" role="status">
        {{ copy('正在加载详情…', 'Loading details…') }}
      </p>
      <p v-if="errorDetailError" class="text-sm text-[var(--atelier-err)]" role="alert">{{ errorDetailError }}</p>
      <template v-if="errorDetail">
        <dl class="grid grid-cols-[110px_minmax(0,1fr)] gap-x-5 gap-y-2.5 text-xs">
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('时间', 'Time') }}</dt>
          <dd class="font-mono text-[var(--atelier-ink)]">{{ new Date(errorDetail.created_at).toLocaleString(locale, { hour12: false }) }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('模型', 'Model') }}</dt>
          <dd class="break-all font-mono text-[var(--atelier-ink)]">{{ errorDetail.model }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('类型', 'Type') }}</dt>
          <dd class="text-[var(--atelier-ink)]">{{ usageErrorRequestLabel(errorDetail, !zh) }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('状态码', 'Status code') }}</dt>
          <dd class="font-mono text-[var(--atelier-ink)]">{{ errorDetail.status_code }}</dd>
          <template v-if="errorDetail.upstream_status_code">
            <dt class="text-[var(--atelier-ink-muted)]">{{ copy('上游状态码', 'Upstream status code') }}</dt>
            <dd class="font-mono text-[var(--atelier-ink)]">{{ errorDetail.upstream_status_code }}</dd>
          </template>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('分类', 'Category') }}</dt>
          <dd class="text-[var(--atelier-ink)]">{{ categoryOptions[errorDetail.category] || errorDetail.category }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">API Key</dt>
          <dd class="text-[var(--atelier-ink)]">{{ errorDetail.key_name }}<span v-if="errorDetail.key_deleted">{{ copy('（已删除）', ' (deleted)') }}</span></dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('分组', 'Group') }}</dt>
          <dd class="text-[var(--atelier-ink)]">{{ errorDetail.group_name || '—' }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('端点', 'Endpoint') }}</dt>
          <dd class="break-all font-mono text-[var(--atelier-ink)]">{{ errorDetail.inbound_endpoint || '—' }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">IP</dt>
          <dd class="font-mono text-[var(--atelier-ink)]">{{ errorDetail.client_ip || '—' }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('平台', 'Platform') }}</dt>
          <dd class="text-[var(--atelier-ink)]">{{ errorDetail.platform }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">{{ copy('消息', 'Message') }}</dt>
          <dd class="break-words text-[var(--atelier-ink)]">{{ errorDetail.message }}</dd>
          <dt class="text-[var(--atelier-ink-muted)]">User-Agent</dt>
          <dd class="break-all text-[var(--atelier-ink)]">{{ errorDetail.user_agent || '—' }}</dd>
        </dl>
        <h3 class="mt-5 mb-2 text-sm font-semibold text-[var(--atelier-ink)]">{{ copy('错误响应', 'Error response') }}</h3>
        <pre class="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] p-3.5 font-mono text-xs text-[var(--atelier-ink)]">{{ errorDetail.error_body || copy('未记录响应内容', 'No response body recorded') }}</pre>
      </template>
    </PortalDialog>
  </div>
</template>
