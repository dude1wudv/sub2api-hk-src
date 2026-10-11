<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import type { UserErrorRequest } from '@/types'
import { portalErrorColumns, type PortalErrorColumnKey, usageErrorRequestLabel, usageColumnLabel } from '@/utils/portalUsage'
const props = defineProps<{ rows: UserErrorRequest[]; columns: PortalErrorColumnKey[]; sortBy: string; sortOrder: 'asc' | 'desc' }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const emit = defineEmits<{ sort: [key: string]; detail: [row: UserErrorRequest] }>()
const mobile = useMediaQuery('(max-width: 739px)')
const visible = computed(() => portalErrorColumns.filter(column => props.columns.includes(column.key)))
const primaryKeys = new Set(['created_at', 'type', 'model', 'status_code', 'category', 'message'])
const extras = computed(() => visible.value.filter(column => !primaryKeys.has(column.key)))
const sortableVisible = computed(() => visible.value.filter(column => 'sortable' in column && column.sortable))
function shown(key: string) { return props.columns.includes(key as PortalErrorColumnKey) }
function date(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? '—' : parsed.toLocaleString(locale.value, { hour12: false }) }
const categories = computed<Record<string, string>>(() => zh.value ? { auth: '认证', rate_limit: '速率限制', quota: '额度', invalid_request: '请求参数', service_unavailable: '服务不可用', upstream: '上游', internal: '内部错误', cyber: '安全策略' } : { auth: 'Authentication', rate_limit: 'Rate limit', quota: 'Quota', invalid_request: 'Invalid request', service_unavailable: 'Service unavailable', upstream: 'Upstream', internal: 'Internal', cyber: 'Security policy' })
function categoryName(value: string) { return categories.value[value] || value || '—' }
</script>

<template>
  <div v-if="mobile" class="error-cards">
    <div v-if="sortableVisible.length" class="error-card-sort">
      <button v-for="column in sortableVisible" :key="column.key" type="button" class="error-sort" @click="emit('sort', column.key)">
        {{ usageColumnLabel(column.key, !zh) }} <span aria-hidden="true">{{ sortBy === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span>
      </button>
    </div>
    <p v-if="!rows.length" class="error-empty">{{ copy('暂无错误请求', 'No error requests') }}</p>
    <article v-for="row in rows" :key="row.id" class="error-card">
      <header class="error-card-head">
        <span v-if="shown('status_code')" class="error-status">{{ row.status_code }}</span>
        <span v-if="shown('category')">{{ categoryName(row.category) }}</span>
        <button v-if="shown('created_at')" type="button" class="error-detail" @click="emit('detail', row)">{{ copy('详情', 'Details') }}</button>
      </header>
      <time v-if="shown('created_at')" class="error-card-time">{{ date(row.created_at) }}</time>
      <p v-if="shown('model')" class="error-card-model">{{ row.model || '—' }}</p>
      <p v-if="shown('type')" class="error-card-type">{{ usageErrorRequestLabel(row, !zh) }}</p>
      <p v-if="shown('message')" class="error-message">{{ row.message }}</p>
      <dl v-if="extras.length" class="error-card-extra">
        <div v-for="column in extras" :key="column.key">
          <dt>{{ usageColumnLabel(column.key, !zh) }}</dt>
          <dd>
            <template v-if="column.key === 'key_name'">{{ row.key_name || '—' }}<small v-if="row.key_deleted">{{ copy('已删除', 'Deleted') }}</small></template>
            <template v-else-if="column.key === 'endpoint'">{{ row.inbound_endpoint || '—' }}</template>
            <template v-else-if="column.key === 'client_ip'">{{ row.client_ip || '—' }}</template>
            <template v-else-if="column.key === 'group_name'">{{ row.group_name || '—' }}</template>
            <template v-else-if="column.key === 'platform'">{{ row.platform || '—' }}</template>
            <template v-else-if="column.key === 'user_agent'">{{ row.user_agent || '—' }}</template>
          </dd>
        </div>
      </dl>
    </article>
  </div>
  <div v-else class="portal-table-wrap error-table-wrap"><table class="portal-table error-table"><thead><tr><th v-for="column in visible" :key="column.key" :aria-sort="'sortable' in column && column.sortable ? (sortBy === column.key ? sortOrder === 'asc' ? 'ascending' : 'descending' : 'none') : undefined"><button v-if="'sortable' in column && column.sortable" class="error-sort" @click="emit('sort', column.key)">{{ usageColumnLabel(column.key, !zh) }} <span aria-hidden="true">{{ sortBy === column.key ? sortOrder === 'asc' ? '↑' : '↓' : '↕' }}</span></button><template v-else>{{ usageColumnLabel(column.key, !zh) }}</template></th></tr></thead><tbody><tr v-if="!rows.length"><td :colspan="visible.length" class="error-empty">{{ copy('暂无错误请求', 'No error requests') }}</td></tr><tr v-for="row in rows" :key="row.id"><td v-for="column in visible" :key="column.key">
    <template v-if="column.key === 'created_at'"><time>{{ date(row.created_at) }}</time><button class="error-detail" @click="emit('detail', row)">{{ copy('详情', 'Details') }}</button></template>
    <template v-else-if="column.key === 'type'">{{ usageErrorRequestLabel(row, !zh) }}</template>
    <template v-else-if="column.key === 'model'"><span class="error-wrap">{{ row.model || '—' }}</span></template>
    <template v-else-if="column.key === 'status_code'"><span class="error-status">{{ row.status_code }}</span></template>
    <template v-else-if="column.key === 'category'">{{ categoryName(row.category) }}</template>
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
.error-table { width:max-content; min-width:100%; white-space: nowrap; font-size: 12px; }
.error-table th { font-size: 11px; }
.error-table td { padding: 20px 12px; }
.error-sort { display: inline-flex; align-items: center; gap: 8px; background: transparent; border: 0; font: inherit; }
.error-sort > span { color: #b6b2a8; }
.error-detail, .error-table small { display: block; font-size: 11px; color: #929088; margin-top: 8px; }
.error-detail { text-decoration: underline; }
.error-status { background: #f3f0eb; padding: 4px 8px; border: 1px solid #efede3; border-radius: 5px; }
.error-message { display: -webkit-box; max-width: 260px; min-width: 180px; white-space: normal; overflow: hidden; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow-wrap: break-word; }
.error-wrap { display: block; max-width: 220px; min-width: 140px; white-space: normal; overflow-wrap: break-word; }
.error-empty { height: 170px; text-align: center; color: #929088; }
.error-cards { display:grid; gap:12px; }
.error-card-sort { display:flex; flex-wrap:wrap; gap:8px 16px; }
.error-card { display:grid; gap:8px; padding:14px; border:1px solid #efede3; border-radius:12px; background:#fffdf7; }
.error-card-head { display:flex; align-items:center; gap:8px; }
.error-card-head .error-detail { margin-top:0; margin-left:auto; }
.error-card-time { color:#929088; font-size:12px; }
.error-card-model { margin:0; font-weight:600; overflow-wrap:anywhere; }
.error-card-type { margin:0; color:#77746d; font-size:12px; }
.error-card .error-message { max-width:none; min-width:0; margin:0; }
.error-card-extra { display:grid; gap:8px; margin:0; }
.error-card-extra div { display:grid; grid-template-columns:72px minmax(0,1fr); gap:8px; }
.error-card-extra dt { color:#929088; font-size:11px; }
.error-card-extra dd { margin:0; overflow-wrap:anywhere; font-size:12px; }
.error-cards .error-empty { display:grid; place-items:center; margin:0; border:1px solid #efede3; border-radius:12px; }
</style>
