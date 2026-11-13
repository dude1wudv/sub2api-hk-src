<template>
  <section class="portal-api-models" :aria-label="text('模型目录', 'Model catalogue')" :aria-busy="loading">
    <aside class="portal-api-sidebar">
      <div class="portal-api-sidebar-heading"><h2>{{ text('接入分组', 'Connections') }}</h2><span>{{ connections.length }}</span></div>
      <label class="portal-api-model-search portal-api-group-search"><Icon name="search" size="sm" aria-hidden="true" /><input v-model="groupSearch" type="search" :aria-label="text('搜索分组', 'Search connections')" :placeholder="text('搜索分组', 'Search connections')" /></label>
      <select class="portal-api-mobile-groups input" :value="selectedGroupId ?? ''" :aria-label="text('接入分组', 'Connections')" @change="$emit('update:selectedGroupId', Number(($event.target as HTMLSelectElement).value))">
        <option v-for="connection in connections" :key="connection.id" :value="connection.id">{{ connection.name }} · {{ connection.count }}</option>
      </select>
      <nav class="portal-api-group-nav" :aria-label="text('接入分组', 'Connections')">
        <button v-for="connection in visibleConnections" :key="connection.id" type="button" :aria-current="selectedGroupId === connection.id ? 'true' : undefined" :aria-pressed="selectedGroupId === connection.id" @click="$emit('update:selectedGroupId', connection.id)">
          <span class="portal-api-connection-name">{{ connection.name }}</span>
          <span class="portal-api-connection-meta"><span>{{ connection.count }} {{ text('模型', 'models') }}</span><span v-if="connection.rate != null">×{{ connection.rate }}</span></span>
          <small v-if="showCreate && !bindableGroupIds.includes(connection.id)">{{ text('仅价格预览', 'Pricing preview only') }}</small>
        </button>
        <p v-if="!visibleConnections.length" class="portal-api-sidebar-empty">{{ text('没有匹配的分组', 'No matching connections') }}</p>
      </nav>
    </aside>

    <div class="portal-api-catalogue">
      <header class="portal-api-catalogue-header">
        <div><p class="portal-api-eyebrow">{{ text('模型目录 / 按分组接入', 'MODEL CATALOGUE / CONNECTION') }}</p><h2>{{ selectedGroup?.name || connections.find(group => group.id === selectedGroupId)?.name || text('选择接入分组', 'Choose a connection') }}</h2></div>
        <label class="portal-api-model-search"><Icon name="search" size="sm" aria-hidden="true" /><input type="search" :value="search" :aria-label="text('搜索模型', 'Search models')" :placeholder="text('搜索模型或厂商', 'Search models or vendors')" @input="$emit('update:search', ($event.target as HTMLInputElement).value)" /></label>
      </header>
      <template v-if="selectedGroup">
        <p v-if="selectedGroup.description" class="portal-api-group-description">{{ selectedGroup.description }}</p>
        <div class="portal-api-group-facts">
          <span>{{ selectedGroup.models.length }} {{ text('个模型', 'models') }}</span>
          <span>{{ selectedGroup.subscription_type === 'subscription' ? text('订阅分组', 'Subscription') : text('按量计费', 'Pay as you go') }}</span>
          <span v-if="selectedGroup.is_exclusive">{{ text('专属分组', 'Exclusive') }}</span>
          <span>{{ selectedGroup.user_rate_multiplier != null ? text('专属倍率', 'Personal rate') : text('分组倍率', 'Group rate') }} ×{{ selectedGroup.user_rate_multiplier ?? selectedGroup.rate_multiplier }}</span>
          <span v-if="selectedGroup.image_rate_independent">{{ text('图片', 'Image') }} ×{{ selectedGroup.image_rate_multiplier }}</span>
          <span v-if="selectedGroup.video_rate_independent">{{ text('视频', 'Video') }} ×{{ selectedGroup.video_rate_multiplier }}</span>
        </div>
      </template>

      <div v-if="!pricingEnabled" class="portal-api-model-state">{{ text('模型价格暂未开放。你仍可创建和管理 API Key。', 'Model pricing is not published. You can still create and manage API keys.') }}</div>
      <div v-else-if="loading" class="portal-api-model-state" role="status">{{ text('正在加载模型价格…', 'Loading model pricing…') }}</div>
      <div v-else-if="error" class="portal-error" role="alert">{{ error }} <button type="button" class="portal-inline-button" @click="$emit('retry')">{{ text('重试', 'Retry') }}</button></div>
      <template v-else>
        <div v-if="platforms.length" class="portal-model-platforms" role="group" :aria-label="text('模型厂商', 'Model vendors')">
          <button type="button" :aria-pressed="!selectedPlatform" @click="selectedPlatform = ''">{{ text('全部模型', 'All models') }}<span>{{ searchedModels.length }}</span></button>
          <button v-for="platform in platforms" :key="platform.id" type="button" :aria-pressed="selectedPlatform === platform.id" @click="selectedPlatform = platform.id"><ModelIcon :model="platform.models[0]!.name" size="18px" />{{ vendorLabel(platform) }}<span>{{ platform.models.length }}</span></button>
        </div>
        <div class="portal-api-table-heading"><span>{{ text('标准时段实际价格', 'Standard-period prices') }}</span><span>USD / 1M tokens · {{ text('其他计费单位单独标注', 'Other billing units are labelled') }}</span></div>
        <div class="portal-api-model-table-wrap">
          <table class="portal-api-model-table">
            <thead><tr><th scope="col">{{ text('模型', 'Model') }}</th><th scope="col">{{ text('输入', 'Input') }}</th><th scope="col">{{ text('缓存写入', 'Cache write') }}</th><th scope="col">{{ text('缓存读取', 'Cache read') }}</th><th scope="col">{{ text('输出', 'Output') }}</th></tr></thead>
            <tbody v-for="platform in visiblePlatforms" :key="platform.id">
              <tr class="portal-api-vendor-row"><th colspan="5" scope="rowgroup"><span>{{ vendorLabel(platform) }}</span><small>{{ platform.models.length }} {{ text('个模型', 'models') }}</small></th></tr>
              <tr v-for="model in platform.models" :key="model.platform + ':' + model.name" class="portal-api-model-row">
                <td>
                  <button type="button" class="portal-api-model-name" @click="detailModel = model"><ModelIcon :model="model.name" size="20px" /><span>{{ model.name }}</span><Icon name="chevronRight" size="xs" aria-hidden="true" /></button>
                  <span v-if="model.context_window || model.reasoning || model.tool_call" class="portal-api-model-capabilities"><span v-if="model.context_window">{{ compactTokens(model.context_window) }} {{ text('上下文', 'context') }}</span><span v-if="model.reasoning">{{ text('推理', 'Reasoning') }}</span><span v-if="model.tool_call">{{ text('工具调用', 'Tools') }}</span></span>
                  <span v-if="modelBadges(model).length" class="portal-api-model-badges"><span v-for="badge in modelBadges(model)" :key="badge">{{ badge }}</span></span>
                </td>
                <td v-if="!model.pricing" colspan="4" class="portal-api-price-missing">{{ text('尚未提供实际单价', 'Actual pricing is not provided') }}</td>
                <template v-else-if="model.pricing.billing_mode === 'token'">
                  <td>{{ rangePrice(model, 'input_price') }}</td>
                  <td><span>{{ rangePrice(model, 'cache_write_price') }}</span><small v-if="hasCacheDuration(model, 'cache_write_1h_price')">1h {{ rangePrice(model, 'cache_write_1h_price') }}</small><small v-if="hasCacheDuration(model, 'cache_write_12h_price')">12h {{ rangePrice(model, 'cache_write_12h_price') }}</small></td>
                  <td>{{ rangePrice(model, 'cache_read_price') }}</td>
                  <td>{{ rangePrice(model, 'output_price') }}</td>
                </template>
                <td v-else colspan="4"><span>{{ billingLabel(model) }} · {{ rangePrice(model, 'per_request_price') }} {{ unitLabel(model) }}</span></td>
              </tr>
            </tbody>
            <tbody v-if="!visiblePlatforms.length"><tr><td colspan="5" class="portal-api-model-empty"><Icon name="search" size="lg" /><p>{{ emptyMessage }}</p></td></tr></tbody>
          </table>
        </div>
        <p class="portal-api-pricing-note">{{ text('— 表示未提供单价，0 表示免费。点击模型查看完整能力、阶梯与分时定价；实际费用以请求命中的计费规则为准。', '— means unpriced; 0 means free. Select a model for capabilities, tiers and schedules. Charges follow the rules matched by each request.') }}</p>
        <details v-if="descriptionHtml" class="portal-api-catalogue-notes"><summary>{{ text('价格说明', 'Pricing notes') }}</summary><div class="portal-api-price-description" v-html="descriptionHtml" /></details>
      </template>
      <div v-if="showCreate" class="portal-api-model-create">
        <button type="button" class="portal-button" :disabled="creationDisabled || selectedCannotBind" @click="$emit('create')">＋ {{ text('创建 API Key', 'Create API key') }}</button>
        <span v-if="selectedCannotBind">{{ text('此分组可查看价格，当前账户无法绑定。请选择可用分组。', 'Pricing is visible, but this account cannot bind this group. Choose an available group.') }}</span>
      </div>
    </div>

    <PortalDialog :open="!!detailModel" :title="detailModel?.name || text('定价详情', 'Pricing details')" wide @close="detailModel = null">
      <div v-if="detailModel" class="portal-api-price-detail">
        <ModelCapabilities :model="detailModel" />
        <p v-if="!detailModel.pricing">{{ text('此模型尚未提供实际单价。', 'Actual pricing is not provided for this model.') }}</p>
        <template v-else-if="selectedGroup">
          <dl class="portal-api-price-summary">
            <div><dt>{{ text('接入分组', 'Group') }}</dt><dd>{{ selectedGroup.name }}</dd></div>
            <div><dt>{{ text('计费方式', 'Billing mode') }}</dt><dd>{{ billingLabel(detailModel) }}</dd></div>
            <div><dt>{{ text('生效倍率', 'Effective multiplier') }}</dt><dd>{{ effectiveRate(detailModel) == null ? '—' : '×' + effectiveRate(detailModel) }} <span v-if="independentMediaRate(detailModel)">{{ text('（独立媒体倍率）', '(independent media rate)') }}</span></dd></div>
            <div v-if="selectedGroup.user_rate_multiplier != null && !independentMediaRate(detailModel)"><dt>{{ text('用户专属倍率', 'User-specific rate') }}</dt><dd>{{ text('已使用专属倍率替代分组默认倍率', 'Replaces the default group multiplier') }} ×{{ selectedGroup.rate_multiplier }}</dd></div>
          </dl>
          <p>{{ text('以下为当前分组标准时段实际价格。— 表示未提供单价；0 表示价格为零。', 'These are actual standard-period prices for this group. — means not provided; 0 is a zero price.') }}</p>
          <p v-if="detailModel.pricing.billing_mode === 'token'">{{ text('单位：USD / 1M tokens。缓存写入按 5 分钟、1 小时及 12 小时分别列出；未配置价格显示 —。', 'Unit: USD / 1M tokens. Cache writes are listed for 5m, 1h and 12h; unconfigured prices show —.') }}</p>
          <p v-if="detailTiers.length > 1">{{ detailModel.long_context_basis === 'marginal' ? text('阶梯采用边际计价：仅超出阈值的部分按对应档位计费。', 'Marginal tiers: only usage beyond a threshold is charged at that tier.') : text('阶梯采用整单计价：整个请求按所处档位的单价计费。', 'Whole-request tiers: the entire request uses the matching tier price.') }}</p>
          <p v-if="detailModel.pricing.billing_mode === 'token' && selectedGroup.long_context_pricing_enabled === false">{{ text('此分组未启用长上下文阶梯，当前展示基础档位。', 'Long-context tiering is disabled for this group; the base tier is shown.') }}</p>
          <div class="portal-api-detail-table-wrap">
            <table class="portal-api-detail-table">
              <thead><tr><th>{{ text('档位', 'Tier') }}</th><template v-if="detailModel.pricing.billing_mode === 'token'"><th>{{ text('输入', 'Input') }}</th><th>{{ text('缓存写 5m', 'Cache write 5m') }}</th><th>{{ text('缓存写 1h', 'Cache write 1h') }}</th><th>{{ text('缓存写 12h', 'Cache write 12h') }}</th><th>{{ text('缓存读', 'Cache read') }}</th><th>{{ text('输出', 'Output') }}</th></template><th v-else>{{ text('价格', 'Price') }} {{ unitLabel(detailModel) }}</th></tr></thead>
              <tbody><tr v-for="(tier, index) in detailTiers" :key="index"><td>{{ tierLabel(tier) }}</td><template v-if="detailModel.pricing.billing_mode === 'token'"><td v-for="field in tokenFields" :key="field">{{ actualPrice(detailModel, tier[field]) }}</td></template><td v-else>{{ actualPrice(detailModel, tier.per_request_price) }}</td></tr></tbody>
            </table>
          </div>
          <details v-if="detailModel.pricing.billing_mode === 'token' && detailModel.official_pricing" class="portal-api-catalogue-notes">
            <summary>{{ text('官方参考价 · USD / 1M tokens', 'Official reference · USD / 1M tokens') }}</summary>
            <p>{{ text('参考价不参与本站扣费，也不应用分组倍率。', 'Reference prices are not charged here and do not apply group multipliers.') }}</p>
            <div class="portal-api-detail-table-wrap"><table class="portal-api-detail-table">
              <thead><tr><th>{{ text('档位', 'Tier') }}</th><th>{{ text('输入', 'Input') }}</th><th>{{ text('缓存写 5m', 'Cache write 5m') }}</th><th>{{ text('缓存写 1h', 'Cache write 1h') }}</th><th>{{ text('缓存写 12h', 'Cache write 12h') }}</th><th>{{ text('缓存读', 'Cache read') }}</th><th>{{ text('输出', 'Output') }}</th></tr></thead>
              <tbody><tr v-for="(tier, index) in officialTiers" :key="index"><td>{{ tierLabel(tier) }}</td><td v-for="field in tokenFields" :key="field">{{ tier[field] == null ? '—' : formatScaled(tier[field]!, 1_000_000, 2) }}</td></tr></tbody>
            </table></div>
          </details>
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
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelCapabilities from '@/components/modelPlaza/ModelCapabilities.vue'
import { matchesCatalogModel, modelVendor } from '@/utils/modelCatalog'
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
type PriceField = 'input_price' | 'cache_write_price' | 'cache_write_1h_price' | 'cache_write_12h_price' | 'cache_read_price' | 'output_price' | 'per_request_price'
const tokenFields = ['input_price', 'cache_write_price', 'cache_write_1h_price', 'cache_write_12h_price', 'cache_read_price', 'output_price'] as const
const detailModel = ref<PlazaModel | null>(null)
const groupSearch = ref('')
const connections = computed(() => {
  const result = new Map(props.availableConnections.map(group => [group.id, group]))
  for (const group of props.groups) result.set(group.id, group)
  return [...result.values()].map(connection => {
    const group = props.groups.find(group => group.id === connection.id)
    return { ...connection, count: group?.models.length ?? 0, rate: group ? group.user_rate_multiplier ?? group.rate_multiplier : null }
  })
})
const visibleConnections = computed(() => connections.value.filter(group => group.name.toLowerCase().includes(groupSearch.value.trim().toLowerCase())))
const selectedGroup = computed(() => props.groups.find(group => group.id === props.selectedGroupId))
const selectedPlatform = ref('')
const searchedModels = computed(() => (selectedGroup.value?.models || []).filter(model => matchesCatalogModel(model, props.search)))
const platforms = computed(() => {
  const result = new Map<string, { id: string; label: string; models: PlazaModel[] }>()
  for (const model of searchedModels.value) {
    const vendor = modelVendor(model)
    if (!result.has(vendor.id)) result.set(vendor.id, { ...vendor, models: [] })
    result.get(vendor.id)!.models.push(model)
  }
  return [...result.values()]
})
const visiblePlatforms = computed(() => platforms.value.filter(platform => !selectedPlatform.value || platform.id === selectedPlatform.value))
const selectedCannotBind = computed(() => props.selectedGroupId != null && !props.bindableGroupIds.includes(props.selectedGroupId))
const vendorLabel = (vendor: { id: string; label: string }) => vendor.id === 'other' ? text('其他模型', 'Other models') : vendor.label
const compactTokens = (value: number) => new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
const emptyMessage = computed(() => props.search.trim()
  ? text('没有找到匹配的模型。', 'No matching models.')
  : text('此分组暂未公布模型价格。', 'No model pricing is published for this group.'))
const detailTiers = computed(() => detailModel.value ? pricingTiers(detailModel.value) : [])
const officialTiers = computed<UserPricingInterval[]>(() => {
  const pricing = detailModel.value?.official_pricing
  if (!pricing) return []
  return pricing.intervals?.length
    ? [...pricing.intervals].sort((a, b) => a.min_tokens - b.min_tokens).map(tier => resolveIntervalPrices(tier, pricing))
    : [{ min_tokens: 0, max_tokens: null, per_request_price: null, tier_label: text('基础价', 'Base'), ...pricing }]
})
const peakWindow = computed(() => formatPeakRateWindow(selectedGroup.value, serverTimezoneLabel(props.serverUtcOffset) || text('服务器时区', 'server timezone')))

watch(connections, groups => {
  if (!groups.some(group => group.id === props.selectedGroupId)) emit('update:selectedGroupId', groups[0]?.id ?? null)
}, { immediate: true })
watch(() => props.selectedGroupId, () => { detailModel.value = null; selectedPlatform.value = '' })
watch(() => props.search, () => { selectedPlatform.value = '' })

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
function hasCacheDuration(model: PlazaModel, field: 'cache_write_1h_price' | 'cache_write_12h_price') {
  return pricingTiers(model).some(tier => tier[field] != null)
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
.portal-api-models { display:grid; grid-template-columns:230px minmax(0,1fr); border:1px solid rgb(var(--line)); border-radius:16px; background:rgb(var(--surface)); min-width:0; overflow:clip; }
.portal-api-sidebar { padding:22px 12px; border-right:1px solid rgb(var(--line)); background:rgb(var(--surface-2) / .35); min-width:0; }
.portal-api-sidebar-heading { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:0 10px; margin-bottom:16px; }
.portal-api-sidebar-heading h2 { font-size:14px; font-weight:650; margin:0; }
.portal-api-sidebar-heading > span { font-size:12px; color:rgb(var(--ink-3)); font-variant-numeric:tabular-nums; }
.portal-api-model-search { display:flex; align-items:center; gap:8px; min-width:0; width:240px; min-height:40px; padding:0 12px; border:1px solid rgb(var(--line)); border-radius:9px; background:rgb(var(--surface)); color:rgb(var(--ink-3)); }
.portal-api-model-search input { min-width:0; width:100%; border:0; outline:none; background:transparent; color:rgb(var(--ink)); font:inherit; font-size:13px; }
.portal-api-model-search:focus-within { border-color:rgb(var(--focus-ring)); box-shadow:0 0 0 3px rgb(var(--focus-ring) / .14); }
.portal-api-group-search { width:100%; margin-bottom:12px; }
.portal-api-group-nav { display:flex; flex-direction:column; gap:5px; max-height:850px; overflow:auto; }
.portal-api-group-nav button { position:relative; display:flex; flex-direction:column; gap:7px; width:100%; padding:12px; border:1px solid transparent; border-radius:9px; background:transparent; text-align:left; color:rgb(var(--ink-2)); cursor:pointer; }
.portal-api-group-nav button:hover { background:rgb(var(--table-hover)); }
.portal-api-group-nav button[aria-pressed='true'] { background:rgb(var(--brand-moss) / .1); border-color:rgb(var(--brand-moss) / .25); color:rgb(var(--ink)); }
.portal-api-connection-name { font-size:13px; line-height:1.5; font-weight:550; overflow-wrap:anywhere; }
.portal-api-connection-meta { display:flex; justify-content:space-between; gap:8px; font-size:11.5px; color:rgb(var(--ink-3)); font-variant-numeric:tabular-nums; }
.portal-api-group-nav small,.portal-api-sidebar-empty { font-size:12px; color:rgb(var(--ink-3)); }
.portal-api-mobile-groups { display:none; }
.portal-api-catalogue { padding:28px; min-width:0; }
.portal-api-catalogue-header { display:flex; justify-content:space-between; align-items:center; gap:20px; margin-bottom:14px; }
.portal-api-catalogue-header h2 { margin:5px 0 0; font-size:21px; font-weight:650; line-height:1.45; overflow-wrap:anywhere; }
.portal-api-eyebrow { margin:0; color:rgb(var(--ink-3)); font-size:10px; font-weight:600; letter-spacing:.08em; }
.portal-api-group-description { margin:0 0 12px; font-size:13px; color:rgb(var(--ink-3)); line-height:1.7; white-space:pre-line; overflow-wrap:anywhere; }
.portal-api-group-facts { display:flex; flex-wrap:wrap; gap:8px 18px; padding-bottom:20px; border-bottom:1px solid rgb(var(--line)); font-size:12px; color:rgb(var(--ink-3)); }
.portal-model-platforms { display:flex; align-items:center; flex-wrap:wrap; gap:8px; margin:22px 0; }
.portal-model-platforms > button { display:inline-flex; align-items:center; gap:7px; min-height:38px; padding:7px 11px; border:1px solid rgb(var(--line)); border-radius:9px; background:rgb(var(--surface)); color:rgb(var(--ink-2)); font-size:13px; cursor:pointer; }
.portal-model-platforms > button[aria-pressed='true'] { border-color:rgb(var(--ink)); color:rgb(var(--ink)); background:rgb(var(--surface-2)); font-weight:600; }
.portal-model-platforms button > span { font-size:11px; color:rgb(var(--ink-3)); font-variant-numeric:tabular-nums; }
.portal-api-table-heading { display:flex; justify-content:space-between; flex-wrap:wrap; gap:8px; color:rgb(var(--ink-3)); font-size:11.5px; padding:0 0 12px; }
.portal-api-model-table-wrap { overflow-x:auto; }
.portal-api-model-table { width:100%; border-collapse:separate; border-spacing:0; text-align:left; font-size:13px; }
.portal-api-model-table thead th { padding:13px 12px; color:rgb(var(--ink-3)); border-bottom:1px solid rgb(var(--line)); font-size:12px; font-weight:500; white-space:nowrap; text-align:right; }
.portal-api-model-table thead th:first-child { text-align:left; }
.portal-api-model-table td { min-width:90px; padding:17px 12px; border-bottom:1px solid rgb(var(--line)); font-variant-numeric:tabular-nums; vertical-align:middle; text-align:right; }
.portal-api-model-table td:first-child { width:42%; min-width:220px; text-align:left; }
.portal-api-vendor-row th { padding:10px 12px; background:rgb(var(--surface-2) / .7); font-size:13px; font-weight:600; text-align:left; border-block:8px solid rgb(var(--surface)); }
.portal-api-vendor-row small { margin-left:10px; color:rgb(var(--ink-3)); font-size:11px; font-weight:400; }
.portal-api-model-row:hover td { background:rgb(var(--table-hover)); }
.portal-api-model-table td small { display:block; margin-top:5px; white-space:nowrap; font-size:11px; color:rgb(var(--ink-3)); }
.portal-api-model-name { display:inline-flex; align-items:center; gap:10px; min-height:30px; padding:0; border:0; background:none; color:rgb(var(--ink)); font-size:13px; font-weight:550; text-align:left; overflow-wrap:anywhere; cursor:pointer; }
.portal-api-model-name > :last-child { color:rgb(var(--ink-3)); opacity:.6; }
.portal-api-model-name :deep(svg) { flex-shrink:0; }
.portal-api-model-name:hover { color:rgb(var(--brand-moss)); }
.portal-api-model-capabilities { display:flex; flex-wrap:wrap; gap:10px; margin:4px 0 0 30px; font-size:11px; color:rgb(var(--ink-3)); }
.portal-api-model-badges { display:flex; flex-wrap:wrap; gap:5px; margin:6px 0 0 30px; }
.portal-api-model-badges span { padding:1px 6px; border-radius:4px; background:rgb(var(--surface-2)); font-size:10.5px; color:rgb(var(--ink-2)); }
.portal-api-price-missing { color:rgb(var(--ink-3)); }
.portal-api-model-table .portal-api-model-empty { height:180px; text-align:center; color:rgb(var(--ink-3)); }
.portal-api-model-empty > svg { margin:0 auto 12px; }
.portal-api-model-state { margin:22px 0; padding:48px 20px; border:1px dashed rgb(var(--line)); border-radius:12px; color:rgb(var(--ink-3)); text-align:center; font-size:13px; }
.portal-api-pricing-note { margin:16px 0 0; font-size:11.5px; line-height:1.7; color:rgb(var(--ink-3)); }
.portal-api-catalogue-notes { margin-top:18px; font-size:13px; color:rgb(var(--ink-2)); }
.portal-api-catalogue-notes summary { cursor:pointer; }
.portal-api-price-description { margin:12px 0 0; color:rgb(var(--ink-3)); font-size:13px; line-height:1.7; overflow-wrap:anywhere; }
.portal-api-model-create { display:flex; align-items:center; flex-wrap:wrap; gap:12px; margin-top:20px; }
.portal-api-model-create > span { max-width:420px; font-size:13px; color:rgb(var(--ink-3)); }
.portal-api-models :is(button,select,summary):focus-visible { outline:2px solid rgb(var(--focus-ring)); outline-offset:3px; }
.portal-api-price-detail { font-size:13.5px; line-height:1.7; }
.portal-api-price-summary { display:grid; gap:8px; margin:0 0 20px; padding:14px 16px; border:1px solid rgb(var(--line)); border-radius:10px; background:rgb(var(--surface-2) / .5); }
.portal-api-price-summary > div { display:flex; flex-wrap:wrap; gap:8px 18px; }
.portal-api-price-summary dt { min-width:105px; color:rgb(var(--ink-3)); }
.portal-api-price-summary dd { margin:0; font-weight:550; }
.portal-api-detail-table-wrap { overflow:auto; margin-top:12px; border:1px solid rgb(var(--line)); border-radius:10px; }
.portal-api-detail-table { width:100%; border-collapse:collapse; font-size:13px; text-align:left; font-variant-numeric:tabular-nums; }
.portal-api-detail-table th { background:rgb(var(--surface-2)); font-weight:600; color:rgb(var(--ink-2)); }
.portal-api-detail-table :is(th,td) { padding:10px 12px; border-bottom:1px solid rgb(var(--line)); white-space:nowrap; }
.portal-api-detail-table tr:last-child td { border-bottom:0; }
.portal-api-pricing-rule { margin-top:20px; padding-top:16px; border-top:1px solid rgb(var(--line)); }
.portal-api-pricing-rule h3 { margin:0 0 8px; font-size:14px; font-weight:600; }
.portal-api-pricing-rule p { margin:6px 0; }
.portal-api-pricing-rule ul { margin:8px 0; padding-left:20px; list-style:disc; }
@media(max-width:1100px) { .portal-api-models { grid-template-columns:205px minmax(0,1fr); } .portal-api-catalogue { padding:22px 18px; } .portal-api-catalogue-header { flex-wrap:wrap; } .portal-api-model-search { width:100%; } }
@media(max-width:760px) { .portal-api-models { grid-template-columns:minmax(0,1fr); } .portal-api-sidebar { border-right:0; border-bottom:1px solid rgb(var(--line)); padding:16px; } .portal-api-sidebar-heading { padding:0; margin-bottom:10px; } .portal-api-group-nav,.portal-api-group-search { display:none; } .portal-api-mobile-groups { display:block; width:100%; } .portal-api-catalogue { padding:20px 16px; } .portal-api-catalogue-header h2 { font-size:19px; } .portal-model-platforms { flex-wrap:nowrap; overflow-x:auto; padding-bottom:6px; } .portal-model-platforms > button { flex-shrink:0; } .portal-api-model-table td:first-child { min-width:200px; } }
</style>
