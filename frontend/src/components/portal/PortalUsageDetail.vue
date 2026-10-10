<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageLog } from '@/types'
import PortalDialog from './PortalDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import {
  finiteNumber, isTokenUsage, usageBillingMode, usageBillingLabel, usageRowRequestLabel,
  formatUsageNumber, formatUsageMoney, formatUsageDuration, formatUsageRate, formatUsagePercent,
  usageCacheRate, usageOutputStageRate, usageOutputRate,
} from '@/utils/portalUsage'
const props = defineProps<{ open: boolean; row: UsageLog | null; loading?: boolean; error?: string }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const emit = defineEmits<{ close: [] }>()
const metadata = computed(() => {
  const row = props.row
  if (!row) return []
  return [
    [copy('时间', 'Time'), new Date(row.created_at).toLocaleString(locale.value, { hour12: false })],
    [copy('请求 ID', 'Request ID'), row.request_id || '—'], ['API Key', row.api_key?.name || `#${row.api_key_id}`],
    [copy('分组', 'Group'), row.group?.name || (row.group_id ? `#${row.group_id}` : '—')],
    [copy('模型', 'Model'), row.model], [copy('请求类型', 'Request type'), usageRowRequestLabel(row, !zh.value)], [copy('计费模式', 'Billing mode'), usageBillingLabel(row, !zh.value)],
    [copy('扣费方式', 'Billing source'), row.billing_type === 1 ? copy('订阅额度', 'Subscription quota') : copy('账户余额', 'Account balance')],
    [copy('推理强度', 'Reasoning effort'), row.reasoning_effort || '—'], [copy('推理强度来源', 'Reasoning source'), row.reasoning_effort_source || '—'],
    [copy('服务等级', 'Service tier'), row.service_tier || '—'], [copy('入站端点', 'Inbound endpoint'), row.inbound_endpoint || '—'],
    ['IP', row.ip_address || '—'], [copy('原生压缩', 'Native compaction'), row.native_compaction_v2 ? copy('是', 'Yes') : copy('否', 'No')],
  ]
})
const tokenRows = computed(() => {
  const row = props.row
  if (!row) return []
  const rows: Array<[string, string]> = [
    [copy('输入 Token', 'Input tokens'), formatUsageNumber(row.input_tokens)], [copy('输出 Token', 'Output tokens'), formatUsageNumber(row.output_tokens)],
    [copy('缓存读取', 'Cache read'), formatUsageNumber(row.cache_read_tokens)], [copy('缓存写入', 'Cache write'), formatUsageNumber(row.cache_creation_tokens)],
  ]
  if (row.cache_creation_5m_tokens > 0) rows.push([copy('5 分钟缓存写入', '5-minute cache write'), formatUsageNumber(row.cache_creation_5m_tokens)])
  if (row.cache_creation_1h_tokens > 0) rows.push([copy('1 小时缓存写入', '1-hour cache write'), formatUsageNumber(row.cache_creation_1h_tokens)])
  if (row.image_input_tokens > 0) rows.push([copy('其中图像输入 Token', 'Image input tokens (included)'), formatUsageNumber(row.image_input_tokens)])
  if (row.image_output_tokens > 0) rows.push([copy('其中图像输出 Token', 'Image output tokens (included)'), formatUsageNumber(row.image_output_tokens)])
  return rows
})
const costRows = computed(() => {
  const row = props.row
  if (!row) return []
  return [
    [copy('输入费用', 'Input cost'), row.input_cost], [copy('输出费用', 'Output cost'), row.output_cost],
    [copy('缓存读取费用', 'Cache read cost'), row.cache_read_cost], [copy('缓存写入费用', 'Cache write cost'), row.cache_creation_cost],
    ...(row.image_input_cost > 0 ? [[copy('图像输入费用', 'Image input cost'), row.image_input_cost]] : []),
    ...(row.image_output_cost > 0 ? [[copy('图像输出费用', 'Image output cost'), row.image_output_cost]] : []),
  ] as Array<[string, number]>
})
const imageSizes = computed(() => Object.entries(props.row?.image_size_breakdown || {}).map(([size, count]) => `${size} × ${count}`).join('、'))
const sizeSource = computed(() => {
  const labels: Record<string, string> = { output: copy('实际输出', 'Actual output'), input: copy('输入参数', 'Input parameter'), default: copy('默认', 'Default'), legacy: copy('历史记录', 'Legacy') }
  return labels[props.row?.image_size_source || ''] || copy('未记录', 'Not recorded')
})
</script>

<template>
  <PortalDialog :open="open" :title="copy('请求详情', 'Request details')" wide @close="emit('close')">
    <p v-if="loading" class="portal-muted" role="status">{{ copy('正在加载完整记录…', 'Loading the complete record…') }}</p>
    <p v-if="error" class="portal-error" role="alert">{{ error }}</p>
    <template v-if="row">
      <div class="usage-detail-model"><ModelIcon :model="row.model" size="28px" aria-hidden="true" /><strong>{{ row.model }}</strong><span>{{ usageRowRequestLabel(row, !zh) }}</span></div>
      <dl class="usage-detail-grid"><template v-for="[label, value] in metadata" :key="label"><dt>{{ label }}</dt><dd>{{ value }}</dd></template></dl>
      <section v-if="isTokenUsage(row)" class="usage-detail-section"><h3><Icon name="cpu" size="sm" class="usage-token-icon" aria-hidden="true" />{{ copy('Token 明细', 'Token details') }}</h3><dl class="usage-detail-grid"><template v-for="[label, value] in tokenRows" :key="label"><dt>{{ label }}</dt><dd>{{ value }}</dd></template><dt>{{ copy('缓存命中率', 'Cache hit rate') }}</dt><dd>{{ formatUsagePercent(usageCacheRate(row)) }}</dd><dt>{{ copy('输出阶段速率', 'Output stage rate') }}</dt><dd>{{ formatUsageRate(usageOutputStageRate(row)) }}</dd><dt v-if="row.cache_ttl_overridden">{{ copy('缓存 TTL', 'Cache TTL') }}</dt><dd v-if="row.cache_ttl_overridden">{{ copy('已覆盖为', 'Overridden to') }} {{ row.cache_creation_1h_tokens > 0 ? copy('1 小时', '1 hour') : copy('5 分钟', '5 minutes') }}</dd></dl><p class="usage-detail-note">{{ copy('缓存命中率 = 缓存读取 /（输入 + 缓存写入 + 缓存读取）', 'Cache hit rate = cache read / (input + cache write + cache read)') }}<br />{{ copy('输出阶段速率 = 输出 Token /（总耗时 − 首字耗时）', 'Output stage rate = output tokens / (duration - first token)') }}</p></section>
      <section v-else-if="usageBillingMode(row) === 'per_request' && !row.image_count" class="usage-detail-section"><h3>{{ copy('按次计费', 'Billed per request') }}</h3><p class="usage-detail-note">{{ copy('此记录按请求次数计费。以下为服务返回的原始 Token 计数。', 'This record is billed per request. These are the raw token counts returned by the service.') }}</p><dl class="usage-detail-grid"><template v-for="[label, value] in tokenRows" :key="label"><dt>{{ label }}</dt><dd>{{ value }}</dd></template></dl></section>
      <section v-else-if="usageBillingMode(row) === 'video'" class="usage-detail-section"><h3>{{ copy('视频计费', 'Video billing') }}</h3><p class="portal-muted">{{ copy('此记录按视频计费。下方耗时为请求耗时，Token 吞吐指标不适用。', 'This request uses video billing. Duration is the request duration; Token throughput does not apply.') }}</p></section>
      <section v-else class="usage-detail-section"><h3>{{ copy('图片计费', 'Image billing') }}</h3><dl class="usage-detail-grid"><dt>{{ copy('图片数量', 'Image count') }}</dt><dd>{{ formatUsageNumber(row.image_count) }} {{ copy('张', 'images') }}</dd><dt>{{ copy('计费尺寸', 'Billing size') }}</dt><dd>{{ row.image_size || copy('未记录', 'Not recorded') }}</dd><dt>{{ copy('输入尺寸', 'Input size') }}</dt><dd>{{ row.image_input_size || copy('未记录', 'Not recorded') }}</dd><dt>{{ copy('输出尺寸', 'Output size') }}</dt><dd>{{ row.image_output_size || copy('未记录', 'Not recorded') }}</dd><dt>{{ copy('尺寸来源', 'Size source') }}</dt><dd>{{ sizeSource }}</dd><template v-if="imageSizes"><dt>{{ copy('尺寸分布', 'Size breakdown') }}</dt><dd>{{ imageSizes }}</dd></template><dt>{{ copy('平均标准单价', 'Average standard unit price') }}</dt><dd>{{ row.image_count > 0 ? formatUsageMoney(row.total_cost / row.image_count, 8) : '—' }}</dd></dl></section>
      <section class="usage-detail-section"><h3><Icon name="creditCard" size="sm" class="usage-cost-icon" aria-hidden="true" />{{ copy('费用明细 · USD', 'Cost breakdown · USD') }}</h3><dl class="usage-detail-grid"><template v-if="isTokenUsage(row)"><template v-for="[label, value] in costRows" :key="label"><dt>{{ label }}</dt><dd>{{ formatUsageMoney(value, 8) }}</dd></template></template><dt>{{ copy('标准费用', 'Standard cost') }}</dt><dd>{{ formatUsageMoney(row.total_cost, 8) }}</dd><dt>{{ copy('计费倍率', 'Rate multiplier') }}</dt><dd>{{ finiteNumber(row.rate_multiplier) }}×</dd><dt>{{ copy('实际扣费', 'Billed cost') }}</dt><dd class="usage-detail-strong">{{ formatUsageMoney(row.actual_cost, 8) }}</dd><template v-if="row.long_context_billing_applied"><dt>{{ copy('长上下文计费', 'Long-context pricing') }}</dt><dd>{{ copy('已应用', 'Applied') }}</dd></template></dl></section>
      <section class="usage-detail-section"><h3><Icon name="clock" size="sm" class="usage-latency-icon" aria-hidden="true" />{{ copy('延迟', 'Latency') }}</h3><dl class="usage-detail-grid"><dt>{{ copy('首字', 'First token') }}</dt><dd>{{ formatUsageDuration(row.first_token_ms) }}</dd><dt>{{ copy('总耗时', 'Duration') }}</dt><dd>{{ formatUsageDuration(row.duration_ms) }}</dd><template v-if="isTokenUsage(row)"><dt>{{ copy('输出 TPS', 'Output TPS') }}</dt><dd>{{ formatUsageRate(usageOutputRate(row)) }}</dd></template></dl><p v-if="isTokenUsage(row)" class="usage-detail-note">{{ copy('输出 TPS = 输出 Token / 总耗时', 'Output TPS = output tokens / total duration') }}</p></section>
      <section class="usage-detail-section"><h3>User-Agent</h3><p class="usage-detail-agent">{{ row.user_agent || copy('未记录', 'Not recorded') }}</p></section>
    </template>
  </PortalDialog>
</template>

<style scoped>
.usage-detail-model { display: flex; align-items: center; gap: 12px; padding-bottom: 22px; margin-bottom: 22px; border-bottom: 1px solid #efede3; overflow-wrap: anywhere; }
.usage-detail-model strong { font-weight: 500; min-width: 0; }
.usage-detail-model span { margin-left: auto; color: #77746d; font-size: 12px; }
.usage-detail-model :deep(svg) { flex-shrink: 0; }
.usage-token-icon { color: #8b5cf6; }
.usage-cost-icon { color: #2563eb; }
.usage-latency-icon { color: #059669; }
.usage-detail-grid { display: grid; grid-template-columns: 130px minmax(0, 1fr); gap: 12px 20px; font-size: 13px; line-height: 1.6; }
.usage-detail-grid dt { color: #929088; }
.usage-detail-grid dd { margin: 0; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
.usage-detail-section { margin-top: 24px; padding-top: 22px; border-top: 1px solid #efede3; }
.usage-detail-section h3 { display: flex; align-items: center; gap: 8px; margin: 0 0 16px; font-size: 15px; font-weight: 500; }
.usage-detail-note { color: #929088; font-size: 11px; line-height: 1.8; margin-top: 14px; }
.usage-detail-strong { font-weight: 600; }
.usage-detail-agent { font-size: 12px; color: #77746d; overflow-wrap: anywhere; }
@media (max-width: 560px) { .usage-detail-grid { grid-template-columns: 105px minmax(0, 1fr); gap: 12px; } }
</style>
