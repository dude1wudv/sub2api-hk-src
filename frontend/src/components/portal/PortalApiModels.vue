<template>
  <section class="portal-section portal-api-models" aria-labelledby="portal-api-models-title">
    <div class="portal-section-heading">
      <h2 id="portal-api-models-title">{{ text('选择接入方式', 'Choose a connection') }}</h2>
      <label class="portal-api-model-search"><Icon name="search" size="sm" aria-hidden="true" /><input type="search" :value="search" :aria-label="text('搜索模型', 'Search models')" :placeholder="text('搜索模型', 'Search models')" @input="$emit('update:search', ($event.target as HTMLInputElement).value)" /></label>
    </div>

    <div v-if="connections.length" class="portal-api-group-tabs" role="group" :aria-label="text('接入分组', 'Connection groups')">
      <button v-for="connection in connections" :key="connection.id" type="button" :aria-pressed="selectedGroupId === connection.id" @click="$emit('update:selectedGroupId', connection.id)">
        {{ connection.name }}<span v-if="showCreate && !bindableGroupIds.includes(connection.id)">{{ text('价格预览', 'Pricing preview') }}</span>
      </button>
    </div>
    <p v-if="selectedGroup?.description" class="portal-api-group-description">{{ selectedGroup.description }}</p>

    <div v-if="!pricingEnabled" class="portal-api-model-state">{{ text('模型价格暂未开放。你仍可创建和管理 API Key。', 'Model pricing is not published. You can still create and manage API keys.') }}</div>
    <div v-else-if="loading" class="portal-api-model-state" role="status">{{ text('正在加载模型价格…', 'Loading model pricing…') }}</div>
    <div v-else-if="error" class="portal-error" role="alert">{{ error }} <button type="button" class="portal-inline-button" @click="$emit('retry')">{{ text('重试', 'Retry') }}</button></div>
    <template v-else>
      <p class="portal-api-pricing-note">{{ text('标准时段实际单价，Token 模型按 USD / 1M tokens 展示。阶梯、分时、高峰及推理倍率请查看模型详情。', 'Standard-period prices for this group. Token prices are USD / 1M tokens. Open model details for tiers, schedules, peak and reasoning multipliers.') }}</p>
      <div class="portal-api-model-table-wrap">
        <table class="portal-api-model-table">
          <thead><tr><th>{{ text('模型', 'Model') }}</th><th>{{ text('输入', 'Input') }}</th><th>{{ text('缓存写', 'Cache write') }}</th><th>{{ text('缓存读', 'Cache read') }}</th><th>{{ text('输出', 'Output') }}</th></tr></thead>
          <tbody>
            <tr v-for="model in filteredModels" :key="model.platform + ':' + model.name">
              <td>
                <button type="button" class="portal-api-model-name" @click="detailModel = model">{{ model.name }}</button>
                <span v-if="modelBadges(model).length" class="portal-api-model-badges"><span v-for="badge in modelBadges(model)" :key="badge">{{ badge }}</span></span>
              </td>
              <td v-if="!model.pricing" colspan="4" class="portal-api-price-missing">{{ text('尚未提供实际单价', 'Actual pricing is not provided') }}</td>
              <template v-else-if="model.pricing.billing_mode === 'token'">
                <td>{{ rangePrice(model, 'input_price') }}</td>
                <td><span>{{ rangePrice(model, 'cache_write_price') }}</span><small v-if="hasHourlyCache(model)">1h {{ rangePrice(model, 'cache_write_1h_price') }}</small></td>
                <td>{{ rangePrice(model, 'cache_read_price') }}</td>
                <td>{{ rangePrice(model, 'output_price') }}</td>
              </template>
              <td v-else colspan="4"><span>{{ billingLabel(model) }} · {{ rangePrice(model, 'per_request_price') }} {{ unitLabel(model) }}</span></td>
            </tr>
            <tr v-if="!filteredModels.length"><td colspan="5" class="portal-api-model-empty">{{ emptyMessage }}</td></tr>
          </tbody>
        </table>
      </div>
      <div v-if="descriptionHtml" class="portal-api-price-description" v-html="descriptionHtml" />
    </template>

    <div v-if="showCreate" class="portal-api-model-create">
      <button type="button" class="portal-button" :disabled="creationDisabled || selectedCannotBind" @click="$emit('create')">＋ {{ text('创建 API Key', 'Create API key') }}</button>
      <span v-if="selectedCannotBind">{{ text('此分组可查看价格，当前账户无法绑定。请选择可用分组。', 'Pricing is visible, but this account cannot bind this group. Choose an available group.') }}</span>
    </div>

    <PortalDialog :open="!!detailModel" :title="detailModel?.name || text('定价详情', 'Pricing details')" wide @close="detailModel = null">
      <div v-if="detailModel" class="portal-api-price-detail">
        <p v-if="!detailModel.pricing">{{ text('此模型尚未提供实际单价。', 'Actual pricing is not provided for this model.') }}</p>
        <template v-else-if="selectedGroup">
          <dl class="portal-api-price-summary">
            <div><dt>{{ text('接入分组', 'Group') }}</dt><dd>{{ selectedGroup.name }}</dd></div>
            <div><dt>{{ text('计费方式', 'Billing mode') }}</dt><dd>{{ billingLabel(detailModel) }}</dd></div>
            <div><dt>{{ text('生效倍率', 'Effective multiplier') }}</dt><dd>{{ effectiveRate(detailModel) == null ? '—' : '×' + effectiveRate(detailModel) }} <span v-if="independentMediaRate(detailModel)">{{ text('（独立媒体倍率）', '(independent media rate)') }}</span></dd></div>
            <div v-if="selectedGroup.user_rate_multiplier != null && !independentMediaRate(detailModel)"><dt>{{ text('用户专属倍率', 'User-specific rate') }}</dt><dd>{{ text('已使用专属倍率替代分组默认倍率', 'Replaces the default group multiplier') }} ×{{ selectedGroup.rate_multiplier }}</dd></div>
          </dl>
          <p>{{ text('以下为当前分组标准时段实际价格。— 表示未提供单价；0 表示价格为零。', 'These are actual standard-period prices for this group. — means not provided; 0 is a zero price.') }}</p>
          <p v-if="detailModel.pricing.billing_mode === 'token'">{{ text('单位：USD / 1M tokens。缓存写入默认展示 5 分钟与 1 小时两种有效期。', 'Unit: USD / 1M tokens. Cache-write durations are 5 minutes and 1 hour.') }}</p>
          <p v-if="detailTiers.length > 1">{{ detailModel.long_context_basis === 'marginal' ? text('阶梯采用边际计价：仅超出阈值的部分按对应档位计费。', 'Marginal tiers: only usage beyond a threshold is charged at that tier.') : text('阶梯采用整单计价：整个请求按所处档位的单价计费。', 'Whole-request tiers: the entire request uses the matching tier price.') }}</p>
          <p v-if="detailModel.pricing.billing_mode === 'token' && selectedGroup.long_context_pricing_enabled === false">{{ text('此分组未启用长上下文阶梯，当前展示基础档位。', 'Long-context tiering is disabled for this group; the base tier is shown.') }}</p>
          <div class="portal-api-detail-table-wrap">
            <table class="portal-api-detail-table">
              <thead><tr><th>{{ text('档位', 'Tier') }}</th><template v-if="detailModel.pricing.billing_mode === 'token'"><th>{{ text('输入', 'Input') }}</th><th>{{ text('缓存写 5m', 'Cache write 5m') }}</th><th>{{ text('缓存写 1h', 'Cache write 1h') }}</th><th>{{ text('缓存读', 'Cache read') }}</th><th>{{ text('输出', 'Output') }}</th></template><th v-else>{{ text('价格', 'Price') }} {{ unitLabel(detailModel) }}</th></tr></thead>
              <tbody><tr v-for="(tier, index) in detailTiers" :key="index"><td>{{ tierLabel(tier) }}</td><template v-if="detailModel.pricing.billing_mode === 'token'"><td v-for="field in tokenFields" :key="field">{{ actualPrice(detailModel, tier[field]) }}</td></template><td v-else>{{ actualPrice(detailModel, tier.per_request_price) }}</td></tr></tbody>
            </table>
          </div>
          <div v-if="detailModel.pricing.billing_mode === 'token' && (detailModel.pricing.image_input_price != null || detailModel.pricing.image_output_price != null)" class="portal-api-pricing-rule">
            <h3>{{ text('图像 Token 单价', 'Image token prices') }}</h3>
            <p>{{ text('输入', 'Input') }} {{ actualPrice(detailModel, detailModel.pricing.image_input_price) }} / 1M tokens · {{ text('输出', 'Output') }} {{ actualPrice(detailModel, detailModel.pricing.image_output_price) }} / 1M tokens</p>
          </div>
          <div v-if="detailModel.time_pricing?.periods.length" class="portal-api-pricing-rule">
            <h3>{{ text('分时倍率', 'Time-based multipliers') }}</h3>
            <p>{{ detailModel.time_pricing.timezone }} · {{ detailModel.time_pricing.weekdays_only ? text('仅工作日；周末使用标准价', 'Weekdays only; standard pricing on weekends') : text('每日生效', 'Applies every day') }}</p>
            <ul><li v-for="(period, index) in detailModel.time_pricing.periods" :key="index">{{ period.start_time }}–{{ period.end_time }} · ×{{ period.multiplier }}</li></ul>
            <p>{{ text('命中时段的请求在标准价基础上乘对应倍率；与高峰窗口重叠时，还会乘高峰倍率。', 'Matching requests multiply the standard price by the period rate. Overlapping peak windows add the peak multiplier.') }}</p>
          </div>
          <div v-if="hasPeakRate(selectedGroup)" class="portal-api-pricing-rule">
            <h3>{{ text('高峰时段', 'Peak window') }}</h3><p>{{ peakWindow }}</p>
            <p>{{ text('该窗口内的请求还会乘以上述高峰倍率。', 'Requests in this window also apply the peak multiplier above.') }}</p>
          </div>
          <div v-if="reasoningRates(detailModel).length" class="portal-api-pricing-rule">
            <h3>{{ text('推理强度倍率', 'Reasoning-effort multipliers') }}</h3>
            <ul><li v-for="[effort, multiplier] in reasoningRates(detailModel)" :key="effort">{{ effort }} · ×{{ multiplier }}</li></ul>
          </div>
        </template>
        <footer class="portal-dialog-actions"><button type="button" class="portal-button secondary" @click="detailModel = null">{{ text('关闭', 'Close') }}</button><button v-if="showCreate && detailModel.pricing?.billing_mode === 'token'" type="button" class="portal-button" @click="chooseExampleModel">{{ text('查看调用示例', 'View request example') }}</button></footer>
      </div>
    </PortalDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import PortalDialog from './PortalDialog.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'
import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'

const props = withDefaults(defineProps<{
  groups: ModelPlazaGroup[]
  availableConnections?: { id: number; name: string }[]
  bindableGroupIds?: number[]
  selectedGroupId: number | null
  search?: string
  description?: string
  serverUtcOffset?: string
  pricingEnabled?: boolean
  loading?: boolean
  error?: string
  showCreate?: boolean
  creationDisabled?: boolean
}>(), { availableConnections: () => [], bindableGroupIds: () => [], search: '', description: '', serverUtcOffset: '', pricingEnabled: true, loading: false, error: '', showCreate: false, creationDisabled: false })
const emit = defineEmits<{
  'update:selectedGroupId': [id: number | null]
  'update:search': [search: string]
  retry: []
  create: []
  'select-model': [name: string]
}>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const descriptionHtml = computed(() => DOMPurify.sanitize(marked.parse(props.description, { async: false, breaks: true })))
type PriceField = 'input_price' | 'cache_write_price' | 'cache_write_1h_price' | 'cache_read_price' | 'output_price' | 'per_request_price'
const tokenFields = ['input_price', 'cache_write_price', 'cache_write_1h_price', 'cache_read_price', 'output_price'] as const
const detailModel = ref<PlazaModel | null>(null)
const connections = computed(() => {
  const result = new Map<number, { id: number; name: string }>()
  for (const group of [...props.availableConnections, ...props.groups]) {
    if (!result.has(group.id)) result.set(group.id, { id: group.id, name: group.name })
  }
  return [...result.values()]
})
const selectedGroup = computed(() => props.groups.find(group => group.id === props.selectedGroupId))
const selectedCannotBind = computed(() => props.selectedGroupId != null && !props.bindableGroupIds.includes(props.selectedGroupId))
const filteredModels = computed(() => {
  const search = props.search.trim().toLowerCase()
  return (selectedGroup.value?.models || []).filter(model => !search || (model.name + ' ' + model.platform).toLowerCase().includes(search))
})
const emptyMessage = computed(() => props.search.trim()
  ? text('没有找到匹配的模型。', 'No matching models.')
  : text('此分组暂未公布模型价格。', 'No model pricing is published for this group.'))
const detailTiers = computed(() => detailModel.value ? pricingTiers(detailModel.value) : [])
const peakWindow = computed(() => formatPeakRateWindow(selectedGroup.value, serverTimezoneLabel(props.serverUtcOffset) || text('服务器时区', 'server timezone')))

watch(connections, groups => {
  if (!groups.some(group => group.id === props.selectedGroupId)) emit('update:selectedGroupId', groups[0]?.id ?? null)
}, { immediate: true })
watch(() => props.selectedGroupId, () => { detailModel.value = null })

function independentMediaRate(model: PlazaModel) {
  return (model.pricing?.billing_mode === 'image' && selectedGroup.value?.image_rate_independent === true)
    || (model.pricing?.billing_mode === 'video' && selectedGroup.value?.video_rate_independent === true)
}
function effectiveRate(model: PlazaModel): number | null {
  const group = selectedGroup.value
  if (!group) return null
  const rate = model.pricing?.billing_mode === 'image' && group.image_rate_independent
    ? group.image_rate_multiplier ?? 1
    : model.pricing?.billing_mode === 'video' && group.video_rate_independent
      ? group.video_rate_multiplier ?? 1
      : group.user_rate_multiplier ?? group.rate_multiplier
  return Number.isFinite(rate) && rate >= 0 ? rate : null
}
function pricingTiers(model: PlazaModel): UserPricingInterval[] {
  const pricing = model.pricing
  if (!pricing) return []
  const intervals = [...(pricing.intervals || [])].sort((a, b) => a.min_tokens - b.min_tokens)
  if (!intervals.length) return [{ min_tokens: 0, max_tokens: null, tier_label: text('基础价', 'Base'), ...pricing }]
  if (pricing.billing_mode !== 'token') return intervals.map(interval => ({ ...interval, per_request_price: interval.per_request_price ?? pricing.per_request_price }))
  const effective = selectedGroup.value?.long_context_pricing_enabled === false ? intervals.slice(0, 1) : intervals
  return effective.map(interval => resolveIntervalPrices(interval, pricing))
}
function actualPrice(model: PlazaModel, value: number | null | undefined) {
  const rate = effectiveRate(model)
  if (value == null || !Number.isFinite(value) || value < 0 || rate == null) return '—'
  return formatScaled(value * rate, model.pricing?.billing_mode === 'token' ? 1_000_000 : 1, 2)
}
function rangePrice(model: PlazaModel, field: PriceField) {
  const values = pricingTiers(model).map(tier => tier[field])
  const known = values.filter((value): value is number => value != null && Number.isFinite(value) && value >= 0)
  if (!known.length) return '—'
  if (known.length !== values.length) return text('部分未定价', 'Partly unpriced')
  const low = Math.min(...known)
  const high = Math.max(...known)
  const lowPrice = actualPrice(model, low)
  const highPrice = actualPrice(model, high)
  return lowPrice === highPrice ? lowPrice : lowPrice + '–' + highPrice
}
function hasHourlyCache(model: PlazaModel) {
  return pricingTiers(model).some(tier => tier.cache_write_1h_price != null)
}
function billingLabel(model: PlazaModel) {
  const mode = model.pricing?.billing_mode
  if (mode === 'token') return text('按 Token', 'Per token')
  if (mode === 'image') return text('按图片', 'Per image')
  if (mode === 'video') return text('按视频请求', 'Per video request')
  return text('按请求', 'Per request')
}
function unitLabel(model: PlazaModel) {
  return model.pricing?.billing_mode === 'image' ? text('/ 张', '/ image') : text('/ 次', '/ request')
}
function tierLabel(tier: UserPricingInterval) {
  if (tier.tier_label) return tier.tier_label
  return tier.max_tokens == null ? '>' + tier.min_tokens.toLocaleString() : tier.min_tokens.toLocaleString() + '–' + tier.max_tokens.toLocaleString()
}
function reasoningRates(model: PlazaModel): [string, number][] {
  return Object.entries(model.pricing?.reasoning_effort_multipliers || {}).filter(([, value]) => Number.isFinite(value) && value >= 0)
}
function modelBadges(model: PlazaModel) {
  return [
    pricingTiers(model).length > 1 ? text('阶梯', 'Tiers') : '',
    model.time_pricing?.periods.length ? text('分时', 'Scheduled') : '',
    hasPeakRate(selectedGroup.value) ? text('高峰', 'Peak') : '',
    reasoningRates(model).some(([, value]) => value !== 1) ? text('推理倍率', 'Reasoning') : '',
  ].filter(Boolean)
}
function chooseExampleModel() {
  if (detailModel.value) emit('select-model', detailModel.value.name)
  detailModel.value = null
}
</script>

<style scoped>
.portal-api-model-search { display:flex; align-items:center; gap:8px; min-width:0; width:190px; padding:5px 0; border-bottom:1px solid #efede3; }
.portal-api-model-search input { min-width:0; width:100%; border:0; outline:none; background:transparent; font:inherit; font-size:12px; }
.portal-api-model-search:focus-within { border-bottom-color:#000; }
.portal-api-group-tabs { display:flex; gap:8px; flex-wrap:wrap; margin:16px 0 14px; }
.portal-api-group-tabs button { display:inline-flex; align-items:center; gap:6px; min-height:34px; padding:6px 16px; border:1px solid #efede3; border-radius:8px; background:transparent; color:#000; font:inherit; }
.portal-api-group-tabs button[aria-pressed='true'] { border-color:#000; background:#f3f0eb; }
.portal-api-group-tabs button > span { font-size:10px; opacity:.55; }
.portal-api-group-description,.portal-api-pricing-note { margin:0 0 12px; font-size:12px; line-height:1.7; color:#76736b; }
.portal-api-model-table-wrap { max-height:400px; overflow:auto; border:1px solid #efede3; border-radius:8px; }
.portal-api-model-table { width:100%; border-collapse:separate; border-spacing:0; text-align:left; font-size:12px; }
.portal-api-model-table th { position:sticky; top:0; z-index:1; padding:13px 16px; color:#76736b; background:#f7f5ee; font-weight:400; white-space:nowrap; }
.portal-api-model-table td { min-width:86px; padding:14px 16px; border-bottom:1px solid #efede3; font-variant-numeric:tabular-nums; vertical-align:middle; }
.portal-api-model-table td:first-child { width:42%; min-width:180px; }
.portal-api-model-table tbody tr:last-child td { border-bottom:0; }
.portal-api-model-table td small { display:block; margin-top:3px; white-space:nowrap; font-size:10px; color:#76736b; }
.portal-api-model-name { display:block; padding:0; border:0; background:none; color:#000; text-align:left; overflow-wrap:anywhere; cursor:pointer; }
.portal-api-model-name:hover { text-decoration:underline; text-underline-offset:3px; }
.portal-api-model-badges { display:flex; flex-wrap:wrap; gap:4px; margin-top:5px; }
.portal-api-model-badges span { padding:0 4px; border:1px solid #efede3; border-radius:3px; font-size:9px; color:#76736b; }
.portal-api-price-missing { color:#76736b; }
.portal-api-model-table .portal-api-model-empty { height:128px; text-align:center; color:#76736b; }
.portal-api-model-state { margin:16px 0; padding:32px 20px; border:1px solid #efede3; border-radius:8px; color:#76736b; text-align:center; }
.portal-api-price-description { margin:12px 0 0; color:#76736b; font-size:12px; line-height:1.7; white-space:pre-wrap; overflow-wrap:anywhere; }
.portal-api-model-create { display:flex; align-items:center; flex-wrap:wrap; gap:12px; margin-top:20px; }
.portal-api-model-create > span { max-width:420px; font-size:12px; color:#76736b; }
.portal-api-price-detail { font-size:13px; line-height:1.7; }
.portal-api-price-summary { display:grid; gap:8px; margin:0 0 20px; }
.portal-api-price-summary > div { display:flex; flex-wrap:wrap; gap:8px 18px; }
.portal-api-price-summary dt { min-width:100px; color:#76736b; }
.portal-api-price-summary dd { margin:0; }
.portal-api-detail-table-wrap { overflow:auto; margin-top:12px; border:1px solid #efede3; border-radius:8px; }
.portal-api-detail-table { width:100%; border-collapse:collapse; font-size:12px; text-align:left; }
.portal-api-detail-table th { background:#f7f5ee; font-weight:400; }
.portal-api-detail-table :is(th,td) { padding:10px; border-bottom:1px solid #efede3; white-space:nowrap; }
.portal-api-detail-table tr:last-child td { border-bottom:0; }
.portal-api-pricing-rule { margin-top:20px; padding-top:16px; border-top:1px solid #efede3; }
.portal-api-pricing-rule h3 { margin:0 0 8px; font-size:14px; font-weight:500; }
.portal-api-pricing-rule p { margin:6px 0; }
.portal-api-pricing-rule ul { margin:8px 0; padding-left:20px; list-style:disc; }
@media(max-width:520px) { .portal-api-model-search { width:100%; } .portal-api-models .portal-section-heading { align-items:flex-start; } .portal-api-model-table :is(th,td) { padding-inline:12px; } }
</style>
