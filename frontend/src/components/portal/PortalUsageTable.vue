<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
import type { UsageLog } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import PortalUsageTokens from '@/components/portal/PortalUsageTokens.vue'
import { firstTokenSeverity, durationSeverity, type LatencySeverity } from '@/utils/latencyHealth'
import {
  portalUsageColumns, type PortalUsageColumnKey, isTokenUsage, usageBillingLabel, usageColumnLabel,
  formatUsageMoney, formatUsageDuration, formatUsageRate, usageOutputRate, usageRowRequestLabel,
} from '@/utils/portalUsage'
const props = defineProps<{ rows: UsageLog[]; columns: PortalUsageColumnKey[]; sortBy: string; sortOrder: 'asc' | 'desc' }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const emit = defineEmits<{ sort: [key: string]; detail: [row: UsageLog] }>()
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
const visible = computed(() => portalUsageColumns.filter(column => props.columns.includes(column.key)))
const primaryKeys = new Set(['created_at', 'type', 'model', 'tokens', 'cost', 'latency'])
const extras = computed(() => visible.value.filter(column => !primaryKeys.has(column.key)))
const sortableVisible = computed(() => visible.value.filter(column => 'sortable' in column && column.sortable))
function shown(key: string) { return props.columns.includes(key as PortalUsageColumnKey) }
function date(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? '—' : parsed.toLocaleDateString(locale.value) }
function time(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleTimeString(locale.value, { hour12: false }) }
const latencyColors: Record<LatencySeverity, string> = { good: 'rgb(var(--text-ok))', warn: 'rgb(var(--text-warn))', slow: 'rgb(var(--text-warn))', critical: 'rgb(var(--text-err))' }
function latencyColor(ms: number | null | undefined, first = false) {
  return ms == null || !Number.isFinite(ms) || ms < 0 ? 'rgb(var(--ink-3))' : latencyColors[first ? firstTokenSeverity(ms) : durationSeverity(ms)]
}
</script>

<template>
  <div v-if="mobile" class="usage-cards">
    <div v-if="sortableVisible.length" class="usage-card-sort">
      <button v-for="column in sortableVisible" :key="column.key" type="button" class="usage-sort" @click="emit('sort', column.key)">
        {{ usageColumnLabel(column.key, !zh) }} <span aria-hidden="true">{{ sortBy === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span>
      </button>
    </div>
    <p v-if="!rows.length" class="usage-empty">{{ copy('暂无用量记录', 'No usage records') }}</p>
    <article v-for="row in rows" :key="row.id" class="usage-card">
      <header v-if="shown('model') || shown('type')" class="usage-card-head">
        <span v-if="shown('model')" class="usage-model-name">
          <ModelIcon :model="row.model" size="20px" aria-hidden="true" />
          <span class="usage-model-text">{{ row.model }}</span>
        </span>
        <span v-if="shown('type')" class="usage-badge" :class="`usage-request-${row.request_type || (row.stream ? 'stream' : 'sync')}`">
          <Icon :name="row.stream ? 'bolt' : 'arrowsUpDown'" size="xs" aria-hidden="true" />{{ usageRowRequestLabel(row, !zh) }}
        </span>
      </header>
      <small v-if="shown('model') && row.service_tier">{{ row.service_tier }}</small>
      <small v-if="shown('type') && row.native_compaction_v2">{{ copy('原生压缩', 'Native compaction') }}</small>
      <div v-if="shown('created_at')" class="usage-card-time">
        <time>{{ date(row.created_at) }} {{ time(row.created_at) }}</time>
        <button type="button" class="usage-detail-link" @click="emit('detail', row)">{{ copy('详情', 'Details') }}</button>
      </div>
      <PortalUsageTokens v-if="shown('tokens')" :row="row" @detail="emit('detail', $event)" />
      <div v-if="shown('cost')" class="usage-card-cost">
        <strong class="usage-actual-cost">{{ formatUsageMoney(row.actual_cost) }}</strong>
        <small class="usage-standard-cost">{{ copy('标准', 'Standard') }} {{ formatUsageMoney(row.total_cost) }}</small>
        <small v-if="row.billing_type === 1">{{ copy('订阅额度', 'Subscription quota') }}</small>
        <small v-if="row.long_context_billing_applied">{{ copy('长上下文计费', 'Long-context pricing') }}</small>
      </div>
      <div v-if="shown('latency')" class="usage-card-latency" :style="{ '--latency-first': latencyColor(row.first_token_ms, true), '--latency-duration': latencyColor(row.duration_ms) }">
        <span>{{ copy('首字', 'First token') }} <b :style="{ color: latencyColor(row.first_token_ms, true) }">{{ formatUsageDuration(row.first_token_ms) }}</b></span>
        <span>{{ copy('总耗时', 'Duration') }} <b :style="{ color: latencyColor(row.duration_ms) }">{{ formatUsageDuration(row.duration_ms) }}</b></span>
        <span v-if="isTokenUsage(row)" :title="copy('输出 TPS = 输出 Token / 总耗时', 'Output TPS = output tokens / total duration')">{{ copy('输出 TPS', 'Output TPS') }} <b>{{ formatUsageRate(usageOutputRate(row)) }}</b></span>
      </div>
      <dl v-if="extras.length" class="usage-card-extra">
        <div v-for="column in extras" :key="column.key">
          <dt>{{ usageColumnLabel(column.key, !zh) }}</dt>
          <dd>
            <template v-if="column.key === 'api_key'">{{ row.api_key?.name || `#${row.api_key_id}` }}</template>
            <template v-else-if="column.key === 'group'">{{ row.group?.name || (row.group_id ? `#${row.group_id}` : '—') }}</template>
            <template v-else-if="column.key === 'endpoint'">{{ row.inbound_endpoint || '—' }}</template>
            <template v-else-if="column.key === 'ip_address'">{{ row.ip_address || '—' }}</template>
            <template v-else-if="column.key === 'reasoning_effort'">{{ row.reasoning_effort || '—' }}</template>
            <template v-else-if="column.key === 'billing_mode'">{{ usageBillingLabel(row, !zh) }} · {{ row.billing_type === 1 ? copy('订阅', 'Subscription') : copy('余额', 'Balance') }}</template>
            <template v-else-if="column.key === 'user_agent'">{{ row.user_agent || '—' }}</template>
          </dd>
        </div>
      </dl>
    </article>
  </div>
  <div v-else class="portal-table-wrap usage-table-wrap">
    <table class="portal-table usage-table"><thead><tr><th v-for="column in visible" :key="column.key" :aria-sort="'sortable' in column && column.sortable ? (sortBy === column.key ? sortOrder === 'asc' ? 'ascending' : 'descending' : 'none') : undefined"><button v-if="'sortable' in column && column.sortable" class="usage-sort" @click="emit('sort', column.key)">{{ usageColumnLabel(column.key, !zh) }} <span aria-hidden="true">{{ sortBy === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span></button><template v-else>{{ usageColumnLabel(column.key, !zh) }}</template></th></tr></thead>
      <tbody><tr v-if="!rows.length"><td :colspan="visible.length" class="usage-empty">{{ copy('暂无用量记录', 'No usage records') }}</td></tr><tr v-for="row in rows" :key="row.id">
        <td v-for="column in visible" :key="column.key" :class="`usage-${column.key}`">
          <template v-if="column.key === 'created_at'"><time>{{ date(row.created_at) }}<small>{{ time(row.created_at) }}</small></time><button class="usage-detail-link" @click="emit('detail', row)">{{ copy('详情', 'Details') }}</button></template>
          <template v-else-if="column.key === 'type'"><span class="usage-badge" :class="`usage-request-${row.request_type || (row.stream ? 'stream' : 'sync')}`"><Icon :name="row.stream ? 'bolt' : 'arrowsUpDown'" size="xs" aria-hidden="true" />{{ usageRowRequestLabel(row, !zh) }}</span><small v-if="row.native_compaction_v2">{{ copy('原生压缩', 'Native compaction') }}</small></template>
          <template v-else-if="column.key === 'model'"><span class="usage-model-name"><ModelIcon :model="row.model" size="20px" aria-hidden="true" /><span class="usage-model-text">{{ row.model }}</span></span><small v-if="row.service_tier">{{ row.service_tier }}</small></template>
          <PortalUsageTokens v-else-if="column.key === 'tokens'" :row="row" @detail="emit('detail', $event)" />
          <template v-else-if="column.key === 'cost'"><strong class="usage-actual-cost">{{ formatUsageMoney(row.actual_cost) }}</strong><small class="usage-standard-cost">{{ copy('标准', 'Standard') }} {{ formatUsageMoney(row.total_cost) }}</small><small v-if="row.billing_type === 1">{{ copy('订阅额度', 'Subscription quota') }}</small><small v-if="row.long_context_billing_applied">{{ copy('长上下文计费', 'Long-context pricing') }}</small></template>
          <template v-else-if="column.key === 'latency'"><div class="usage-latency-values" :style="{ '--latency-first': latencyColor(row.first_token_ms, true), '--latency-duration': latencyColor(row.duration_ms) }"><span>{{ copy('首字', 'First token') }}</span><b :style="{ color: latencyColor(row.first_token_ms, true) }">{{ formatUsageDuration(row.first_token_ms) }}</b><span>{{ copy('总耗时', 'Duration') }}</span><b :style="{ color: latencyColor(row.duration_ms) }">{{ formatUsageDuration(row.duration_ms) }}</b><template v-if="isTokenUsage(row)"><span :title="copy('输出 TPS = 输出 Token / 总耗时', 'Output TPS = output tokens / total duration')">{{ copy('输出 TPS', 'Output TPS') }}</span><b>{{ formatUsageRate(usageOutputRate(row)) }}</b></template></div></template>
          <template v-else-if="column.key === 'api_key'">{{ row.api_key?.name || `#${row.api_key_id}` }}</template>
          <template v-else-if="column.key === 'group'"><span class="usage-group-tag">{{ row.group?.name || (row.group_id ? `#${row.group_id}` : '—') }}</span></template>
          <template v-else-if="column.key === 'endpoint'"><span class="usage-long-text" :title="row.inbound_endpoint || ''">{{ row.inbound_endpoint || '—' }}</span></template>
          <template v-else-if="column.key === 'ip_address'">{{ row.ip_address || '—' }}</template>
          <template v-else-if="column.key === 'reasoning_effort'">{{ row.reasoning_effort || '—' }}</template>
          <template v-else-if="column.key === 'billing_mode'"><span class="usage-badge">{{ usageBillingLabel(row, !zh) }}</span><small>{{ row.billing_type === 1 ? copy('订阅', 'Subscription') : copy('余额', 'Balance') }}</small></template>
          <template v-else-if="column.key === 'user_agent'"><span class="usage-long-text" :title="row.user_agent || ''">{{ row.user_agent || '—' }}</span></template>
        </td>
      </tr></tbody>
    </table>
  </div>
</template>

<style scoped>
.usage-table-wrap { padding: 0; border:1px solid rgb(var(--line)); border-radius:var(--radius-card); }
.usage-table { width:max-content; min-width:100%; white-space: nowrap; font-size: 13.5px; }
.usage-table th { font-size: 12.5px; font-weight: 600; color: rgb(var(--ink-2)); text-transform: none; padding: 12px 14px; background: rgb(var(--surface-2)); }
.usage-table td { padding: 15px 14px; font-size: 13.5px; }
.usage-table .usage-api_key { font-weight: 550; }
.usage-group-tag { display:inline-block; max-width:200px; padding:2px 9px; font-size:12.5px; font-weight:550; line-height:1.5; white-space:normal; overflow-wrap:anywhere; border:1px solid rgb(var(--brand-moss) / .24); border-radius:999px; background:rgb(var(--brand-moss) / .08); color:rgb(var(--brand-moss)); }
.usage-sort { display: inline-flex; align-items: center; gap: 6px; background: none; border: 0; font: inherit; color: inherit; cursor: pointer; }
.usage-sort:hover { color: rgb(var(--ink)); }
.usage-sort > span { color: rgb(var(--ink-3)); }
.usage-table small, .usage-card small { display: block; font-size: 12.5px; color: rgb(var(--ink-3)); margin-top: 4px; line-height: 1.45; }
.usage-detail-link { display: inline-block; margin-top: 6px; color: rgb(var(--ink-2)); text-decoration: underline; text-underline-offset: 3px; font-size: 12.5px; cursor: pointer; }
.usage-detail-link:hover { color: rgb(var(--brand-moss)); }
.usage-badge { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; border-radius: 6px; border: 0; background: rgb(var(--status-link-bg)); color:rgb(var(--text-link)); font-size:12px; font-weight:600; white-space:nowrap; }
.usage-model-text { @apply font-mono font-medium; display: block; min-width: 140px; max-width: 220px; white-space: normal; overflow-wrap: break-word; font-size: 13px; }
.usage-model-name { display: flex; align-items: flex-start; gap: 8px; }
.usage-model-name :deep(svg), .usage-model-name :deep(.model-icon-fallback) { flex-shrink: 0; }
.usage-request-stream, .usage-request-ws_v2, .usage-request-live, .usage-request-sync { color: rgb(var(--text-link)); background: rgb(var(--status-link-bg)); }
.usage-actual-cost { @apply font-mono font-semibold; color: rgb(var(--ink)); font-size: 13.5px; font-variant-numeric: tabular-nums; }
.usage-table .usage-standard-cost, .usage-card .usage-standard-cost { color: rgb(var(--ink-3)); font-variant-numeric: tabular-nums; }
.usage-latency-values { position:relative; display: grid; grid-template-columns: max-content max-content; gap: 3px 10px; padding-left: 12px; font-size: 12.5px; }
.usage-latency-values::before { content:''; position:absolute; inset:0 auto 0 0; width:3px; border-radius:3px; background:linear-gradient(to bottom,var(--latency-first),var(--latency-duration)); }
.usage-latency-values span { color: rgb(var(--ink-3)); }
.usage-latency-values b { font-weight: 600; font-variant-numeric: tabular-nums; }
.usage-long-text { display: block; max-width: 220px; overflow: hidden; text-overflow: ellipsis; font-family: var(--font-mono); font-size: 12.5px; }
.usage-empty { height: 170px; text-align: center; color: rgb(var(--ink-3)); }
.usage-cards { display:grid; gap:12px; }
.usage-card-sort { display:flex; flex-wrap:wrap; gap:8px 16px; }
.usage-card { display:grid; gap:10px; padding:16px; border:1px solid rgb(var(--line)); border-radius:var(--radius-card); background:rgb(var(--surface)); box-shadow:var(--shadow-panel); }
.usage-card-head { display:flex; align-items:flex-start; justify-content:space-between; gap:12px; }
.usage-card .usage-model-name { min-width:0; flex:1; }
.usage-card .usage-model-text { min-width:0; max-width:none; }
.usage-card-time { display:flex; align-items:center; justify-content:space-between; gap:12px; color:rgb(var(--ink-3)); font-size:12.5px; font-variant-numeric:tabular-nums; }
.usage-card-time .usage-detail-link { margin-top:0; }
.usage-card-cost { display:flex; flex-wrap:wrap; align-items:baseline; gap:6px 12px; }
.usage-card-latency { display:flex; flex-wrap:wrap; gap:6px 14px; padding-left:10px; border-left:3px solid var(--latency-first); font-size:12.5px; }
.usage-card-latency span { color:rgb(var(--ink-3)); }
.usage-card-latency b { color:rgb(var(--ink)); font-weight:600; font-variant-numeric:tabular-nums; }
.usage-card-extra { display:grid; gap:8px; margin:0; }
.usage-card-extra div { display:grid; grid-template-columns:72px minmax(0,1fr); gap:8px; }
.usage-card-extra dt { color:rgb(var(--ink-3)); font-size:12.5px; }
.usage-card-extra dd { margin:0; overflow-wrap:anywhere; font-size:12.5px; }
.usage-cards .usage-empty { display:grid; place-items:center; margin:0; border:1px solid rgb(var(--line)); border-radius:var(--radius-card); }
</style>
