<template>
  <article class="portal-api-key-card" :data-key-id="apiKey.id" :data-selected="selectable && selected">
    <header><label v-if="selectable" class="portal-api-key-select"><input type="checkbox" :checked="selected" :disabled="busy" :aria-label="text('选择密钥：', 'Select key: ') + apiKey.name" :data-test="'select-key-' + apiKey.id" @change="$emit('select', apiKey.id, ($event.target as HTMLInputElement).checked)" /></label><h3 :title="apiKey.name">{{ apiKey.name }}</h3><span class="portal-api-key-status" :data-status="apiKey.status">{{ statusName }}</span></header>
    <div class="portal-api-key-secret"><code>{{ maskedKey }}</code><button type="button" class="portal-inline-button" :aria-label="text('复制密钥：', 'Copy key: ') + apiKey.name" @click="$emit('copy', apiKey)"><Icon name="copy" size="sm" aria-hidden="true" /></button></div>
    <p class="portal-api-key-group">{{ groupName }}</p>
    <dl class="portal-api-key-dates">
      <div><dt>{{ text('创建时间', 'Created') }}</dt><dd>{{ date(apiKey.created_at) }}</dd></div>
      <div><dt>{{ text('最近使用', 'Last used') }}</dt><dd>{{ apiKey.last_used_at ? date(apiKey.last_used_at) : text('尚未使用', 'Never used') }}</dd></div>
    </dl>
    <details class="portal-api-key-details">
      <summary>{{ text('限额与访问设置', 'Limits and access') }}</summary>
      <dl>
        <div><dt>{{ text('额度用量', 'Quota usage') }}</dt><dd>{{ money(apiKey.quota_used) }} / {{ apiKey.quota > 0 ? money(apiKey.quota) : text('不限', 'Unlimited') }}</dd></div>
        <div><dt>{{ text('到期时间', 'Expires') }}</dt><dd>{{ apiKey.expires_at ? date(apiKey.expires_at) : text('永久', 'Never') }}</dd></div>
        <div v-for="limit in limits" :key="limit.label"><dt>{{ limit.label }}</dt><dd>{{ money(limit.used) }} / {{ money(limit.limit) }}<small v-if="limit.reset">{{ text('重置：', 'Resets: ') }}{{ date(limit.reset) }}</small></dd></div>
        <div><dt>{{ text('当前并发', 'Concurrency') }}</dt><dd>{{ apiKey.current_concurrency || 0 }}</dd></div>
        <div><dt>{{ text('最近使用 IP', 'Last IP') }}</dt><dd>{{ apiKey.last_used_ip || '—' }}</dd></div>
        <div v-if="apiKey.routing_group_ids?.length"><dt>{{ text('路由分组', 'Routing groups') }}</dt><dd>{{ apiKey.routing_group_ids.map(nameForGroup).join(' / ') }}</dd></div>
        <div><dt>{{ text('IP 白名单', 'IP allowlist') }}</dt><dd>{{ apiKey.ip_whitelist?.length ? apiKey.ip_whitelist.join(', ') : text('未设置', 'Not set') }}</dd></div>
        <div><dt>{{ text('IP 黑名单', 'IP blocklist') }}</dt><dd>{{ apiKey.ip_blacklist?.length ? apiKey.ip_blacklist.join(', ') : text('未设置', 'Not set') }}</dd></div>
      </dl>
    </details>
    <footer>
      <button type="button" class="portal-inline-button" :disabled="busy" @click="$emit('edit', apiKey)">{{ text('编辑', 'Edit') }}</button>
      <button type="button" class="portal-inline-button" :disabled="busy" @click="$emit('toggle', apiKey)">{{ apiKey.status === 'active' ? text('停用', 'Disable') : text('启用', 'Enable') }}</button>
      <button type="button" class="portal-inline-button portal-api-key-delete" :disabled="busy" @click="$emit('delete', apiKey)">{{ text('删除', 'Delete') }}</button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ApiKey, Group } from '@/types'
const props = withDefaults(defineProps<{ apiKey: ApiKey; groups: Group[]; busy?: boolean; selectable?: boolean; selected?: boolean }>(), { busy: false, selectable: false, selected: false })
defineEmits<{ copy: [key: ApiKey]; edit: [key: ApiKey]; toggle: [key: ApiKey]; delete: [key: ApiKey]; select: [id: number, checked: boolean] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const maskedKey = computed(() => props.apiKey.key.length > 10 ? props.apiKey.key.slice(0, 6) + '••••••••' + props.apiKey.key.slice(-4) : '••••••••••••')
const statusName = computed(() => ({
  active: text('已启用', 'Active'),
  inactive: text('已停用', 'Inactive'),
  expired: text('已过期', 'Expired'),
  quota_exhausted: text('额度用尽', 'Quota exhausted'),
})[props.apiKey.status])
const groupName = computed(() => props.apiKey.group?.name || (props.apiKey.group_id == null ? text('未绑定分组', 'No group') : nameForGroup(props.apiKey.group_id)))
const limits = computed(() => [
  { label: text('5 小时限额', '5-hour limit'), used: props.apiKey.usage_5h, limit: props.apiKey.rate_limit_5h, reset: props.apiKey.reset_5h_at },
  { label: text('每日限额', 'Daily limit'), used: props.apiKey.usage_1d, limit: props.apiKey.rate_limit_1d, reset: props.apiKey.reset_1d_at },
  { label: text('7 日限额', '7-day limit'), used: props.apiKey.usage_7d, limit: props.apiKey.rate_limit_7d, reset: props.apiKey.reset_7d_at },
].filter(limit => limit.limit > 0))
function nameForGroup(id: number) { return props.groups.find(group => group.id === id)?.name || '#' + id }
function date(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '—' : parsed.toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
function money(value: number) { return '$' + (Number.isFinite(value) ? value : 0).toFixed(2) }
</script>

<style scoped>
.portal-api-key-card { min-width:0; display:flex; flex-direction:column; padding:20px; border:1px solid #efede3; border-radius:12px; background:#fffdf7; }
.portal-api-key-card > header { display:flex; align-items:center; justify-content:space-between; gap:12px; }
.portal-api-key-card[data-selected='true'] { border-color:#000; }
.portal-api-key-select { display:flex; align-items:center; flex-shrink:0; }
.portal-api-key-select input { width:16px; height:16px; accent-color:#000; }
.portal-api-key-select + h3 { flex:1; }
.portal-api-key-card h3 { min-width:0; margin:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:14px; font-weight:500; }
.portal-api-key-status { flex-shrink:0; font-size:10px; color:#76736b; }
.portal-api-key-status[data-status='active'] { color:#000; }
.portal-api-key-status[data-status='expired'],.portal-api-key-status[data-status='quota_exhausted'] { color:#a53d36; }
.portal-api-key-secret { display:flex; min-width:0; align-items:center; justify-content:space-between; gap:8px; margin:16px 0 4px; }
.portal-api-key-secret code { min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:12px; }
.portal-api-key-group { margin:4px 0 18px; font-size:12px; color:#76736b; overflow-wrap:anywhere; }
.portal-api-key-dates { display:grid; gap:7px; margin:0 0 16px; font-size:11px; }
.portal-api-key-dates > div { display:flex; flex-wrap:wrap; align-items:baseline; justify-content:space-between; gap:4px 12px; }
.portal-api-key-dates dt,.portal-api-key-details dt { color:#76736b; }
.portal-api-key-dates dd,.portal-api-key-details dd { margin:0; }
.portal-api-key-details { margin-top:auto; font-size:11px; }
.portal-api-key-details summary { padding:6px 0; cursor:pointer; color:#76736b; }
.portal-api-key-details dl { display:grid; gap:8px; padding:10px 0; }
.portal-api-key-details dl > div { display:grid; grid-template-columns:90px minmax(0,1fr); gap:8px; }
.portal-api-key-details dd { overflow-wrap:anywhere; }
.portal-api-key-details small { display:block; margin-top:3px; font-size:10px; color:#76736b; }
.portal-api-key-card footer { display:flex; align-items:center; gap:12px; margin-top:14px; padding-top:12px; border-top:1px solid #efede3; font-size:12px; }
.portal-api-key-delete { margin-left:auto; color:#a53d36; }
.portal-api-key-card button:disabled { opacity:.45; cursor:wait; }
</style>
