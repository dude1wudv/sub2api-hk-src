<script setup lang="ts">
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
  if (!['balance', 'both'].includes(plan.purchase_mode || '')) return copy('暂未开放余额订阅', 'Balance purchase unavailable')
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
  else document.getElementById('portal-subscription-plans')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
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
    <header class="subscription-header">
      <h1 class="portal-heading">{{ copy('订阅计划', 'Subscription plans') }}</h1>
      <p class="portal-muted">{{ copy('选择适合你的 API 订阅计划，按计划额度使用模型服务。', 'Choose an API plan and use models within your subscription quota.') }}</p>
    </header>
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
        <article v-for="plan in visiblePlans" :key="plan.id" class="subscription-plan">
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
        </article>
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
.subscription-page { max-width: 1200px; margin: 0 auto; }
.subscription-header { margin-bottom: 28px; }
.subscription-header p { margin-top: 12px; }
.subscription-groups { display: flex; gap: 8px; margin-bottom: 20px; flex-wrap: wrap; }
.subscription-groups button { padding: 8px 16px; border-radius: 8px; border: 1px solid #efede3; background: #fffdf7; }
.subscription-groups .active { background: #f3f0eb; color: #000; }
.subscription-plans { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px; scroll-margin-top: 20px; }
.subscription-plan { display: flex; flex-direction: column; border: 1px solid #efede3; border-radius: 16px; padding: 30px; background: #fffdf7; }
.plan-title { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.plan-title h2 { font: 24px/1.3 'Times New Roman', Times, serif; }
.plan-note { white-space: nowrap; color: #777; font-size: 11px; border: 1px solid #efede3; border-radius: 6px; padding: 4px 6px; }
.plan-description { min-height: 38px; font-size: 13px; color: #999; line-height: 1.5; margin: 14px 0 24px; }
.plan-price { display: flex; align-items: baseline; flex-wrap: wrap; gap: 10px; }
.plan-price strong { font-size: 38px; line-height: 1.1; font-weight: 500; }
.plan-price > span { color: #777; font-size: 13px; }
.plan-original { color: #999; text-decoration: line-through; font-size: 13px; margin-top: 7px; }
.plan-button { width: 100%; min-height: 46px; margin: 26px 0; border-radius: 100px; }
.plan-features { display: grid; gap: 18px; margin: 0; padding: 0; list-style: none; }
.plan-features li { display: flex; align-items: flex-start; gap: 11px; line-height: 1.6; font-size: 13px; }
.plan-features li > span { color: #555; }
.plan-sale-end { color: #999; margin-top: 20px; font-size: 11px; }
.current-subscriptions, .subscription-history { margin-top: 42px; }
.current-subscriptions h2, .subscription-history h2 { font: 22px/1.3 'Times New Roman', Times, serif; margin-bottom: 20px; }
.progress-retry { text-decoration: underline; text-underline-offset: 3px; }
.subscription-current { border-bottom: 1px solid #efede3; }
.subscription-current summary { padding: 18px 0; display: flex; justify-content: space-between; align-items: center; gap: 20px; cursor: pointer; list-style: none; }
.subscription-current summary::after { content: '+'; color: #999; margin-left: auto; }
.subscription-current[open] summary::after { content: '−'; }
.subscription-current summary .portal-muted { font-size: 12px; }
.current-content { padding: 10px 0 26px; max-width: 680px; }
.subscription-quota { margin-bottom: 20px; }
.subscription-quota p { display: flex; justify-content: space-between; gap: 20px; font-size: 13px; }
.subscription-quota progress { width: 100%; height: 5px; appearance: none; margin: 10px 0 5px; border: 0; background: #efede3; border-radius: 4px; overflow: hidden; }
.subscription-quota progress::-webkit-progress-bar { background: #efede3; }
.subscription-quota progress::-webkit-progress-value { background: #000; border-radius: 4px; }
.subscription-quota progress::-moz-progress-bar { background: #000; }
.subscription-quota small { color: #999; font-size: 11px; }
.history-heading { display: flex; justify-content: space-between; align-items: center; gap: 20px; }
.history-heading a { font-size: 12px; color: #777; margin-bottom: 20px; }
.subscription-table-wrap { border: 1px solid #efede3; border-radius: 12px; overflow-x: auto; }
.subscription-history-table { width: 100%; text-align: left; white-space: nowrap; border-collapse: collapse; }
.subscription-history-table th, .subscription-history-table td { padding: 18px; border-bottom: 1px solid #efede3; font-size: 12px; }
.subscription-history-table th { font-weight: 400; background: #f7f5ee; color: #777; }
.subscription-history-table tr:last-child td { border-bottom: 0; }
.history-empty { height: 150px; text-align: center; color: #999; }
.subscription-state { display: inline-block; border-radius: 5px; padding: 4px 9px; background: #f3f0eb; color: #777; }
.subscription-state.active { color: #356148; background: #edf4ec; }
.portal-error button { margin: 10px; text-decoration: underline; }
@media (max-width: 780px) { .subscription-plans { grid-template-columns: 1fr; } .subscription-plan { padding: 24px; } .subscription-current summary { flex-wrap: wrap; gap: 8px; } .subscription-current summary .portal-muted { width: 100%; order: 2; } }
</style>
