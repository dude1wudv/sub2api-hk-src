<template>
  <div class="workspace-metrics" :class="{ 'workspace-metrics-simple': isSimple }">
    <article v-if="!isSimple" data-metric="balance" class="workspace-metric workspace-metric-balance">
      <div class="workspace-metric-label"><span>{{ t('dashboard.balance') }}</span><Icon name="creditCard" size="sm" /></div>
      <strong class="metric-value">${{ formatBalance(balance) }}</strong>
      <div class="workspace-metric-note"><span>{{ t('common.available') }}</span><RouterLink to="/purchase">{{ tx('充值余额', 'Top up') }} <span aria-hidden="true">↗</span></RouterLink></div>
      <svg viewBox="0 0 160 76" aria-hidden="true" class="balance-ink"><path d="M-8 62C26 1 68 96 112 28s76 42 89 20" /></svg>
    </article>
    <article data-metric="requests" class="workspace-metric">
      <div class="workspace-metric-label"><span>{{ t('dashboard.todayRequests') }}</span><Icon name="chart" size="sm" /></div>
      <strong class="metric-value">{{ formatNumber(stats.today_requests) }}</strong>
      <p class="workspace-metric-note">{{ t('common.total') }} · {{ formatNumber(stats.total_requests) }}</p>
    </article>
    <article data-metric="cost" class="workspace-metric">
      <div class="workspace-metric-label"><span>{{ t('dashboard.todayCost') }}</span><Icon name="dollar" size="sm" /></div>
      <strong class="metric-value">${{ formatCost(stats.today_actual_cost) }}</strong>
      <p class="workspace-metric-note">{{ t('dashboard.standard') }} · ${{ formatCost(stats.today_cost) }}</p>
    </article>
    <article data-metric="latency" class="workspace-metric">
      <div class="workspace-metric-label"><span>{{ t('dashboard.avgResponse') }}</span><Icon name="clock" size="sm" /></div>
      <strong class="metric-value">{{ formatDuration(stats.average_duration_ms) }}</strong>
      <p class="workspace-metric-note">{{ tx('累计请求平均总耗时', 'Average duration across all requests') }}</p>
    </article>
  </div>
  <div class="workspace-facts" :aria-label="tx('更多账户数据', 'More account statistics')">
    <RouterLink to="/keys" data-metric="keys"><Icon name="key" size="sm" /><span>{{ t('dashboard.apiKeys') }}</span><strong>{{ stats.active_api_keys }} / {{ stats.total_api_keys }}</strong><small>{{ t('common.active') }}</small><span aria-hidden="true">↗</span></RouterLink>
    <div data-metric="tokens"><span>{{ t('dashboard.todayTokens') }}</span><strong>{{ formatTokens(stats.today_tokens) }}</strong><small>{{ t('dashboard.cache') }}: {{ formatTokens(stats.today_cache_creation_tokens + stats.today_cache_read_tokens) }}</small></div>
    <div data-metric="tokens"><span>{{ t('dashboard.totalTokens') }}</span><strong>{{ formatTokens(stats.total_tokens) }}</strong><small>{{ t('dashboard.cache') }}: {{ formatTokens(stats.total_cache_creation_tokens + stats.total_cache_read_tokens) }}</small></div>
    <div data-metric="performance"><span>RPM / TPM</span><strong>{{ formatTokens(stats.rpm) }} / {{ formatTokens(stats.tpm) }}</strong><small>{{ tx('当前吞吐', 'Current throughput') }}</small></div>
  </div>
  <section v-if="!isSimple && platformCards.length" class="workspace-platforms">
    <header class="workspace-section-title"><div><h2>{{ t('dashboard.platformBreakdown') }}</h2></div><span class="portal-tag">{{ t('dashboard.platformCount', { count: platformCount }) }}</span></header>
    <div class="workspace-platform-grid">
      <article v-for="item in platformCards" :key="item.platform" data-testid="platform-card" :data-platform="item.platform" class="workspace-platform" :class="{ 'workspace-platform-other': item.isOther }">
        <header><span class="platform-ink-dot" aria-hidden="true"></span><h3>{{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}</h3><strong :title="t('dashboard.actual')">${{ formatCost(item.total_actual_cost) }}</strong></header>
        <dl><div><dt>{{ t('dashboard.todayCost') }}</dt><dd>${{ formatCost(item.today_actual_cost) }}</dd></div><div><dt>{{ t('dashboard.requests') }}</dt><dd>{{ item.total_requests > 0 ? formatNumber(item.total_requests) : '—' }}</dd></div><div><dt>{{ t('dashboard.tokens') }}</dt><dd>{{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '—' }}</dd></div></dl>
        <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="workspace-quotas">
          <h4>{{ t('dashboard.platformQuota.title') }}</h4>
          <div v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w" class="workspace-quota-row">
            <div class="workspace-quota-copy"><span>{{ t(`dashboard.platformQuota.${w}`) }}</span>
              <strong v-if="quotaVal(item.quota, `${w}_limit_usd`) == null">{{ tx('无限制', 'Unlimited') }}</strong>
              <strong v-else-if="quotaVal(item.quota, `${w}_limit_usd`) === 0" class="quota-disabled">{{ t('dashboard.platformQuota.disabled') }}</strong>
              <strong v-else>${{ formatUsd(Number(quotaVal(item.quota, `${w}_usage_usd`)) || 0) }} / ${{ formatUsd(Number(quotaVal(item.quota, `${w}_limit_usd`))) }}</strong>
              <small v-if="quotaVal(item.quota, `${w}_window_resets_at`)">{{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(String(quotaVal(item.quota, `${w}_window_resets_at`))) }) }}</small>
            </div>
            <div v-if="Number(quotaVal(item.quota, `${w}_limit_usd`)) > 0" class="quota-vessel" role="progressbar" :aria-label="`${platformLabel(item.platform)} ${t(`dashboard.platformQuota.${w}`)}`" :aria-valuenow="calcPercent(Number(quotaVal(item.quota, `${w}_usage_usd`)) || 0, Number(quotaVal(item.quota, `${w}_limit_usd`)))" :aria-valuemin="0" :aria-valuemax="100"><span :style="{ height: calcPercent(Number(quotaVal(item.quota, `${w}_usage_usd`)) || 0, Number(quotaVal(item.quota, `${w}_limit_usd`))) + '%' }"></span></div>
          </div>
        </div>
        <p v-else-if="!item.isOther" class="workspace-unlimited">{{ platformQuotas === null ? tx('额度信息暂不可用', 'Quota information unavailable') : `${tx('平台额度', 'Platform quota')} · ${tx('无限制', 'Unlimited')}` }}</p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { PlatformDashboardStats, UserDashboardStats as UserStatsType } from '@/api/usage'
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
const { t, locale } = useI18n()
const tx = (zh: string, en: string) => locale?.value.startsWith('zh') ? zh : en

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  typesafe: 'TypeSafe / Jev',
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

// 处理"各平台之和 < 总值"的差值：后端按平台聚合时过滤了无法归属平台的行
// （group 与 account 都缺 platform）。这里把差值作为"其他"卡片显式展示，
// 避免 Row 1 总值与 Row 3 平台拆分加总对不上、用户困惑。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  // 建立 by_platform Map
  const byPlat = new Map<string, PlatformDashboardStats>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  // 建立 quota Map。三档全空的记录不产生卡片，挂到卡片上也不渲染配额区。
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // 卡片集合 = 有用量的平台 ∪ 至少配置了一档限额的平台。
  // 三档全空的限额记录等价于不限额，不单独产生卡片。
  // 后端 by_platform / quota 接口均不会返回 platform='__other__'，
  // 无需显式排除；__other__ 由下方差值补差逻辑单独追加。
  const platforms = new Set<string>(byPlat.keys())
  for (const [platform, q] of byQuota) {
    if (hasAnyLimit(q)) platforms.add(platform)
  }

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'typesafe']
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

// 标题右侧的平台计数 = 实际渲染的平台卡片数，不含"其他"差额卡。
const platformCount = computed(() => platformCards.value.filter((c) => !c.isOther).length)

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
