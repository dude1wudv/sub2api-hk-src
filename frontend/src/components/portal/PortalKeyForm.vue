<template>
  <PortalDialog :open="open" :title="editing ? text('编辑 API Key', 'Edit API key') : text('创建 API Key', 'Create API key')" @close="!saving && $emit('close')">
    <form @submit.prevent="submit">
      <p v-if="error" class="portal-error" role="alert">{{ error }}</p>
      <label class="portal-field">{{ text('名称', 'Name') }}<input v-model="form.name" required maxlength="100" :placeholder="text('例如：我的应用', 'e.g. My application')" autofocus /></label>
      <label class="portal-field">{{ text('接入分组', 'Connection group') }}<select v-model="form.group" :required="!form.routing.length" :disabled="form.routing.length > 0"><option value="">{{ text('请选择分组', 'Select a group') }}</option><option v-for="group in groups" :key="group.id" :value="String(group.id)">{{ group.name }}</option></select></label>
      <details class="key-advanced"><summary>{{ text('高级设置', 'Advanced settings') }}</summary>
        <div class="portal-fields two-columns">
          <label class="portal-field">{{ text('额度上限（USD，0 为不限）', 'Quota (USD, 0 = unlimited)') }}<input v-model.number="form.quota" type="number" min="0" step="0.01" required /></label>
          <label v-if="!editing" class="portal-field">{{ text('有效天数（留空为永久）', 'Expires in days (optional)') }}<input v-model.number="form.days" type="number" min="1" step="1" /></label>
          <label v-else class="portal-field">{{ text('到期时间（留空为永久）', 'Expiry (empty = never)') }}<input v-model="form.expiry" type="datetime-local" /></label>
        </div>
        <div class="key-rate-fields"><label v-for="field in rateFields" :key="field.key" class="portal-field">{{ field.label }}<input v-model.number="form[field.key]" type="number" min="0" step="0.01" required /></label></div>
        <div v-if="editing" class="key-resets"><label><input v-model="form.resetQuota" type="checkbox" />{{ text('重置已用额度', 'Reset used quota') }}</label><label><input v-model="form.resetRates" type="checkbox" />{{ text('重置限流计数', 'Reset rate-limit usage') }}</label><small>{{ text('仅勾选后保存才会重置。', 'Resets apply only when selected and saved.') }}</small></div>
        <label class="portal-field">{{ text('IP 白名单（每行一个 IP 或 CIDR）', 'IP allowlist (one IP/CIDR per line)') }}<textarea v-model="form.allow" rows="2" /></label>
        <label class="portal-field">{{ text('IP 黑名单（每行一个 IP 或 CIDR）', 'IP blocklist (one IP/CIDR per line)') }}<textarea v-model="form.block" rows="2" /></label>
        <label v-if="!editing" class="portal-field">{{ text('自定义密钥（可选）', 'Custom key (optional)') }}<input v-model="form.custom" type="password" autocomplete="off" /></label>
        <fieldset v-if="routingGroups.length > 1 || form.routing.length" class="key-routing"><legend>{{ text('智能路由候选分组（最多10个）', 'Smart routing groups (up to 10)') }}</legend><label v-for="group in routingGroups" :key="group.id"><input v-model="form.routing" type="checkbox" :value="group.id" />{{ group.name }}</label>
          <template v-if="form.routing.length"><p class="routing-help">{{ text('按下列顺序依次回退，按实际命中的分组计费。', 'Groups are tried in this order; billing uses the group that serves the request.') }}</p><ol class="routing-order"><li v-for="(id, index) in form.routing" :key="id"><span>{{ index + 1 }}. {{ groups.find(group => group.id === id)?.name || `#${id}` }} <small v-if="!routingGroups.some(group => group.id === id)">{{ text('不可用', 'Unavailable') }}</small><small v-if="index === 0">{{ text('首选', 'Primary') }}</small></span><button type="button" :disabled="index === 0" :aria-label="text(`上移第${index + 1}个分组`, `Move group ${index + 1} up`)" @click="moveRouting(index, -1)">↑</button><button type="button" :disabled="index === form.routing.length - 1" :aria-label="text(`下移第${index + 1}个分组`, `Move group ${index + 1} down`)" @click="moveRouting(index, 1)">↓</button><button type="button" :aria-label="text(`移除第${index + 1}个分组`, `Remove group ${index + 1}`)" @click="form.routing.splice(index, 1)">×</button></li></ol></template>
        </fieldset>
      </details>
      <footer class="portal-dialog-actions"><button type="button" class="portal-button secondary" :disabled="saving" @click="$emit('close')">{{ text('取消', 'Cancel') }}</button><button class="portal-button" :disabled="saving">{{ saving ? text('保存中…', 'Saving…') : text('保存', 'Save') }}</button></footer>
    </form>
  </PortalDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PortalDialog from './PortalDialog.vue'
import { keysAPI } from '@/api/keys'
import { isSmartRoutingGroup } from '@/utils/smartRouting'
import type { ApiKey, Group } from '@/types'
const props = defineProps<{ open: boolean; editing: ApiKey | null; groups: Group[]; selectedGroup?: number }>()
const emit = defineEmits<{ close: []; saved: [key: ApiKey] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const saving = ref(false)
const error = ref('')
const routingGroups = computed(() => props.groups.filter(isSmartRoutingGroup))
const form = reactive({ name: '', group: '', quota: 0, days: '' as number | string, expiry: '', limit5h: 0, limit1d: 0, limit7d: 0, allow: '', block: '', custom: '', routing: [] as number[], resetQuota: false, resetRates: false })
const rateFields = computed(() => [ { key: 'limit5h' as const, label: text('5小时限额（USD）', '5-hour limit (USD)') }, { key: 'limit1d' as const, label: text('每日限额（USD）', 'Daily limit (USD)') }, { key: 'limit7d' as const, label: text('7日限额（USD）', '7-day limit (USD)') } ])
function localDate(iso: string | null) { if (!iso) return ''; const date = new Date(iso); return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0,16) }
watch(() => props.open, (open) => {
  if (!open) return
  const key = props.editing
  Object.assign(form, { name:key?.name || '', group:String(key?.group_id ?? props.selectedGroup ?? ''), quota:key?.quota || 0, days:'', expiry:localDate(key?.expires_at ?? null), limit5h:key?.rate_limit_5h || 0, limit1d:key?.rate_limit_1d || 0, limit7d:key?.rate_limit_7d || 0, allow:key?.ip_whitelist?.join('\n') || '', block:key?.ip_blacklist?.join('\n') || '', custom:'', routing:[...(key?.routing_group_ids || [])], resetQuota:false, resetRates:false })
  error.value = ''
}, { immediate: true })
function moveRouting(index: number, direction: number) {
  const target = index + direction
  if (target < 0 || target >= form.routing.length) return
  const [id] = form.routing.splice(index, 1)
  form.routing.splice(target, 0, id)
}
async function submit() {
  if (saving.value || !form.name.trim()) return
  const groupId = form.routing[0] ?? (form.group ? Number(form.group) : null)
  if (groupId === null || !props.groups.some(group => group.id === groupId)) {
    error.value = text('请选择可用的接入分组。', 'Select an available connection group.')
    return
  }
  if (form.routing.length > 10 || form.routing.some(id => !routingGroups.value.some(group => group.id === id))) {
    error.value = text('请选择最多10个支持智能路由的分组。', 'Select up to 10 groups that support smart routing.')
    return
  }
  saving.value = true; error.value = ''
  const lines = (value: string) => value.split(/[\n,]+/).map(v => v.trim()).filter(Boolean)
  const limits = { rate_limit_5h:Number(form.limit5h), rate_limit_1d:Number(form.limit1d), rate_limit_7d:Number(form.limit7d) }
  try {
    const key = props.editing
      ? await keysAPI.update(props.editing.id, { name:form.name.trim(), group_id:groupId, quota:Number(form.quota), expires_at:form.expiry ? new Date(form.expiry).toISOString() : '', ip_whitelist:lines(form.allow), ip_blacklist:lines(form.block), routing_group_ids:form.routing, ...limits, ...(form.resetQuota ? { reset_quota:true } : {}), ...(form.resetRates ? { reset_rate_limit_usage:true } : {}) })
      : await keysAPI.create(form.name.trim(), groupId, form.custom || undefined, lines(form.allow), lines(form.block), Number(form.quota), form.days ? Number(form.days) : undefined, limits, form.routing)
    emit('saved', key)
  } catch (e) { error.value = (e as {message?: string}).message || text('保存失败，请重试。', 'Could not save. Please retry.') }
  finally { saving.value = false }
}
</script>
<style scoped>
.key-advanced { margin-top:20px; } .key-advanced summary { cursor:pointer; padding:8px 0; } .key-rate-fields { display:grid; grid-template-columns:repeat(3,1fr); gap:12px; } .key-routing { display:grid; gap:8px; margin-top:20px; } .key-routing label { display:flex; gap:8px; } @media(max-width:500px) { .key-rate-fields { grid-template-columns:1fr; } }
.key-resets { display:grid; gap:10px; margin:10px 0 20px; } .key-resets label { display:flex; align-items:center; gap:8px; } .key-resets small,.routing-help { color:#8d8c87; font-size:12px; line-height:1.6; }
.routing-order { display:grid; gap:8px; padding:0; margin:5px 0 0; list-style:none; } .routing-order li { display:flex; align-items:center; gap:8px; padding:8px 10px; border:1px solid #efede3; border-radius:6px; } .routing-order li>span { flex:1; } .routing-order small { margin-left:8px; color:#8d8c87; } .routing-order button { width:28px; height:28px; border:1px solid #efede3; border-radius:4px; } .routing-order button:disabled { opacity:.3; }
</style>
