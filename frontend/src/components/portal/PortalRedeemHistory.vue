<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { redeemAPI, type RedeemHistoryItem } from '@/api/redeem'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatPortalBalance, maskPortalRedeemCode } from '@/utils/portalPurchase'

const props = withDefaults(defineProps<{ compact?: boolean; refreshKey?: number }>(), { compact: false, refreshKey: 0 })
const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
type HistoryRow = Omit<RedeemHistoryItem, 'code'> & { maskedCode: string }
const rows = ref<HistoryRow[]>([])
const loading = ref(true)
const error = ref('')
const page = ref(1)
const pageSize = ref(props.compact ? 5 : 20)
const selectedPageSize = ref(pageSize.value)
const total = ref(0)
const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
let request = 0
let failedTarget: { page: number; size: number } | null = null

function typeLabel(type: string) {
  const names: Record<string, [string, string]> = {
    balance: ['余额兑换', 'Balance redemption'],
    subscription: ['订阅兑换', 'Subscription redemption'],
    concurrency: ['并发额度兑换', 'Concurrency redemption'],
    admin_balance: ['余额调整', 'Balance adjustment'],
    admin_concurrency: ['并发额度调整', 'Concurrency adjustment'],
    invitation: ['邀请兑换', 'Invitation redemption']
  }
  const label = names[type]
  return label ? copy(...label) : copy('其他兑换', 'Other redemption')
}

function valueLabel(item: HistoryRow) {
  if (item.type === 'balance' || item.type === 'admin_balance') {
    return `${item.value > 0 ? '+' : ''}${formatPortalBalance(item.value, locale.value)}`
  }
  if (item.type === 'subscription') {
    const days = item.validity_days ?? item.value
    return Number.isFinite(days) && days > 0 ? copy(`${days} 天`, `${days} days`) : '—'
  }
  if (item.type === 'concurrency' || item.type === 'admin_concurrency') {
    return Number.isFinite(item.value) ? `${item.value > 0 ? '+' : ''}${item.value} ${copy('并发', 'concurrent requests')}` : '—'
  }
  return '—'
}

function formattedDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', { hour12: false })
}

async function load(targetPage = page.value, targetSize = pageSize.value) {
  const currentRequest = ++request
  loading.value = true
  error.value = ''
  failedTarget = null
  try {
    const result = await redeemAPI.getHistory(targetPage, targetSize)
    if (currentRequest !== request) return
    const resultTotal = Number.isFinite(result.total) ? Math.max(0, result.total) : 0
    const lastPage = Math.max(1, Math.ceil(resultTotal / targetSize))
    if (targetPage > lastPage) { await load(lastPage, targetSize); return }
    rows.value = (result.items || []).map(({ code, ...item }) => ({
      ...item,
      maskedCode: item.type.startsWith('admin_') ? '—' : maskPortalRedeemCode(code)
    }))
    total.value = resultTotal
    page.value = targetPage
    pageSize.value = targetSize
    selectedPageSize.value = targetSize
  } catch (cause) {
    if (currentRequest !== request) return
    selectedPageSize.value = pageSize.value
    failedTarget = { page: targetPage, size: targetSize }
    error.value = extractApiErrorMessage(cause, copy('兑换记录加载失败，请重试。', 'Could not load redemption history. Please retry.'))
  } finally {
    if (currentRequest === request) loading.value = false
  }
}

function retry() { void load(failedTarget?.page ?? page.value, failedTarget?.size ?? pageSize.value) }
watch(() => props.refreshKey, () => { void load(1, pageSize.value) })
onMounted(() => { void load(1, pageSize.value) })
onBeforeUnmount(() => { request++ })
</script>

<template>
  <section class="redeem-history" :aria-label="copy('本站兑换记录', 'Redemption history on this site')">
    <header class="redeem-history-heading">
      <h2>{{ compact ? copy('最近兑换', 'Recent redemptions') : copy('本站兑换记录', 'Redemption history') }}</h2>
      <RouterLink v-if="compact" to="/orders" class="redeem-view-all">{{ copy('查看全部', 'View all') }} <span aria-hidden="true">→</span></RouterLink>
      <button v-else type="button" class="portal-button secondary" :disabled="loading" @click="load()">{{ copy('刷新', 'Refresh') }}</button>
    </header>
    <div v-if="error" class="portal-error redeem-history-error" role="alert"><p>{{ error }}</p><button type="button" class="portal-button secondary" :disabled="loading" @click="retry">{{ copy('重试', 'Retry') }}</button></div>
    <div class="portal-table-wrap" :aria-busy="loading">
      <table class="portal-table redeem-history-table">
        <thead><tr><th scope="col">{{ copy('时间', 'Date') }}</th><th scope="col">{{ copy('类型', 'Type') }}</th><th scope="col">{{ copy('金额 / 额度', 'Amount / allowance') }}</th><th scope="col">{{ copy('兑换码', 'Code') }}</th></tr></thead>
        <tbody>
          <tr v-if="loading"><td colspan="4" class="redeem-history-empty" role="status">{{ copy('正在加载兑换记录…', 'Loading redemption history…') }}</td></tr>
          <tr v-else-if="!rows.length"><td colspan="4" class="redeem-history-empty">{{ error ? copy('记录暂时无法显示', 'History is temporarily unavailable') : copy('暂无兑换记录', 'No redemptions yet') }}</td></tr>
          <tr v-for="item in loading ? [] : rows" :key="item.id">
            <td><time :datetime="item.used_at || item.created_at">{{ formattedDate(item.used_at || item.created_at) }}</time><small>#{{ item.id }}</small></td>
            <td>{{ typeLabel(item.type) }}<small v-if="item.type === 'subscription' && item.group?.name">{{ item.group.name }}</small></td>
            <td class="redeem-history-value">{{ valueLabel(item) }}</td>
            <td class="redeem-history-code">{{ item.maskedCode }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="!compact" class="portal-pagination redeem-history-pagination">
      <label>{{ copy('每页', 'Per page') }} <select v-model.number="selectedPageSize" :disabled="loading" @change="load(1, selectedPageSize)"><option :value="20">20</option><option :value="50">50</option><option :value="100">100</option></select></label>
      <span class="portal-muted">{{ copy(`共 ${total} 条`, `${total} records`) }} · {{ page }} / {{ pages }}</span>
      <button type="button" :disabled="loading || page <= 1" @click="load(page - 1)">{{ copy('上一页', 'Previous') }}</button>
      <button type="button" :disabled="loading || page >= pages" @click="load(page + 1)">{{ copy('下一页', 'Next') }}</button>
    </div>
  </section>
</template>

<style scoped>
.redeem-history-heading { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-bottom: 18px; }
.redeem-history-heading h2 { margin: 0; font-size: 18px; font-weight: 400; }
.redeem-view-all { font-size: 13px; color: #000; }
.redeem-view-all:hover { text-decoration: underline; text-underline-offset: 3px; }
.redeem-history-error { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-bottom: 18px; }
.redeem-history-error p { margin: 0; }
.redeem-history-error button { flex-shrink: 0; }
.redeem-history-table { min-width: 560px; }
.redeem-history-table td { white-space: nowrap; }
.redeem-history-table small { display: block; color: #8d8c87; font-size: 12px; margin-top: 5px; }
.redeem-history-empty { height: 150px; text-align: center; color: #8d8c87; }
.redeem-history-value { font-variant-numeric: tabular-nums; }
.redeem-history-code { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px; color: #8d8c87; }
.redeem-history-pagination { font-size: 12px; }
.redeem-history-pagination label { margin-right: auto; display: flex; align-items: center; gap: 10px; }
@media (max-width: 600px) {
  .redeem-history-heading { gap: 12px; }
  .redeem-history-pagination { gap: 10px; }
  .redeem-history-pagination label { width: 100%; }
  .redeem-history-error { align-items: flex-start; flex-direction: column; gap: 12px; }
}
</style>
