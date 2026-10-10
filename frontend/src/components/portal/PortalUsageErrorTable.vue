<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserErrorRequest } from '@/types'
import { portalErrorColumns, type PortalErrorColumnKey, usageErrorRequestLabel, usageColumnLabel } from '@/utils/portalUsage'
const props = defineProps<{ rows: UserErrorRequest[]; columns: PortalErrorColumnKey[]; sortBy: string; sortOrder: 'asc' | 'desc' }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const emit = defineEmits<{ sort: [key: string]; detail: [row: UserErrorRequest] }>()
const visible = computed(() => portalErrorColumns.filter(column => props.columns.includes(column.key)))
function date(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString(locale.value, { hour12: false }) }
const categories = computed<Record<string, string>>(() => zh.value ? { auth: '认证', rate_limit: '速率限制', quota: '额度', invalid_request: '请求参数', service_unavailable: '服务不可用', upstream: '上游', internal: '内部错误', cyber: '安全策略' } : { auth: 'Authentication', rate_limit: 'Rate limit', quota: 'Quota', invalid_request: 'Invalid request', service_unavailable: 'Service unavailable', upstream: 'Upstream', internal: 'Internal', cyber: 'Security policy' })
</script>

<template>
  <div class="portal-table-wrap error-table-wrap"><table class="portal-table error-table"><thead><tr><th v-for="column in visible" :key="column.key" :aria-sort="'sortable' in column && column.sortable ? (sortBy === column.key ? sortOrder === 'asc' ? 'ascending' : 'descending' : 'none') : undefined"><button v-if="'sortable' in column && column.sortable" class="error-sort" @click="emit('sort', column.key)">{{ usageColumnLabel(column.key, !zh) }} <span aria-hidden="true">{{ sortBy === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span></button><template v-else>{{ usageColumnLabel(column.key, !zh) }}</template></th></tr></thead><tbody><tr v-if="!rows.length"><td :colspan="visible.length" class="error-empty">{{ copy('暂无错误请求', 'No error requests') }}</td></tr><tr v-for="row in rows" :key="row.id"><td v-for="column in visible" :key="column.key">
    <template v-if="column.key === 'created_at'"><time>{{ date(row.created_at) }}</time><button class="error-detail" @click="emit('detail', row)">{{ copy('详情', 'Details') }}</button></template>
    <template v-else-if="column.key === 'type'">{{ usageErrorRequestLabel(row, !zh) }}</template>
    <template v-else-if="column.key === 'model'"><span class="error-wrap">{{ row.model || '—' }}</span></template>
    <template v-else-if="column.key === 'status_code'"><span class="error-status">{{ row.status_code }}</span></template>
    <template v-else-if="column.key === 'category'">{{ categories[row.category] || row.category || '—' }}</template>
    <template v-else-if="column.key === 'message'"><span class="error-message" :title="row.message">{{ row.message }}</span></template>
    <template v-else-if="column.key === 'key_name'">{{ row.key_name || '—' }}<small v-if="row.key_deleted">{{ copy('已删除', 'Deleted') }}</small></template>
    <template v-else-if="column.key === 'endpoint'"><span class="error-wrap">{{ row.inbound_endpoint || '—' }}</span></template>
    <template v-else-if="column.key === 'client_ip'">{{ row.client_ip || '—' }}</template>
    <template v-else-if="column.key === 'group_name'">{{ row.group_name || '—' }}</template>
    <template v-else-if="column.key === 'platform'">{{ row.platform || '—' }}</template>
    <template v-else-if="column.key === 'user_agent'"><span class="error-message" :title="row.user_agent">{{ row.user_agent || '—' }}</span></template>
  </td></tr></tbody></table></div>
</template>

<style scoped>
.error-table-wrap { padding: 8px; }
.error-table { white-space: nowrap; font-size: 12px; }
.error-table th { font-size: 11px; }
.error-table td { padding: 20px 12px; }
.error-sort { display: inline-flex; align-items: center; gap: 8px; background: transparent; border: 0; font: inherit; }
.error-sort > span { color: #b6b2a8; }
.error-detail, .error-table small { display: block; font-size: 11px; color: #929088; margin-top: 8px; }
.error-detail { text-decoration: underline; }
.error-status { background: #f3f0eb; padding: 4px 8px; border: 1px solid #efede3; border-radius: 5px; }
.error-message { display: -webkit-box; max-width: 260px; min-width: 180px; white-space: normal; overflow: hidden; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow-wrap: anywhere; }
.error-wrap { display: block; max-width: 180px; white-space: normal; overflow-wrap: anywhere; }
.error-empty { height: 170px; text-align: center; color: #929088; }
</style>
