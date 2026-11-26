<template>
  <div class="space-y-6" data-admin-page="dashboard">
    <!-- Loading State -->
    <div v-if="loading" class="flex items-center justify-center py-16">
      <LoadingSpinner />
    </div>

    <template v-else-if="stats">
      <!-- Single-Row Atelier Metric Ribbon (replaces 4-equal-grid KPI boxes) -->
      <UiMetricRibbon
        :primary-label="t('admin.dashboard.todayRequests')"
        :primary-value="formatNumber(stats.today_requests)"
        :primary-meta="`${t('common.total')}: ${formatNumber(stats.total_requests)}`"
        :metrics="ribbonMetrics"
        show-dots
      >
        <template #primaryAction>
          <div class="flex flex-wrap items-center gap-2">
            <span
              v-if="stats.stats_stale"
              role="status"
              class="inline-flex items-center gap-1.5 rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-warn-bg)] px-2.5 py-1 text-xs font-medium text-[var(--atelier-warn)]"
            >
              <UiStatusDot tone="warn" />
              {{ zh ? '统计待更新' : 'Statistics awaiting refresh' }}
            </span>
            <UiButton
              variant="secondary"
              size="sm"
              :disabled="loading || chartsLoading"
              @click="loadDashboardStats"
            >
              <Icon name="refresh" size="sm" :class="chartsLoading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </UiButton>
            <router-link to="/admin/accounts" class="atelier-btn atelier-btn-secondary h-8 px-3 text-xs">
              <Icon name="server" size="sm" />
              {{ t('nav.accounts') }}
            </router-link>
          </div>
        </template>
      </UiMetricRibbon>

      <!-- Asymmetric Operational Ledger (7 / 5 split) -->
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
        <!-- Left: Fleet, Keys, Users & Consumption Ledger -->
        <section class="atelier-surface flex flex-col justify-between p-5 lg:col-span-7">
          <div>
            <div class="flex items-center justify-between gap-3 border-b border-[var(--atelier-border)] pb-3">
              <div>
                <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
                  {{ zh ? '资源与核心运营指标' : 'Resources & core operational metrics' }}
                </h2>
                <p class="mt-0.5 text-xs text-[var(--atelier-ink-muted)]">
                  {{ zh ? '实时统计口径，点击异常账号可直接筛选排查' : 'Live operational totals — drill directly into error accounts or admin modules' }}
                </p>
              </div>
              <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
                {{ formatTokens(stats.rpm) }} RPM · {{ formatTokens(stats.tpm) }} TPM
              </span>
            </div>

            <dl class="divide-y divide-[var(--atelier-border-subtle)]">
              <!-- API Keys -->
              <div data-metric="keys" class="flex flex-wrap items-center justify-between gap-3 py-3.5">
                <dt class="flex items-center gap-2.5 text-sm text-[var(--atelier-ink-secondary)]">
                  <Icon name="key" size="sm" class="text-[var(--atelier-ink-muted)]" />
                  <span>{{ t('admin.dashboard.apiKeys') }}</span>
                </dt>
                <dd class="flex items-baseline gap-3">
                  <span class="font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
                    {{ formatNumber(stats.total_api_keys) }}
                  </span>
                  <span class="inline-flex items-center gap-1.5 text-xs text-[var(--atelier-ok)]">
                    <UiStatusDot tone="ok" />
                    <span class="font-mono tabular-nums">{{ formatNumber(stats.active_api_keys) }}</span>
                    {{ t('common.active') }}
                  </span>
                </dd>
              </div>

              <!-- Accounts -->
              <div data-metric="accounts" class="flex flex-wrap items-center justify-between gap-3 py-3.5">
                <dt class="flex items-center gap-2.5 text-sm text-[var(--atelier-ink-secondary)]">
                  <Icon name="server" size="sm" class="text-[var(--atelier-ink-muted)]" />
                  <span>{{ t('admin.dashboard.accounts') }}</span>
                </dt>
                <dd class="flex flex-wrap items-center gap-3">
                  <span class="font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
                    {{ formatNumber(stats.total_accounts) }}
                  </span>
                  <span class="inline-flex items-center gap-1.5 text-xs text-[var(--atelier-ok)]">
                    <UiStatusDot tone="ok" />
                    <span class="font-mono tabular-nums">{{ formatNumber(stats.normal_accounts) }}</span>
                    {{ t('common.active') }}
                  </span>
                  <button
                    v-if="stats.error_accounts > 0"
                    type="button"
                    data-test="error-account-drill"
                    class="inline-flex items-center gap-1 rounded-full bg-[var(--atelier-err-bg)] px-2.5 py-0.5 text-xs font-medium text-[var(--atelier-err)] transition-opacity hover:opacity-85"
                    @click="goToErrorAccounts"
                  >
                    <UiStatusDot tone="err" />
                    <span class="font-mono tabular-nums">{{ stats.error_accounts }}</span>
                    {{ t('common.error') }}
                    <Icon name="arrowRight" size="xs" />
                  </button>
                </dd>
              </div>

              <!-- Users & Balance -->
              <div data-metric="users" class="flex flex-wrap items-center justify-between gap-3 py-3.5">
                <dt class="flex items-center gap-2.5 text-sm text-[var(--atelier-ink-secondary)]">
                  <Icon name="userPlus" size="sm" class="text-[var(--atelier-ink-muted)]" />
                  <span>{{ t('admin.dashboard.users') }}</span>
                </dt>
                <dd class="flex flex-wrap items-baseline gap-3">
                  <span class="font-mono text-base font-semibold tabular-nums text-[var(--atelier-ok)]">
                    +{{ formatNumber(stats.today_new_users) }}
                  </span>
                  <span class="text-xs text-[var(--atelier-ink-muted)]">
                    {{ t('common.total') }}:
                    <span class="font-mono font-medium tabular-nums text-[var(--atelier-ink-secondary)]">{{ formatNumber(stats.total_users) }}</span>
                  </span>
                  <span data-metric="user-balance" class="text-xs text-[var(--atelier-ink-muted)]">
                    {{ zh ? '用户余额' : 'User balance' }}:
                    <span class="font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">${{ formatCost(stats.users_with_balance_total) }}</span>
                    ({{ stats.users_with_balance ?? 0 }})
                  </span>
                </dd>
              </div>

              <!-- Today Tokens & Cost Breakdown -->
              <div data-metric="tokens" class="flex flex-wrap items-center justify-between gap-3 py-3.5">
                <dt class="flex items-center gap-2.5 text-sm text-[var(--atelier-ink-secondary)]">
                  <Icon name="cube" size="sm" class="text-[var(--atelier-ink-muted)]" />
                  <span>{{ t('admin.dashboard.todayTokens') }}</span>
                </dt>
                <dd class="flex flex-wrap items-baseline gap-3">
                  <span class="font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
                    {{ formatTokens(stats.today_tokens) }}
                  </span>
                  <span class="font-mono text-xs tabular-nums">
                    <span class="font-medium text-[var(--atelier-ok)]" :title="t('admin.dashboard.actual')">${{ formatCost(stats.today_actual_cost) }}</span>
                    <span class="mx-1 text-[var(--atelier-ink-muted)]">/</span>
                    <span class="font-medium text-[var(--atelier-warn)]" :title="t('admin.dashboard.accountCost')">${{ formatCost(stats.today_account_cost) }}</span>
                    <span class="mx-1 text-[var(--atelier-ink-muted)]">/</span>
                    <span class="text-[var(--atelier-ink-muted)]" :title="t('admin.dashboard.standard')">${{ formatCost(stats.today_cost) }}</span>
                  </span>
                </dd>
              </div>

              <!-- Total Tokens & Cost Breakdown -->
              <div data-metric="total-tokens" class="flex flex-wrap items-center justify-between gap-3 py-3.5">
                <dt class="flex items-center gap-2.5 text-sm text-[var(--atelier-ink-secondary)]">
                  <Icon name="database" size="sm" class="text-[var(--atelier-ink-muted)]" />
                  <span>{{ t('admin.dashboard.totalTokens') }}</span>
                </dt>
                <dd class="flex flex-wrap items-baseline gap-3">
                  <span class="font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
                    {{ formatTokens(stats.total_tokens) }}
                  </span>
                  <span class="font-mono text-xs tabular-nums">
                    <span class="font-medium text-[var(--atelier-ok)]" :title="t('admin.dashboard.actual')">${{ formatCost(stats.total_actual_cost) }}</span>
                    <span class="mx-1 text-[var(--atelier-ink-muted)]">/</span>
                    <span class="font-medium text-[var(--atelier-warn)]" :title="t('admin.dashboard.accountCost')">${{ formatCost(stats.total_account_cost) }}</span>
                    <span class="mx-1 text-[var(--atelier-ink-muted)]">/</span>
                    <span class="text-[var(--atelier-ink-muted)]" :title="t('admin.dashboard.standard')">${{ formatCost(stats.total_cost) }}</span>
                  </span>
                </dd>
              </div>
            </dl>
          </div>

          <!-- Compact Operations Shortcuts -->
          <nav
            class="mt-4 flex flex-wrap items-center gap-2 border-t border-[var(--atelier-border)] pt-4"
            :aria-label="zh ? '运营快捷入口' : 'Operations shortcuts'"
          >
            <router-link to="/admin/users" class="atelier-btn atelier-btn-ghost h-8 px-2.5 text-xs">
              <Icon name="users" size="sm" />
              {{ t('admin.users.title') }}
            </router-link>
            <router-link to="/admin/groups" class="atelier-btn atelier-btn-ghost h-8 px-2.5 text-xs">
              <Icon name="grid" size="sm" />
              {{ t('admin.groups.title') }}
            </router-link>
            <router-link to="/admin/usage" class="atelier-btn atelier-btn-ghost h-8 px-2.5 text-xs">
              <Icon name="chart" size="sm" />
              {{ t('admin.usage.title') }}
            </router-link>
            <router-link to="/admin/audit-logs" class="atelier-btn atelier-btn-ghost h-8 px-2.5 text-xs">
              <Icon name="shield" size="sm" />
              {{ t('admin.audit.title') }}
            </router-link>
            <button
              v-if="canUseBatchImage"
              type="button"
              class="atelier-btn atelier-btn-ghost ml-auto h-8 px-2.5 text-xs"
              @click="router.push('/batch-image')"
            >
              <Icon name="sparkles" size="sm" />
              {{ t('admin.dashboard.batchImage') }}
            </button>
          </nav>
        </section>

        <!-- Right: Upstream Balance & Consumption Panel -->
        <section class="atelier-surface flex flex-col justify-between p-5 lg:col-span-5">
          <div>
            <div class="flex items-start justify-between gap-3 border-b border-[var(--atelier-border)] pb-3.5">
              <div>
                <p class="text-xs font-medium text-[var(--atelier-ink-muted)]">
                  {{ t('admin.dashboard.upstreamBalance.title') }}
                </p>
                <p class="mt-1 font-mono text-2xl font-semibold tracking-tight tabular-nums text-[var(--atelier-ink)]">
                  ${{ formatCost(upstreamBalances?.total || 0) }}
                </p>
                <p class="mt-0.5 text-xs text-[var(--atelier-ink-muted)]">
                  {{ t('admin.dashboard.upstreamBalance.autoRefresh') }}
                </p>
              </div>
              <UiButton variant="ghost" size="sm" @click="router.push('/admin/groups')">
                {{ t('admin.dashboard.groupPricing') }}
                <Icon name="chevronRight" size="xs" />
              </UiButton>
            </div>

            <div v-if="upstreamBalancesLoading" class="flex justify-center py-8">
              <LoadingSpinner size="sm" />
            </div>
            <template v-else-if="upstreamBalances?.items?.length">
              <ul class="mt-3 max-h-48 divide-y divide-[var(--atelier-border-subtle)] overflow-y-auto pr-1">
                <li
                  v-for="item in upstreamBalances.items"
                  :key="item.id"
                  class="flex items-center justify-between gap-3 py-2.5"
                >
                  <div class="flex min-w-0 items-center gap-2">
                    <UiStatusDot :tone="item.error ? 'err' : 'ok'" />
                    <div class="min-w-0">
                      <p class="truncate text-sm font-medium text-[var(--atelier-ink)]">
                        {{ item.group_name || item.name }}
                      </p>
                      <p class="truncate text-xs text-[var(--atelier-ink-muted)]">{{ item.name }}</p>
                    </div>
                  </div>
                  <div class="shrink-0 text-right">
                    <span
                      v-if="item.error"
                      class="rounded-full bg-[var(--atelier-err-bg)] px-2 py-0.5 text-xs font-medium text-[var(--atelier-err)]"
                    >
                      {{ t('admin.dashboard.upstreamBalance.error') }}
                    </span>
                    <span
                      v-else
                      class="font-mono text-sm font-semibold tabular-nums text-[var(--atelier-ink)]"
                    >
                      ${{ formatCost(item.balance) }}
                    </span>
                  </div>
                </li>
              </ul>

              <div class="mt-4 grid grid-cols-2 gap-2.5 border-t border-[var(--atelier-border)] pt-4">
                <div
                  v-for="entry in upstreamConsumptionCards"
                  :key="entry.label"
                  class="atelier-surface-subtle px-3 py-2.5"
                >
                  <p class="text-[11px] font-medium text-[var(--atelier-ink-muted)]">{{ entry.label }}</p>
                  <p class="mt-0.5 font-mono text-sm font-semibold tabular-nums text-[var(--atelier-ink)]">
                    ${{ formatCost(entry.value) }}
                  </p>
                </div>
              </div>
            </template>
            <p v-else class="py-8 text-center text-sm text-[var(--atelier-ink-muted)]">
              {{ t('admin.dashboard.upstreamBalance.empty') }}
            </p>
          </div>
        </section>
      </div>

      <!-- Analytics Filter Bar -->
      <section class="atelier-surface p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ t('admin.dashboard.timeRange') }}
            </span>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="onDateRangeChange"
            />
          </div>

          <div class="ml-auto flex flex-wrap items-center gap-2.5">
            <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
              {{ t('admin.dashboard.granularity') }}
            </span>
            <UiSegmented
              :model-value="granularity"
              :options="granularityOptions"
              size="sm"
              :aria-label="t('admin.dashboard.granularity')"
              @update:model-value="onGranularityChange"
            />
          </div>
        </div>

        <div class="mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--atelier-border-subtle)] pt-3">
          <span class="text-xs text-[var(--atelier-ink-muted)]">
            {{ zh ? '以下图表跟随所选时间；顶部指标保持今日或累计口径。' : 'Charts follow this range; headline metrics retain their today or all-time scope.' }}
          </span>
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-2" data-test="dashboard-drill-model">
              <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('usage.model') }}</span>
              <div class="w-48">
                <UiSelect
                  :model-value="drillModel"
                  :options="drillModelOptions"
                  size="sm"
                  :aria-label="t('usage.model')"
                  @update:model-value="(val) => (drillModel = String(val ?? ''))"
                />
              </div>
            </div>
            <UiButton
              variant="secondary"
              size="sm"
              data-test="usage-range-drill"
              @click="goToUsage(drillModel)"
            >
              {{ zh ? '查看区间明细' : 'View requests in range' }}
              <Icon name="arrowRight" size="xs" />
            </UiButton>
          </div>
        </div>
      </section>

      <!-- Asymmetric Charts Grid (5 / 7 split) -->
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
        <section class="atelier-surface flex flex-col justify-between p-5 lg:col-span-5">
          <ModelDistributionChart
            :model-stats="modelStats"
            :enable-ranking-view="true"
            :ranking-items="rankingItems"
            :ranking-total-actual-cost="rankingTotalActualCost"
            :ranking-total-requests="rankingTotalRequests"
            :ranking-total-tokens="rankingTotalTokens"
            :loading="chartsLoading"
            :ranking-loading="rankingLoading"
            :ranking-error="rankingError"
            :start-date="startDate"
            :end-date="endDate"
            @ranking-click="goToUserUsage"
          />

          <details
            v-if="modelStats.length"
            class="mt-4 border-t border-[var(--atelier-border-subtle)] pt-3"
          >
            <summary class="flex cursor-pointer list-none items-center justify-between text-xs font-medium text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)]">
              <span>{{ zh ? '按模型查看请求明细' : 'Explore requests by model' }}</span>
              <span class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-secondary)]">
                {{ modelStats.length }}
              </span>
            </summary>
            <ul class="mt-2.5 max-h-56 divide-y divide-[var(--atelier-border-subtle)] overflow-y-auto">
              <li v-for="model in modelStats" :key="model.model">
                <button
                  type="button"
                  data-test="model-usage-drill"
                  class="flex w-full items-center justify-between gap-2 rounded-lg px-2 py-2 text-left text-xs transition-colors hover:bg-[var(--atelier-hover)]"
                  @click="goToUsage(model.model)"
                >
                  <span class="flex min-w-0 items-center gap-2 text-[var(--atelier-ink)]">
                    <BrandModelIcon :model="model.model" size="xs" />
                    <span class="truncate font-mono">{{ model.model }}</span>
                  </span>
                  <span class="flex shrink-0 items-center gap-1.5 font-mono tabular-nums text-[var(--atelier-ink-muted)]">
                    {{ formatNumber(model.requests) }} {{ zh ? '次' : 'requests' }}
                    <Icon name="arrowRight" size="xs" />
                  </span>
                </button>
              </li>
            </ul>
          </details>
        </section>

        <section class="atelier-surface flex flex-col justify-between p-5 lg:col-span-7">
          <div class="min-h-0 flex-1">
            <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" fill />
          </div>

          <div
            v-if="trendDays.length"
            class="mt-4 flex flex-wrap items-center justify-end gap-2.5 border-t border-[var(--atelier-border-subtle)] pt-3"
          >
            <div class="flex items-center gap-2" data-test="dashboard-drill-date">
              <span class="text-xs font-medium text-[var(--atelier-ink-secondary)]">
                {{ zh ? '选择日期' : 'Choose a day' }}
              </span>
              <div class="w-40">
                <UiSelect
                  :model-value="drillDate"
                  :options="drillDateOptions"
                  size="sm"
                  :aria-label="zh ? '选择日期' : 'Choose a day'"
                  @update:model-value="(val) => (drillDate = String(val ?? ''))"
                />
              </div>
            </div>
            <UiButton
              variant="secondary"
              size="sm"
              :disabled="!drillDate"
              data-test="trend-usage-drill"
              @click="goToUsage(drillModel, drillDate)"
            >
              {{ zh ? '查看当天明细' : 'View this day' }}
              <Icon name="arrowRight" size="xs" />
            </UiButton>
          </div>
        </section>
      </div>

      <!-- User Usage Trend (Full Width) -->
      <section class="atelier-surface p-5">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h3 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ t('admin.dashboard.recentUsage') }} (Top 12)
          </h3>
          <UiSegmented
            :model-value="userTrendMetric"
            :options="userTrendMetricOptions"
            size="sm"
            :aria-label="t('admin.dashboard.recentUsage')"
            @update:model-value="setUserTrendMetric"
          />
        </div>
        <div class="h-64">
          <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
            <LoadingSpinner size="md" />
          </div>
          <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
          <div
            v-else
            class="flex h-full items-center justify-center text-sm text-[var(--atelier-ink-muted)]"
          >
            {{ t('admin.dashboard.noDataAvailable') }}
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import type { UpstreamBalanceSummary } from '@/api/admin/dashboard'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import {
  BrandModelIcon,
  UiButton,
  UiMetricRibbon,
  UiSegmented,
  UiSelect,
  UiStatusDot,
  type RibbonMetric,
  type SegmentedOption,
  type UiSelectOption
} from '@/ui'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const chartTheme = useChartTheme()
const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const drillModel = ref('')
const drillDate = ref('')
const trendDays = computed(() => [...new Set(trendData.value.map(item => item.date.slice(0, 10)).filter(day => /^\d{4}-\d{2}-\d{2}$/.test(day)))])
watch(trendDays, days => { if (!days.includes(drillDate.value)) drillDate.value = days[0] || '' })
watch(modelStats, models => { if (!models.some(model => model.model === drillModel.value)) drillModel.value = '' })
const userTrend = ref<UserUsageTrendPoint[]>([])
const userTrendMetric = ref<'tokens' | 'actual_cost'>('tokens')
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
const upstreamBalances = ref<UpstreamBalanceSummary | null>(null)
const upstreamBalancesLoading = ref(false)
let upstreamBalanceLoadSeq = 0
let upstreamBalanceRefreshTimer: ReturnType<typeof setInterval> | null = null
const upstreamConsumptionCards = computed(() => [
  { label: t('admin.dashboard.upstreamBalance.last24h'), value: upstreamBalances.value?.consumption?.last_24h || 0 },
  { label: t('admin.dashboard.upstreamBalance.yesterday'), value: upstreamBalances.value?.consumption?.yesterday || 0 },
  { label: t('admin.dashboard.upstreamBalance.today'), value: upstreamBalances.value?.consumption?.today || 0 },
  { label: t('admin.dashboard.upstreamBalance.totalConsumption'), value: upstreamBalances.value?.consumption?.total || 0 }
])
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

const granularityOptions = computed<SegmentedOption<'day' | 'hour'>[]>(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

const userTrendMetricOptions = computed<SegmentedOption<'tokens' | 'actual_cost'>[]>(() => [
  { value: 'tokens', label: t('admin.dashboard.tokens') },
  { value: 'actual_cost', label: t('admin.dashboard.actualSpending') }
])

const drillModelOptions = computed<UiSelectOption[]>(() => [
  { value: '', label: zh.value ? '全部模型' : 'All models' },
  ...modelStats.value.map((m) => ({ value: m.model, label: m.model }))
])

const drillDateOptions = computed<UiSelectOption[]>(() =>
  trendDays.value.map((day) => ({ value: day, label: day }))
)

const onGranularityChange = (val: 'day' | 'hour') => {
  if (granularity.value === val) return
  granularity.value = val
  loadChartData()
}

// Ribbon headline metrics
const ribbonMetrics = computed<RibbonMetric[]>(() => {
  if (!stats.value) return []
  return [
    {
      label: t('admin.dashboard.todayTokens'),
      value: formatTokens(stats.value.today_tokens),
      sub: `$${formatCost(stats.value.today_actual_cost)} / $${formatCost(stats.value.today_account_cost)}`
    },
    {
      label: t('admin.dashboard.performance'),
      value: `${formatTokens(stats.value.rpm)} RPM`,
      sub: `${formatTokens(stats.value.tpm)} TPM`,
      dot: 'info'
    },
    {
      label: t('admin.dashboard.avgResponse'),
      value: formatDuration(stats.value.average_duration_ms),
      sub: `${stats.value.active_users} ${t('admin.dashboard.activeUsers')}`
    },
    {
      label: t('admin.dashboard.accounts'),
      value: `${stats.value.normal_accounts}/${stats.value.total_accounts}`,
      dot: stats.value.error_accounts > 0 ? 'err' : 'ok',
      delta: stats.value.error_accounts > 0 ? `${stats.value.error_accounts} ${t('common.error')}` : undefined,
      deltaTone: stats.value.error_accounts > 0 ? 'err' : 'neutral'
    }
  ]
})

// Dark mode detection
const userTrendPalette = useChartPalette()

// Chart colors
const chartColors = computed(() => ({
  text: chartTheme.value.text,
  grid: chartTheme.value.grid
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  animation: false as const,
  maintainAspectRatio: false,
  elements: {
    point: {
      radius: 2.5,
      hoverRadius: 5,
      hitRadius: 12,
      borderWidth: 2,
      backgroundColor: chartTheme.value.pointBorder
    },
    line: { borderWidth: 2 }
  },
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
        boxWidth: 8,
        padding: 15,
        font: {
          size: 12.5
        }
      }
    },
    tooltip: {
      ...chartTheme.value.tooltip,
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatUserTrendValue(Number(context.raw))}`
        }
      }
    }
  },
  scales: {
    x: {
      border: {
        display: false
      },
      grid: {
        display: false
      },
      ticks: {
        color: chartColors.value.text,
        maxRotation: 0,
        maxTicksLimit: 8,
        font: {
          size: 12
        }
      }
    },
    y: {
      beginAtZero: true,
      border: {
        display: false
      },
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 12
        },
        callback: (value: string | number) => formatUserTrendValue(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, userTrendMetric.value === 'tokens' ? point.tokens : point.actual_cost)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = userTrendPalette.value
  const pointBorder = chartTheme.value.pointBorder
  const pointRadius = sortedDates.length > 48 ? 0 : 2.5

  const datasets = Array.from(userGroups.values()).map((group, idx) => {
    const borderColor = colors[idx % colors.length]
    return {
      label: group.name,
      data: sortedDates.map((date) => group.data.get(date) || 0),
      borderColor,
      backgroundColor: `${borderColor}20`,
      borderWidth: 2,
      pointRadius,
      pointHoverRadius: 5,
      pointBackgroundColor: pointBorder,
      pointBorderColor: borderColor,
      pointBorderWidth: 2,
      fill: false,
      tension: 0.3
    }
  })

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatUserTrendValue = (value: number): string =>
  userTrendMetric.value === 'tokens' ? formatTokens(value) : `$${formatCost(value)}`

const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

const goToErrorAccounts = () => {
  void router.push({ path: '/admin/accounts', query: { status: 'error' } })
}

const goToUsage = (model = '', day = '') => {
  void router.push({
    path: '/admin/usage',
    query: {
      start_date: day || startDate.value,
      end_date: day || endDate.value,
      ...(model ? { model } : {})
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const setUserTrendMetric = (metric: 'tokens' | 'actual_cost') => {
  if (userTrendMetric.value === metric) return
  userTrendMetric.value = metric
  userTrend.value = []
  loadUsersTrend()
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12,
      metric: userTrendMetric.value
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadUpstreamBalances = async () => {
  const currentSeq = ++upstreamBalanceLoadSeq
  upstreamBalancesLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUpstreamBalances()
    if (currentSeq === upstreamBalanceLoadSeq) upstreamBalances.value = response
  } catch (error) {
    if (currentSeq === upstreamBalanceLoadSeq) upstreamBalances.value = null
    console.error('Error loading upstream balances:', error)
  } finally {
    if (currentSeq === upstreamBalanceLoadSeq) upstreamBalancesLoading.value = false
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking(),
    loadUpstreamBalances()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  void refreshBatchImageAccess()
  loadDashboardStats()
  upstreamBalanceRefreshTimer = setInterval(loadUpstreamBalances, 60 * 1000)
})

onUnmounted(() => {
  if (upstreamBalanceRefreshTimer) clearInterval(upstreamBalanceRefreshTimer)
})
</script>
