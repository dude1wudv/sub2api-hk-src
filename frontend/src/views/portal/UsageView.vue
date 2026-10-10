<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
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
import PortalUsageTable from '@/components/portal/PortalUsageTable.vue'
import PortalUsageErrorTable from '@/components/portal/PortalUsageErrorTable.vue'
import PortalUsageDetail from '@/components/portal/PortalUsageDetail.vue'
import PortalDialog from '@/components/portal/PortalDialog.vue'

const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const auth = useAuthStore()
const app = useAppStore()
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
const dates = usageMonthWindow()
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
const showFilters = ref(false)
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

function defaultFilters() {
  return { start: dates.currentStart, end: dates.end, key: '', group: '', model: '', requestType: '', billingType: '', billingMode: '', compaction: '', status: '', category: '' }
}
const draft = reactive(defaultFilters())
const applied = ref(defaultFilters())
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
  if (!draft.start || !draft.end || !Number.isFinite(Date.parse(draft.start)) || !Number.isFinite(Date.parse(draft.end)) || draft.start > draft.end) {
    filterError.value = copy('请选择有效的开始日期和结束日期。', 'Choose a valid start and end date.')
    return
  }
  filterError.value = ''
  applied.value = { ...draft }
  pagination.usage.page = 1
  pagination.errors.page = 1
  showFilters.value = false
  void loadRows()
}
function resetFilters() { Object.assign(draft, defaultFilters()); applyFilters() }
function changePage(page: number) { pageState.value.page = page; void loadRows() }
function changePageSize(event: Event) { pageState.value.size = Number((event.target as HTMLSelectElement).value); pageState.value.page = 1; void loadRows() }
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
</script>

<template>
  <div class="portal-page usage-page">
    <h1 class="portal-heading">{{ copy('用量信息', 'Usage information') }}</h1>
    <div class="usage-overview">
      <section class="usage-summary"><div><span class="portal-muted">{{ copy('账户余额', 'Account balance') }}</span><strong>{{ auth.user ? formatUsageMoney(auth.user.balance, 2) : '—' }}</strong></div><RouterLink v-if="!auth.isSimpleMode" to="/purchase" class="portal-button">{{ copy('充值', 'Recharge') }}</RouterLink></section>
      <section class="usage-summary"><div><span class="portal-muted">{{ copy('本月消费', 'This month') }}</span><strong>{{ overviewLoading ? '—' : monthStats ? formatUsageMoney(monthStats.total_actual_cost, 2) : '—' }}</strong><small v-if="monthStats && !overviewLoading">{{ formatUsageNumber(monthStats.total_requests) }} {{ copy('次请求', 'requests') }}</small></div><span class="usage-summary-symbol" aria-hidden="true">↗</span></section>
    </div>
    <p v-if="monthError" class="portal-error" role="alert">{{ monthError }} <button @click="loadOverview">{{ copy('重新加载', 'Reload') }}</button></p>
    <section class="usage-monthly" aria-labelledby="usage-monthly-title"><div class="usage-section-heading"><h2 id="usage-monthly-title">{{ copy('每月用量', 'Monthly usage') }}</h2><button class="portal-inline-button" :disabled="overviewLoading" @click="loadOverview">{{ overviewLoading ? copy('加载中…', 'Loading…') : copy('刷新', 'Refresh') }}</button></div><div v-if="overviewLoading" class="usage-chart-loading" role="status">{{ copy('正在加载每月用量…', 'Loading monthly usage…') }}</div><div v-else-if="trendError" class="portal-error" role="alert">{{ trendError }} <button @click="loadOverview">{{ copy('重新加载', 'Reload') }}</button></div><PortalUsageChart v-else :months="months" /></section>
    <section class="usage-records" aria-labelledby="usage-records-title">
      <div class="usage-section-heading"><h2 id="usage-records-title">{{ copy('明细', 'Details') }}</h2><div class="usage-toolbar"><button class="portal-button secondary" :aria-expanded="showFilters" aria-controls="usage-filters" @click="showFilters = !showFilters">{{ copy('筛选', 'Filters') }}<span v-if="filtersActive" class="usage-filter-dot" :aria-label="copy('已应用筛选', 'Filters applied')"></span></button><button class="portal-button secondary" @click="showColumns = true">{{ copy('显示列', 'Columns') }}</button><button class="portal-button secondary" :disabled="exporting" @click="exportCsv">{{ exporting ? copy('导出中…', 'Exporting…') : copy('导出 CSV', 'Export CSV') }}</button></div></div>
      <div v-if="errorsEnabled" class="usage-record-tabs" role="tablist" :aria-label="copy('明细类别', 'Record type')"><button role="tab" :aria-selected="activeTab === 'usage'" :class="{ active: activeTab === 'usage' }" @click="activeTab = 'usage'">{{ copy('用量记录', 'Usage logs') }}</button><button role="tab" :aria-selected="activeTab === 'errors'" :class="{ active: activeTab === 'errors' }" @click="activeTab = 'errors'">{{ copy('错误请求', 'Error requests') }}</button></div>
      <form v-if="showFilters" id="usage-filters" class="usage-filter-panel" @submit.prevent="applyFilters">
        <div class="usage-filter-grid"><label class="portal-field">{{ copy('开始日期', 'Start date') }}<input v-model="draft.start" type="date" :max="draft.end || dates.end" required /></label><label class="portal-field">{{ copy('结束日期', 'End date') }}<input v-model="draft.end" type="date" :min="draft.start" :max="dates.end" required /></label><label class="portal-field">API Key<select v-model="draft.key"><option value="">{{ copy('全部 Key', 'All keys') }}</option><option v-for="key in keys" :key="key.id" :value="String(key.id)">{{ key.name }}</option></select></label><label class="portal-field">{{ copy('模型', 'Model') }}<input v-model="draft.model" list="portal-usage-models" :placeholder="copy('全部模型', 'All models')" /><datalist id="portal-usage-models"><option v-for="model in modelNames" :key="model" :value="model" /></datalist></label>
          <template v-if="activeTab === 'usage'"><label class="portal-field">{{ copy('分组', 'Group') }}<select v-model="draft.group"><option value="">{{ copy('全部分组', 'All groups') }}</option><option v-for="group in groups" :key="group.id" :value="String(group.id)">{{ group.name }}</option></select></label><label class="portal-field">{{ copy('请求类型', 'Request type') }}<select v-model="draft.requestType"><option value="">{{ copy('全部类型', 'All types') }}</option><option value="sync">{{ copy('同步', 'Sync') }}</option><option value="stream">{{ copy('流式', 'Stream') }}</option><option value="ws_v2">WS</option><option value="live">Live</option><option value="cyber">Cyber</option></select></label><label v-if="subscriptionEnabled" class="portal-field">{{ copy('扣费方式', 'Billing source') }}<select v-model="draft.billingType"><option value="">{{ copy('全部方式', 'All sources') }}</option><option value="0">{{ copy('余额', 'Balance') }}</option><option value="1">{{ copy('订阅', 'Subscription') }}</option></select></label><label class="portal-field">{{ copy('计费模式', 'Billing mode') }}<select v-model="draft.billingMode"><option value="">{{ copy('全部模式', 'All modes') }}</option><option value="token">Token</option><option value="per_request">{{ copy('按次', 'Per request') }}</option><option value="image">{{ copy('图片', 'Image') }}</option><option value="video">{{ copy('视频', 'Video') }}</option></select></label><label class="portal-field">{{ copy('原生压缩', 'Native compaction') }}<select v-model="draft.compaction"><option value="">{{ copy('全部请求', 'All requests') }}</option><option value="true">{{ copy('仅原生压缩', 'Compaction only') }}</option><option value="false">{{ copy('非原生压缩', 'No compaction') }}</option></select></label></template>
          <template v-else><label class="portal-field">{{ copy('错误分类', 'Error category') }}<select v-model="draft.category"><option value="">{{ copy('全部分类', 'All categories') }}</option><option v-for="(label, key) in categoryOptions" :key="key" :value="key">{{ label }}</option></select></label><label class="portal-field">{{ copy('状态码', 'Status code') }}<select v-model="draft.status"><option value="">{{ copy('全部状态', 'All status codes') }}</option><option v-for="code in COMMON_ERROR_STATUS_CODES" :key="code" :value="String(code)">{{ code }}</option></select></label></template>
        </div><p v-if="optionsError" class="usage-options-note">{{ optionsError }} <button type="button" @click="loadOptions">{{ copy('重试', 'Retry') }}</button></p><p v-if="filterError" class="portal-error" role="alert">{{ filterError }}</p><div class="usage-filter-actions"><span>{{ copy('筛选仅应用于明细', 'Filters apply to details only') }} · {{ timezone }}</span><button class="portal-button secondary" type="button" @click="resetFilters">{{ copy('重置', 'Reset') }}</button><button class="portal-button" type="submit">{{ copy('应用筛选', 'Apply filters') }}</button></div>
      </form>
      <div class="usage-range"><span>{{ applied.start }} — {{ applied.end }}</span><button class="portal-inline-button" :disabled="rowsLoading" @click="loadRows">{{ copy('刷新明细', 'Refresh details') }}</button></div>
      <p v-if="exportProgress" class="usage-export-status" role="status">{{ exportProgress }} <button v-if="exporting" @click="cancelExport">{{ copy('取消导出', 'Cancel export') }}</button></p><p v-if="exportError" class="portal-error" role="alert">{{ exportError }}</p>
      <div v-if="rowsLoading" class="usage-list-loading" role="status">{{ copy('正在加载明细…', 'Loading details…') }}</div><p v-else-if="rowsError" class="portal-error" role="alert">{{ rowsError }} <button @click="loadRows">{{ copy('重试', 'Retry') }}</button></p><PortalUsageTable v-else-if="activeTab === 'usage'" :rows="usageRows" :columns="usageColumns" :sort-by="sorting.usage.by" :sort-order="sorting.usage.order" @sort="changeSort" @detail="openDetail" /><PortalUsageErrorTable v-else-if="errorsEnabled" :rows="errorRows" :columns="errorColumns" :sort-by="sorting.errors.by" :sort-order="sorting.errors.order" @sort="changeSort" @detail="openErrorDetail" />
      <div v-if="!rowsError" class="portal-pagination usage-pagination"><label>{{ copy('每页', 'Per page') }}<select :value="pageState.size" :disabled="rowsLoading" @change="changePageSize"><option :value="10">10</option><option :value="20">20</option><option :value="50">50</option></select></label><span class="portal-muted">{{ zh ? `共 ${pageState.total} 条` : `${pageState.total} records` }} · {{ pageState.page }} / {{ totalPages }}</span><button :disabled="rowsLoading || pageState.page <= 1" @click="changePage(pageState.page - 1)" :aria-label="copy('上一页', 'Previous page')">←</button><button :disabled="rowsLoading || pageState.page >= totalPages" @click="changePage(pageState.page + 1)" :aria-label="copy('下一页', 'Next page')">→</button></div>
    </section>
    <PortalDialog :open="showColumns" :title="copy('显示列', 'Visible columns')" @close="showColumns = false"><div class="usage-column-list"><label v-for="column in columnsAvailable" :key="column.key"><input type="checkbox" :checked="columnKeys.includes(column.key)" :disabled="'fixed' in column && column.fixed" @change="toggleColumn(column.key)" />{{ usageColumnLabel(column.key, !zh) }}</label></div></PortalDialog>
    <PortalUsageDetail :open="detailOpen" :row="detailRow" :loading="detailLoading" :error="detailError" @close="closeDetail" />
    <PortalDialog :open="errorDetailOpen && errorsEnabled" :title="copy('错误请求详情', 'Error request details')" wide @close="closeErrorDetail"><p v-if="errorDetailLoading" class="portal-muted" role="status">{{ copy('正在加载详情…', 'Loading details…') }}</p><p v-if="errorDetailError" class="portal-error" role="alert">{{ errorDetailError }}</p><template v-if="errorDetail"><dl class="usage-error-metadata"><dt>{{ copy('时间', 'Time') }}</dt><dd>{{ new Date(errorDetail.created_at).toLocaleString(locale, { hour12: false }) }}</dd><dt>{{ copy('模型', 'Model') }}</dt><dd>{{ errorDetail.model }}</dd><dt>{{ copy('类型', 'Type') }}</dt><dd>{{ usageErrorRequestLabel(errorDetail, !zh) }}</dd><dt>{{ copy('状态码', 'Status code') }}</dt><dd>{{ errorDetail.status_code }}</dd><dt v-if="errorDetail.upstream_status_code">{{ copy('上游状态码', 'Upstream status code') }}</dt><dd v-if="errorDetail.upstream_status_code">{{ errorDetail.upstream_status_code }}</dd><dt>{{ copy('分类', 'Category') }}</dt><dd>{{ categoryOptions[errorDetail.category] || errorDetail.category }}</dd><dt>API Key</dt><dd>{{ errorDetail.key_name }}<span v-if="errorDetail.key_deleted">{{ copy('（已删除）', ' (deleted)') }}</span></dd><dt>{{ copy('分组', 'Group') }}</dt><dd>{{ errorDetail.group_name || '—' }}</dd><dt>{{ copy('端点', 'Endpoint') }}</dt><dd>{{ errorDetail.inbound_endpoint || '—' }}</dd><dt>IP</dt><dd>{{ errorDetail.client_ip || '—' }}</dd><dt>{{ copy('平台', 'Platform') }}</dt><dd>{{ errorDetail.platform }}</dd><dt>{{ copy('消息', 'Message') }}</dt><dd>{{ errorDetail.message }}</dd><dt>User-Agent</dt><dd>{{ errorDetail.user_agent || '—' }}</dd></dl><h3 class="usage-error-body-title">{{ copy('错误响应', 'Error response') }}</h3><pre class="usage-error-body">{{ errorDetail.error_body || copy('未记录响应内容', 'No response body recorded') }}</pre></template></PortalDialog>
  </div>
</template>

<style scoped>
.usage-overview { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 22px; }
.usage-summary { display: flex; align-items: center; justify-content: space-between; gap: 22px; border: 1px solid #efede3; border-radius: 12px; padding: 24px; min-height: 132px; }
.usage-summary > div { display: grid; gap: 12px; }
.usage-summary strong { font-size: 30px; font-weight: 500; line-height: 1; font-variant-numeric: tabular-nums; }
.usage-summary small { color: #929088; font-size: 11px; }
.usage-summary-symbol { font-size: 30px; color: #c9c5bb; }
.usage-monthly { margin-top: 32px; }
.usage-section-heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 20px; }
.usage-section-heading h2 { font-size: 16px; font-weight: 500; }
.usage-section-heading > button { font-size: 12px; color: #929088; }
.usage-chart-loading { height: 265px; display: grid; place-items: center; color: #929088; }
.usage-records { margin-top: 34px; }
.usage-toolbar { display: flex; gap: 8px; flex-wrap: wrap; }
.usage-toolbar .portal-button { min-height: 32px; font-size: 12px; padding: 6px 13px; }
.usage-filter-dot { width: 5px; height: 5px; background: #000; border-radius: 50%; }
.usage-record-tabs { display: flex; gap: 22px; margin: 0 0 20px; border-bottom: 1px solid #efede3; }
.usage-record-tabs button { padding: 0 0 12px; color: #929088; border-bottom: 2px solid transparent; }
.usage-record-tabs .active { color: #000; border-bottom-color: #000; }
.usage-filter-panel { border: 1px solid #efede3; border-radius: 12px; padding: 20px; margin-bottom: 16px; }
.usage-filter-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; }
.usage-filter-grid label { font-size: 12px; color: #77746d; }
.usage-filter-actions { display: flex; gap: 10px; align-items: center; justify-content: flex-end; margin-top: 20px; }
.usage-filter-actions > span { color: #929088; font-size: 11px; margin-right: auto; }
.usage-options-note, .usage-export-status { color: #77746d; font-size: 12px; margin: 14px 0; }
.usage-options-note button, .usage-export-status button, .portal-error button { text-decoration: underline; margin-left: 8px; }
.usage-range { display: flex; align-items: center; justify-content: space-between; color: #929088; font-size: 11px; margin-bottom: 12px; }
.usage-list-loading { display: grid; place-items: center; min-height: 280px; border: 1px solid #efede3; border-radius: 12px; color: #929088; }
.usage-pagination { font-size: 12px; }
.usage-pagination label { margin-right: auto; }
.usage-pagination select { margin-left: 8px; }
.usage-column-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }
.usage-column-list label { display: flex; align-items: center; gap: 10px; }
.usage-column-list input { accent-color: #000; }
.usage-error-metadata { display: grid; grid-template-columns: 110px minmax(0, 1fr); gap: 12px 20px; font-size: 13px; }
.usage-error-metadata dt { color: #929088; }
.usage-error-metadata dd { overflow-wrap: anywhere; }
.usage-error-body-title { margin: 26px 0 14px; font-size: 15px; }
.usage-error-body { padding: 16px; background: #f7f5ee; border: 1px solid #efede3; border-radius: 8px; font-size: 12px; white-space: pre-wrap; overflow-wrap: anywhere; max-height: 340px; overflow: auto; }
.portal-error { margin: 16px 0; }
@media (max-width: 1100px) { .usage-filter-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 739px) { .usage-overview { gap: 12px; } .usage-summary { padding: 18px 14px; min-height: 122px; flex-wrap: wrap; gap: 14px; } .usage-summary strong { font-size: 25px; } .usage-summary .portal-button { font-size: 12px; padding: 5px 12px; min-height: 30px; } .usage-summary-symbol { display: none; } .usage-summary > div > .portal-muted { font-size: 12px; } .usage-section-heading { flex-wrap: wrap; } .usage-toolbar { width: 100%; } .usage-filter-panel { padding: 16px; } .usage-filter-actions { flex-wrap: wrap; } .usage-filter-actions > span { width: 100%; margin-bottom: 6px; } }
@media (max-width: 440px) { .usage-filter-grid { grid-template-columns: 1fr; } .usage-pagination { gap: 8px; } .usage-error-metadata { grid-template-columns: 80px minmax(0, 1fr); gap: 12px; } }
</style>
