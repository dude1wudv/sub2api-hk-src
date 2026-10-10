<template>
  <PortalDialog :open="open" :title="text('批量编辑 API Keys', 'Edit API keys in bulk')" wide @close="close">
    <form class="portal-api-bulk-form" @submit.prevent="submit">
      <p class="portal-api-bulk-intro">{{ text('仅修改勾选的字段，其余设置保持不变。', 'Only checked fields will change. Other settings stay as they are.') }}</p>
      <details class="portal-api-bulk-selection"><summary>{{ text('本次选择 ', 'Selected: ') }}{{ initialKeys.length }}{{ text(' 个密钥', ' keys') }}</summary><ul><li v-for="key in initialKeys" :key="key.id">#{{ key.id }} {{ key.name }}</li></ul></details>

      <fieldset v-if="pendingKeys.length" :disabled="submitting" class="portal-api-bulk-fields">
        <div class="portal-api-bulk-field">
          <label class="portal-api-bulk-check"><input v-model="enabled.group_id" type="checkbox" :disabled="!canChangeGroup && !enabled.group_id" data-test="enable-group" />{{ text('接入分组', 'Connection group') }}</label>
          <label v-if="enabled.group_id" class="portal-field"><span class="sr-only">{{ text('选择可绑定分组', 'Choose an available group') }}</span><select v-model.number="groupId" required data-test="group-input"><option :value="null" disabled>{{ text('请选择分组', 'Select a group') }}</option><option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option></select></label>
          <p v-if="routingKeys.length" class="portal-api-bulk-hint">{{ text('选择中包含智能路由密钥。请通过单个密钥的“编辑”调整路由；这里可继续修改其他字段。', 'This selection contains smart-routing keys. Edit those keys individually to change routing; other fields can still be updated here.') }}</p>
          <p v-else-if="groupsLoading || groupsError || !groups.length" class="portal-api-bulk-hint">{{ groupsLoading ? text('分组加载中，其他字段仍可编辑。', 'Groups are loading. Other fields can still be edited.') : text('暂无可用的分组列表，请关闭后刷新分组。其他字段仍可编辑。', 'No available group list. Close this dialog and refresh groups; other fields can still be edited.') }}</p>
        </div>
        <div class="portal-api-bulk-field">
          <label class="portal-api-bulk-check"><input v-model="enabled.status" type="checkbox" data-test="enable-status" />{{ text('启停状态', 'Status') }}</label>
          <label v-if="enabled.status" class="portal-field"><span class="sr-only">{{ text('密钥状态', 'Key status') }}</span><select v-model="status" data-test="status-input"><option value="active">{{ text('启用', 'Enable') }}</option><option value="inactive">{{ text('停用', 'Disable') }}</option></select></label>
        </div>
        <div v-for="field in limitFields" :key="field.key" class="portal-api-bulk-field">
          <label class="portal-api-bulk-check"><input v-model="enabled[field.key]" type="checkbox" :data-test="'enable-' + field.key" />{{ field.label }}</label>
          <label v-if="enabled[field.key]" class="portal-field"><span class="sr-only">{{ field.label }}</span><input v-model="limits[field.key]" type="number" min="0" step="any" required :data-test="field.key + '-input'" /><small>{{ text('单位 USD；0 表示不限。不会重置已有用量。', 'USD. 0 means unlimited. Existing usage is not reset.') }}</small></label>
        </div>
        <div class="portal-api-bulk-field portal-api-bulk-full">
          <label class="portal-api-bulk-check"><input v-model="enabled.expires_at" type="checkbox" data-test="enable-expiration" />{{ text('到期时间', 'Expiration') }}</label>
          <div v-if="enabled.expires_at" class="portal-api-bulk-expiry">
            <label class="portal-api-bulk-check"><input v-model="neverExpires" type="checkbox" data-test="never-expires" />{{ text('永久有效，清除到期时间', 'Never expires — clear the expiry date') }}</label>
            <label v-if="!neverExpires" class="portal-field"><span class="sr-only">{{ text('到期日期和时间', 'Expiry date and time') }}</span><input v-model="expirationDate" type="datetime-local" required data-test="expiration-input" /><small>{{ text('按浏览器本地时区填写。', 'Enter a time in your browser’s local timezone.') }}</small></label>
          </div>
        </div>
        <div v-for="field in ipFields" :key="field.key" class="portal-api-bulk-field">
          <label class="portal-api-bulk-check"><input v-model="enabled[field.key]" type="checkbox" :data-test="'enable-' + field.key" />{{ field.label }}</label>
          <label v-if="enabled[field.key]" class="portal-field"><span class="sr-only">{{ field.label }}</span><textarea v-model="ipLists[field.key]" rows="3" :data-test="field.key + '-input'" /><small>{{ text('每行一个 IP 或 CIDR。勾选后留空会清除此列表。', 'One IP or CIDR per line. An empty checked list clears the existing list.') }}</small></label>
        </div>
      </fieldset>

      <p v-if="validationError && pendingKeys.length" class="portal-error" role="alert">{{ validationError }}</p>
      <p v-if="requestError" class="portal-error" role="alert">{{ requestError }}</p>
      <section v-if="attempted" class="portal-api-bulk-results" aria-live="polite">
        <p>{{ text('已成功 ', 'Succeeded: ') }}<strong>{{ succeededKeys.length }}</strong>{{ text(' 个；待重试 ', '; pending retry: ') }}<strong>{{ pendingKeys.length }}</strong>{{ text(' 个。', '.') }}</p>
        <p v-if="pendingKeys.length" class="portal-api-bulk-hint">{{ text('再次提交只处理下方失败的密钥，已成功的密钥不会重复提交。你可修改勾选字段后重试。', 'Submitting again only updates the failed keys below. Successful keys are not sent again. You may adjust the checked fields before retrying.') }}</p>
        <details v-if="succeededKeys.length" class="portal-api-bulk-successes"><summary>{{ text('查看成功项', 'Show successful keys') }}</summary><ul><li v-for="key in succeededKeys" :key="key.id">#{{ key.id }} {{ key.name }}</li></ul></details>
        <ul v-if="failures.length" class="portal-api-bulk-failures" role="alert"><li v-for="failure in failures" :key="failure.id">#{{ failure.id }} {{ failure.name }}：{{ failure.message }}</li></ul>
      </section>
      <footer class="portal-dialog-actions">
        <button type="button" class="portal-button secondary" :disabled="submitting" @click="close">{{ attempted ? text('关闭', 'Close') : text('取消', 'Cancel') }}</button>
        <button v-if="pendingKeys.length" type="submit" class="portal-button" :disabled="!canSubmit" data-test="submit">{{ submitting ? text('更新中…', 'Updating…') : attempted ? text('仅重试失败项', 'Retry failed keys only') : text('应用到 ', 'Apply to ') + pendingKeys.length + text(' 个密钥', ' keys') }}</button>
      </footer>
    </form>
  </PortalDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PortalDialog from './PortalDialog.vue'
import { keysAPI } from '@/api/keys'
import { useAppStore } from '@/stores/app'
import type { ApiKey, Group, UpdateApiKeyRequest } from '@/types'

type SelectedKey = Pick<ApiKey, 'id' | 'name' | 'routing_group_ids'>
type LimitField = 'quota' | 'rate_limit_5h' | 'rate_limit_1d' | 'rate_limit_7d'
type IPField = 'ip_whitelist' | 'ip_blacklist'
type EditableField = LimitField | IPField | 'group_id' | 'status' | 'expires_at'
const props = withDefaults(defineProps<{ open: boolean; selectedKeys: SelectedKey[]; groups: Group[]; groupsLoading?: boolean; groupsError?: string }>(), { groupsLoading: false, groupsError: '' })
const emit = defineEmits<{ close: []; updated: [succeededIds: number[]] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const appStore = useAppStore()
const submitting = ref(false)
const attempted = ref(false)
const initialKeys = ref<SelectedKey[]>([])
const pendingKeys = ref<SelectedKey[]>([])
const succeededKeys = ref<SelectedKey[]>([])
const failures = ref<Array<{ id: number; name: string; message: string }>>([])
const requestError = ref('')
const enabled = reactive<Record<EditableField, boolean>>({ group_id: false, status: false, quota: false, rate_limit_5h: false, rate_limit_1d: false, rate_limit_7d: false, expires_at: false, ip_whitelist: false, ip_blacklist: false })
const groupId = ref<number | null>(null)
const status = ref<'active' | 'inactive'>('active')
const limits = reactive<Record<LimitField, string | number>>({ quota: '', rate_limit_5h: '', rate_limit_1d: '', rate_limit_7d: '' })
const ipLists = reactive<Record<IPField, string>>({ ip_whitelist: '', ip_blacklist: '' })
const neverExpires = ref(false)
const expirationDate = ref('')
const limitFields = computed<Array<{ key: LimitField; label: string }>>(() => [
  { key: 'quota', label: text('额度上限', 'Quota') },
  { key: 'rate_limit_5h', label: text('5 小时限额', '5-hour limit') },
  { key: 'rate_limit_1d', label: text('每日限额', 'Daily limit') },
  { key: 'rate_limit_7d', label: text('7 日限额', '7-day limit') },
])
const ipFields = computed<Array<{ key: IPField; label: string }>>(() => [
  { key: 'ip_whitelist', label: text('IP 白名单', 'IP allowlist') },
  { key: 'ip_blacklist', label: text('IP 黑名单', 'IP blocklist') },
])
const routingKeys = computed(() => pendingKeys.value.filter(key => key.routing_group_ids?.length))
const canChangeGroup = computed(() => !routingKeys.value.length && !props.groupsLoading && !props.groupsError && props.groups.length > 0)
const validationError = computed(() => {
  if (enabled.group_id && (!canChangeGroup.value || !props.groups.some(group => group.id === groupId.value))) return text('请选择账户可绑定的分组；智能路由密钥请逐个编辑。', 'Choose an available group. Smart-routing keys must be edited individually.')
  for (const { key } of limitFields.value) {
    if (!enabled[key]) continue
    const value = String(limits[key]).trim()
    if (!value || !Number.isFinite(Number(value)) || Number(value) < 0) return text('勾选的额度或限额必须填写有限的非负数字。', 'Checked quota and limits require a finite non-negative number.')
  }
  if (enabled.expires_at && !neverExpires.value && !Number.isFinite(Date.parse(expirationDate.value))) return text('请选择有效的到期时间，或勾选永久有效。', 'Choose a valid expiration time or select Never expires.')
  return ''
})
const canSubmit = computed(() => pendingKeys.value.length > 0 && Object.values(enabled).some(Boolean) && !validationError.value && !submitting.value)
let session = 0
let disposed = false

watch(() => props.open, open => {
  session++
  if (!open) return
  const uniqueKeys = new Map(props.selectedKeys.map(key => [key.id, { id: key.id, name: key.name, routing_group_ids: [...(key.routing_group_ids || [])] }]))
  initialKeys.value = [...uniqueKeys.values()]
  pendingKeys.value = [...initialKeys.value]
  succeededKeys.value = []
  failures.value = []
  requestError.value = ''
  attempted.value = false
  submitting.value = false
  for (const field of Object.keys(enabled) as EditableField[]) enabled[field] = false
  for (const { key } of limitFields.value) limits[key] = ''
  for (const { key } of ipFields.value) ipLists[key] = ''
  groupId.value = null
  status.value = 'active'
  neverExpires.value = false
  expirationDate.value = ''
}, { immediate: true })
onBeforeUnmount(() => { disposed = true })

function close() { if (!submitting.value) emit('close') }
function errorMessage(error: unknown) {
  const message = (error as { message?: unknown } | null)?.message
  return typeof message === 'string' && message ? message : text('更新失败，请重试。', 'Could not update this key. Please retry.')
}
async function submit() {
  if (!canSubmit.value) return
  const updates: UpdateApiKeyRequest = {}
  if (enabled.group_id) updates.group_id = groupId.value
  if (enabled.status) updates.status = status.value
  for (const { key } of limitFields.value) if (enabled[key]) updates[key] = Number(limits[key])
  for (const { key } of ipFields.value) if (enabled[key]) updates[key] = ipLists[key].split('\n').map(ip => ip.trim()).filter(Boolean)
  if (enabled.expires_at) updates.expires_at = neverExpires.value ? '' : new Date(expirationDate.value).toISOString()
  const attemptKeys = [...pendingKeys.value]
  const currentSession = session
  submitting.value = true
  requestError.value = ''
  try {
    const result = await keysAPI.bulkUpdate(attemptKeys.map(key => key.id), updates)
    if (disposed || session !== currentSession) return
    attempted.value = true
    const succeeded = new Set(result.succeededIds)
    const failed = new Set(result.failures.map(item => item.id))
    succeededKeys.value.push(...attemptKeys.filter(key => succeeded.has(key.id)))
    failures.value = result.failures.map(({ id, error }) => ({ id, name: attemptKeys.find(key => key.id === id)?.name || '', message: errorMessage(error) }))
    pendingKeys.value = attemptKeys.filter(key => failed.has(key.id) && !succeeded.has(key.id))
    if (result.succeededIds.length) emit('updated', result.succeededIds)
    if (!pendingKeys.value.length) appStore.showSuccess(text('已更新 ', 'Updated ') + succeededKeys.value.length + text(' 个密钥', ' keys'))
  } catch (error) {
    if (!disposed && session === currentSession) requestError.value = errorMessage(error)
  } finally {
    if (!disposed && session === currentSession) submitting.value = false
  }
}
</script>

<style scoped>
.portal-api-bulk-intro { margin:0 0 12px; color:#76736b; font-size:13px; line-height:1.7; }
.portal-api-bulk-selection { margin-bottom:20px; font-size:12px; }
.portal-api-bulk-selection summary,.portal-api-bulk-successes summary { cursor:pointer; }
.portal-api-bulk-selection ul,.portal-api-bulk-successes ul { max-height:128px; overflow:auto; margin:10px 0 0; padding-left:20px; list-style:disc; overflow-wrap:anywhere; }
.portal-api-bulk-fields { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:20px 24px; padding:18px 0; border:0; border-block:1px solid #efede3; }
.portal-api-bulk-fields:disabled { opacity:.6; }
.portal-api-bulk-field { min-width:0; }
.portal-api-bulk-full { grid-column:1 / -1; }
.portal-api-bulk-check { display:flex; align-items:center; gap:9px; font-size:13px; line-height:1.7; }
.portal-api-bulk-check input { flex-shrink:0; width:15px; height:15px; accent-color:#000; }
.portal-api-bulk-form .portal-field { margin-top:12px; font-size:13px; }
.portal-api-bulk-form .portal-field small { font-size:11px; line-height:1.6; color:#76736b; }
.portal-api-bulk-hint { margin:10px 0 0; color:#76736b; font-size:12px; line-height:1.7; }
.portal-api-bulk-expiry { margin-top:12px; }
.portal-api-bulk-results { margin-top:20px; padding:16px; border:1px solid #efede3; border-radius:8px; background:#f7f5ee; font-size:13px; }
.portal-api-bulk-results > p { margin:0 0 8px; }
.portal-api-bulk-successes { margin-top:12px; font-size:12px; }
.portal-api-bulk-failures { display:grid; gap:6px; margin:12px 0 0; padding-left:20px; color:#b8253a; list-style:disc; overflow-wrap:anywhere; }
.portal-api-bulk-form > .portal-error { margin-top:16px; }
@media(max-width:520px) { .portal-api-bulk-fields { grid-template-columns:1fr; gap:18px; } }
</style>
