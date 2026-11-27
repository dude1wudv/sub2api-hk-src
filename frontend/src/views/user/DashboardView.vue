<template>
  <div class="space-y-6">
    <!-- Minimal Page Header -->
    <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2.5">
          <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
            {{ zh ? '工作台概览' : 'Workspace Overview' }}
          </h1>
          <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
            {{ formattedToday }}
          </span>
        </div>
        <p class="mt-0.5 text-sm text-[var(--atelier-ink-secondary)]">
          {{ zh ? '实时掌握账户余额、调用趋势、主力模型分布与常用密钥。' : 'Monitor balance, usage trends, model mix, and frequent API keys.' }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <UiButton variant="ghost" size="sm" :loading="loading" @click="refreshAll">
          <Icon name="refresh" size="xs" aria-hidden="true" />
          {{ t('common.refresh') }}
        </UiButton>
        <UiButton variant="secondary" size="sm" @click="router.push('/playground')">
          <Icon name="play" size="xs" aria-hidden="true" />
          {{ zh ? 'API 调试台' : 'Playground' }}
        </UiButton>
        <UiButton variant="primary" size="sm" @click="router.push('/keys?create=1')">
          <Icon name="plus" size="xs" aria-hidden="true" />
          {{ zh ? '创建密钥' : 'Create key' }}
        </UiButton>
      </div>
    </header>

    <!-- Compact Base URL Quick-Connect Strip -->
    <div
      v-if="openAIBase"
      class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-4 py-2.5"
    >
      <div class="flex min-w-0 flex-wrap items-center gap-2.5">
        <span class="rounded-md bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono text-[11px] font-medium text-[var(--atelier-ink-secondary)]">
          Base URL
        </span>
        <code class="truncate font-mono text-xs text-[var(--atelier-ink)] sm:text-[13px]">
          {{ openAIBase }}
        </code>
      </div>
      <div class="flex items-center gap-1.5">
        <UiButton variant="ghost" size="sm" @click="copyBaseUrl">
          <Icon name="copy" size="xs" aria-hidden="true" />
          {{ zh ? '复制地址' : 'Copy URL' }}
        </UiButton>
        <UiButton variant="secondary" size="sm" @click="router.push('/keys?quickstart=1')">
          <Icon name="bolt" size="xs" aria-hidden="true" />
          {{ zh ? '一键接入配置' : 'Quickstart' }}
        </UiButton>
      </div>
    </div>

    <!-- Error Alert -->
    <div
      v-if="error"
      role="alert"
      class="flex items-center justify-between gap-3 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]"
    >
      <span>{{ error }}</span>
      <UiButton variant="secondary" size="sm" :disabled="loading" @click="refreshAll">
        {{ zh ? '重试' : 'Retry' }}
      </UiButton>
    </div>

    <!-- Initial Loading Skeleton -->
    <div
      v-if="loading && !stats"
      class="atelier-surface flex items-center justify-center py-14"
      role="status"
    >
      <LoadingSpinner />
    </div>

    <template v-if="stats">
      <!-- Top: Single-Row Borderless Balance & Core Metrics Ribbon (NO 4-equal-grid KPI boxes) -->
      <UiMetricRibbon
        :primary-label="ribbonPrimaryLabel"
        :primary-value="ribbonPrimaryValue"
        :primary-meta="ribbonPrimaryMeta"
        :metrics="ribbonMetrics"
        :show-dots="true"
      >
        <template v-if="!authStore.isSimpleMode" #primaryAction>
          <div class="flex items-center gap-2">
            <UiButton variant="primary" size="sm" @click="router.push('/redeem')">
              {{ zh ? '兑换卡密' : 'Redeem' }}
            </UiButton>
            <UiButton variant="secondary" size="sm" @click="router.push('/purchase')">
              {{ zh ? '充值' : 'Top up' }}
            </UiButton>
          </div>
        </template>
      </UiMetricRibbon>

      <!-- Onboarding Checklist for Zero-Usage New Accounts -->
      <UserOnboardingChecklist
        v-if="stats.total_requests === 0"
        :key-count="stats.total_api_keys"
      />

      <!-- Middle: Asymmetric 7:5 Two-Column Layout -->
      <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-12">
        <!-- Left (7 cols): Usage Trend Chart -->
        <section class="atelier-surface flex min-w-0 flex-col p-5 lg:col-span-7">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
                {{ zh ? '用量趋势' : 'Usage trend' }}
              </h2>
              <p class="mt-0.5 font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">
                {{ startDate }} — {{ endDate }}
                <span v-if="trendSummaryText" class="ml-2 text-[var(--atelier-ink-secondary)]">
                  · {{ trendSummaryText }}
                </span>
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <UiSegmented
                v-model="rangePreset"
                :options="rangeOptions"
                size="sm"
                :aria-label="zh ? '时间范围' : 'Time range'"
                @update:model-value="onRangeChange"
              />
              <UiSegmented
                v-model="chartMetric"
                :options="metricOptions"
                size="sm"
                :aria-label="zh ? '指标维度' : 'Metric'"
              />
            </div>
          </div>

          <div
            v-if="chartError"
            role="alert"
            class="mt-4 flex items-center justify-between gap-2 rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-3 py-2 text-xs text-[var(--atelier-err)]"
          >
            <span>{{ chartError }}</span>
            <UiButton variant="ghost" size="sm" @click="loadCharts">
              {{ zh ? '重试' : 'Retry' }}
            </UiButton>
          </div>

          <div class="relative mt-5 h-[260px] w-full flex-1" :aria-busy="loadingCharts">
            <div
              v-if="loadingCharts && !trendData.length"
              class="flex h-full items-center justify-center"
              role="status"
            >
              <LoadingSpinner />
            </div>
            <div
              v-else-if="!trendData.length"
              class="flex h-full flex-col items-center justify-center text-center text-sm text-[var(--atelier-ink-muted)]"
            >
              <p>{{ t('dashboard.noDataAvailable') }}</p>
            </div>
            <Line
              v-else
              :data="chartJsData"
              :options="chartJsOptions"
              role="img"
              :aria-label="zh ? '所选时间内的用量趋势，点击数据点查看明细' : 'Usage trend over the selected period. Click a point to view details.'"
            />
          </div>

          <footer class="mt-4 flex flex-wrap items-center justify-between gap-2 border-t border-[var(--atelier-border-subtle)] pt-3 text-xs text-[var(--atelier-ink-muted)]">
            <span class="font-mono tabular-nums">
              {{ zh ? '点击图表数据点可下钻明细' : 'Click any chart point to drill down' }}
            </span>
            <RouterLink
              to="/usage"
              class="inline-flex items-center gap-1 font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:text-[var(--atelier-ink)]"
            >
              {{ zh ? '查看完整用量明细' : 'View full usage' }}
              <Icon name="arrowRight" size="xs" aria-hidden="true" />
            </RouterLink>
          </footer>
        </section>

        <!-- Right (5 cols): Active Model Usage Breakdown -->
        <section class="atelier-surface flex min-w-0 flex-col p-5 lg:col-span-5">
          <div class="flex items-center justify-between gap-2">
            <div>
              <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
                {{ zh ? '主力模型分布' : 'Active models' }}
              </h2>
              <p class="mt-0.5 text-xs text-[var(--atelier-ink-muted)]">
                {{ zh ? '按实际费用占比 · 点击下钻' : 'By actual spend share · Click to inspect' }}
              </p>
            </div>
            <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">
              {{ formatUsageMoney(totalModelCost, 4) }}
            </span>
          </div>

          <div
            v-if="loadingCharts && !modelStats.length"
            class="flex flex-1 items-center justify-center py-10"
            role="status"
          >
            <LoadingSpinner />
          </div>
          <div
            v-else-if="!rankedModels.length"
            class="flex flex-1 items-center justify-center py-10 text-sm text-[var(--atelier-ink-muted)]"
          >
            {{ t('dashboard.noDataAvailable') }}
          </div>
          <ul v-else class="mt-4 flex-1 space-y-3.5">
            <li v-for="item in rankedModels" :key="item.model">
              <button
                type="button"
                class="group block w-full rounded-lg p-1.5 -mx-1.5 text-left transition-colors hover:bg-[var(--atelier-hover)]"
                @click="drilldown({ start: startDate, end: endDate, model: item.model })"
              >
                <div class="flex items-center justify-between gap-2">
                  <div class="flex min-w-0 items-center gap-2">
                    <BrandModelIcon :model="item.model" size="sm" />
                    <span class="truncate text-sm font-medium text-[var(--atelier-ink)]">
                      {{ item.model }}
                    </span>
                  </div>
                  <div class="shrink-0 font-mono text-xs tabular-nums">
                    <span class="font-medium text-[var(--atelier-ink)]">
                      {{ formatUsageMoney(item.actual_cost, 2) }}
                    </span>
                    <span class="text-[var(--atelier-ink-muted)]">
                      · {{ item.sharePct }}%
                    </span>
                  </div>
                </div>
                <div class="mt-2 flex items-center gap-3">
                  <UiProgressBar :value="item.sharePct" class="flex-1" />
                  <span class="w-16 shrink-0 text-right font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
                    {{ formatCompact(item.requests) }} {{ zh ? '次' : 'req' }}
                  </span>
                </div>
              </button>
            </li>
          </ul>

          <!-- Compact Platform Quota Watermarks (when configured) -->
          <div
            v-if="!authStore.isSimpleMode && platformQuotaRows.length"
            class="mt-5 border-t border-[var(--atelier-border-subtle)] pt-4"
          >
            <h3 class="text-xs font-medium text-[var(--atelier-ink-muted)]">
              {{ t('dashboard.platformQuota.title') }}
            </h3>
            <div class="mt-2.5 space-y-2.5">
              <div
                v-for="row in platformQuotaRows"
                :key="row.key"
                class="space-y-1"
              >
                <div class="flex items-center justify-between gap-2 text-xs">
                  <span class="flex min-w-0 items-center gap-1.5 font-medium text-[var(--atelier-ink)]">
                    <BrandPlatformIcon :platform="row.platform" size="xs" />
                    <span class="truncate">{{ row.label }}</span>
                  </span>
                  <span class="shrink-0 font-mono tabular-nums">
                    <span class="text-[var(--atelier-ink)]">${{ row.used.toFixed(2) }}</span>
                    <span class="text-[var(--atelier-ink-muted)]"> / ${{ row.limit.toFixed(2) }}</span>
                  </span>
                </div>
                <UiProgressBar
                  :value="row.pct"
                  :tone="row.pct >= 100 ? 'err' : row.pct >= 80 ? 'warn' : 'ink'"
                />
              </div>
            </div>
          </div>
        </section>
      </div>

      <!-- Bottom: Separated by >=32px gap & hairline divider -->
      <div class="mt-8 grid grid-cols-1 gap-6 border-t border-[var(--atelier-border)] pt-8 lg:grid-cols-12">
        <!-- Bottom Left: Quick-Copy Frequent API Keys Strip (up to 3 keys) -->
        <section class="min-w-0 lg:col-span-6">
          <div class="mb-3 flex items-center justify-between gap-2">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ zh ? '常用 API 密钥' : 'Frequent API keys' }}
            </h2>
            <RouterLink
              to="/keys"
              class="inline-flex items-center gap-1 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:text-[var(--atelier-ink)]"
            >
              {{ zh ? '管理全部密钥' : 'Manage all keys' }}
              <Icon name="arrowRight" size="xs" aria-hidden="true" />
            </RouterLink>
          </div>

          <div
            v-if="loadingKeys && !quickKeys.length"
            class="atelier-surface flex items-center justify-center py-10"
            role="status"
          >
            <LoadingSpinner />
          </div>
          <div
            v-else-if="!quickKeys.length"
            class="atelier-surface relative overflow-hidden px-5 py-8 text-center"
          >
            <DotMatrixBackdrop mask="edges" />
            <div class="relative z-10 flex flex-col items-center gap-2.5">
              <p class="text-sm font-medium text-[var(--atelier-ink)]">
                {{ zh ? '暂无活跃 API 密钥' : 'No active API keys yet' }}
              </p>
              <p class="text-xs text-[var(--atelier-ink-muted)]">
                {{ zh ? '创建一个密钥，立即接入 OpenAI、Claude 或 Gemini 客户端。' : 'Create a key to connect OpenAI, Claude, or Gemini clients.' }}
              </p>
              <UiButton variant="primary" size="sm" class="mt-1" @click="router.push('/keys?create=1')">
                <Icon name="plus" size="xs" aria-hidden="true" />
                {{ zh ? '创建密钥' : 'Create key' }}
              </UiButton>
            </div>
          </div>
          <ul
            v-else
            class="atelier-surface divide-y divide-[var(--atelier-border-subtle)] overflow-hidden"
          >
            <li
              v-for="apiKey in quickKeys"
              :key="apiKey.id"
              class="flex items-center justify-between gap-3 px-4 py-3 transition-colors hover:bg-[var(--atelier-hover)]"
            >
              <div class="flex min-w-0 items-center gap-3">
                <BrandPlatformIcon :platform="apiKey.group?.platform" size="sm" />
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="truncate text-sm font-medium text-[var(--atelier-ink)]">
                      {{ apiKey.name }}
                    </span>
                    <UiStatusDot :tone="apiKey.status === 'active' ? 'ok' : 'neutral'" />
                    <span
                      v-if="apiKey.group?.name"
                      class="hidden truncate text-xs text-[var(--atelier-ink-muted)] sm:inline"
                    >
                      {{ apiKey.group.name }}
                    </span>
                  </div>
                  <div class="mt-0.5 flex items-center gap-2 font-mono text-xs text-[var(--atelier-ink-muted)]">
                    <span class="truncate">{{ maskApiKey(apiKey.key) }}</span>
                    <span v-if="apiKey.quota > 0" class="tabular-nums">
                      · ${{ apiKey.quota_used.toFixed(2) }} / ${{ apiKey.quota.toFixed(2) }}
                    </span>
                  </div>
                </div>
              </div>
              <div class="flex shrink-0 items-center gap-1.5">
                <UiButton
                  variant="secondary"
                  size="sm"
                  @click="copyQuickKey(apiKey)"
                >
                  <Icon :name="copiedKeyId === apiKey.id ? 'check' : 'copy'" size="xs" aria-hidden="true" />
                  {{ copiedKeyId === apiKey.id ? (zh ? '已复制' : 'Copied') : (zh ? '复制' : 'Copy') }}
                </UiButton>
              </div>
            </li>
          </ul>
        </section>

        <!-- Bottom Right: Recent Activity / Announcements Feed -->
        <section class="min-w-0 lg:col-span-6">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ zh ? '近期动态与公告' : 'Recent activity & notices' }}
            </h2>
            <UiSegmented
              v-model="feedTab"
              :options="feedTabOptions"
              size="sm"
              :aria-label="zh ? '动态视图' : 'Activity view'"
            />
          </div>

          <!-- Recent Usage Activity -->
          <div v-if="feedTab === 'activity'" class="atelier-surface overflow-hidden">
            <div
              v-if="recentError"
              role="alert"
              class="flex items-center justify-between gap-2 px-4 py-3 text-xs text-[var(--atelier-err)]"
            >
              <span>{{ recentError }}</span>
              <UiButton variant="ghost" size="sm" @click="loadRecent">
                {{ zh ? '重试' : 'Retry' }}
              </UiButton>
            </div>
            <div
              v-else-if="loadingUsage && !recentUsage.length"
              class="flex items-center justify-center py-10"
              role="status"
            >
              <LoadingSpinner />
            </div>
            <div
              v-else-if="!recentUsage.length"
              class="px-4 py-8 text-center text-xs text-[var(--atelier-ink-muted)]"
            >
              {{ t('dashboard.noUsageRecords') }}
            </div>
            <ul v-else class="divide-y divide-[var(--atelier-border-subtle)]">
              <li v-for="log in recentUsage" :key="log.id">
                <RouterLink
                  :to="usageLocation(log)"
                  class="flex items-center justify-between gap-3 px-4 py-3 transition-colors hover:bg-[var(--atelier-hover)]"
                >
                  <div class="flex min-w-0 items-center gap-2.5">
                    <UiStatusDot tone="ok" />
                    <BrandModelIcon :model="log.model" size="xs" />
                    <div class="min-w-0">
                      <p class="truncate text-sm font-medium text-[var(--atelier-ink)]">
                        {{ log.model }}
                      </p>
                      <p class="font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
                        {{ formatDateTime(log.created_at) }}
                      </p>
                    </div>
                  </div>
                  <div class="shrink-0 text-right font-mono tabular-nums">
                    <p class="text-xs font-medium text-[var(--atelier-ink)]">
                      {{ formatUsageMoney(log.actual_cost, 4) }}
                    </p>
                    <p class="text-[11px] text-[var(--atelier-ink-muted)]">
                      {{ formatCompact(logTotalTokens(log)) }} tok
                    </p>
                  </div>
                </RouterLink>
              </li>
            </ul>
          </div>

          <!-- System Announcements Feed -->
          <div v-else class="atelier-surface overflow-hidden">
            <div
              v-if="loadingAnnouncements && !announcements.length"
              class="flex items-center justify-center py-10"
              role="status"
            >
              <LoadingSpinner />
            </div>
            <div
              v-else-if="!announcements.length"
              class="px-4 py-8 text-center text-xs text-[var(--atelier-ink-muted)]"
            >
              {{ zh ? '暂无系统公告' : 'No announcements right now' }}
            </div>
            <ul v-else class="divide-y divide-[var(--atelier-border-subtle)]">
              <li v-for="item in announcements" :key="item.id">
                <button
                  type="button"
                  class="block w-full px-4 py-3 text-left transition-colors hover:bg-[var(--atelier-hover)]"
                  @click="toggleAnnouncement(item)"
                >
                  <div class="flex items-center justify-between gap-3">
                    <div class="flex min-w-0 items-center gap-2.5">
                      <UiStatusDot :tone="item.read_at ? 'neutral' : 'info'" />
                      <span
                        class="truncate text-sm text-[var(--atelier-ink)]"
                        :class="item.read_at ? 'font-normal' : 'font-medium'"
                      >
                        {{ item.title }}
                      </span>
                    </div>
                    <span class="shrink-0 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
                      {{ formatDateTime(item.created_at) }}
                    </span>
                  </div>
                  <p
                    v-if="expandedAnnouncementId === item.id"
                    class="mt-2 whitespace-pre-line pl-4 text-xs leading-relaxed text-[var(--atelier-ink-secondary)]"
                  >
                    {{ item.content }}
                  </p>
                </button>
              </li>
            </ul>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Filler,
  type ChartOptions,
  type ScriptableContext
} from 'chart.js'
import {
  UiMetricRibbon,
  UiButton,
  UiSegmented,
  UiProgressBar,
  UiStatusDot,
  BrandModelIcon,
  BrandPlatformIcon,
  DotMatrixBackdrop,
  type RibbonMetric,
  type SegmentedOption
} from '@/ui'
import { usageDrilldownQuery, validUsageDate, type UsageDrilldown } from '@/views/portal/usageRoute'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { keysAPI } from '@/api/keys'
import announcementsAPI from '@/api/announcements'
import { getMyPlatformQuotas } from '@/api/user'
import { useColorMode } from '@/composables/useColorMode'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useClipboard } from '@/composables/useClipboard'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserOnboardingChecklist from '@/components/user/dashboard/UserOnboardingChecklist.vue'
import { formatDateLocalInput, formatDateTime } from '@/utils/format'
import { formatUsageMoney } from '@/utils/portalUsage'
import { maskApiKey } from '@/utils/maskApiKey'
import type {
  ApiKey,
  UsageLog,
  TrendDataPoint,
  ModelStat,
  PlatformQuotaItem,
  UserAnnouncement
} from '@/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Filler)

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const authStore = useAuthStore()
const appStore = useAppStore()
const router = useRouter()
const { isDark } = useColorMode()
const { openAIBase } = usePatrickApiBase()
const { copyToClipboard } = useClipboard()

const user = computed(() => authStore.user)
const formattedToday = computed(() =>
  new Date().toLocaleDateString(locale.value, { month: 'short', day: 'numeric', weekday: 'short' })
)

const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const loadingKeys = ref(false)
const loadingAnnouncements = ref(false)
const error = ref('')
const chartError = ref('')
const recentError = ref('')

const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const quickKeys = ref<ApiKey[]>([])
const announcements = ref<UserAnnouncement[]>([])
const expandedAnnouncementId = ref<number | null>(null)
const copiedKeyId = ref<number | null>(null)

type RangePreset = '24h' | '7d' | '30d'
type ChartMetric = 'cost' | 'tokens'
type FeedTab = 'activity' | 'notices'

const rangePreset = ref<RangePreset>('7d')
const chartMetric = ref<ChartMetric>('cost')
const feedTab = ref<FeedTab>('activity')

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref<'day' | 'hour'>('day')

let chartRequest = 0
let disposed = false
let copyResetTimer: ReturnType<typeof setTimeout> | undefined
const today = formatDateLocalInput(new Date())

const rangeOptions = computed<SegmentedOption<RangePreset>[]>(() => [
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' }
])

const metricOptions = computed<SegmentedOption<ChartMetric>[]>(() => [
  { value: 'cost', label: zh.value ? '费用' : 'Cost' },
  { value: 'tokens', label: 'Tokens' }
])

const unreadNoticeCount = computed(() => announcements.value.filter((a) => !a.read_at).length)

const feedTabOptions = computed<SegmentedOption<FeedTab>[]>(() => [
  { value: 'activity', label: zh.value ? '最近请求' : 'Activity' },
  {
    value: 'notices',
    label: unreadNoticeCount.value > 0
      ? (zh.value ? `系统公告 (${unreadNoticeCount.value})` : `Notices (${unreadNoticeCount.value})`)
      : (zh.value ? '系统公告' : 'Notices')
  }
])

function onRangeChange(next: RangePreset) {
  const now = Date.now()
  if (next === '24h') {
    startDate.value = formatDateLocalInput(new Date(now - 86400000))
    endDate.value = formatDateLocalInput(new Date(now))
    granularity.value = 'hour'
  } else if (next === '30d') {
    startDate.value = formatDateLocalInput(new Date(now - 29 * 86400000))
    endDate.value = formatDateLocalInput(new Date(now))
    granularity.value = 'day'
  } else {
    startDate.value = formatDateLocalInput(new Date(now - 6 * 86400000))
    endDate.value = formatDateLocalInput(new Date(now))
    granularity.value = 'day'
  }
  void loadCharts()
}

function drilldown(value: UsageDrilldown) {
  void router.push({ path: '/usage', query: usageDrilldownQuery(value) })
}

async function copyBaseUrl() {
  if (openAIBase.value && (await copyToClipboard(openAIBase.value))) {
    appStore.showSuccess(zh.value ? '已复制 API Base URL' : 'API Base URL copied')
  }
}

async function copyQuickKey(apiKey: ApiKey) {
  if (await copyToClipboard(apiKey.key)) {
    copiedKeyId.value = apiKey.id
    appStore.showSuccess(zh.value ? `已复制密钥：${apiKey.name}` : `Copied key: ${apiKey.name}`)
    clearTimeout(copyResetTimer)
    copyResetTimer = setTimeout(() => {
      if (copiedKeyId.value === apiKey.id) copiedKeyId.value = null
    }, 1800)
  }
}

async function toggleAnnouncement(item: UserAnnouncement) {
  expandedAnnouncementId.value = expandedAnnouncementId.value === item.id ? null : item.id
  if (!item.read_at) {
    try {
      await announcementsAPI.markRead(item.id)
      item.read_at = new Date().toISOString()
    } catch {
      // Non-critical read marker failure
    }
  }
}

// Formatting helpers
function formatBalance(val: number): string {
  return Number.isFinite(val) ? val.toFixed(2) : '0.00'
}

function formatNumber(val: number): string {
  return Number.isFinite(val) ? val.toLocaleString() : '0'
}

function formatCompact(val: number): string {
  if (!Number.isFinite(val)) return '0'
  if (val >= 1_000_000) return `${(val / 1_000_000).toFixed(1)}M`
  if (val >= 1_000) return `${(val / 1_000).toFixed(1)}K`
  return val.toLocaleString()
}

function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}

function logTotalTokens(log: UsageLog): number {
  return (
    (log.input_tokens || 0) +
    (log.output_tokens || 0) +
    (log.cache_creation_tokens || 0) +
    (log.cache_read_tokens || 0)
  )
}

function usageLocation(log: UsageLog) {
  const day = formatDateLocalInput(new Date(log.created_at))
  return {
    path: '/usage',
    query: usageDrilldownQuery({ start: day, end: day, model: log.model, key: log.api_key_id })
  }
}

// Metric Ribbon Computed
const ribbonPrimaryLabel = computed(() =>
  !authStore.isSimpleMode
    ? (zh.value ? '可用余额' : 'Available balance')
    : (zh.value ? '今日请求' : 'Today requests')
)

const ribbonPrimaryValue = computed(() => {
  if (!authStore.isSimpleMode) {
    return `$${formatBalance(user.value?.balance || 0)}`
  }
  return formatNumber(stats.value?.today_requests || 0)
})

const ribbonPrimaryMeta = computed(() => {
  if (!stats.value) return ''
  if (!authStore.isSimpleMode) {
    const balance = user.value?.balance || 0
    const avgDaily = trendData.value.length
      ? trendData.value.reduce((s, d) => s + (d.actual_cost || 0), 0) / Math.max(1, rangePreset.value === '30d' ? 30 : rangePreset.value === '24h' ? 1 : 7)
      : stats.value.today_actual_cost
    if (balance > 0 && avgDaily > 0.01) {
      const days = Math.min(999, Math.floor(balance / avgDaily))
      return zh.value ? `约 ${days} 天续航` : `~${days}d runway`
    }
    return t('common.available')
  }
  return `${t('common.total')} ${formatNumber(stats.value.total_requests)}`
})

const ribbonMetrics = computed<RibbonMetric[]>(() => {
  const s = stats.value
  if (!s) return []
  const todayCache = (s.today_cache_creation_tokens || 0) + (s.today_cache_read_tokens || 0)
  const items: RibbonMetric[] = [
    {
      label: zh.value ? '今日消费' : "Today's spend",
      value: formatUsageMoney(s.today_actual_cost, 4),
      sub: s.today_cost > s.today_actual_cost
        ? `${zh.value ? '标准价' : 'Std'} ${formatUsageMoney(s.today_cost, 2)}`
        : `${zh.value ? '累计' : 'Total'} ${formatUsageMoney(s.total_actual_cost, 2)}`
    },
    {
      label: zh.value ? '今日调用 / Tokens' : 'Today calls / tokens',
      value: formatNumber(s.today_requests),
      sub: `${formatCompact(s.today_tokens)} tok` + (todayCache > 0 ? ` · ${formatCompact(todayCache)} ${zh.value ? '缓存' : 'cache'}` : '')
    },
    {
      label: zh.value ? '平均响应 / 吞吐' : 'Avg latency / rate',
      value: formatDuration(s.average_duration_ms),
      sub: `RPM ${formatCompact(s.rpm)} · TPM ${formatCompact(s.tpm)}`
    },
    {
      label: zh.value ? '密钥状态' : 'API keys',
      value: `${s.active_api_keys} / ${s.total_api_keys}`,
      dot: s.active_api_keys > 0 ? 'ok' : 'neutral',
      sub: s.active_api_keys > 0
        ? (zh.value ? '运行正常' : 'Operational')
        : (zh.value ? '未启用密钥' : 'No active keys')
    }
  ]
  return items
})

// Chart.js Data & Options (monotone smooth spline + whisper-light vertical gradient fill)
const trendSummaryText = computed(() => {
  if (!trendData.value.length) return ''
  if (chartMetric.value === 'cost') {
    const sum = trendData.value.reduce((acc, item) => acc + (item.actual_cost || 0), 0)
    return `${zh.value ? '区间合计' : 'Period'} ${formatUsageMoney(sum, 4)}`
  }
  const sumTokens = trendData.value.reduce((acc, item) => acc + (item.total_tokens || 0), 0)
  return `${zh.value ? '区间合计' : 'Period'} ${formatCompact(sumTokens)} tok`
})

const chartJsData = computed(() => {
  const dark = isDark.value
  const stroke = dark ? '#f5f5f7' : '#14171f'
  const values = trendData.value.map((item) =>
    chartMetric.value === 'cost' ? Number(item.actual_cost || 0) : Number(item.total_tokens || 0)
  )
  return {
    labels: trendData.value.map((item) =>
      granularity.value === 'hour' ? item.date.slice(11, 16) || item.date : item.date.slice(5)
    ),
    datasets: [
      {
        label: chartMetric.value === 'cost'
          ? (zh.value ? '实际消费 (USD)' : 'Actual spend (USD)')
          : (zh.value ? '总 Tokens' : 'Total tokens'),
        data: values,
        borderColor: stroke,
        borderWidth: 1.75,
        fill: true,
        cubicInterpolationMode: 'monotone' as const,
        tension: 0.35,
        pointRadius: trendData.value.length > 36 ? 0 : 2.5,
        pointHoverRadius: 5,
        pointHitRadius: 14,
        pointBackgroundColor: dark ? '#15171a' : '#ffffff',
        pointBorderColor: stroke,
        pointBorderWidth: 1.75,
        backgroundColor: (ctx: ScriptableContext<'line'>) => {
          const chart = ctx.chart
          const { ctx: canvasCtx, chartArea } = chart
          if (!chartArea) {
            return dark ? 'rgba(245, 245, 247, 0.08)' : 'rgba(20, 23, 31, 0.06)'
          }
          const gradient = canvasCtx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
          if (dark) {
            gradient.addColorStop(0, 'rgba(245, 245, 247, 0.08)')
            gradient.addColorStop(1, 'rgba(245, 245, 247, 0)')
          } else {
            gradient.addColorStop(0, 'rgba(20, 23, 31, 0.06)')
            gradient.addColorStop(1, 'rgba(20, 23, 31, 0)')
          }
          return gradient
        }
      }
    ]
  }
})

const chartJsOptions = computed<ChartOptions<'line'>>(() => {
  const dark = isDark.value
  const tickColor = dark ? '#7e8592' : '#868c98'
  const gridColor = dark ? 'rgba(255, 255, 255, 0.06)' : 'rgba(20, 23, 31, 0.06)'
  return {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    interaction: { intersect: false, mode: 'index' },
    onClick: (_event, items) => {
      const point = trendData.value[items[0]?.index ?? -1]
      if (!point) return
      const day = point.date.slice(0, 10)
      if (validUsageDate(day)) drilldown({ start: day, end: day })
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: dark ? '#1a1c20' : '#ffffff',
        titleColor: dark ? '#f5f5f7' : '#14171f',
        bodyColor: dark ? '#a0a6b1' : '#525866',
        borderColor: dark ? '#363a42' : '#e4e4de',
        borderWidth: 1,
        padding: 10,
        displayColors: false,
        callbacks: {
          title: (items) => {
            const idx = items[0]?.dataIndex ?? -1
            return trendData.value[idx]?.date || items[0]?.label || ''
          },
          label: (item) => {
            const val = Number(item.parsed.y || 0)
            return chartMetric.value === 'cost'
              ? formatUsageMoney(val, 4)
              : `${val.toLocaleString()} tokens`
          }
        }
      }
    },
    scales: {
      x: {
        border: { display: false },
        grid: { display: false },
        ticks: {
          color: tickColor,
          maxTicksLimit: 7,
          maxRotation: 0,
          font: { size: 11, family: "'JetBrains Mono', ui-monospace, monospace" }
        }
      },
      y: {
        beginAtZero: true,
        border: { display: false },
        grid: { color: gridColor },
        ticks: {
          color: tickColor,
          padding: 8,
          maxTicksLimit: 5,
          font: { size: 11, family: "'JetBrains Mono', ui-monospace, monospace" },
          callback: (value) => {
            const n = Number(value)
            return chartMetric.value === 'cost' ? `$${n < 1 ? n.toFixed(2) : n.toFixed(0)}` : formatCompact(n)
          }
        }
      }
    }
  }
})

// Active Model Breakdown
const totalModelCost = computed(() =>
  modelStats.value.reduce((sum, item) => sum + (item.actual_cost || 0), 0)
)
const totalModelRequests = computed(() =>
  modelStats.value.reduce((sum, item) => sum + (item.requests || 0), 0)
)

const rankedModels = computed(() => {
  const sorted = [...modelStats.value].sort((a, b) => {
    const diff = (b.actual_cost || 0) - (a.actual_cost || 0)
    return diff !== 0 ? diff : (b.requests || 0) - (a.requests || 0)
  })
  const useCost = totalModelCost.value > 0
  const denom = useCost ? totalModelCost.value : totalModelRequests.value
  return sorted.slice(0, 6).map((item) => {
    const numer = useCost ? (item.actual_cost || 0) : (item.requests || 0)
    const sharePct = denom > 0 ? Math.round((numer / denom) * 100) : 0
    return { ...item, sharePct }
  })
})

// Platform Quotas (only configured limits)
const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax'
}

const platformQuotaRows = computed(() => {
  const rows: Array<{ key: string; platform: string; label: string; used: number; limit: number; pct: number }> = []
  for (const q of platformQuotas.value ?? []) {
    const windows: Array<{ k: 'daily' | 'weekly' | 'monthly'; zh: string; en: string }> = [
      { k: 'daily', zh: '日额', en: 'Daily' },
      { k: 'weekly', zh: '周额', en: 'Weekly' },
      { k: 'monthly', zh: '月額', en: 'Monthly' }
    ]
    for (const w of windows) {
      const limit = Number(q[`${w.k}_limit_usd` as keyof PlatformQuotaItem])
      if (Number.isFinite(limit) && limit > 0) {
        const used = Number(q[`${w.k}_usage_usd` as keyof PlatformQuotaItem]) || 0
        const pct = Math.min(100, Math.round((used / limit) * 100))
        rows.push({
          key: `${q.platform}-${w.k}`,
          platform: q.platform,
          label: `${PLATFORM_LABELS[q.platform] ?? q.platform} · ${zh.value ? w.zh : w.en}`,
          used,
          limit,
          pct
        })
      }
    }
  }
  return rows.slice(0, 4)
})

// Real API Data Loaders
async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    await authStore.refreshUser()
    const data = await usageAPI.getDashboardStats()
    if (!disposed) stats.value = data
  } catch (cause) {
    if (!disposed) {
      error.value = zh.value ? '暂时无法获取账户概览，请重试。' : 'Unable to load your overview. Please try again.'
    }
    console.error('Failed to load dashboard stats:', cause)
  } finally {
    if (!disposed) loading.value = false
  }
}

async function loadCharts() {
  const request = ++chartRequest
  loadingCharts.value = true
  chartError.value = ''
  try {
    const [trend, models] = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value
      }),
      usageAPI.getDashboardModels({
        start_date: startDate.value,
        end_date: endDate.value
      })
    ])
    if (disposed || request !== chartRequest) return
    trendData.value = trend.trend || []
    modelStats.value = models.models || []
  } catch (cause) {
    if (!disposed && request === chartRequest) {
      chartError.value = zh.value ? '趋势暂时无法加载，请重试。' : 'Unable to load trends. Please retry.'
    }
    console.error('Failed to load charts:', cause)
  } finally {
    if (!disposed && request === chartRequest) loadingCharts.value = false
  }
}

async function loadRecent() {
  loadingUsage.value = true
  recentError.value = ''
  try {
    const res = await usageAPI.getByDateRange(
      formatDateLocalInput(new Date(Date.now() - 6 * 86400000)),
      today
    )
    if (!disposed) recentUsage.value = res.items.slice(0, 5)
  } catch (cause) {
    if (!disposed) {
      recentError.value = zh.value ? '最近请求加载失败，请重试。' : 'Unable to load recent requests. Please retry.'
    }
    console.error('Failed to load recent usage:', cause)
  } finally {
    if (!disposed) loadingUsage.value = false
  }
}

async function loadPlatformQuotas() {
  try {
    const data = await getMyPlatformQuotas()
    if (!disposed) platformQuotas.value = data.platform_quotas ?? []
  } catch (cause) {
    console.warn('Failed to load platform quotas:', cause)
    if (!disposed) platformQuotas.value = null
  }
}

async function loadQuickKeys() {
  loadingKeys.value = true
  try {
    const res = await keysAPI.list(1, 3, {
      status: 'active',
      sort_by: 'created_at',
      sort_order: 'desc'
    })
    if (!disposed) quickKeys.value = res.items || []
  } catch {
    if (!disposed) quickKeys.value = []
  } finally {
    if (!disposed) loadingKeys.value = false
  }
}

async function loadAnnouncements() {
  loadingAnnouncements.value = true
  try {
    const items = await announcementsAPI.list()
    if (!disposed) announcements.value = (items || []).slice(0, 5)
  } catch {
    if (!disposed) announcements.value = []
  } finally {
    if (!disposed) loadingAnnouncements.value = false
  }
}

function refreshAll() {
  void loadStats()
  void loadCharts()
  void loadRecent()
  void loadPlatformQuotas()
  void loadQuickKeys()
  void loadAnnouncements()
}

onMounted(refreshAll)
onBeforeUnmount(() => {
  disposed = true
  chartRequest++
  clearTimeout(copyResetTimer)
})
</script>
