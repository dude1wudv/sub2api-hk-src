<template>
  <div class="portal-api-table-wrap">
    <table class="portal-api-table">
      <caption class="portal-api-table-caption">{{ text('API 密钥', 'API keys') }}</caption>
      <thead>
        <tr>
          <th v-if="selectable" class="portal-api-select-col">
            <input
              type="checkbox"
              :checked="allSelected"
              :indeterminate.prop="someSelected"
              :disabled="!keys.length || busy"
              :aria-label="text('当前页全选', 'Select this page')"
              data-test="table-select-all"
              @change="emit('select-all', ($event.target as HTMLInputElement).checked)"
            />
          </th>
          <th>{{ text('名称', 'Name') }}</th>
          <th>{{ text('API 密钥', 'API key') }}</th>
          <th>{{ text('分组', 'Group') }}</th>
          <th>{{ text('当前并发', 'Concurrency') }}</th>
          <th>{{ text('用量', 'Usage') }}</th>
          <th>{{ text('过期时间', 'Expires') }}</th>
          <th>{{ text('状态', 'Status') }}</th>
          <th>{{ text('创建时间', 'Created') }}</th>
          <th class="portal-api-actions-col">{{ text('操作', 'Actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="apiKey in keys"
          :key="apiKey.id"
          :data-key-id="apiKey.id"
          :data-selected="selectable && selectedIds.includes(apiKey.id)"
        >
          <td v-if="selectable" class="portal-api-select-col">
            <input
              type="checkbox"
              :checked="selectedIds.includes(apiKey.id)"
              :disabled="busyIds.has(apiKey.id)"
              :aria-label="text('选择密钥：', 'Select key: ') + apiKey.name"
              :data-test="'select-key-' + apiKey.id"
              @change="emit('select', apiKey.id, ($event.target as HTMLInputElement).checked)"
            />
          </td>
          <td class="portal-api-name" :title="apiKey.name">{{ apiKey.name }}</td>
          <td>
            <span class="portal-api-secret">
              <code>{{ masked(apiKey.key) }}</code>
              <button type="button" :aria-label="text('复制密钥：', 'Copy key: ') + apiKey.name" @click="emit('copy', apiKey)">
                <Icon name="copy" size="sm" aria-hidden="true" />
              </button>
            </span>
          </td>
          <td>
            <GroupBadge
              v-if="groupFor(apiKey)"
              :name="groupFor(apiKey)?.name || ''"
              :platform="groupFor(apiKey)?.platform"
              :subscription-type="groupFor(apiKey)?.subscription_type"
              :rate-multiplier="groupFor(apiKey)?.rate_multiplier"
              :user-rate-multiplier="groupRates[groupFor(apiKey)?.id ?? -1] ?? null"
              :peak-rate-enabled="groupFor(apiKey)?.peak_rate_enabled"
              :peak-start="groupFor(apiKey)?.peak_start"
              :peak-end="groupFor(apiKey)?.peak_end"
              :peak-rate-multiplier="groupFor(apiKey)?.peak_rate_multiplier"
            />
            <span v-else class="portal-api-muted">{{ text('未绑定分组', 'No group') }}</span>
          </td>
          <td><span class="portal-api-concurrency">{{ apiKey.current_concurrency || 0 }}</span></td>
          <td>
            <div v-if="usageFor(apiKey.id)" class="portal-api-usage">
              <span>{{ text('今日', 'Today') }} <strong>${{ money(usageFor(apiKey.id)?.today_actual_cost ?? 0) }}</strong></span>
              <span>{{ text('累计', 'Total') }} <strong>${{ money(usageFor(apiKey.id)?.total_actual_cost ?? 0) }}</strong></span>
            </div>
            <span v-else class="portal-api-muted">—</span>
          </td>
          <td>
            <span class="portal-api-expiry" :class="apiKey.status === 'expired' ? 'portal-api-expired' : 'portal-api-muted'">
              {{ apiKey.expires_at ? date(apiKey.expires_at) : text('永久有效', 'Never expires') }}
            </span>
          </td>
          <td><span class="portal-api-status" :data-status="apiKey.status">{{ statusName(apiKey.status) }}</span></td>
          <td class="portal-api-created">{{ date(apiKey.created_at) }}</td>
          <td class="portal-api-actions-col">
            <div class="portal-api-actions">
              <button type="button" data-test="use-key" :disabled="busyIds.has(apiKey.id)" @click="emit('use', apiKey)">
                <Icon name="terminal" size="sm" aria-hidden="true" />
                <span>{{ text('使用密钥', 'Use key') }}</span>
              </button>
              <button v-if="!hideCcs" type="button" data-test="import-ccs" :disabled="busyIds.has(apiKey.id)" @click="emit('import-ccs', apiKey)">
                <Icon name="upload" size="sm" aria-hidden="true" />
                <span>{{ text('导入到 CCS', 'Import to CCS') }}</span>
              </button>
              <button type="button" :disabled="busyIds.has(apiKey.id)" @click="emit('toggle', apiKey)">
                <Icon :name="apiKey.status === 'active' ? 'ban' : 'play'" size="sm" aria-hidden="true" />
                <span>{{ apiKey.status === 'active' ? text('禁用', 'Disable') : text('启用', 'Enable') }}</span>
              </button>
              <button type="button" :disabled="busyIds.has(apiKey.id)" @click="emit('edit', apiKey)">
                <Icon name="edit" size="sm" aria-hidden="true" />
                <span>{{ text('编辑', 'Edit') }}</span>
              </button>
              <button type="button" class="portal-api-delete" :disabled="busyIds.has(apiKey.id)" @click="emit('delete', apiKey)">
                <Icon name="trash" size="sm" aria-hidden="true" />
                <span>{{ text('删除', 'Delete') }}</span>
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { maskApiKey } from '@/utils/maskApiKey'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { ApiKey, Group } from '@/types'

const props = withDefaults(defineProps<{
  keys: ApiKey[]
  groups: Group[]
  groupRates: Record<number, number>
  usageStats: Record<string, BatchApiKeyUsageStats>
  busyIds: Set<number>
  selectable?: boolean
  selectedIds: number[]
  hideCcs?: boolean
}>(), {
  selectable: false,
  hideCcs: false,
})

const emit = defineEmits<{
  select: [id: number, checked: boolean]
  'select-all': [checked: boolean]
  copy: [key: ApiKey]
  use: [key: ApiKey]
  'import-ccs': [key: ApiKey]
  edit: [key: ApiKey]
  toggle: [key: ApiKey]
  delete: [key: ApiKey]
}>()

const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const busy = computed(() => props.busyIds.size > 0)
const selectedOnPage = computed(() => props.keys.filter(key => props.selectedIds.includes(key.id)))
const allSelected = computed(() => props.keys.length > 0 && selectedOnPage.value.length === props.keys.length)
const someSelected = computed(() => selectedOnPage.value.length > 0 && !allSelected.value)

function usageFor(id: number) {
  return props.usageStats[String(id)]
}
function groupFor(apiKey: ApiKey) {
  return apiKey.group || (apiKey.group_id == null ? undefined : props.groups.find(group => group.id === apiKey.group_id))
}
function masked(value: string) {
  return maskApiKey(value)
}
function statusName(status: ApiKey['status']) {
  return {
    active: text('活跃', 'Active'),
    inactive: text('停用', 'Inactive'),
    expired: text('过期', 'Expired'),
    quota_exhausted: text('额度用尽', 'Quota exhausted'),
  }[status]
}
function date(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '—'
  return parsed.toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}
function money(value: number) {
  return (Number.isFinite(value) ? value : 0).toFixed(4)
}
</script>

<style scoped>
.portal-api-table-caption { position:absolute; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0,0,0,0); white-space:nowrap; border:0; }
.portal-api-table-wrap { overflow:auto; border:1px solid #e4e9ef; border-radius:14px; background:#fff; }
.portal-api-table { width:max-content; min-width:100%; border-collapse:separate; border-spacing:0; text-align:left; font-size:13px; color:#1c2430; }
.portal-api-table th { padding:12px 10px; color:#8b93a1; background:#f7f9fb; font-size:12px; font-weight:500; white-space:nowrap; border-bottom:1px solid #e8edf2; }
.portal-api-table td { padding:14px 10px; border-bottom:1px solid #eef2f6; vertical-align:middle; background:#fff; }
.portal-api-table tbody tr:last-child td { border-bottom:0; }
.portal-api-table tbody tr:hover td { background:#f8fbff; }
.portal-api-table tbody tr[data-selected='true'] td { background:#f2f7ff; }
.portal-api-select-col { width:42px; }
.portal-api-select-col input { width:15px; height:15px; accent-color:#1d4ed8; }
.portal-api-name { max-width:160px; overflow:hidden; font-weight:600; text-overflow:ellipsis; white-space:nowrap; }
.portal-api-secret { display:inline-flex; max-width:210px; align-items:center; gap:4px; padding:3px 4px 3px 8px; border-radius:8px; background:#eef6ff; color:#2458b5; }
.portal-api-secret code { min-width:0; overflow:hidden; font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace; font-size:12px; text-overflow:ellipsis; white-space:nowrap; }
.portal-api-secret button { display:inline-flex; padding:2px; border:0; border-radius:5px; color:inherit; background:transparent; cursor:pointer; }
.portal-api-secret button:hover { background:#dceaff; }
.portal-api-muted { color:#8b93a1; }
.portal-api-expiry { white-space:nowrap; }
.portal-api-expired { color:#d14343; }
.portal-api-concurrency { display:inline-flex; min-width:28px; justify-content:center; padding:2px 8px; border-radius:8px; background:#f1f3f6; color:#5d6675; font-variant-numeric:tabular-nums; }
.portal-api-usage { display:grid; gap:2px; color:#5d6675; font-size:12px; white-space:nowrap; }
.portal-api-usage strong { color:#1c2430; font-weight:600; }
.portal-api-status { display:inline-flex; align-items:center; padding:2px 8px; border-radius:999px; font-size:12px; font-weight:600; white-space:nowrap; }
.portal-api-status[data-status='active'] { color:#178a45; background:#e7f8ee; }
.portal-api-status[data-status='inactive'] { color:#6b7280; background:#f1f2f4; }
.portal-api-status[data-status='expired'],
.portal-api-status[data-status='quota_exhausted'] { color:#d14343; background:#fdecec; }
.portal-api-created { min-width:132px; color:#5d6675; white-space:nowrap; font-variant-numeric:tabular-nums; }
.portal-api-actions-col { position:sticky; right:0; z-index:1; box-shadow:-10px 0 12px -14px rgba(28,36,48,.45); }
.portal-api-table th.portal-api-actions-col { z-index:2; background:#f7f9fb; }
.portal-api-actions { display:flex; align-items:flex-start; gap:2px; }
.portal-api-actions > button { position:relative; display:inline-flex; min-width:46px; flex-direction:column; align-items:center; gap:3px; padding:4px; border:0; border-radius:8px; color:#4b5568; background:transparent; font:inherit; font-size:11px; line-height:1.2; cursor:pointer; }
.portal-api-actions > button:hover { background:#eef3f8; color:#111827; }
.portal-api-actions > button:disabled { opacity:.45; cursor:wait; }
.portal-api-actions .portal-api-delete { color:#d14343; }
.portal-api-actions .portal-api-delete:hover { background:#fdecec; }
@media(max-width:739px) {
  .portal-api-actions > button span { position:absolute; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0,0,0,0); white-space:nowrap; border:0; }
  .portal-api-actions > button { min-width:34px; }
}
</style>
