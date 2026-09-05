<template>
  <AppLayout>
    <div class="space-y-5 sm:space-y-6">
      <UsageStatsCards :stats="usageStats" :show-account-cost="false" :strike-standard-cost="true" />

      <div
        v-if="statsLoadError || modelStatsLoadError || chartLoadError"
        class="flex flex-col gap-2 rounded-xl border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-100 sm:flex-row sm:flex-wrap sm:items-center"
        role="alert"
      >
        <template v-if="statsLoadError">
          <span>{{ t('usage.statsFailedToLoad') }}</span>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadStats({ background: true })">
            {{ t('common.retry') }}
          </button>
        </template>
        <template v-if="modelStatsLoadError">
          <span>{{ t('usage.modelsFailedToLoad') }}</span>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadModelStats({ background: true })">
            {{ t('common.retry') }}
          </button>
        </template>
        <template v-if="chartLoadError">
          <span>{{ t('usage.chartsFailedToLoad') }}</span>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadChartData({ background: true })">
            {{ t('common.retry') }}
          </button>
        </template>
      </div>

      <div class="space-y-4">
        <div class="card p-4 sm:p-5">
          <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('admin.dashboard.timeRange') }}:</span>
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="onDateRangeChange"
              />
            </div>
            <div class="flex items-center gap-2 sm:ml-auto">
              <span class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('admin.dashboard.granularity') }}:</span>
              <div class="w-28">
                <Select v-model="granularity" :options="granularityOptions" @change="onGranularityChange" />
              </div>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <ModelDistributionChart
            v-model:metric="modelDistributionMetric"
            :model-stats="requestedModelStats"
            :loading="modelStatsLoading"
            :show-source-toggle="false"
            :show-metric-toggle="true"
            :enable-breakdown="false"
            :show-account-cost="false"
            :start-date="startDate"
            :end-date="endDate"
          />
          <GroupDistributionChart
            v-model:metric="groupDistributionMetric"
            :group-stats="groupStats"
            :loading="chartsLoading"
            :show-metric-toggle="true"
            :enable-breakdown="false"
            :show-account-cost="false"
            :start-date="startDate"
            :end-date="endDate"
          />
        </div>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <EndpointDistributionChart
            v-model:source="endpointDistributionSource"
            v-model:metric="endpointDistributionMetric"
            :endpoint-stats="inboundEndpointStats"
            :upstream-endpoint-stats="upstreamEndpointStats"
            :endpoint-path-stats="endpointPathStats"
            :loading="endpointStatsLoading"
            :show-source-toggle="false"
            :show-metric-toggle="true"
            :enable-breakdown="false"
            :title="t('usage.endpointDistribution')"
            :start-date="startDate"
            :end-date="endDate"
          />
          <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
        </div>
      </div>

      <div class="card p-4 sm:p-5">
        <div class="flex flex-col gap-4">
          <div v-if="activeTab === 'errors'" class="grid flex-1 grid-cols-1 items-end gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <div class="w-full sm:w-auto sm:min-w-[220px]">
              <label class="input-label">{{ t('usage.errors.keyName') }}</label>
              <Select v-model="errorFilter.api_key_id" :options="errorKeyOptions" @change="applyErrorFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[220px]">
              <label class="input-label">{{ t('usage.errors.model') }}</label>
              <Select
                v-model="errorFilter.model"
                :options="errorModelOptions"
                searchable
                creatable
                clearable
                :placeholder="t('usage.errors.modelPlaceholder')"
                @change="applyErrorFilters"
              />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[200px]">
              <label class="input-label">{{ t('usage.errors.category') }}</label>
              <Select v-model="errorFilter.category" :options="errorCategoryOptions" @change="applyErrorFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[180px]">
              <label class="input-label">{{ t('usage.errors.status') }}</label>
              <Select v-model="errorFilter.status_code" :options="errorStatusOptions" @change="applyErrorFilters" />
            </div>
          </div>
          <div v-else class="grid flex-1 grid-cols-1 items-end gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-6">
            <div class="w-full sm:w-auto sm:min-w-[220px]">
              <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
              <Select v-model="filters.api_key_id" :options="apiKeyOptions" @change="applyFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[220px]">
              <label class="input-label">{{ t('usage.model') }}</label>
              <Select v-model="filters.model" :options="modelOptions" searchable @change="applyFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[200px]">
              <label class="input-label">{{ t('admin.usage.group') }}</label>
              <Select v-model="filters.group_id" :options="groupOptions" searchable @change="applyFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[180px]">
              <label class="input-label">{{ t('usage.type') }}</label>
              <Select v-model="filters.request_type" :options="requestTypeOptions" @change="applyFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[180px]">
              <label class="input-label">{{ t('usage.compactionFilter') }}</label>
              <Select v-model="filters.native_compaction_v2" :options="compactionOptions" @change="applyFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[200px]">
              <label class="input-label">{{ t('admin.usage.billingType') }}</label>
              <Select v-model="filters.billing_type" :options="billingTypeOptions" @change="applyFilters" />
            </div>
            <div class="w-full sm:w-auto sm:min-w-[200px]">
              <label class="input-label">{{ t('admin.usage.billingMode') }}</label>
              <Select v-model="filters.billing_mode" :options="billingModeOptions" @change="applyFilters" />
            </div>
          </div>

          <div class="flex w-full flex-wrap items-center justify-end gap-2 border-t border-gray-100 pt-4 dark:border-dark-700 sm:gap-3">
            <button type="button" @click="refreshData()" :disabled="activeTab === 'errors' ? (errorLoading || errorRefreshing) : (loading || logsRefreshing)" class="btn btn-secondary">
              <Icon name="refresh" size="sm" :class="(activeTab === 'errors' ? (errorLoading || errorRefreshing) : (loading || logsRefreshing)) ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
            <button type="button" @click="resetFilters" class="btn btn-secondary">
              {{ t('common.reset') }}
            </button>
            <div class="relative" ref="columnDropdownRef">
              <button
                type="button"
                data-testid="usage-column-settings"
                @click="showColumnDropdown = !showColumnDropdown"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('admin.users.columnSettings')"
              >
                <Icon name="grid" size="sm" />
                <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
              </button>
              <div
                v-if="showColumnDropdown"
                class="absolute right-0 top-full z-50 mt-2 max-h-80 w-56 overflow-y-auto rounded-xl border border-gray-200 bg-white/95 py-1.5 shadow-lg backdrop-blur dark:border-dark-600 dark:bg-dark-800/95"
              >
                <button
                  v-for="col in currentToggleableColumns"
                  :key="col.key"
                  type="button"
                  :data-testid="`usage-column-toggle-${col.key}`"
                  @click="toggleCurrentColumn(col.key)"
                  class="flex min-h-10 w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 transition-colors hover:bg-gray-100 focus-visible:bg-primary-50 focus-visible:outline-none dark:text-dark-200 dark:hover:bg-dark-700 dark:focus-visible:bg-primary-950/30"
                >
                  <span>{{ col.label }}</span>
                  <Icon v-if="isCurrentColumnVisible(col.key)" name="check" size="sm" class="text-primary-500" />
                </button>
              </div>
            </div>
            <button v-if="activeTab !== 'errors'" type="button" @click="exportToCSV" :disabled="exporting" class="btn btn-primary">
              {{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="errorViewEnabled" class="flex gap-1 rounded-xl border border-gray-200 bg-gray-50/70 p-1 dark:border-dark-700 dark:bg-dark-800/60">
        <button class="tab" :class="{ 'tab-active': activeTab === 'usage' }" @click="activeTab = 'usage'">
          {{ t('usage.tabs.usage') }}
        </button>
        <button class="tab" :class="{ 'tab-active': activeTab === 'errors' }" @click="switchToErrors">
          {{ t('usage.tabs.errors') }}
        </button>
      </div>

      <template v-if="activeTab === 'usage'">
        <div
          v-if="usageLoadError"
          class="mb-3 flex flex-wrap items-center gap-3 rounded-xl border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-100"
          role="alert"
        >
          <span>{{ t('usage.failedToLoad') }}</span>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadLogs({ background: true })">
            {{ t('common.retry') }}
          </button>
        </div>
        <UsageTable
          :data="usageLogs"
          :loading="loading"
          :refreshing="logsRefreshing"
          :columns="visibleColumns"
          :server-side-sort="true"
          :show-account-billing="false"
          :show-upstream-endpoint="false"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
          @ipGeoBatchFailed="handleIpGeoBatchFailed"
        />

        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>

      <div
        v-if="activeTab === 'errors' && errorLoadError"
        class="mb-3 flex flex-wrap items-center gap-3 rounded-xl border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-100"
        role="alert"
      >
        <span>{{ t('usage.errors.failedToLoad') }}</span>
        <button type="button" class="btn btn-secondary btn-sm" @click="loadErrors({ background: true })">
          {{ t('common.retry') }}
        </button>
      </div>
      <UserErrorRequestsTable
        v-if="activeTab === 'errors' && errorViewEnabled"
        :rows="errorRows"
        :total="errorTotal"
        :loading="errorLoading"
        :page="errorPage"
        :page-size="errorPageSize"
        :visible-column-keys="errVisibleColumnKeys"
        @sort="onErrorSort"
        @update:page="onErrorPage"
        @update:pageSize="onErrorPageSize"
        @ipGeoBatchFailed="handleIpGeoBatchFailed"
      />
    </div>
    <ExportProgressDialog
      :show="exportProgress.show"
      :progress="exportProgress.progress"
      :current="exportProgress.current"
      :total="exportProgress.total"
      :estimated-time="exportProgress.estimatedTime"
      @cancel="cancelExport"
    />
  </AppLayout>

</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { keysAPI, usageAPI, userGroupsAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import UsageStatsCards from '@/components/admin/usage/UsageStatsCards.vue'
import UsageTable from '@/components/admin/usage/UsageTable.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import GroupDistributionChart from '@/components/charts/GroupDistributionChart.vue'
import EndpointDistributionChart from '@/components/charts/EndpointDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import Icon from '@/components/icons/Icon.vue'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import ExportProgressDialog from '@/components/common/ExportProgressDialog.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatReasoningEffort } from '@/utils/format'
import { getBillingModeLabel, getDisplayBillingMode as resolveDisplayBillingMode } from '@/utils/billingMode'
import { resolveUsageRequestType, requestTypeToLegacyStream } from '@/utils/usageRequestType'
import type {
  ApiKey,
  EndpointStat,
  Group,
  GroupStat,
  ModelStat,
  TrendDataPoint,
  UsageLog,
  UsageQueryParams,
  UsageStatsResponse,
  UserErrorRequest,
} from '@/types'
import type { Column } from '@/components/common/types'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'

const { t } = useI18n()
const appStore = useAppStore()

type DistributionMetric = 'tokens' | 'actual_cost'
type EndpointSource = 'inbound' | 'upstream' | 'path'
type ErrorFilters = { model: string | null; category: string; api_key_id: number | null; status_code: number | null }

interface UsageContext {
  start_date?: string
  end_date?: string
  granularity?: 'day' | 'hour'
  filters?: Partial<UsageQueryParams>
  errorFilters?: Partial<ErrorFilters>
}

const USAGE_CONTEXT_STORAGE_KEY = 'user-usage-query-context'
const readUsageContext = (): UsageContext => {
  try {
    const saved = localStorage.getItem(USAGE_CONTEXT_STORAGE_KEY)
    if (!saved) return {}
    const context = JSON.parse(saved)
    return context && typeof context === 'object' ? context as UsageContext : {}
  } catch {
    return {}
  }
}
const savedUsageContext = readUsageContext()

const usageStats = ref<UsageStatsResponse | null>(null)
const usageLogs = ref<UsageLog[]>([])
const trendData = ref<TrendDataPoint[]>([])
const requestedModelStats = ref<ModelStat[]>([])
const groupStats = ref<GroupStat[]>([])
const inboundEndpointStats = ref<EndpointStat[]>([])
const upstreamEndpointStats = ref<EndpointStat[]>([])
const endpointPathStats = ref<EndpointStat[]>([])

const loading = ref(false)
const logsRefreshing = ref(false)
const chartsLoading = ref(false)
const modelStatsLoading = ref(false)
const endpointStatsLoading = ref(false)
const exporting = ref(false)
const exportProgress = reactive({ show: false, progress: 0, current: 0, total: 0, estimatedTime: '' })
const usageLoadError = ref(false)
const statsLoadError = ref(false)
const modelStatsLoadError = ref(false)
const chartLoadError = ref(false)
const errorLoadError = ref(false)
const errorRows = ref<UserErrorRequest[]>([])
const errorLoading = ref(false)
const errorRefreshing = ref(false)
const errorPage = ref(1)
const errorPageSize = ref(20)
const errorSortBy = ref('created_at')
const errorSortOrder = ref<'asc' | 'desc'>('desc')
const errorTotal = ref(0)
const errorFilter = ref<ErrorFilters>({
  model: typeof savedUsageContext.errorFilters?.model === 'string' ? savedUsageContext.errorFilters.model : '',
  category: typeof savedUsageContext.errorFilters?.category === 'string' ? savedUsageContext.errorFilters.category : '',
  api_key_id: typeof savedUsageContext.errorFilters?.api_key_id === 'number' ? savedUsageContext.errorFilters.api_key_id : null,
  status_code: typeof savedUsageContext.errorFilters?.status_code === 'number' ? savedUsageContext.errorFilters.status_code : null,
})

const errorKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allKeys') },
  ...apiKeys.value.map((k) => ({ value: k.id, label: k.name })),
])

// 模型候选取自当前已加载错误中出现过的模型；creatable 允许输入任意片段做后端模糊。
const errorModelOptions = computed<SelectOption[]>(() => {
  const seen = new Set<string>()
  const opts: SelectOption[] = []
  for (const r of errorRows.value) {
    if (r.model && !seen.has(r.model)) {
      seen.add(r.model)
      opts.push({ value: r.model, label: r.model })
    }
  }
  return opts
})

const errorCategoryCodes = ['auth', 'rate_limit', 'quota', 'invalid_request', 'service_unavailable', 'upstream', 'internal', 'cyber']

const errorCategoryOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('usage.errors.allCategories') },
  ...errorCategoryCodes.map((c) => ({ value: c, label: t('usage.errors.categories.' + c) })),
])

// 状态码候选用固定常用列表(与管理端 UsageFilters 共用常量),不受当前页数据限制:
// 后端 status_code 过滤对全量生效,若只列当前页出现过的码,用户就选不到仅在后续页的码。
const errorStatusOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allStatuses') },
  ...COMMON_ERROR_STATUS_CODES.map((c) => ({ value: c, label: String(c) })),
])

const applyErrorFilters = () => {
  errorPage.value = 1
  persistUsageContext()
  void loadErrors()
}

let abortController: AbortController | null = null
let exportAbortController: AbortController | null = null
let logsReqSeq = 0
let chartReqSeq = 0
let statsReqSeq = 0
let modelStatsReqSeq = 0
let errorReqSeq = 0
let isUnmounted = false
let lastLogsQueryKey = ''
let lastErrorQueryKey = ''

const formatLocalDate = (date: Date): string =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

const getLast24HoursRangeDates = () => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return { start: formatLocalDate(start), end: formatLocalDate(end) }
}

const getGranularityForRange = (start: string, end: string): 'day' | 'hour' => {
  const startTime = new Date(`${start}T00:00:00`).getTime()
  const endTime = new Date(`${end}T00:00:00`).getTime()
  return Math.ceil((endTime - startTime) / (1000 * 60 * 60 * 24)) <= 1 ? 'hour' : 'day'
}

const defaultRange = getLast24HoursRangeDates()
const startDate = ref(
  typeof savedUsageContext.start_date === 'string' ? savedUsageContext.start_date : defaultRange.start
)
const endDate = ref(
  typeof savedUsageContext.end_date === 'string' ? savedUsageContext.end_date : defaultRange.end
)
const granularity = ref<'day' | 'hour'>(
  savedUsageContext.granularity === 'day' || savedUsageContext.granularity === 'hour'
    ? savedUsageContext.granularity
    : getGranularityForRange(startDate.value, endDate.value)
)

const modelDistributionMetric = ref<DistributionMetric>('tokens')
const groupDistributionMetric = ref<DistributionMetric>('tokens')
const endpointDistributionMetric = ref<DistributionMetric>('tokens')
const endpointDistributionSource = ref<EndpointSource>('inbound')
const activeTab = ref<'usage' | 'errors'>('usage')
const errorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false)

const filters = ref<UsageQueryParams>({
  start_date: startDate.value,
  end_date: endDate.value,
  native_compaction_v2: null,
  billing_type: null,
  billing_mode: null,
  ...savedUsageContext.filters,
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc',
})

const granularityOptions = computed<SelectOption[]>(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') },
])
const requestTypeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allTypes') },
  { value: 'ws_v2', label: t('usage.ws') },
  { value: 'live', label: t('usage.live') },
  { value: 'stream', label: t('usage.stream') },
  { value: 'sync', label: t('usage.sync') },
])
const compactionOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.allCompactionTypes') },
  { value: true, label: t('usage.compactionOnly') },
])
const billingTypeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allBillingTypes') },
  { value: 0, label: t('admin.usage.billingTypeBalance') },
  { value: 1, label: t('admin.usage.billingTypeSubscription') },
])
const billingModeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allBillingModes') },
  { value: 'token', label: t('admin.usage.billingModeToken') },
  { value: 'per_request', label: t('admin.usage.billingModePerRequest') },
  { value: 'image', label: t('admin.usage.billingModeImage') },
  { value: 'video', label: t('admin.usage.billingModeVideo') },
])

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const modelOptionValues = ref<string[]>([])

const apiKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.allApiKeys') },
  ...apiKeys.value.map((key) => ({ value: key.id, label: key.name })),
])
const groupOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allGroups') },
  ...groups.value.map((group) => ({ value: group.id, label: group.name })),
])
const modelOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allModels') },
  ...modelOptionValues.value.map((model) => ({ value: model, label: model })),
])

const normalizedFilters = computed<UsageQueryParams>(() => {
  const requestType = filters.value.request_type
  const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
  return {
    ...filters.value,
    start_date: startDate.value,
    end_date: endDate.value,
    stream: legacyStream === null ? undefined : legacyStream,
  }
})

const buildUsageListParams = (page: number, pageSize: number): UsageQueryParams => ({
  page,
  page_size: pageSize,
  ...normalizedFilters.value,
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order,
})

const persistUsageContext = () => {
  try {
    localStorage.setItem(USAGE_CONTEXT_STORAGE_KEY, JSON.stringify({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      filters: filters.value,
      errorFilters: errorFilter.value,
    }))
  } catch (error) {
    console.error('Failed to save usage query context:', error)
  }
}

const requestKey = (params: object) => JSON.stringify(params)

const buildErrorListParams = () => ({
  page: errorPage.value,
  page_size: errorPageSize.value,
  start_date: startDate.value,
  end_date: endDate.value,
  model: (errorFilter.value.model ?? '').trim() || undefined,
  category: errorFilter.value.category || undefined,
  api_key_id: errorFilter.value.api_key_id ?? undefined,
  status_code: errorFilter.value.status_code ?? undefined,
  sort_by: errorSortBy.value,
  sort_order: errorSortOrder.value,
})

const loadLogs = async ({ background = false }: { background?: boolean } = {}) => {
  abortController?.abort()
  const controller = new AbortController()
  const seq = ++logsReqSeq
  const params = buildUsageListParams(pagination.page, pagination.page_size)
  const isBackgroundRefresh = background && lastLogsQueryKey === requestKey(params)
  abortController = controller
  loading.value = !isBackgroundRefresh
  logsRefreshing.value = isBackgroundRefresh
  try {
    const res = await usageAPI.query(params, { signal: controller.signal })
    if (controller.signal.aborted || isUnmounted || seq !== logsReqSeq) return
    usageLogs.value = res.items
    pagination.total = res.total
    lastLogsQueryKey = requestKey(params)
    usageLoadError.value = false
  } catch (error: any) {
    if (controller.signal.aborted || isUnmounted || seq !== logsReqSeq) return
    usageLoadError.value = true
    appStore.showError(t('usage.failedToLoad'))
  } finally {
    if (!isUnmounted && seq === logsReqSeq) {
      loading.value = false
      logsRefreshing.value = false
    }
  }
}

const loadStats = async ({ background = false }: { background?: boolean } = {}) => {
  const seq = ++statsReqSeq
  const showLoading = !background || usageStats.value === null
  if (showLoading) endpointStatsLoading.value = true
  try {
    const stats = await usageAPI.getStats({ ...normalizedFilters.value })
    if (isUnmounted || seq !== statsReqSeq) return
    usageStats.value = stats
    inboundEndpointStats.value = stats.endpoints || []
    upstreamEndpointStats.value = []
    endpointPathStats.value = []
    statsLoadError.value = false
  } catch (error) {
    if (isUnmounted || seq !== statsReqSeq) return
    console.error('Failed to load usage stats:', error)
    statsLoadError.value = true
    appStore.showError(t('usage.statsFailedToLoad'))
  } finally {
    if (!isUnmounted && seq === statsReqSeq && showLoading) endpointStatsLoading.value = false
  }
}

const loadModelStats = async ({ background = false }: { background?: boolean } = {}) => {
  const seq = ++modelStatsReqSeq
  const showLoading = !background || requestedModelStats.value.length === 0
  if (showLoading) modelStatsLoading.value = true
  try {
    const response = await usageAPI.getDashboardModels({
      ...normalizedFilters.value,
      model_source: 'requested',
    })
    if (isUnmounted || seq !== modelStatsReqSeq) return
    requestedModelStats.value = response.models || []
    refreshModelOptions(response.models || [])
    modelStatsLoadError.value = false
  } catch (error) {
    if (isUnmounted || seq !== modelStatsReqSeq) return
    console.error('Failed to load model stats:', error)
    modelStatsLoadError.value = true
    appStore.showError(t('usage.modelsFailedToLoad'))
  } finally {
    if (!isUnmounted && seq === modelStatsReqSeq && showLoading) modelStatsLoading.value = false
  }
}

const loadChartData = async ({ background = false }: { background?: boolean } = {}) => {
  const seq = ++chartReqSeq
  const showLoading = !background || (trendData.value.length === 0 && groupStats.value.length === 0)
  if (showLoading) chartsLoading.value = true
  try {
    const snapshot = await usageAPI.getDashboardSnapshotV2({
      ...normalizedFilters.value,
      granularity: granularity.value,
      include_trend: true,
      include_model_stats: false,
      include_group_stats: true,
    })
    if (isUnmounted || seq !== chartReqSeq) return
    trendData.value = snapshot.trend || []
    groupStats.value = snapshot.groups || []
    chartLoadError.value = false
  } catch (error) {
    if (isUnmounted || seq !== chartReqSeq) return
    console.error('Failed to load chart data:', error)
    chartLoadError.value = true
    appStore.showError(t('usage.chartsFailedToLoad'))
  } finally {
    if (!isUnmounted && seq === chartReqSeq && showLoading) chartsLoading.value = false
  }
}

const refreshModelOptions = (models: ModelStat[]) => {
  const current = filters.value.model
  const set = new Set(modelOptionValues.value)
  models.forEach((item) => {
    if (item.model) set.add(item.model)
  })
  if (current) set.add(current)
  modelOptionValues.value = Array.from(set).sort()
}

const applyFilters = () => {
  pagination.page = 1
  persistUsageContext()
  void loadLogs()
  void loadStats()
  void loadModelStats()
  void loadChartData()
  resetErrorRows()
}

const refreshData = () => {
  void loadLogs({ background: true })
  void loadStats({ background: true })
  void loadModelStats({ background: true })
  void loadChartData({ background: true })
  if (activeTab.value === 'errors') void loadErrors({ background: true })
}

const onGranularityChange = () => {
  persistUsageContext()
  void loadChartData()
}

const resetFilters = () => {
  const range = getLast24HoursRangeDates()
  startDate.value = range.start
  endDate.value = range.end
  filters.value = {
    start_date: range.start,
    end_date: range.end,
    request_type: undefined,
    native_compaction_v2: null,
    billing_type: null,
    billing_mode: null,
  }
  granularity.value = getGranularityForRange(range.start, range.end)
  applyFilters()
  if (activeTab.value === 'errors') {
    errorFilter.value = { model: '', category: '', api_key_id: null, status_code: null }
    applyErrorFilters()
  }
}

const onDateRangeChange = (range: { startDate: string; endDate: string; preset: string | null }) => {
  startDate.value = range.startDate
  endDate.value = range.endDate
  filters.value.start_date = range.startDate
  filters.value.end_date = range.endDate
  granularity.value = getGranularityForRange(range.startDate, range.endDate)
  applyFilters()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  void loadLogs()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadLogs()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadLogs()
}

const handleIpGeoBatchFailed = () => {
  appStore.showError(t('usage.ipGeo.batchFailed'))
}

const getRequestTypeExportText = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'Cyber'
  if (requestType === 'live') return 'Live'
  if (requestType === 'ws_v2') return 'WS'
  if (requestType === 'stream') return 'Stream'
  if (requestType === 'sync') return 'Sync'
  return 'Unknown'
}

const getDisplayBillingMode = (
  row: Pick<UsageLog, 'billing_mode' | 'image_count'> | null | undefined
): string | null | undefined => resolveDisplayBillingMode(row)

const escapeCSVValue = (value: unknown): string => {
  if (value == null) return ''
  const str = String(value)
  const escaped = str.replace(/"/g, '""')
  if (/^[=+\-@\t\r]/.test(str)) return `"\'${escaped}"`
  if (/[,"\n\r]/.test(str)) return `"${escaped}"`
  return str
}

const cancelExport = () => exportAbortController?.abort()

const exportToCSV = async () => {
  if (pagination.total === 0) {
    appStore.showWarning(t('usage.noDataToExport'))
    return
  }

  const controller = new AbortController()
  const params = { ...buildUsageListParams(1, 100) }
  const filenameStartDate = startDate.value
  const filenameEndDate = endDate.value
  const startedAt = Date.now()
  exportAbortController = controller
  exporting.value = true
  exportProgress.show = true
  exportProgress.progress = 0
  exportProgress.current = 0
  exportProgress.total = pagination.total
  exportProgress.estimatedTime = ''
  appStore.showInfo(t('usage.preparingExport'))

  try {
    const allLogs: UsageLog[] = []
    let page = 1
    let total = pagination.total
    let exportedCount = 0

    while (!controller.signal.aborted) {
      const response = await usageAPI.query({ ...params, page, page_size: 100 }, { signal: controller.signal })
      if (controller.signal.aborted) return

      if (page === 1) {
        total = response.total
        exportProgress.total = total
        if (total === 0) {
          appStore.showWarning(t('usage.noDataToExport'))
          return
        }
      }

      allLogs.push(...response.items)
      exportedCount += response.items.length
      exportProgress.current = exportedCount
      exportProgress.progress = Math.min(100, Math.round(exportedCount / total * 100))
      if (exportedCount > 0 && exportedCount < total) {
        const elapsed = Math.max(Date.now() - startedAt, 1)
        const remainingSeconds = Math.ceil(((total - exportedCount) * elapsed) / exportedCount / 1000)
        exportProgress.estimatedTime = remainingSeconds > 0 ? `${remainingSeconds}s` : ''
      }

      if (exportedCount >= total || response.items.length < 100) break
      page += 1
    }

    if (controller.signal.aborted || allLogs.length === 0) return

    const headers = [
      'Time',
      'API Key Name',
      'Model',
      'Reasoning Effort',
      'Inbound Endpoint',
      'IP Address',
      'Type',
      'Billing Mode',
      'Input Tokens',
      'Output Tokens',
      'Cache Read Tokens',
      'Cache Creation Tokens',
      'Rate Multiplier',
      'Billed Cost',
      'Original Cost',
      'First Token (ms)',
      'Duration (ms)',
    ]
    const rows = allLogs.map((log) => [
      log.created_at,
      log.api_key?.name || '',
      log.model,
      formatReasoningEffort(log.reasoning_effort),
      log.inbound_endpoint || '',
      log.ip_address || '',
      getRequestTypeExportText(log),
      getBillingModeLabel(getDisplayBillingMode(log), t),
      log.input_tokens,
      log.output_tokens,
      log.cache_read_tokens,
      log.cache_creation_tokens,
      log.rate_multiplier,
      log.actual_cost.toFixed(8),
      log.total_cost.toFixed(8),
      log.first_token_ms ?? '',
      log.duration_ms ?? '',
    ].map(escapeCSVValue))
    const csvContent = [
      headers.map(escapeCSVValue).join(','),
      ...rows.map((row) => row.join(',')),
    ].join('\n')
    const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage_${filenameStartDate}_to_${filenameEndDate}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('usage.exportSuccess'))
  } catch (error: any) {
    if (controller.signal.aborted || error?.name === 'AbortError' || error?.code === 'ERR_CANCELED') return
    console.error('CSV Export failed:', error)
    appStore.showError(t('usage.exportFailed'))
  } finally {
    if (exportAbortController === controller) {
      exportAbortController = null
      exporting.value = false
      exportProgress.show = false
    }
  }
}

const ALWAYS_VISIBLE = ['created_at']
const DEFAULT_HIDDEN_COLUMNS = ['user_agent']
const HIDDEN_COLUMNS_KEY = 'user-usage-hidden-columns'

const allColumns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'model', label: t('usage.model'), sortable: true },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), sortable: false },
  { key: 'endpoint', label: t('usage.endpoint'), sortable: false },
  { key: 'ip_address', label: 'IP', sortable: false },
  { key: 'group', label: t('admin.usage.group'), sortable: false },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), sortable: false },
  { key: 'tokens', label: t('usage.tokens'), sortable: false },
  { key: 'cost', label: t('usage.cost'), sortable: false },
  { key: 'latency', label: t('usage.latency'), sortable: false },
  { key: 'created_at', label: t('usage.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false },
])

const hiddenColumns = reactive<Set<string>>(new Set())
const toggleableColumns = computed(() => allColumns.value.filter((col) => !ALWAYS_VISIBLE.includes(col.key)))
const visibleColumns = computed(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE.includes(col.key) || !hiddenColumns.has(col.key))
)
const isColumnVisible = (key: string) => !hiddenColumns.has(key)
const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) hiddenColumns.delete(key)
  else hiddenColumns.add(key)
  localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
}
const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    const values = saved ? JSON.parse(saved) as string[] : DEFAULT_HIDDEN_COLUMNS
    values.forEach((key) => hiddenColumns.add(key))
  } catch {
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

// 错误请求 tab 独立列设置(机制同用量列设置,存储互不影响)
const ERR_ALWAYS_VISIBLE = ['status', 'created_at']
const ERR_DEFAULT_HIDDEN_COLUMNS = ['user_agent']
const ERR_HIDDEN_COLUMNS_KEY = 'user-usage-error-hidden-columns'

// key 须与 UserErrorRequestsTable 的 allColumns 一致
const errAllColumns = computed<Column[]>(() => [
  { key: 'key_name', label: t('usage.errors.keyName') },
  { key: 'model', label: t('usage.errors.model') },
  { key: 'endpoint', label: t('usage.errors.endpoint') },
  { key: 'client_ip', label: 'IP' },
  { key: 'group', label: t('admin.usage.group') },
  { key: 'type', label: t('usage.type') },
  { key: 'platform', label: t('usage.errors.platform') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('usage.errors.status') },
  { key: 'message', label: t('usage.errors.message') },
  { key: 'created_at', label: t('usage.errors.time') },
  { key: 'user_agent', label: t('usage.userAgent') },
])

const errHiddenColumns = reactive<Set<string>>(new Set())
const errToggleableColumns = computed(() =>
  errAllColumns.value.filter((col) => !ERR_ALWAYS_VISIBLE.includes(col.key))
)
const errVisibleColumnKeys = computed(() =>
  errAllColumns.value
    .filter((col) => ERR_ALWAYS_VISIBLE.includes(col.key) || !errHiddenColumns.has(col.key))
    .map((col) => col.key)
)
const isErrColumnVisible = (key: string) => !errHiddenColumns.has(key)
const toggleErrColumn = (key: string) => {
  if (errHiddenColumns.has(key)) errHiddenColumns.delete(key)
  else errHiddenColumns.add(key)
  localStorage.setItem(ERR_HIDDEN_COLUMNS_KEY, JSON.stringify([...errHiddenColumns]))
}
const loadSavedErrColumns = () => {
  try {
    const saved = localStorage.getItem(ERR_HIDDEN_COLUMNS_KEY)
    const values = saved ? (JSON.parse(saved) as string[]) : ERR_DEFAULT_HIDDEN_COLUMNS
    values.forEach((key) => errHiddenColumns.add(key))
  } catch {
    ERR_DEFAULT_HIDDEN_COLUMNS.forEach((key) => errHiddenColumns.add(key))
  }
}

// 列设置下拉按当前 tab 分发
const currentToggleableColumns = computed(() =>
  activeTab.value === 'errors' ? errToggleableColumns.value : toggleableColumns.value
)
const isCurrentColumnVisible = (key: string) =>
  activeTab.value === 'errors' ? isErrColumnVisible(key) : isColumnVisible(key)
const toggleCurrentColumn = (key: string) => {
  if (activeTab.value === 'errors') toggleErrColumn(key)
  else toggleColumn(key)
}

const showColumnDropdown = ref(false)
const columnDropdownRef = ref<HTMLElement | null>(null)
const handleColumnClickOutside = (event: MouseEvent) => {
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(event.target as HTMLElement)) {
    showColumnDropdown.value = false
  }
}

const loadFilterOptions = async () => {
  try {
    const [keys, availableGroups] = await Promise.all([
      keysAPI.list(1, 100),
      userGroupsAPI.getAvailable(),
    ])
    apiKeys.value = keys.items
    groups.value = availableGroups
  } catch (error) {
    console.error('Failed to load usage filter options:', error)
  }
}

const resetErrorRows = () => {
  errorPage.value = 1
  if (activeTab.value === 'errors') {
    void loadErrors()
  } else {
    errorRows.value = []
    errorTotal.value = 0
  }
}

const loadErrors = async ({ background = false }: { background?: boolean } = {}) => {
  const seq = ++errorReqSeq
  const params = buildErrorListParams()
  const isBackgroundRefresh = background && lastErrorQueryKey === requestKey(params)
  errorLoading.value = !isBackgroundRefresh
  errorRefreshing.value = isBackgroundRefresh
  try {
    const resp = await usageAPI.listMyErrorRequests(params)
    if (isUnmounted || seq !== errorReqSeq) return
    errorRows.value = resp.items
    errorTotal.value = resp.total
    lastErrorQueryKey = requestKey(params)
    errorLoadError.value = false
  } catch (error) {
    if (isUnmounted || seq !== errorReqSeq) return
    console.error('[UsageView] loadErrors failed:', error)
    errorLoadError.value = true
    appStore.showError(t('usage.errors.failedToLoad'))
  } finally {
    if (!isUnmounted && seq === errorReqSeq) {
      errorLoading.value = false
      errorRefreshing.value = false
    }
  }
}

const onErrorSort = (sortBy: string, sortOrder: 'asc' | 'desc') => {
  errorSortBy.value = sortBy
  errorSortOrder.value = sortOrder
  errorPage.value = 1
  void loadErrors()
}

const onErrorPage = (page: number) => {
  errorPage.value = page
  void loadErrors()
}

const onErrorPageSize = (pageSize: number) => {
  errorPageSize.value = pageSize
  errorPage.value = 1
  void loadErrors()
}

const switchToErrors = () => {
  activeTab.value = 'errors'
  if (errorRows.value.length === 0) void loadErrors()
}

onMounted(() => {
  loadSavedColumns()
  loadSavedErrColumns()
  document.addEventListener('click', handleColumnClickOutside)
  void loadFilterOptions()
  refreshData()
})

onUnmounted(() => {
  isUnmounted = true
  abortController?.abort()
  exportAbortController?.abort()
  document.removeEventListener('click', handleColumnClickOutside)
})

watch(endpointDistributionSource, () => {
  // Endpoint source switching is handled by the chart component using already loaded stats.
})
</script>
