<script setup lang="ts">
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { paymentAPI } from '@/api/payment'
import subscriptionsAPI from '@/api/subscriptions'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatPaymentAmount } from '@/components/payment/currency'
import { planValiditySuffix } from '@/components/payment/validity'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'
import { getRemainingDurationParts, isOneTimeDailyQuota } from '@/utils/subscriptionQuota'
import type { CheckoutInfoResponse, SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import InkSurface from '@/components/common/InkSurface.vue'
import PortalSubscriptionConfirm from '@/components/portal/PortalSubscriptionConfirm.vue'

const { t, locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore()
const store = useSubscriptionStore()
const route = useRoute()
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
const active = computed(() => owned.value.filter(subscription => subscription.status === 'active'))
const usd = (value: number) => formatPaymentAmount(value, 'USD')
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
  // Loading current subscriptions independently keeps purchase guards current without hiding history on a cache error.
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
  <div class="portal-page subscription-page">
    <PortalPageHeader :title="copy('订阅', 'Subscriptions')" />
    <p v-if="route.query.online_payment === 'retired'" class="portal-error" role="status">{{ copy('在线支付已停用。未完成的订单不会继续结算，订阅仍可使用余额购买。', 'Online payment is no longer available. Unfinished orders will not be completed. Subscriptions can still be bought with account balance.') }}</p>
    <div v-if="!enabled" class="portal-empty">{{ copy('订阅服务暂未开放。', 'Subscriptions are not available.') }}</div>
    <div v-else-if="loading" class="portal-empty" role="status">{{ copy('正在加载订阅信息…', 'Loading subscriptions…') }}</div>
    <template v-else>
      <PortalSubscriptionConfirm :plan="selected" :unavailable="selected ? unavailable(selected) : ''" @close="selected = null" @purchased="purchased" />
      <p v-if="purchaseNotice" class="portal-success" role="status">{{ purchaseNotice }}</p>
      <div v-if="planError" class="portal-error" role="alert"><p>{{ planError }}</p><button class="portal-button secondary" @click="load">{{ copy('重新加载', 'Reload') }}</button></div>
      <div v-if="groups.length > 1" class="subscription-groups" :aria-label="copy('筛选订阅分组', 'Filter plan groups')">
        <button :class="{ active: !group }" @click="group = ''">{{ copy('全部计划', 'All plans') }}</button>
        <button v-for="[id, name] in groups" :key="id" :class="{ active: group === id }" @click="group = id">{{ name }}</button>
      </div>
      <section id="portal-subscription-plans" class="subscription-plans" :aria-label="copy('可购买的订阅计划', 'Available subscription plans')">
        <InkSurface as="article" interactive v-for="plan in visiblePlans" :key="plan.id" class="subscription-plan"><span class="plan-ink-accent" aria-hidden="true"></span>
          <div class="plan-title"><h2>{{ plan.name }}</h2><span v-if="plan.one_purchase_per_user" class="plan-note">{{ copy('每人限购一次', 'One purchase per user') }}</span></div>
          <p class="plan-description">{{ plan.description || plan.group_name }}</p>
          <p class="plan-price"><strong>{{ formatPaymentAmount(plan.price, plan.currency || 'USD') }}</strong><span>/ {{ plan.fixed_expires_at ? copy('固定有效期', 'Fixed term') : planValiditySuffix(plan, t) }}</span></p>
          <p v-if="plan.original_price && plan.original_price > plan.price" class="plan-original">{{ formatPaymentAmount(plan.original_price, plan.currency || 'USD') }}</p>
          <button class="portal-button plan-button" :disabled="!!unavailable(plan) || !!historyError" @click="selectPlan(plan)">{{ unavailable(plan) || (active.some(subscription => subscription.group_id === plan.group_id) ? copy('续订计划', 'Renew plan') : copy('订阅计划', 'Subscribe')) }}</button>
          <ul class="plan-features">
            <li v-if="plan.daily_limit_usd"><span>✓</span>{{ usd(plan.daily_limit_usd) }} {{ copy('每日额度', 'daily quota') }}</li>
            <li v-if="plan.weekly_limit_usd"><span>✓</span>{{ usd(plan.weekly_limit_usd) }} {{ copy('每周额度', 'weekly quota') }}</li>
            <li v-if="plan.monthly_limit_usd"><span>✓</span>{{ usd(plan.monthly_limit_usd) }} {{ copy('每月额度', 'monthly quota') }}</li>
            <li v-if="!plan.daily_limit_usd && !plan.weekly_limit_usd && !plan.monthly_limit_usd"><span>✓</span>{{ copy('套餐未设置周期额度上限', 'No periodic quota limit') }}</li>
            <li v-if="plan.rate_multiplier != null"><span>✓</span>{{ copy('计费倍率', 'Rate multiplier') }} {{ plan.rate_multiplier }}×</li>
            <li v-if="hasPeakRate(plan)"><span>✓</span>{{ formatPeakRateWindow(plan, serverTimezoneLabel(app.cachedPublicSettings?.server_utc_offset)) }}</li>
            <li v-for="feature in features(plan)" :key="feature"><span>✓</span>{{ feature }}</li>
            <li v-if="plan.fixed_expires_at"><span>✓</span>{{ copy('有效至', 'Valid until') }} {{ date(plan.fixed_expires_at) }}</li>
          </ul>
          <p v-if="plan.sale_ends_at" class="plan-sale-end">{{ copy('销售截止', 'Sale ends') }} {{ date(plan.sale_ends_at) }}</p>
        </InkSurface>
      </section>
      <div v-if="!planError && !visiblePlans.length" class="portal-empty">{{ copy('暂无可购买的订阅计划', 'No subscription plans for sale') }}</div>
      <section v-if="active.length" class="current-subscriptions" aria-labelledby="current-subscriptions-title">
        <h2 id="current-subscriptions-title">{{ copy('当前订阅', 'Current subscriptions') }}</h2>
        <p v-if="!progressLoaded" class="portal-muted">{{ copy('额度重置时间暂不可用。', 'Quota reset times are unavailable.') }} <button class="progress-retry" @click="load">{{ copy('重新加载', 'Reload') }}</button></p>
        <details v-for="subscription in active" :key="subscription.id" class="subscription-current">
          <summary><span>{{ subscription.group?.name || `#${subscription.group_id}` }}</span><span class="portal-muted">{{ copy('到期时间', 'Expires') }} {{ date(subscription.expires_at) }}</span></summary>
          <div class="current-content">
            <div v-for="row in quotaRows(subscription)" :key="row.key" class="subscription-quota">
              <p><span>{{ row.label }}</span><span>{{ usd(row.used || 0) }} / {{ usd(row.limit || 0) }}</span></p>
              <progress :value="row.used || 0" :max="row.limit || 1" :aria-label="row.label"></progress><small>{{ resetLabel(subscription, row) }}</small>
            </div>
            <p v-if="!quotaRows(subscription).length" class="portal-muted">{{ copy('此订阅未设置周期额度上限。', 'This subscription has no periodic quota limit.') }}</p>
            <button v-if="availablePlans.some(plan => plan.group_id === subscription.group_id && !unavailable(plan)) && !historyError" class="portal-button secondary" @click="renew(subscription)">{{ copy('续订计划', 'Renew plan') }}</button>
          </div>
        </details>
      </section>
      <section class="subscription-history" aria-labelledby="subscription-history-title">
        <div class="history-heading"><h2 id="subscription-history-title">{{ copy('历史订阅', 'Subscription history') }}</h2><RouterLink to="/orders">{{ copy('兑换记录', 'Redemption history') }} ↗</RouterLink></div>
        <p v-if="historyError" class="portal-error" role="alert">{{ historyError }} <button @click="load">{{ copy('重新加载', 'Reload') }}</button></p>
        <div class="subscription-table-wrap"><table class="portal-table subscription-history-table">
          <thead><tr><th>{{ copy('创建时间', 'Created') }}</th><th>{{ copy('订阅', 'Subscription') }}</th><th>{{ copy('开始时间', 'Start') }}</th><th>{{ copy('有效期', 'Expiry') }}</th><th>{{ copy('状态', 'Status') }}</th></tr></thead>
          <tbody>
            <tr v-if="!owned.length && !historyError"><td colspan="5" class="history-empty">{{ copy('暂无订阅记录', 'No subscriptions yet') }}</td></tr>
            <tr v-for="subscription in owned" :key="subscription.id"><td>{{ date(subscription.created_at) }}</td><td>{{ subscription.group?.name || `#${subscription.group_id}` }}</td><td>{{ date(subscription.starts_at) }}</td><td>{{ date(subscription.expires_at) }}</td><td><span class="subscription-state" :class="{ active: subscription.status === 'active' }">{{ ({ active: copy('有效', 'Active'), expired: copy('已到期', 'Expired'), revoked: copy('已撤销', 'Revoked'), suspended: copy('已暂停', 'Suspended') })[subscription.status] || subscription.status }}</span></td></tr>
          </tbody>
        </table></div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.subscription-page { max-width: 1240px; margin: 0 auto; }
.subscription-header { margin-bottom: 26px; }
.subscription-header p { margin-top: 10px; }
.subscription-groups { display: flex; gap: 8px; margin-bottom: 22px; flex-wrap: wrap; }
.subscription-groups button { padding: 7px 16px; border-radius: var(--radius-control); border: 1px solid rgb(var(--line)); background: rgb(var(--surface)); color: rgb(var(--ink-2)); font-size: 13.5px; font-weight: 500; cursor: pointer; transition: border-color var(--motion-fast) ease, background-color var(--motion-fast) ease, color var(--motion-fast) ease; }
.subscription-groups button:hover { border-color: rgb(var(--line-strong)); color: rgb(var(--ink)); }
.subscription-groups .active { background: rgb(var(--action-bg)); color: rgb(var(--action-text)); border-color: rgb(var(--action-bg)); font-weight: 600; }
.subscription-plans { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 22px; scroll-margin-top: 20px; }
.subscription-plan { display: flex; flex-direction: column; border: 1px solid rgb(var(--line)); border-radius: var(--radius-card); padding: 26px; background: rgb(var(--surface)); box-shadow: var(--shadow-panel); position: relative; overflow: hidden; isolation: isolate; transition: border-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease; }
.subscription-plan:hover { border-color: rgb(var(--line-strong)); box-shadow: var(--shadow-card-hover); }
.plan-ink-accent { position: absolute; left: 0; right: 0; top: 0; height: 3px; background: linear-gradient(90deg, rgb(var(--brand-moss)) 0%, rgb(var(--focus-ring) / .55) 100%); pointer-events: none; z-index: 0; }
.plan-title { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.plan-title h2 { font: 650 22px/1.3 var(--font-sans); letter-spacing: -0.02em; margin: 0; }
.plan-note { white-space: nowrap; color: rgb(var(--brand-moss)); background: rgb(var(--brand-moss) / .1); font-size: 12px; font-weight: 600; border: 1px solid rgb(var(--brand-moss) / .24); border-radius: 999px; padding: 3px 9px; }
.plan-description { min-height: 36px; font-size: 13.5px; color: rgb(var(--ink-2)); line-height: 1.6; margin: 12px 0 22px; }
.plan-price { display: flex; align-items: baseline; flex-wrap: wrap; gap: 8px; }
.plan-price strong { font-size: 32px; line-height: 1.2; font-weight: 650; letter-spacing: -0.025em; font-variant-numeric: tabular-nums; }
.plan-price > span { color: rgb(var(--ink-3)); font-size: 13.5px; }
.plan-original { color: rgb(var(--ink-3)); text-decoration: line-through; font-size: 13px; margin-top: 6px; font-variant-numeric: tabular-nums; }
.plan-button { width: 100%; min-height: 44px; margin: 22px 0; border-radius: var(--radius-control); }
.plan-features { display: grid; gap: 13px; margin: 0; padding: 18px 0 0; border-top: 1px solid rgb(var(--line)); list-style: none; }
.plan-features li { display: flex; align-items: flex-start; gap: 10px; line-height: 1.55; font-size: 13.5px; color: rgb(var(--ink-2)); }
.plan-features li > span { color: rgb(var(--brand-moss)); font-weight: 700; }
.plan-sale-end { color: rgb(var(--ink-3)); margin-top: 18px; font-size: 12.5px; }
.current-subscriptions, .subscription-history { margin-top: 38px; }
.current-subscriptions h2, .subscription-history h2 { font: 650 20px/1.3 var(--font-sans); letter-spacing: -0.015em; margin-bottom: 18px; }
.progress-retry { text-decoration: underline; text-underline-offset: 3px; }
.subscription-current { border: 1px solid rgb(var(--line)); border-radius: var(--radius-card); background: rgb(var(--surface)); box-shadow: var(--shadow-panel); padding: 0 20px; margin-bottom: 12px; }
.subscription-current summary { padding: 16px 0; display: flex; justify-content: space-between; align-items: center; gap: 20px; cursor: pointer; list-style: none; font-weight: 600; }
.subscription-current summary::after { content: '+'; color: rgb(var(--ink-3)); margin-left: auto; font-size: 16px; }
.subscription-current[open] summary { border-bottom: 1px solid rgb(var(--line)); }
.subscription-current[open] summary::after { content: '−'; }
.subscription-current summary .portal-muted { font-size: 13px; font-weight: 400; }
.current-content { padding: 18px 0 22px; max-width: 680px; }
.subscription-quota { margin-bottom: 18px; }
.subscription-quota p { display: flex; justify-content: space-between; gap: 20px; font-size: 13px; font-variant-numeric: tabular-nums; margin: 0; }
.subscription-quota progress { width: 100%; height: 6px; appearance: none; margin: 8px 0 5px; border: 0; background: rgb(var(--surface-2)); border-radius: 999px; overflow: hidden; }
.subscription-quota progress::-webkit-progress-bar { background: rgb(var(--surface-2)); border-radius: 999px; }
.subscription-quota progress::-webkit-progress-value { background: rgb(var(--brand-moss)); border-radius: 999px; }
.subscription-quota progress::-moz-progress-bar { background: rgb(var(--brand-moss)); border-radius: 999px; }
.subscription-quota small { color: rgb(var(--ink-3)); font-size: 12.5px; }
.history-heading { display: flex; justify-content: space-between; align-items: center; gap: 20px; }
.history-heading a { font-size: 13px; font-weight: 550; color: rgb(var(--brand-moss)); margin-bottom: 18px; text-decoration: none; }
.history-heading a:hover { text-decoration: underline; text-underline-offset: 3px; }
.subscription-table-wrap { border: 1px solid rgb(var(--line)); border-radius: var(--radius-card); background: rgb(var(--surface)); box-shadow: var(--shadow-panel); overflow-x: auto; }
.subscription-history-table { width: 100%; text-align: left; white-space: nowrap; border-collapse: collapse; font-variant-numeric: tabular-nums; }
.subscription-history-table th, .subscription-history-table td { padding: 14px 16px; border-bottom: 1px solid rgb(var(--line)); font-size: 13.5px; }
.subscription-history-table th { font-size: 12.5px; font-weight: 600; background: rgb(var(--surface-2)); color: rgb(var(--ink-2)); }
.subscription-history-table tr:last-child td { border-bottom: 0; }
.history-empty { height: 150px; text-align: center; color: rgb(var(--ink-3)); }
.subscription-state { display: inline-flex; align-items: center; border-radius: 999px; padding: 3px 10px; font-size: 12.5px; font-weight: 600; background: rgb(var(--surface-2)); color: rgb(var(--ink-3)); }
.subscription-state.active { color: rgb(var(--text-ok)); background: rgb(var(--status-ok-bg)); }
.portal-error button { margin: 10px; text-decoration: underline; }
@media (max-width: 780px) { .subscription-plans { grid-template-columns: 1fr; } .subscription-plan { padding: 22px; } .subscription-current summary { flex-wrap: wrap; gap: 8px; } .subscription-current summary .portal-muted { width: 100%; order: 2; } }
</style>
