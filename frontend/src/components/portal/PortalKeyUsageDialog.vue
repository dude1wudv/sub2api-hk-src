<template>
  <PortalDialog :open="!!apiKey" :title="text('密钥用量详情', 'Key usage')" wide @close="$emit('close')">
    <div class="key-usage" v-if="apiKey">
      <p class="key-usage-name">{{ apiKey.name }} <span>· {{ timezone }}</span></p>
      <div v-if="billingInfo" class="key-billing-strip">
        <div>
          <small>{{ text('当前生效倍率', 'Effective multiplier') }}</small>
          <strong>{{ formatMultiplier(billingInfo.effective_rate_multiplier) }}</strong>
        </div>
        <div>
          <small>{{ text('分组倍率', 'Group rate') }}</small>
          <strong>{{ formatMultiplier(billingInfo.group_rate_multiplier) }}</strong>
        </div>
        <div v-if="billingInfo.user_rate_multiplier != null">
          <small>{{ text('个人专属倍率', 'Personal override') }}</small>
          <strong>{{ formatMultiplier(billingInfo.user_rate_multiplier) }}</strong>
        </div>
        <div v-if="billingInfo.peak_rate_enabled && billingInfo.peak_start && billingInfo.peak_end">
          <small>{{ peakActive ? text('高峰时段（生效中）', 'Peak window (active)') : text('高峰时段', 'Peak window') }}</small>
          <strong>{{ billingInfo.peak_start }}–{{ billingInfo.peak_end }} ({{ formatMultiplier(billingInfo.peak_rate_multiplier) }})</strong>
        </div>
      </div>
      <div class="key-usage-controls" role="group" :aria-label="text('时间范围', 'Date range')">
        <button v-for="item in presets" :key="item.days" type="button" :aria-pressed="period === item.days" @click="setPeriod(item.days)">{{ item.label }}</button>
        <button type="button" :aria-pressed="period === 0" @click="period = 0">{{ text('自定义', 'Custom') }}</button>
      </div>
      <form v-if="period === 0" class="key-usage-dates" @submit.prevent="load">
        <label>{{ text('开始', 'From') }} <input v-model="start" type="date" required /></label>
        <label>{{ text('结束', 'To') }} <input v-model="end" type="date" required :min="start" /></label>
        <button class="portal-button secondary" :disabled="loading || !start || !end || start > end">{{ text('查询', 'Apply') }}</button>
      </form>
      <p v-if="loading" role="status">{{ text('正在读取用量…', 'Loading usage…') }}</p>
      <p v-else-if="error" class="portal-error" role="alert">{{ error }} <button type="button" class="portal-inline-button" @click="load">{{ text('重试', 'Retry') }}</button></p>
      <template v-else-if="data">
        <p class="key-usage-range">{{ data.start_date }} – {{ data.end_date }} · {{ data.timezone }}</p>
        <div class="key-usage-summary">
          <div><small>{{ text('实际费用', 'Actual cost') }}</small><strong>{{ money(data.range.actual_cost) }}</strong></div>
          <div><small>{{ text('请求数', 'Requests') }}</small><strong>{{ number(data.range.requests) }}</strong></div>
          <div><small>Tokens</small><strong>{{ number(data.range.total_tokens) }}</strong></div>
          <div><small>{{ text('今日费用', 'Today') }}</small><strong>{{ money(data.today.actual_cost) }}</strong></div>
        </div>
        <div class="key-usage-scroll"><div v-if="mobile" class="mobile-records"><dl v-for="row in data.items" :key="row.date" class="mobile-record">
  <div  class="mobile-record-field"><dt>{{ text('日期', 'Date') }}</dt><dd >{{ row.date }}</dd></div>
  <div  class="mobile-record-field"><dt>{{ text('请求', 'Requests') }}</dt><dd >{{ number(row.requests) }}</dd></div>
  <div  class="mobile-record-field"><dt>Tokens</dt><dd >{{ number(row.total_tokens) }}</dd></div>
  <div  class="mobile-record-field"><dt>{{ text('实际费用', 'Actual cost') }}</dt><dd >{{ money(row.actual_cost) }}</dd></div>
</dl>
</div>
<table v-else><thead><tr><th>{{ text('日期', 'Date') }}</th><th>{{ text('请求', 'Requests') }}</th><th>Tokens</th><th>{{ text('实际费用', 'Actual cost') }}</th></tr></thead><tbody><tr v-for="row in data.items" :key="row.date"><td>{{ row.date }}</td><td>{{ number(row.requests) }}</td><td>{{ number(row.total_tokens) }}</td><td>{{ money(row.actual_cost) }}</td></tr></tbody></table></div>
        <p v-if="!data.items.length">{{ text('该时间范围没有用量记录。', 'No usage in this range.') }}</p>
      </template>
    </div>
    <template #footer><div class="portal-dialog-actions"><button type="button" class="portal-button secondary" @click="$emit('close')">{{ text('关闭', 'Close') }}</button></div></template>
  </PortalDialog>
</template>
<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getKeyUsageDetail, type KeyUsageDetail } from '@/api/usage'
import { formatMultiplier, getKeyBillingInfo, isPeakRateActive, type KeyBillingInfo } from '@/api/keyBilling'
import type { ApiKey } from '@/types'
import PortalDialog from './PortalDialog.vue'
const props = defineProps<{ apiKey: ApiKey | null }>()
defineEmits<{ close: [] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
const period = ref(7), start = ref(''), end = ref('')
const presets = computed(() => [{days:1,label:text('今日','Today')},{days:7,label:text('近 7 天','7 days')},{days:30,label:text('近 30 天','30 days')}])
const data = ref<KeyUsageDetail | null>(null), loading = ref(false), error = ref('')
const billingInfo = ref<KeyBillingInfo | null>(null)
const peakActive = computed(() => isPeakRateActive(billingInfo.value))
let controller: AbortController | null = null
let billingController: AbortController | null = null
let sequence = 0
const number = (value: number) => new Intl.NumberFormat(locale.value).format(value)
const money = (value: number) => `$${Number(value).toFixed(4)}`
function setPeriod(days: number) { period.value = days; void load() }
async function loadBilling(secret: string | undefined) {
  billingController?.abort()
  billingInfo.value = null
  if (!secret) return
  billingController = new AbortController()
  const res = await getKeyBillingInfo(secret, billingController.signal)
  if (res.data) billingInfo.value = res.data
}
async function load() {
  const id = props.apiKey?.id
  if (!id || (period.value === 0 && (!start.value || !end.value || start.value > end.value))) return
  const current = ++sequence
  controller?.abort(); controller = new AbortController()
  const signal = controller.signal
  data.value = null; error.value = ''; loading.value = true
  try {
    const result = await getKeyUsageDetail(id, period.value ? { days: period.value, timezone } : { start_date: start.value, end_date: end.value, timezone }, signal)
    if (current === sequence && !signal.aborted && props.apiKey?.id === id && result.api_key_id === id) data.value = result
  } catch {
    if (current === sequence && !signal.aborted) error.value = text('用量加载失败，请重试。', 'Unable to load usage. Please retry.')
  } finally { if (current === sequence) loading.value = false }
}
watch(() => props.apiKey?.id, id => { ++sequence; controller?.abort(); data.value = null; error.value = ''; loading.value = false; period.value = 7; void loadBilling(props.apiKey?.key); if (id) void load() }, { immediate: true })
onBeforeUnmount(() => { ++sequence; controller?.abort(); billingController?.abort() })
</script>
<style scoped>
.key-usage { padding: 0 24px 24px; } .key-usage-name { font-weight:600; } .key-usage-name span,.key-usage-range { color:rgb(var(--ink-3)); font-size:12px; font-weight:400; }
.key-billing-strip { display:grid; grid-template-columns:repeat(auto-fit,minmax(130px,1fr)); gap:8px; margin:12px 0; padding:10px 12px; border-radius:12px; background:var(--md-sys-color-primary-container); color:var(--md-sys-color-on-primary-container); }
.key-billing-strip > div { display:flex; flex-direction:column; gap:2px; }
.key-billing-strip small { font-size:11.5px; opacity:0.8; }
.key-billing-strip strong { font-size:14px; font-family:var(--font-mono); }
.key-usage-controls,.key-usage-dates { display:flex; flex-wrap:wrap; gap:8px; margin:16px 0; align-items:center; }
.key-usage-controls button { padding:7px 12px; border:1px solid rgb(var(--line)); border-radius:6px; } .key-usage-controls button[aria-pressed=true] { background:rgb(var(--action-bg)); color:rgb(var(--action-text)); }
.key-usage-dates label { display:flex; align-items:center; gap:6px; } .key-usage-summary { display:grid; grid-template-columns:repeat(4,1fr); gap:10px; margin:16px 0; } .key-usage-summary div { display:grid; gap:6px; padding:12px; background:rgb(var(--surface-2)); border-radius:8px; } .key-usage-summary small { color:rgb(var(--ink-3)); } .key-usage-scroll { overflow:auto; max-height:300px; } table { width:100%; border-collapse:collapse; font-size:13px; } th,td { padding:10px; border-bottom:1px solid rgb(var(--line)); text-align:right; white-space:nowrap; } th:first-child,td:first-child { text-align:left; }
@media (max-width:767px), (max-width:1023px) and (max-height:500px) and (pointer:coarse) { .key-usage { padding:0 0 16px; } .key-usage-summary { grid-template-columns:repeat(2,minmax(0,1fr)); overflow-wrap:anywhere; } .key-usage-scroll { max-height:none; overflow:visible; } .key-usage-dates label { flex-wrap:wrap; min-width:0; } .key-usage-dates input { min-width:0; max-width:100%; min-height:44px; } .key-usage-controls button { min-height:44px; } }
</style>
