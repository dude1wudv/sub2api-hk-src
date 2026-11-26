<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { paymentAPI } from '@/api/payment'
import subscriptionsAPI from '@/api/subscriptions'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatPaymentAmount } from '@/components/payment/currency'
import { planValiditySuffix } from '@/components/payment/validity'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'
import { getRemainingDurationParts, isOneTimeDailyQuota } from '@/utils/subscriptionQuota'
import type { CheckoutInfoResponse, SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import PortalSubscriptionConfirm from '@/components/portal/PortalSubscriptionConfirm.vue'
import { UiButton, UiProgressBar, UiSegmented, UiStatusDot, UiTabs } from '@/ui'

const { t, locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore()
const auth = useAuthStore()
const store = useSubscriptionStore()
const route = useRoute()
const router = useRouter()
const checkout = ref<CheckoutInfoResponse | null>(null)
const owned = ref<UserSubscription[]>([])
const loading = ref(true)
const planError = ref('')
const historyError = ref('')
type QuotaPeriod = 'daily' | 'weekly' | 'monthly'
interface QuotaProgress { used_usd: number; limit_usd: number; resets_at: string }
const progress = ref<Record<number, Partial<Record<QuotaPeriod, QuotaProgress>>>>({})
const progressLoaded = ref(false)
const purchaseNotice = ref('')
const selected = ref<SubscriptionPlan | null>(null)
const group = ref('')
const enabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription))
const availablePlans = computed(() => (checkout.value?.plans || []).filter(plan => plan.for_sale).sort((a, b) => a.sort_order - b.sort_order))
const visiblePlans = computed(() => availablePlans.value.filter(plan => !group.value || String(plan.group_id) === group.value))
const groups = computed(() => [...new Map(availablePlans.value.map(plan => [String(plan.group_id), plan.group_name || plan.name])).entries()])
const groupOptions = computed(() => [
  { value: '', label: copy('全部计划', 'All plans') },
  ...groups.value.map(([id, name]) => ({ value: id, label: name })),
])
const active = computed(() => owned.value.filter(subscription => subscription.status === 'active'))
const usd = (value: number) => formatPaymentAmount(value, 'USD')

const billingTabs = computed(() => {
  const tabs: Array<{ value: string; label: string }> = []
  if (!auth.isSimpleMode) {
    tabs.push({ value: 'purchase', label: copy('充值与兑换', 'Top up & redeem') })
    tabs.push({ value: 'orders', label: copy('兑换记录', 'Redemption history') })
  }
  if (enabled.value) {
    tabs.push({ value: 'subscriptions', label: copy('订阅计划', 'Subscriptions') })
  }
  return tabs
})

function handleTabChange(val: string) {
  if (val === 'purchase') void router.push('/purchase')
  else if (val === 'orders') void router.push('/orders')
}

function date(value?: string | null) { if (!value) return copy('长期有效', 'No expiry'); const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? '—' : parsed.toLocaleString(locale.value, { hour12: false }) }
function features(plan: SubscriptionPlan): string[] {
  const value: unknown = plan.features
  if (Array.isArray(value)) return value.filter((feature): feature is string => typeof feature === 'string')
  if (typeof value === 'string') {
    try { const parsed = JSON.parse(value); if (Array.isArray(parsed)) return parsed.filter((feature): feature is string => typeof feature === 'string') } catch { /* Legacy feature strings are newline separated. */ }
    return value.split('\n').map(feature => feature.trim()).filter(Boolean)
  }
  return []
}
function unavailable(plan: SubscriptionPlan) {
  if (!['balance', 'both'].includes(plan.purchase_mode || '')) return copy('在线支付已停用', 'Online payment is unavailable')
  if (plan.sale_ends_at && Date.parse(plan.sale_ends_at) <= Date.now()) return copy('销售已结束', 'Sale ended')
  if (plan.fixed_expires_at && Date.parse(plan.fixed_expires_at) <= Date.now()) return copy('套餐已到期', 'Plan expired')
  if (plan.one_purchase_per_user && owned.value.some(subscription => subscription.group_id === plan.group_id)) return copy('已购买', 'Already purchased')
  return ''
}
function selectPlan(plan: SubscriptionPlan) { if (!unavailable(plan)) selected.value = plan }
function renew(subscription: UserSubscription) {
  group.value = String(subscription.group_id)
  const plans = availablePlans.value.filter(plan => plan.group_id === subscription.group_id && !unavailable(plan))
  if (plans.length === 1) selected.value = plans[0]
  else document.getElementById('portal-subscription-plans')?.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start' })
}
function quotaRows(subscription: UserSubscription) {
  return [
    { key: 'daily' as const, label: isOneTimeDailyQuota(subscription) ? copy('周期额度', 'Term quota') : copy('每日额度', 'Daily quota'), used: subscription.daily_usage_usd, limit: subscription.group?.daily_limit_usd, window: subscription.daily_window_start },
    { key: 'weekly' as const, label: copy('每周额度', 'Weekly quota'), used: subscription.weekly_usage_usd, limit: subscription.group?.weekly_limit_usd, window: subscription.weekly_window_start },
    { key: 'monthly' as const, label: copy('每月额度', 'Monthly quota'), used: subscription.monthly_usage_usd, limit: subscription.group?.monthly_limit_usd, window: subscription.monthly_window_start },
  ].map(row => {
    const current = progress.value[subscription.id]?.[row.key]
    return { ...row, used: current?.used_usd ?? row.used, limit: current?.limit_usd ?? row.limit, resetsAt: current?.resets_at }
  }).filter(row => row.limit && row.limit > 0)
}
function quotaPercent(used: number | undefined, limit: number | undefined): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, ((used || 0) / limit) * 100))
}
function quotaTone(used: number | undefined, limit: number | undefined): 'ink' | 'warn' | 'err' {
  const pct = quotaPercent(used, limit)
  if (pct >= 95) return 'err'
  if (pct >= 80) return 'warn'
  return 'ink'
}
function resetLabel(subscription: UserSubscription, row: ReturnType<typeof quotaRows>[number]) {
  if (!row.resetsAt) return progressLoaded.value && progress.value[subscription.id] && !row.window
    ? copy('使用后开始计算周期', 'Period starts after first use')
    : copy('重置时间暂不可用', 'Reset time unavailable')
  const left = getRemainingDurationParts(row.resetsAt)
  if (!left) return copy('等待周期更新', 'Waiting for period reset')
  const duration = `${left.days}d ${left.hours}h ${left.minutes}m`
  return row.key === 'daily' && isOneTimeDailyQuota(subscription) ? copy(`${duration} 后到期`, `Expires in ${duration}`) : copy(`${duration} 后重置`, `Resets in ${duration}`)
}
function parseProgress(value: unknown) {
  if (!Array.isArray(value)) throw new Error('Invalid subscription progress response')
  const result: typeof progress.value = {}
  for (const item of value) {
    const id = item?.subscription?.id
    if (!Number.isInteger(id) || !item?.progress || typeof item.progress !== 'object') continue
    const periods: Partial<Record<QuotaPeriod, QuotaProgress>> = {}
    for (const period of ['daily', 'weekly', 'monthly'] as const) {
      const current = item.progress[period]
      if (current && Number.isFinite(current.used_usd) && Number.isFinite(current.limit_usd)
        && typeof current.resets_at === 'string' && Number.isFinite(Date.parse(current.resets_at))) {
        periods[period] = { used_usd: current.used_usd, limit_usd: current.limit_usd, resets_at: current.resets_at }
      }
    }
    result[id] = periods
  }
  return result
}
async function load() {
  loading.value = true
  planError.value = ''
  historyError.value = ''
  progress.value = {}
  progressLoaded.value = false
  const outcomes = await Promise.allSettled([
    subscriptionsAPI.getMySubscriptions(),
    paymentAPI.getCheckoutInfo(),
    store.fetchActiveSubscriptions(true),
    subscriptionsAPI.getSubscriptionsProgress(),
  ])
  if (outcomes[0].status === 'fulfilled') owned.value = outcomes[0].value
  else historyError.value = extractI18nErrorMessage(outcomes[0].reason, t, 'payment.errors', copy('订阅记录加载失败。', 'Could not load subscription history.'))
  if (outcomes[1].status === 'fulfilled') checkout.value = outcomes[1].value?.data || null
  else planError.value = extractI18nErrorMessage(outcomes[1].reason, t, 'payment.errors', copy('订阅计划加载失败。', 'Could not load subscription plans.'))
  if (outcomes[2].status === 'rejected' && !historyError.value) historyError.value = copy('当前订阅状态暂未刷新，请重试后购买。', 'Subscription status is unavailable. Refresh before purchasing.')
  if (outcomes[3].status === 'fulfilled') {
    try { progress.value = parseProgress(outcomes[3].value); progressLoaded.value = true } catch { /* Unknown progress stays unavailable rather than estimating a reset. */ }
  }
  loading.value = false
}
async function purchased(warning: string) { selected.value = null; purchaseNotice.value = warning || copy('订阅购买成功。', 'Subscription purchased.'); await load() }
onMounted(async () => {
  if (!enabled.value) { loading.value = false; return }
  await load()
  if (typeof route.query.group === 'string' && groups.value.some(([id]) => id === route.query.group)) {
    group.value = route.query.group
  }
})
</script>

<template>
  <div class="mx-auto max-w-[1240px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ copy('订阅计划', 'Subscriptions') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ copy('使用账户余额订阅周期额度套餐，并随时追踪已生效的订阅进度。', 'Subscribe to periodic quota plans with your account balance and track active allowances.') }}
      </p>
    </header>

    <UiTabs
      v-if="billingTabs.length > 1"
      model-value="subscriptions"
      :tabs="billingTabs"
      :aria-label="copy('计费与兑换导航', 'Billing and redemption navigation')"
      @update:model-value="handleTabChange"
    />

    <p
      v-if="route.query.online_payment === 'retired'"
      class="rounded-xl border border-[var(--atelier-warn)]/30 bg-[var(--atelier-warn-bg)] px-4 py-3 text-sm text-[var(--atelier-warn)]"
      role="status"
    >
      {{ copy('在线支付已停用。未完成的订单不会继续结算，订阅仍可使用余额购买。', 'Online payment is no longer available. Unfinished orders will not be completed. Subscriptions can still be bought with account balance.') }}
    </p>

    <div v-if="!enabled" class="atelier-surface py-14 text-center text-sm text-[var(--atelier-ink-muted)]">
      {{ copy('订阅服务暂未开放。', 'Subscriptions are not available.') }}
    </div>
    <div v-else-if="loading" class="atelier-surface py-14 text-center text-sm text-[var(--atelier-ink-muted)]" role="status">
      {{ copy('正在加载订阅信息…', 'Loading subscriptions…') }}
    </div>

    <template v-else>
      <PortalSubscriptionConfirm
        :plan="selected"
        :unavailable="selected ? unavailable(selected) : ''"
        @close="selected = null"
        @purchased="purchased"
      />

      <p
        v-if="purchaseNotice"
        class="rounded-xl border border-[var(--atelier-ok)]/25 bg-[var(--atelier-ok-bg)] px-4 py-3 text-sm text-[var(--atelier-ok)]"
        role="status"
      >
        {{ purchaseNotice }}
      </p>

      <div
        v-if="planError"
        class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]"
        role="alert"
      >
        <p class="m-0">{{ planError }}</p>
        <UiButton variant="secondary" size="sm" @click="load">{{ copy('重新加载', 'Reload') }}</UiButton>
      </div>

      <div v-if="groups.length > 1" class="flex flex-wrap items-center">
        <UiSegmented
          v-model="group"
          :options="groupOptions"
          :aria-label="copy('筛选订阅分组', 'Filter plan groups')"
        />
      </div>

      <!-- Plans Grid -->
      <section
        id="portal-subscription-plans"
        class="grid scroll-mt-6 grid-cols-1 gap-5 md:grid-cols-2"
        :aria-label="copy('可购买的订阅计划', 'Available subscription plans')"
      >
        <article
          v-for="plan in visiblePlans"
          :key="plan.id"
          class="atelier-surface flex flex-col p-6 transition-colors hover:border-[var(--atelier-border-strong)]"
        >
          <div class="flex items-start justify-between gap-3">
            <h2 class="text-lg font-semibold tracking-tight text-[var(--atelier-ink)]">{{ plan.name }}</h2>
            <span
              v-if="plan.one_purchase_per_user"
              class="shrink-0 rounded-full border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2.5 py-0.5 text-[11px] font-medium text-[var(--atelier-ink-secondary)]"
            >
              {{ copy('每人限购一次', 'One purchase per user') }}
            </span>
          </div>

          <p class="mt-2 min-h-[36px] text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
            {{ plan.description || plan.group_name }}
          </p>

          <div class="mt-5 flex flex-wrap items-baseline gap-2">
            <strong class="font-mono text-3xl font-semibold tracking-tight tabular-nums text-[var(--atelier-ink)]">
              {{ formatPaymentAmount(plan.price, plan.currency || 'USD') }}
            </strong>
            <span class="text-xs text-[var(--atelier-ink-muted)]">
              / {{ plan.fixed_expires_at ? copy('固定有效期', 'Fixed term') : planValiditySuffix(plan, t) }}
            </span>
          </div>
          <p
            v-if="plan.original_price && plan.original_price > plan.price"
            class="mt-1 font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)] line-through"
          >
            {{ formatPaymentAmount(plan.original_price, plan.currency || 'USD') }}
          </p>

          <div class="my-5">
            <UiButton
              variant="primary"
              size="lg"
              block
              :disabled="!!unavailable(plan) || !!historyError"
              @click="selectPlan(plan)"
            >
              {{ unavailable(plan) || (active.some(subscription => subscription.group_id === plan.group_id) ? copy('续订计划', 'Renew plan') : copy('订阅计划', 'Subscribe')) }}
            </UiButton>
          </div>

          <ul class="space-y-2.5 border-t border-[var(--atelier-border)] pt-4 text-sm text-[var(--atelier-ink-secondary)]">
            <li v-if="plan.daily_limit_usd" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span><span class="font-mono tabular-nums">{{ usd(plan.daily_limit_usd) }}</span> {{ copy('每日额度', 'daily quota') }}</span>
            </li>
            <li v-if="plan.weekly_limit_usd" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span><span class="font-mono tabular-nums">{{ usd(plan.weekly_limit_usd) }}</span> {{ copy('每周额度', 'weekly quota') }}</span>
            </li>
            <li v-if="plan.monthly_limit_usd" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span><span class="font-mono tabular-nums">{{ usd(plan.monthly_limit_usd) }}</span> {{ copy('每月额度', 'monthly quota') }}</span>
            </li>
            <li v-if="!plan.daily_limit_usd && !plan.weekly_limit_usd && !plan.monthly_limit_usd" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span>{{ copy('套餐未设置周期额度上限', 'No periodic quota limit') }}</span>
            </li>
            <li v-if="plan.rate_multiplier != null" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span>{{ copy('计费倍率', 'Rate multiplier') }} <span class="font-mono tabular-nums">{{ plan.rate_multiplier }}×</span></span>
            </li>
            <li v-if="hasPeakRate(plan)" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span>{{ formatPeakRateWindow(plan, serverTimezoneLabel(app.cachedPublicSettings?.server_utc_offset)) }}</span>
            </li>
            <li v-for="feature in features(plan)" :key="feature" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span>{{ feature }}</span>
            </li>
            <li v-if="plan.fixed_expires_at" class="flex items-start gap-2.5">
              <span class="font-semibold text-[var(--atelier-ink)]">✓</span>
              <span>{{ copy('有效至', 'Valid until') }} <span class="font-mono tabular-nums">{{ date(plan.fixed_expires_at) }}</span></span>
            </li>
          </ul>

          <p v-if="plan.sale_ends_at" class="mt-4 text-xs text-[var(--atelier-ink-muted)]">
            {{ copy('销售截止', 'Sale ends') }} <span class="font-mono tabular-nums">{{ date(plan.sale_ends_at) }}</span>
          </p>
        </article>
      </section>

      <div v-if="!planError && !visiblePlans.length" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]">
        {{ copy('暂无可购买的订阅计划', 'No subscription plans for sale') }}
      </div>

      <!-- Active Subscriptions -->
      <section v-if="active.length" class="space-y-3 pt-2" aria-labelledby="current-subscriptions-title">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 id="current-subscriptions-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ copy('当前订阅', 'Current subscriptions') }}
          </h2>
          <p v-if="!progressLoaded" class="text-xs text-[var(--atelier-ink-muted)]">
            {{ copy('额度重置时间暂不可用。', 'Quota reset times are unavailable.') }}
            <button type="button" class="ml-1 underline hover:text-[var(--atelier-ink)]" @click="load">
              {{ copy('重新加载', 'Reload') }}
            </button>
          </p>
        </div>

        <details
          v-for="subscription in active"
          :key="subscription.id"
          open
          class="atelier-surface px-5"
        >
          <summary class="flex cursor-pointer list-none flex-wrap items-center justify-between gap-3 py-4 text-sm font-semibold text-[var(--atelier-ink)]">
            <span class="flex items-center gap-2">
              <UiStatusDot tone="ok" />
              {{ subscription.group?.name || `#${subscription.group_id}` }}
            </span>
            <span class="font-mono text-xs font-normal tabular-nums text-[var(--atelier-ink-muted)]">
              {{ copy('到期时间', 'Expires') }} {{ date(subscription.expires_at) }}
            </span>
          </summary>

          <div class="max-w-2xl space-y-4 border-t border-[var(--atelier-border)] py-4">
            <div v-for="row in quotaRows(subscription)" :key="row.key" class="space-y-1.5">
              <div class="flex items-center justify-between gap-4 text-xs">
                <span class="font-medium text-[var(--atelier-ink-secondary)]">{{ row.label }}</span>
                <span class="font-mono tabular-nums text-[var(--atelier-ink)]">
                  {{ usd(row.used || 0) }} / {{ usd(row.limit || 0) }}
                </span>
              </div>
              <UiProgressBar :value="quotaPercent(row.used ?? undefined, row.limit ?? undefined)" :tone="quotaTone(row.used ?? undefined, row.limit ?? undefined)" />
              <p class="text-[11px] text-[var(--atelier-ink-muted)]">{{ resetLabel(subscription, row) }}</p>
            </div>

            <p v-if="!quotaRows(subscription).length" class="text-xs text-[var(--atelier-ink-muted)]">
              {{ copy('此订阅未设置周期额度上限。', 'This subscription has no periodic quota limit.') }}
            </p>

            <UiButton
              v-if="availablePlans.some(plan => plan.group_id === subscription.group_id && !unavailable(plan)) && !historyError"
              variant="secondary"
              size="sm"
              @click="renew(subscription)"
            >
              {{ copy('续订计划', 'Renew plan') }}
            </UiButton>
          </div>
        </details>
      </section>

      <!-- Subscription History -->
      <section class="atelier-surface overflow-hidden" aria-labelledby="subscription-history-title">
        <header class="flex items-center justify-between gap-3 border-b border-[var(--atelier-border)] px-5 py-4">
          <h2 id="subscription-history-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ copy('历史订阅', 'Subscription history') }}
          </h2>
          <RouterLink
            v-if="!auth.isSimpleMode"
            to="/orders"
            class="text-xs font-medium text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)] hover:underline"
          >
            {{ copy('兑换记录', 'Redemption history') }} ↗
          </RouterLink>
        </header>

        <div class="p-5">
          <p v-if="historyError" class="mb-4 rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]" role="alert">
            {{ historyError }}
            <button type="button" class="ml-2 underline" @click="load">{{ copy('重新加载', 'Reload') }}</button>
          </p>

          <div class="overflow-x-auto">
            <div v-if="mobile" class="space-y-3">
              <div v-if="!owned.length && !historyError" class="py-10 text-center text-xs text-[var(--atelier-ink-muted)]">
                {{ copy('暂无订阅记录', 'No subscriptions yet') }}
              </div>
              <dl
                v-for="subscription in owned"
                :key="subscription.id"
                class="space-y-2 rounded-xl border border-[var(--atelier-border)] p-3.5 text-xs"
              >
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ copy('创建时间', 'Created') }}</dt>
                  <dd class="font-mono tabular-nums">{{ date(subscription.created_at) }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ copy('订阅', 'Subscription') }}</dt>
                  <dd class="font-medium">{{ subscription.group?.name || `#${subscription.group_id}` }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ copy('开始时间', 'Start') }}</dt>
                  <dd class="font-mono tabular-nums">{{ date(subscription.starts_at) }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ copy('有效期', 'Expiry') }}</dt>
                  <dd class="font-mono tabular-nums">{{ date(subscription.expires_at) }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ copy('状态', 'Status') }}</dt>
                  <dd>
                    <span
                      class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium"
                      :class="subscription.status === 'active' ? 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]' : 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-muted)]'"
                    >
                      <UiStatusDot :tone="subscription.status === 'active' ? 'ok' : 'neutral'" />
                      {{ ({ active: copy('有效', 'Active'), expired: copy('已到期', 'Expired'), revoked: copy('已撤销', 'Revoked'), suspended: copy('已暂停', 'Suspended') })[subscription.status] || subscription.status }}
                    </span>
                  </dd>
                </div>
              </dl>
            </div>

            <table v-else class="atelier-table text-xs">
              <thead>
                <tr>
                  <th>{{ copy('创建时间', 'Created') }}</th>
                  <th>{{ copy('订阅', 'Subscription') }}</th>
                  <th>{{ copy('开始时间', 'Start') }}</th>
                  <th>{{ copy('有效期', 'Expiry') }}</th>
                  <th>{{ copy('状态', 'Status') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="!owned.length && !historyError">
                  <td colspan="5" class="py-12 text-center text-[var(--atelier-ink-muted)]">
                    {{ copy('暂无订阅记录', 'No subscriptions yet') }}
                  </td>
                </tr>
                <tr v-for="subscription in owned" :key="subscription.id">
                  <td class="whitespace-nowrap font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ date(subscription.created_at) }}</td>
                  <td class="font-medium text-[var(--atelier-ink)]">{{ subscription.group?.name || `#${subscription.group_id}` }}</td>
                  <td class="whitespace-nowrap font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ date(subscription.starts_at) }}</td>
                  <td class="whitespace-nowrap font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ date(subscription.expires_at) }}</td>
                  <td>
                    <span
                      class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium"
                      :class="subscription.status === 'active' ? 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]' : 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-muted)]'"
                    >
                      <UiStatusDot :tone="subscription.status === 'active' ? 'ok' : 'neutral'" />
                      {{ ({ active: copy('有效', 'Active'), expired: copy('已到期', 'Expired'), revoked: copy('已撤销', 'Revoked'), suspended: copy('已暂停', 'Suspended') })[subscription.status] || subscription.status }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
