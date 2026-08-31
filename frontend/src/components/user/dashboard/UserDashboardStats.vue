<template>
  <!-- Row 1: Core Stats -->
  <div class="grid grid-cols-2 gap-3.5 sm:gap-4 lg:grid-cols-4">
    <!-- Balance -->
    <div v-if="!isSimple" class="card relative overflow-hidden p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-100/80 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
          </svg>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.balance') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">${{ formatBalance(balance) }}</p>
          <p class="text-xs text-gray-400 dark:text-dark-400">{{ t('common.available') }}</p>
        </div>
      </div>
    </div>

    <!-- API Keys -->
    <div class="card relative overflow-hidden p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-link-100/70 text-link-700 dark:bg-link-900/30 dark:text-link-300">
          <Icon name="key" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.apiKeys') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">{{ stats?.total_api_keys || 0 }}</p>
          <p class="text-xs font-medium text-emerald-600 dark:text-emerald-400">{{ stats?.active_api_keys || 0 }} {{ t('common.active') }}</p>
        </div>
      </div>
    </div>

    <!-- Today Requests -->
    <div class="card relative overflow-hidden p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200">
          <Icon name="chart" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.todayRequests') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">{{ stats?.today_requests || 0 }}</p>
          <p class="truncate text-xs text-gray-400 dark:text-dark-400">{{ t('common.total') }}: <span class="tabular-nums font-medium text-gray-600 dark:text-dark-200">{{ formatNumber(stats?.total_requests || 0) }}</span></p>
        </div>
      </div>
    </div>

    <!-- Today Cost -->
    <div class="card relative overflow-hidden p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
          <Icon name="dollar" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.todayCost') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">
            <span class="text-primary-700 dark:text-primary-300" :title="t('dashboard.actual')">${{ formatCost(stats?.today_actual_cost || 0) }}</span>
            <span class="text-xs font-normal text-gray-400 dark:text-dark-400" :title="t('dashboard.standard')"> / ${{ formatCost(stats?.today_cost || 0) }}</span>
          </p>
          <p class="truncate text-xs text-gray-400 dark:text-dark-400">
            <span>{{ t('common.total') }}: </span>
            <span class="font-medium text-primary-700 dark:text-primary-300 tabular-nums" :title="t('dashboard.actual')">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
            <span class="tabular-nums text-gray-400 dark:text-dark-400" :title="t('dashboard.standard')"> / ${{ formatCost(stats?.total_cost || 0) }}</span>
          </p>
        </div>
      </div>
    </div>
  </div>

  <!-- Row 2: Token Stats -->
  <div class="grid grid-cols-2 gap-3.5 sm:gap-4 lg:grid-cols-4">
    <!-- Today Tokens -->
    <div class="card p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300">
          <Icon name="cube" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.todayTokens') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">{{ formatTokens(stats?.today_tokens || 0) }}</p>
          <p class="truncate text-[11px] text-gray-400 dark:text-dark-400" :title="`${t('dashboard.input')}: ${formatTokens(stats?.today_input_tokens || 0)} / ${t('dashboard.output')}: ${formatTokens(stats?.today_output_tokens || 0)} / ${t('dashboard.cache')}: ${formatTokens((stats?.today_cache_creation_tokens || 0) + (stats?.today_cache_read_tokens || 0))}`">
            <span class="tabular-nums">{{ t('dashboard.input') }}: {{ formatTokens(stats?.today_input_tokens || 0) }}</span> /
            <span class="tabular-nums">{{ t('dashboard.output') }}: {{ formatTokens(stats?.today_output_tokens || 0) }}</span> /
            <span class="tabular-nums">{{ t('dashboard.cache') }}: {{ formatTokens((stats?.today_cache_creation_tokens || 0) + (stats?.today_cache_read_tokens || 0)) }}</span>
          </p>
        </div>
      </div>
    </div>

    <!-- Total Tokens -->
    <div class="card p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200">
          <Icon name="database" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.totalTokens') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">{{ formatTokens(stats?.total_tokens || 0) }}</p>
          <p class="truncate text-[11px] text-gray-400 dark:text-dark-400" :title="`${t('dashboard.input')}: ${formatTokens(stats?.total_input_tokens || 0)} / ${t('dashboard.output')}: ${formatTokens(stats?.total_output_tokens || 0)} / ${t('dashboard.cache')}: ${formatTokens((stats?.total_cache_creation_tokens || 0) + (stats?.total_cache_read_tokens || 0))}`">
            <span class="tabular-nums">{{ t('dashboard.input') }}: {{ formatTokens(stats?.total_input_tokens || 0) }}</span> /
            <span class="tabular-nums">{{ t('dashboard.output') }}: {{ formatTokens(stats?.total_output_tokens || 0) }}</span> /
            <span class="tabular-nums">{{ t('dashboard.cache') }}: {{ formatTokens((stats?.total_cache_creation_tokens || 0) + (stats?.total_cache_read_tokens || 0)) }}</span>
          </p>
        </div>
      </div>
    </div>

    <!-- Performance (RPM/TPM) -->
    <div class="card p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-link-50 text-link-700 dark:bg-link-950/30 dark:text-link-300">
          <Icon name="bolt" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.performance') }}</p>
          <div class="flex items-baseline gap-1.5">
            <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">{{ formatTokens(stats?.rpm || 0) }}</p>
            <span class="text-xs font-semibold uppercase text-gray-400 dark:text-dark-400">RPM</span>
          </div>
          <div class="flex items-baseline gap-1.5">
            <p class="text-xs font-semibold text-link-700 dark:text-link-300 tabular-nums">{{ formatTokens(stats?.tpm || 0) }}</p>
            <span class="text-[10px] uppercase text-gray-400 dark:text-dark-400">TPM</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Avg Response Time -->
    <div class="card p-4 transition-colors hover:border-gray-300 dark:hover:border-dark-500">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200">
          <Icon name="clock" size="md" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.avgResponse') }}</p>
          <p class="text-xl font-bold tracking-tight text-gray-900 tabular-nums dark:text-white sm:text-2xl">{{ formatDuration(stats?.average_duration_ms || 0) }}</p>
          <p class="text-xs text-gray-400 dark:text-dark-400">{{ t('dashboard.averageTime') }}</p>
        </div>
      </div>
    </div>
  </div>

  <!-- Row 3: Per-platform breakdown -->
  <div v-if="!isSimple && platformCards.length > 0" class="card p-4 sm:p-5">
    <div class="mb-3.5 flex items-center justify-between">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.platformBreakdown') }}</h3>
      <span class="badge badge-gray text-xs">
        {{ t('dashboard.platformCount', { count: sortedPlatforms.length }) }}
      </span>
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="item in platformCards"
        :key="item.platform"
        :class="[
          'rounded-xl border p-3.5 transition-colors',
          item.isOther
            ? 'border-dashed border-gray-300 bg-gray-50 text-gray-900 dark:border-dark-600 dark:bg-dark-800/40 dark:text-white'
            : 'border-gray-200 bg-white/60 hover:border-gray-300 dark:border-dark-700 dark:bg-dark-800/60 dark:hover:border-dark-500'
        ]"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">
            {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
          </span>
          <span class="shrink-0 font-mono text-sm font-semibold tabular-nums text-primary-700 dark:text-primary-300" :title="t('dashboard.actual')">
            ${{ formatCost(item.total_actual_cost) }}
          </span>
        </div>
        <div class="mt-2.5 space-y-1.5 text-xs">
          <div class="flex items-center justify-between">
            <span class="text-gray-500 dark:text-dark-300">{{ t('dashboard.todayCost') }}</span>
            <span class="font-mono font-medium tabular-nums text-gray-900 dark:text-white">${{ formatCost(item.today_actual_cost) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-500 dark:text-dark-300">{{ t('dashboard.requests') }}</span>
            <span class="font-mono tabular-nums text-gray-600 dark:text-dark-200">
              {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
            </span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-500 dark:text-dark-300">{{ t('dashboard.tokens') }}</span>
            <span class="font-mono tabular-nums text-gray-600 dark:text-dark-200">
              {{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}
            </span>
          </div>
        </div>

        <!-- Quota 区 -->
        <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-3.5 space-y-2 border-t border-gray-100 pt-2.5 dark:border-dark-700">
          <p class="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-dark-400">
            {{ t('dashboard.platformQuota.title') }}
          </p>
          <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
            <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="space-y-1">
              <!-- limit=0：完全禁用 -->
              <template v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-dark-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="font-mono text-xs font-semibold text-err-600 dark:text-err-400">{{ t('dashboard.platformQuota.disabled') }}</span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full w-full rounded-full bg-err-500" />
                </div>
              </template>
              <!-- limit>0：正常用量进度条 -->
              <template v-else>
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-dark-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="font-mono text-xs tabular-nums text-gray-900 dark:text-white">
                    ${{ formatUsd((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0) }} / ${{ formatUsd(quotaVal(item.quota, `${w}_limit_usd`) as number) }}
                  </span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div
                    class="h-full rounded-full transition-[width] duration-300"
                    :class="quotaBarClass(calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number))"
                    :style="{ width: calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number) + '%' }"
                  />
                </div>
                <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="text-[10px] text-gray-400 dark:text-dark-400">
                  {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
                </p>
              </template>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()
const { t } = useI18n()

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity'
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

const sortedPlatforms = computed(() => {
  const list = props.stats?.by_platform ?? []
  return [...list].sort((a, b) => b.total_actual_cost - a.total_actual_cost)
})

// 处理"各平台之和 < 总值"的差值：后端按平台聚合时过滤了无法归属平台的行
// （group 与 account 都缺 platform）。这里把差值作为"其他"卡片显式展示，
// 避免 Row 1 总值与 Row 3 平台拆分加总对不上、用户困惑。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  // 建立 by_platform Map
  const byPlat = new Map<string, (typeof sortedPlatforms.value)[number]>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  // 建立 quota Map
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // union 平台集合。后端 by_platform / quota 接口均不会返回 platform='__other__'，
  // 无需显式排除；__other__ 由下方差值补差逻辑单独追加。
  const platforms = new Set<string>([...byPlat.keys(), ...byQuota.keys()])

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
  const cards: FusedPlatformCard[] = []

  for (const p of platforms) {
    const stat = byPlat.get(p)
    cards.push({
      platform: p,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(p),
    })
  }

  // 排序：按 PLATFORM_ORDER，未知平台按名称排序
  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  // __other__ 补差逻辑：只对 by_platform 有 usage 数据的总和计算
  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const sumTotal = cards.reduce((s, c) => s + c.total_actual_cost, 0)
  const sumToday = cards.reduce((s, c) => s + c.today_actual_cost, 0)
  const diffTotal = Math.max(0, total - sumTotal)
  const diffToday = Math.max(0, today - sumToday)

  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

// Quota helpers

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

function quotaVal(q: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return q?.[key]
}

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function quotaBarClass(p: number): string {
  if (p >= 95) return 'bg-red-500'
  if (p >= 75) return 'bg-amber-500'
  return 'bg-green-500'
}

// 与 formatBalance 一致使用 Intl.NumberFormat 做半偶舍入，避免 toFixed 在不同 JS 引擎
// 下偶发截断而非四舍五入（与后端展示精度不一致）。
const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
function formatUsd(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return usdFormatter.format(n)
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(b)

const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>
