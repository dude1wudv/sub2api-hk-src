<template>
  <PortalDialog :open="open" :title="editing ? text('编辑 API Key', 'Edit API key') : text('创建 API Key', 'Create API key')" wide @close="!saving && $emit('close')">
    <form class="key-editor" @submit.prevent="submit">
      <div class="key-editor-body">
        <section class="key-picker" :aria-label="text('选择分组', 'Choose a group')">
          <h3><span class="key-step">1</span>{{ text('选择分组', 'Choose a group') }}</h3>
          <div class="key-mode" role="group" :aria-label="text('接入模式', 'Connection mode')">
            <button type="button" :aria-pressed="!form.smart" @click="setMode(false)">{{ text('固定分组', 'Fixed group') }}</button>
            <button type="button" :aria-pressed="form.smart" @click="setMode(true)">{{ text('智能路由', 'Smart routing') }}</button>
          </div>
          <p v-if="form.smart" class="key-hint">{{ text('点击分组添加候选，再在右侧调整顺序。最多 10 个，按实际命中的分组计费。', 'Add candidate groups, then arrange their order on the right. Up to 10; billing uses the group that serves the request.') }}</p>
          <label class="key-search"><span aria-hidden="true">⌕</span><input v-model="search" type="search" :aria-label="text('搜索分组', 'Search groups')" :placeholder="text('搜索分组名称或说明', 'Search group names or descriptions')" /></label>
          <div class="key-providers" role="group" :aria-label="text('分组类别', 'Group categories')"><button v-for="provider in providers" :key="provider.value" type="button" :aria-pressed="category === provider.value" @click="category = provider.value">{{ provider.label }} <span>{{ provider.count }}</span></button></div>
          <div class="key-group-list">
            <button v-for="group in filteredGroups" :key="group.id" type="button" class="key-group-card" :class="{ selected: isSelected(group.id) }" :aria-pressed="isSelected(group.id)" :disabled="form.smart && (form.routing.includes(group.id) || form.routing.length >= 10)" :data-group-id="group.id" @click="selectGroup(group.id)">
              <span class="key-radio" aria-hidden="true">{{ isSelected(group.id) ? '✓' : '+' }}</span>
              <span class="key-group-copy"><strong>{{ group.name }}</strong><span v-if="group.description" class="key-description">{{ group.description }}</span><small v-if="group.subscription_type === 'subscription'">{{ text('订阅分组', 'Subscription group') }}</small></span>
              <span class="key-group-rate"><span>{{ text('分组倍率', 'Group rate') }} <b>{{ rate(group) }}×</b></span><small v-if="hasUserRate(group)">{{ text('专属倍率', 'Your rate') }}</small><small v-if="group.peak_rate_enabled">{{ peakText(group) }}</small></span>
            </button>
            <p v-if="!filteredGroups.length" class="key-empty">{{ text('没有匹配的可用分组', 'No matching groups') }}</p>
          </div>
          <p class="key-count">{{ text('显示 ', 'Showing ') }}{{ filteredGroups.length }} / {{ eligibleGroups.length }}<span v-if="form.smart">{{ text('已添加 ', 'Added ') }}{{ form.routing.length }} / 10</span></p>
        </section>
        <section class="key-configuration" :aria-label="text('配置密钥', 'Configure key')">
          <h3><span class="key-step">2</span>{{ text('配置 API Key', 'Configure API key') }}</h3>
          <p v-if="error" class="portal-error" role="alert">{{ error }}</p>
          <div v-if="form.smart" class="key-routing">
            <div class="key-routing-heading"><h4>{{ text('候选分组顺序', 'Routing order') }}</h4><span>{{ form.routing.length }} / 10</span></div>
            <ol v-if="form.routing.length" class="routing-order">
              <li v-for="(id, index) in form.routing" :key="id" :class="{ unavailable: !routingGroups.some(group => group.id === id) }">
                <span class="route-number">{{ index + 1 }}</span><div class="route-copy"><strong>{{ groups.find(group => group.id === id)?.name || `#${id}` }}</strong><small v-if="!routingGroups.some(group => group.id === id)">{{ text('不可用，请移除或替换', 'Unavailable — remove or replace') }}</small><small v-else-if="index === 0">{{ text('首选分组', 'Primary group') }}</small><small v-else>{{ text('回退候选', 'Fallback group') }}</small></div>
                <div class="route-actions"><button type="button" :disabled="index === 0" :aria-label="text(`上移第${index + 1}个分组`, `Move group ${index + 1} up`)" @click="moveRouting(index, -1)">↑</button><button type="button" :disabled="index === form.routing.length - 1" :aria-label="text(`下移第${index + 1}个分组`, `Move group ${index + 1} down`)" @click="moveRouting(index, 1)">↓</button><button type="button" :aria-label="text(`移除第${index + 1}个分组`, `Remove group ${index + 1}`)" @click="removeRouting(index)">×</button></div>
              </li>
            </ol>
            <p v-else class="key-selected-empty">{{ text('从左侧添加路由候选分组。', 'Add routing candidates from the group list.') }}</p>
            <p class="key-hint">{{ text('候选按顺序回退；不同分组的模型与接口支持范围可能不同。', 'Candidates are tried in order. Model and endpoint support can differ by group.') }}</p>
          </div>
          <div v-else class="key-selected"><h4>{{ text('所选分组', 'Selected group') }}</h4><div v-if="selectedGroup" class="key-selected-summary"><strong>{{ selectedGroup.name }}</strong><span>{{ text('分组倍率 ', 'Group rate ') }}{{ rate(selectedGroup) }}×</span><small v-if="selectedGroup.description">{{ selectedGroup.description }}</small><small v-if="selectedGroup.peak_rate_enabled">{{ peakText(selectedGroup) }}</small></div><p v-else class="key-selected-empty">{{ text('从左侧选择分组，查看分组倍率与说明。', 'Choose a group to view its rate and description.') }}</p></div>
          <label class="portal-field">{{ text('名称', 'Name') }}<input v-model="form.name" required maxlength="100" :placeholder="text('为密钥起一个便于识别的名称', 'A memorable name for this key')" data-test="key-name" /></label>
          <div class="key-common-fields">
            <label v-if="!editing" class="portal-field">{{ text('有效期', 'Expiry') }}<input v-model.number="form.days" type="number" min="1" step="1" :placeholder="text('永久有效', 'Never expires')" data-test="key-days" /><small>{{ text('有效天数，留空为永久', 'Days until expiry; empty means never') }}</small></label>
            <label v-else class="portal-field">{{ text('有效期', 'Expiry') }}<input v-model="form.expiry" type="datetime-local" /><small>{{ text('留空为永久有效', 'Empty means never expires') }}</small></label>
            <label class="portal-field">{{ text('额度（USD）', 'Quota (USD)') }}<input v-model.number="form.quota" type="number" min="0" step="0.01" required data-test="key-quota" /><small>{{ text('0 为不限额度', '0 means unlimited') }}</small></label>
          </div>
          <details class="key-advanced"><summary>{{ text('高级设置', 'Advanced settings') }}</summary>
            <div class="key-rate-fields"><label v-for="field in rateFields" :key="field.key" class="portal-field">{{ field.label }}<input v-model.number="form[field.key]" type="number" min="0" step="0.01" required /></label></div><p class="key-hint">{{ text('窗口限额按 USD 计算，0 为不限。', 'Window limits are in USD; 0 means unlimited.') }}</p>
            <div v-if="editing" class="key-resets"><label><input v-model="form.resetQuota" type="checkbox" />{{ text('重置已用额度', 'Reset used quota') }}</label><label><input v-model="form.resetRates" type="checkbox" />{{ text('重置限流计数', 'Reset rate-limit usage') }}</label><small>{{ text('仅勾选后保存才会重置。', 'Resets apply only when selected and saved.') }}</small></div>
            <label class="portal-field">{{ text('IP 白名单（每行一个 IP 或 CIDR）', 'IP allowlist (one IP/CIDR per line)') }}<textarea v-model="form.allow" rows="2" /></label>
            <label class="portal-field">{{ text('IP 黑名单（每行一个 IP 或 CIDR）', 'IP blocklist (one IP/CIDR per line)') }}<textarea v-model="form.block" rows="2" /></label>
            <label v-if="!editing" class="portal-field">{{ text('自定义密钥（可选）', 'Custom key (optional)') }}<input v-model="form.custom" type="password" autocomplete="off" /><small>{{ text('至少16位，仅支持字母、数字、下划线和连字符。', 'At least 16 characters: letters, numbers, underscores or hyphens.') }}</small></label>
          </details>
        </section>
      </div>
      <footer class="key-editor-footer"><span aria-live="polite">{{ form.smart ? text(`已选择 ${form.routing.length} 个路由分组`, `${form.routing.length} routing groups selected`) : selectedGroup?.name || text('请选择分组', 'Select a group') }}</span><button type="button" class="portal-button secondary" :disabled="saving" @click="$emit('close')">{{ text('取消', 'Cancel') }}</button><button class="portal-button" :disabled="saving">{{ saving ? text('保存中…', 'Saving…') : editing ? text('保存修改', 'Save changes') : text('确认并创建', 'Create API key') }}</button></footer>
    </form>
  </PortalDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PortalDialog from './PortalDialog.vue'
import { keysAPI } from '@/api/keys'
import { isSmartRoutingGroup } from '@/utils/smartRouting'
import { getKeyGroupProvider, type KeyGroupProvider } from '@/utils/keyGroupProviders'
import type { ApiKey, Group } from '@/types'
const props = withDefaults(defineProps<{ open: boolean; editing: ApiKey | null; groups: Group[]; selectedGroup?: number; groupRates?: Record<number, number> }>(), { groupRates: () => ({}) })
const emit = defineEmits<{ close: []; saved: [key: ApiKey] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const saving = ref(false)
const error = ref('')
const search = ref('')
const category = ref<'all' | KeyGroupProvider>('all')
const routingGroups = computed(() => props.groups.filter(isSmartRoutingGroup))
const form = reactive({ name: '', group: '', smart: false, quota: 0, days: '' as number | string, expiry: '', limit5h: 0, limit1d: 0, limit7d: 0, allow: '', block: '', custom: '', routing: [] as number[], resetQuota: false, resetRates: false })
const eligibleGroups = computed(() => form.smart ? routingGroups.value : props.groups)
const selectedGroup = computed(() => props.groups.find(group => String(group.id) === form.group))
const providers = computed(() => [
  { value: 'all' as const, label: text('全部', 'All'), count: eligibleGroups.value.length },
  ...(['anthropic', 'openai', 'domestic', 'other'] as const).map(value => ({ value, label: ({ anthropic: 'Claude', openai: 'OpenAI', domestic: text('国产模型', 'Chinese models'), other: text('其他', 'Other') })[value], count: eligibleGroups.value.filter(group => getKeyGroupProvider(group.platform) === value).length })),
].filter(provider => provider.value === 'all' || provider.count > 0))
const filteredGroups = computed(() => eligibleGroups.value.filter(group => (category.value === 'all' || getKeyGroupProvider(group.platform) === category.value) && `${group.name} ${group.description || ''}`.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())))
const rateFields = computed(() => [ { key: 'limit5h' as const, label: text('5 小时限额', '5-hour limit') }, { key: 'limit1d' as const, label: text('每日限额', 'Daily limit') }, { key: 'limit7d' as const, label: text('7 日限额', '7-day limit') } ])
function rate(group: Group) { return props.groupRates[group.id] ?? group.rate_multiplier ?? 1 }
function hasUserRate(group: Group) { return props.groupRates[group.id] != null && props.groupRates[group.id] !== group.rate_multiplier }
function peakText(group: Group) { return text('高峰 ', 'Peak ') + `${group.peak_start || '—'}–${group.peak_end || '—'} · ${group.peak_rate_multiplier ?? 1}× ` + text('（服务器时间）', '(server time)') }
function localDate(iso: string | null) { if (!iso) return ''; const date = new Date(iso); return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0,16) }
watch(() => props.open, (open) => {
  if (!open) return
  const key = props.editing
  Object.assign(form, { name:key?.name || '', group:String(key?.group_id ?? props.selectedGroup ?? ''), smart:!!key?.routing_group_ids?.length, quota:key?.quota || 0, days:'', expiry:localDate(key?.expires_at ?? null), limit5h:key?.rate_limit_5h || 0, limit1d:key?.rate_limit_1d || 0, limit7d:key?.rate_limit_7d || 0, allow:key?.ip_whitelist?.join('\n') || '', block:key?.ip_blacklist?.join('\n') || '', custom:'', routing:[...(key?.routing_group_ids || [])], resetQuota:false, resetRates:false })
  error.value = ''; search.value = ''; category.value = 'all'
}, { immediate: true })
function isSelected(id: number) { return form.smart ? form.routing.includes(id) : form.group === String(id) }
function setMode(smart: boolean) {
  if (smart && !form.routing.length && routingGroups.value.some(group => String(group.id) === form.group)) form.routing = [Number(form.group)]
  form.smart = smart; category.value = 'all'; error.value = ''
}
function selectGroup(id: number) {
  if (form.smart) {
    if (form.routing.length < 10 && !form.routing.includes(id) && routingGroups.value.some(group => group.id === id)) form.routing.push(id)
  } else form.group = String(id)
  error.value = ''
}
function removeRouting(index: number) { form.routing.splice(index, 1) }
function moveRouting(index: number, direction: number) {
  const target = index + direction
  if (target < 0 || target >= form.routing.length) return
  const [id] = form.routing.splice(index, 1)
  form.routing.splice(target, 0, id)
}
async function submit() {
  if (saving.value || !form.name.trim()) return
  const routing = form.smart ? [...form.routing] : []
  if (form.smart && (!routing.length || routing.length > 10 || routing.some(id => !routingGroups.value.some(group => group.id === id)))) {
    error.value = text('请选择 1–10 个支持智能路由的分组。', 'Select 1–10 groups that support smart routing.')
    return
  }
  const groupId = routing[0] ?? (form.group ? Number(form.group) : null)
  if (groupId === null || !props.groups.some(group => group.id === groupId)) {
    error.value = text('请选择可用的接入分组。', 'Select an available connection group.')
    return
  }
  if (![form.quota, form.limit5h, form.limit1d, form.limit7d].every(value => Number.isFinite(Number(value)) && Number(value) >= 0) || (!props.editing && form.days !== '' && (!Number.isInteger(Number(form.days)) || Number(form.days) <= 0))) {
    error.value = text('请输入有效的额度与有效天数。', 'Enter valid limits and expiry days.')
    return
  }
  if (!props.editing && form.custom && !/^[A-Za-z0-9_-]{16,}$/.test(form.custom)) {
    error.value = text('自定义密钥至少16位，仅支持字母、数字、下划线和连字符。', 'Custom keys need at least 16 letters, numbers, underscores or hyphens.')
    return
  }
  saving.value = true; error.value = ''
  const lines = (value: string) => value.split(/[\n,]+/).map(v => v.trim()).filter(Boolean)
  const limits = { rate_limit_5h:Number(form.limit5h), rate_limit_1d:Number(form.limit1d), rate_limit_7d:Number(form.limit7d) }
  try {
    const key = props.editing
      ? await keysAPI.update(props.editing.id, { name:form.name.trim(), group_id:groupId, quota:Number(form.quota), expires_at:form.expiry ? new Date(form.expiry).toISOString() : '', ip_whitelist:lines(form.allow), ip_blacklist:lines(form.block), routing_group_ids:routing, ...limits, ...(form.resetQuota ? { reset_quota:true } : {}), ...(form.resetRates ? { reset_rate_limit_usage:true } : {}) })
      : await keysAPI.create(form.name.trim(), groupId, form.custom || undefined, lines(form.allow), lines(form.block), Number(form.quota), form.days ? Number(form.days) : undefined, limits, routing)
    emit('saved', key)
  } catch (e) { error.value = (e as {message?: string}).message || text('保存失败，请重试。', 'Could not save. Please retry.') }
  finally { saving.value = false }
}
</script>
<style scoped>
:global(dialog.portal-dialog.wide:has(.key-editor)) { width: min(1120px, calc(100vw - 48px)); max-width:1120px; padding:0; overflow:hidden; }
:global(dialog.portal-dialog:has(.key-editor) > .portal-dialog-header) { margin:0; padding:24px 28px; border-bottom:1px solid #efede3; }
.key-editor { color:#151513; background:#fffdf7; }
.key-editor-body { display:grid; grid-template-columns:1fr 1fr; max-height:calc(90dvh - 160px); overflow-y:auto; }
.key-picker,.key-configuration { min-width:0; padding:26px 28px; }
.key-picker { border-right:1px solid #efede3; }
.key-editor h3 { display:flex; align-items:center; gap:10px; margin:0 0 24px; font-size:16px; font-weight:600; }
.key-step { display:grid; place-items:center; width:27px; height:27px; border-radius:50%; background:#171715; color:#fffdf7; font-size:13px; font-weight:400; }
.key-mode { display:flex; padding:4px; margin-bottom:18px; border:1px solid #efede3; border-radius:10px; background:#f7f5ee; }
.key-mode button { flex:1; padding:10px; border:0; border-radius:7px; font:inherit; font-size:13px; color:#76736b; background:transparent; cursor:pointer; }
.key-mode button[aria-pressed=true] { background:#fffdf7; color:#111; box-shadow:0 1px 3px #0000000a; }
.key-search { display:flex; align-items:center; gap:10px; padding:0 13px; border:1px solid #e6e3d9; border-radius:8px; }
.key-search>span { font-size:25px; color:#76736b; }
.key-search input { width:100%; min-width:0; padding:13px 0; border:0; background:transparent; font:inherit; font-size:13px; outline:none; }
.key-search:focus-within { border-color:#151513; }
.key-providers { display:flex; flex-wrap:wrap; gap:7px; margin:16px 0; }
.key-providers button { border:1px solid #efede3; border-radius:6px; background:transparent; padding:5px 9px; color:#76736b; font:inherit; font-size:11px; cursor:pointer; }
.key-providers button[aria-pressed=true] { background:#f3f0eb; color:#151513; border-color:#b9b5aa; }
.key-providers span { margin-left:3px; opacity:.7; }
.key-group-list { display:grid; gap:10px; max-height:460px; overflow-y:auto; padding:2px 5px 2px 2px; }
.key-group-card { display:flex; align-items:flex-start; gap:11px; width:100%; padding:16px 13px; border:1px solid #e9e6dd; border-radius:10px; background:transparent; text-align:left; font:inherit; cursor:pointer; }
.key-group-card:hover { background:#faf8f1; }
.key-group-card.selected { border-color:#34342e; background:#f3f0eb; }
.key-group-card:disabled { cursor:default; }
.key-group-card:disabled:not(.selected) { opacity:.45; }
.key-radio { display:grid; place-items:center; flex-shrink:0; width:18px; height:18px; margin-top:1px; border:1px solid #c6c2b8; border-radius:50%; color:transparent; font-size:11px; }
.key-group-card.selected .key-radio { color:#fffdf7; background:#151513; border-color:#151513; }
.key-group-copy { display:grid; gap:7px; min-width:0; flex:1; }
.key-group-copy strong { font-size:13px; font-weight:600; overflow-wrap:anywhere; }
.key-description { color:#76736b; font-size:12px; line-height:1.65; white-space:pre-line; overflow-wrap:anywhere; }
.key-group-copy small,.key-group-rate small { color:#8d887e; font-size:10px; line-height:1.5; }
.key-group-rate { display:grid; flex-shrink:0; gap:5px; max-width:130px; text-align:right; font-size:11px; color:#76736b; }
.key-group-rate b { color:#151513; font-weight:500; }
.key-count { display:flex; justify-content:space-between; margin:16px 0 0; padding-top:14px; border-top:1px solid #efede3; color:#8d887e; font-size:11px; }
.key-editor h4 { margin:0 0 12px; font-size:13px; font-weight:500; }
.key-selected { margin-bottom:22px; }
.key-selected-summary { display:flex; flex-wrap:wrap; gap:8px 16px; padding:15px; border:1px solid #e9e6dd; border-radius:8px; background:#f7f5ee; font-size:12px; }
.key-selected-summary strong { flex:1; overflow-wrap:anywhere; font-weight:600; }
.key-selected-summary small { flex-basis:100%; color:#76736b; line-height:1.6; }
.key-selected-empty { margin:0 0 20px; padding:22px 16px; border:1px dashed #e6e3d9; border-radius:8px; color:#8d887e; font-size:12px; line-height:1.7; }
.key-common-fields { display:grid; grid-template-columns:1fr 1fr; gap:14px; }
.key-editor .portal-field { margin-bottom:18px; font-size:13px; gap:9px; }
.key-editor .portal-field input,.key-editor .portal-field textarea { width:100%; min-width:0; padding:11px 12px; border-radius:8px; font-size:13px; }
.key-editor .portal-field small { color:#8d887e; font-size:11px; line-height:1.5; }
.key-advanced { margin-top:4px; border-top:1px solid #efede3; }
.key-advanced summary { cursor:pointer; padding:16px 0; color:#76736b; font-size:12px; }
.key-rate-fields { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:10px; }
.key-rate-fields .portal-field { margin-bottom:0; }
.key-resets { display:grid; gap:10px; margin:10px 0 20px; font-size:12px; }
.key-resets label { display:flex; align-items:center; gap:8px; }
.key-resets input { accent-color:#151513; }
.key-resets small,.key-hint { color:#8d887e; font-size:11px; line-height:1.7; }
.key-hint { margin:8px 0 16px; }
.key-empty { padding:30px 10px; color:#8d887e; font-size:12px; text-align:center; }
.key-routing-heading { display:flex; align-items:baseline; justify-content:space-between; font-size:11px; color:#76736b; }
.key-routing-heading h4 { color:#151513; }
.routing-order { display:grid; gap:8px; padding:0; margin:0; list-style:none; }
.routing-order li { display:flex; align-items:center; gap:10px; padding:11px; border:1px solid #e9e6dd; border-radius:8px; }
.routing-order li.unavailable { border-color:#c88f49; }
.route-number { display:grid; place-items:center; width:22px; height:22px; flex-shrink:0; border-radius:50%; background:#f3f0eb; font-size:11px; }
.route-copy { display:grid; gap:5px; flex:1; min-width:0; }
.route-copy strong { overflow-wrap:anywhere; font-size:12px; font-weight:500; }
.route-copy small { color:#8d887e; font-size:10px; }
.route-actions { display:flex; flex-shrink:0; gap:2px; }
.route-actions button { width:27px; height:30px; border:0; border-radius:5px; background:transparent; color:#45443f; cursor:pointer; }
.route-actions button:hover { background:#f3f0eb; }
.route-actions button:disabled { opacity:.25; cursor:default; }
.key-editor-footer { display:flex; align-items:center; gap:10px; padding:18px 28px; border-top:1px solid #efede3; background:#fffdf7; }
.key-editor-footer>span { flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:12px; color:#76736b; }
.key-editor-footer .portal-button { font-size:12px; }
.key-editor button:focus-visible { outline:2px solid #151513; outline-offset:2px; }
@media(max-width:739px) { :global(dialog.portal-dialog.wide:has(.key-editor)) { width:calc(100vw - 20px); } :global(dialog.portal-dialog:has(.key-editor) > .portal-dialog-header) { padding:20px; } .key-editor-body { grid-template-columns:1fr; max-height:calc(90dvh - 154px); } .key-picker,.key-configuration { padding:22px 20px; } .key-picker { border-right:0; border-bottom:1px solid #efede3; } .key-group-list { max-height:300px; } .key-editor-footer { padding:15px 20px; } .key-editor-footer>span { display:none; } .key-editor-footer .portal-button:first-of-type { margin-left:auto; } }
@media(max-width:420px) { .key-common-fields { grid-template-columns:1fr; gap:0; } .key-group-card { flex-wrap:wrap; } .key-group-rate { flex-basis:100%; max-width:none; text-align:left; padding-left:29px; } .key-rate-fields { grid-template-columns:1fr; } }
</style>
