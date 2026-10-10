<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageLog } from '@/types'
import { firstTokenSeverity, durationSeverity, type LatencySeverity } from '@/utils/latencyHealth'
import {
  portalUsageColumns, type PortalUsageColumnKey, isTokenUsage, usageBillingLabel, usageBillingMode, usageColumnLabel,
  formatUsageNumber, formatUsageMoney, formatUsageDuration, formatUsageRate, formatUsagePercent,
  usageCacheRate, usageOutputStageRate, usageOutputRate, usageRowRequestLabel,
} from '@/utils/portalUsage'
const props = defineProps<{ rows: UsageLog[]; columns: PortalUsageColumnKey[]; sortBy: string; sortOrder: 'asc' | 'desc' }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const emit = defineEmits<{ sort: [key: string]; detail: [row: UsageLog] }>()
const visible = computed(() => portalUsageColumns.filter(column => props.columns.includes(column.key)))
function date(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString(locale.value) }
function time(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '' : date.toLocaleTimeString(locale.value, { hour12: false }) }
function compact(value: number) { return value >= 1000000 ? `${(value / 1000000).toFixed(1)}M` : value >= 1000 ? `${(value / 1000).toFixed(1)}K` : formatUsageNumber(value) }
const latencyColors: Record<LatencySeverity, string> = { good: '#16804a', warn: '#a66c00', slow: '#c25117', critical: '#c83131' }
function latencyColor(ms: number | null | undefined, first = false) {
  return ms == null || !Number.isFinite(ms) || ms < 0 ? '#929088' : latencyColors[first ? firstTokenSeverity(ms) : durationSeverity(ms)]
}
</script>

<template>
  <div class="portal-table-wrap usage-table-wrap">
    <table class="portal-table usage-table"><thead><tr><th v-for="column in visible" :key="column.key" :aria-sort="'sortable' in column && column.sortable ? (sortBy === column.key ? sortOrder === 'asc' ? 'ascending' : 'descending' : 'none') : undefined"><button v-if="'sortable' in column && column.sortable" class="usage-sort" @click="emit('sort', column.key)">{{ usageColumnLabel(column.key, !zh) }} <span aria-hidden="true">{{ sortBy === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span></button><template v-else>{{ usageColumnLabel(column.key, !zh) }}</template></th></tr></thead>
      <tbody><tr v-if="!rows.length"><td :colspan="visible.length" class="usage-empty">{{ copy('暂无用量记录', 'No usage records') }}</td></tr><tr v-for="row in rows" :key="row.id">
        <td v-for="column in visible" :key="column.key" :class="`usage-${column.key}`">
          <template v-if="column.key === 'created_at'"><time>{{ date(row.created_at) }}<small>{{ time(row.created_at) }}</small></time><button class="usage-detail-link" @click="emit('detail', row)">{{ copy('详情', 'Details') }}</button></template>
          <template v-else-if="column.key === 'type'"><span class="usage-badge">{{ usageRowRequestLabel(row, !zh) }}</span><small v-if="row.native_compaction_v2">{{ copy('原生压缩', 'Native compaction') }}</small></template>
          <template v-else-if="column.key === 'model'"><span class="usage-model-text">{{ row.model }}</span><small v-if="row.service_tier">{{ row.service_tier }}</small></template>
          <template v-else-if="column.key === 'tokens'">
            <div v-if="isTokenUsage(row)" class="usage-token-cell">
              <div class="usage-token-counts"><span :title="`${copy('输入', 'Input')} ${formatUsageNumber(row.input_tokens)} Token`"><b aria-hidden="true">↓</b> {{ formatUsageNumber(row.input_tokens) }}</span><span :title="`${copy('输出', 'Output')} ${formatUsageNumber(row.output_tokens)} Token`"><b aria-hidden="true">↑</b> {{ formatUsageNumber(row.output_tokens) }}</span></div>
              <div v-if="row.cache_read_tokens > 0 || row.cache_creation_tokens > 0" class="usage-cache"><span v-if="row.cache_read_tokens > 0" :title="`${copy('缓存读取', 'Cache read')} ${formatUsageNumber(row.cache_read_tokens)} Token`">▣ {{ compact(row.cache_read_tokens) }}</span><span v-if="row.cache_creation_tokens > 0" :title="`${copy('缓存写入', 'Cache write')} ${formatUsageNumber(row.cache_creation_tokens)} Token`">▧ {{ compact(row.cache_creation_tokens) }}<i v-if="row.cache_creation_1h_tokens > 0">1h</i><i v-if="row.cache_ttl_overridden" :title="copy('缓存 TTL 已被覆盖', 'Cache TTL overridden')">R</i></span></div>
              <small v-if="row.image_input_tokens > 0">{{ copy('图像输入', 'Image input') }} {{ formatUsageNumber(row.image_input_tokens) }} Token</small><small v-if="row.image_output_tokens > 0">{{ copy('图像输出', 'Image output') }} {{ formatUsageNumber(row.image_output_tokens) }} Token</small>
              <div class="usage-token-rates"><span :title="copy('缓存命中率 = 缓存读取 / (输入 + 缓存写入 + 缓存读取)', 'Cache hit rate = cache read / (input + cache write + cache read)')">{{ copy('缓存', 'Cache') }} {{ formatUsagePercent(usageCacheRate(row)) }}</span><span :title="copy('输出阶段速率 = 输出 Token / (总耗时 − 首字耗时)', 'Output stage rate = output tokens / (duration - first token)')">{{ copy('输出阶段', 'Output stage') }} {{ formatUsageRate(usageOutputStageRate(row)) }}</span></div>
              <small v-if="usageBillingMode(row) === 'per_request'">{{ copy('按次计费', 'Billed per request') }}</small>
            </div>
            <div v-else-if="usageBillingMode(row) === 'per_request' && !row.image_count" class="usage-media"><strong>{{ copy('1 次请求', '1 request') }}</strong><small>{{ copy('按次计费', 'Billed per request') }}</small><button class="usage-detail-link" @click="emit('detail', row)">{{ copy('查看计费详情', 'Billing details') }}</button></div>
            <div v-else class="usage-media"><strong>{{ usageBillingMode(row) === 'video' ? copy('视频计费', 'Video billing') : row.image_count > 0 || usageBillingMode(row) === 'image' ? `${formatUsageNumber(row.image_count)} ${copy('张图片', 'images')}` : usageBillingLabel(row, !zh) }}</strong><small v-if="row.image_size && usageBillingMode(row) !== 'video'">{{ row.image_size }}</small><button class="usage-detail-link" @click="emit('detail', row)">{{ copy('查看计费详情', 'Billing details') }}</button></div>
          </template>
          <template v-else-if="column.key === 'cost'"><strong class="usage-actual-cost">{{ formatUsageMoney(row.actual_cost) }}</strong><small class="usage-standard-cost">{{ copy('标准', 'Standard') }} {{ formatUsageMoney(row.total_cost) }}</small><small v-if="row.billing_type === 1">{{ copy('订阅额度', 'Subscription quota') }}</small><small v-if="row.long_context_billing_applied">{{ copy('长上下文计费', 'Long-context pricing') }}</small></template>
          <template v-else-if="column.key === 'latency'"><div class="usage-latency-values" :style="{ '--latency-first': latencyColor(row.first_token_ms, true), '--latency-duration': latencyColor(row.duration_ms) }"><span>{{ copy('首字', 'First token') }}</span><b :style="{ color: latencyColor(row.first_token_ms, true) }">{{ formatUsageDuration(row.first_token_ms) }}</b><span>{{ copy('总耗时', 'Duration') }}</span><b :style="{ color: latencyColor(row.duration_ms) }">{{ formatUsageDuration(row.duration_ms) }}</b><template v-if="isTokenUsage(row)"><span :title="copy('输出 TPS = 输出 Token / 总耗时', 'Output TPS = output tokens / total duration')">{{ copy('输出 TPS', 'Output TPS') }}</span><b>{{ formatUsageRate(usageOutputRate(row)) }}</b></template></div></template>
          <template v-else-if="column.key === 'api_key'">{{ row.api_key?.name || `#${row.api_key_id}` }}</template>
          <template v-else-if="column.key === 'group'">{{ row.group?.name || (row.group_id ? `#${row.group_id}` : '—') }}</template>
          <template v-else-if="column.key === 'endpoint'"><span class="usage-long-text" :title="row.inbound_endpoint || ''">{{ row.inbound_endpoint || '—' }}</span></template>
          <template v-else-if="column.key === 'ip_address'">{{ row.ip_address || '—' }}</template>
          <template v-else-if="column.key === 'reasoning_effort'">{{ row.reasoning_effort || '—' }}</template>
          <template v-else-if="column.key === 'billing_mode'">{{ usageBillingLabel(row, !zh) }}<small>{{ row.billing_type === 1 ? copy('订阅', 'Subscription') : copy('余额', 'Balance') }}</small></template>
          <template v-else-if="column.key === 'user_agent'"><span class="usage-long-text" :title="row.user_agent || ''">{{ row.user_agent || '—' }}</span></template>
        </td>
      </tr></tbody>
    </table>
  </div>
</template>

<style scoped>
.usage-table-wrap { padding: 8px; }
.usage-table { white-space: nowrap; font-size: 12px; }
.usage-table th { font-size: 11px; text-transform: none; }
.usage-table td { padding: 18px 12px; }
.usage-sort { display: inline-flex; align-items: center; gap: 8px; background: none; border: 0; font: inherit; }
.usage-sort > span { color: #b6b2a8; }
.usage-table small { display: block; font-size: 11px; color: #929088; margin-top: 5px; line-height: 1.45; }
.usage-detail-link { display: inline-block; margin-top: 7px; color: #929088; text-decoration: underline; font-size: 11px; }
.usage-badge { display: inline-block; padding: 3px 8px; border-radius: 5px; border: 1px solid #efede3; background: #f7f5ee; }
.usage-model-text { display: block; max-width: 200px; white-space: normal; overflow-wrap: anywhere; }
.usage-token-counts { display: flex; gap: 12px; font-variant-numeric: tabular-nums; font-size: 13px; }
.usage-token-counts b { font-weight: 400; font-size: 19px; color: #77746d; margin-right: 2px; }
.usage-cache { display: flex; gap: 12px; color: #77746d; margin-top: 5px; }
.usage-cache i { font-size: 9px; font-style: normal; border: 1px solid #dedbd1; padding: 1px 3px; border-radius: 3px; margin-left: 5px; }
.usage-token-rates { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 8px; font-size: 10px; }
.usage-token-rates span { border: 1px solid #e3dfd4; background: #f7f5ee; border-radius: 4px; padding: 2px 5px; color: #77746d; }
.usage-actual-cost { font-weight: 600; font-size: 13px; font-variant-numeric: tabular-nums; }
.usage-standard-cost { font-variant-numeric: tabular-nums; }
.usage-latency-values { position:relative; display: grid; grid-template-columns: max-content max-content; gap: 4px 10px; padding-left: 13px; font-size: 11px; }
.usage-latency-values::before { content:''; position:absolute; inset:0 auto 0 0; width:3px; border-radius:3px; background:linear-gradient(to bottom,var(--latency-first),var(--latency-duration)); }
.usage-latency-values span { color: #929088; }
.usage-latency-values b { font-weight: 400; font-variant-numeric: tabular-nums; }
.usage-long-text { display: block; max-width: 220px; overflow: hidden; text-overflow: ellipsis; }
.usage-media strong { font-size: 12px; font-weight: 500; }
.usage-empty { height: 170px; text-align: center; color: #929088; }
</style>
