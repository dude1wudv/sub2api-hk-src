<template>
  <PortalDialog :open="open" :title="editing ? text('编辑 API Key', 'Edit API key') : text('创建 API Key', 'Create API key')" wide @close="!saving && $emit('close')">
    <form class="key-editor" @submit.prevent="submit">
      <div class="key-editor-body">
        <section class="key-picker" data-tour="key-form-group" :aria-label="text('选择分组', 'Choose a group')">
          <h3><span class="key-step">1</span>{{ text('选择分组', 'Choose a group') }}</h3>
          <div class="key-mode" role="group" :aria-label="text('接入模式', 'Connection mode')">
            <button type="button" :aria-pressed="!form.smart" @click="setMode(false)">{{ text('固定分组', 'Fixed group') }}</button>
            <button type="button" :aria-pressed="form.smart" @click="setMode(true)">{{ text('智能路由', 'Smart routing') }}</button>
          </div>
          <p v-if="form.smart" class="key-hint">{{ text('点击分组添加候选，再在右侧调整顺序。最多 10 个，按实际命中的分组计费。', 'Add candidate groups, then arrange their order on the right. Up to 10; billing uses the group that serves the request.') }}</p>
          <label v-if="eligibleGroups.length > 1" class="key-search"><Icon name="search" size="sm" aria-hidden="true" /><input v-model="search" type="search" :aria-label="text('搜索分组', 'Search groups')" :placeholder="text('搜索分组名称或说明', 'Search group names or descriptions')" /></label>
          <div v-if="eligibleGroups.length > 1" class="key-providers" role="group" :aria-label="text('分组类别', 'Group categories')"><button v-for="provider in providers" :key="provider.value" type="button" :aria-pressed="category === provider.value" @click="category = provider.value">{{ provider.label }} <span>{{ provider.count }}</span></button></div>
          <div class="key-group-list">
            <button v-for="group in filteredGroups" :key="group.id" type="button" class="key-group-card" :class="{ selected: isSelected(group.id) }" :aria-pressed="isSelected(group.id)" :disabled="form.smart && (form.routing.includes(group.id) || form.routing.length >= 10)" :data-group-id="group.id" @click="selectGroup(group.id)">
              <span class="key-radio" aria-hidden="true">{{ isSelected(group.id) ? '✓' : '+' }}</span>
              <span class="key-group-copy"><strong>{{ group.name }} <span v-if="group.is_new" class="key-new">NEW</span></strong><span v-if="group.description" class="key-description">{{ group.description }}</span><small v-if="group.subscription_type === 'subscription'">{{ text('订阅分组', 'Subscription group') }}</small></span>
              <span class="key-group-rate"><span>{{ text('分组倍率', 'Group rate') }} <b>{{ rate(group) }}×</b></span><small v-if="hasUserRate(group)">{{ text('专属倍率', 'Your rate') }}</small><small v-if="group.peak_rate_enabled">{{ peakText(group) }}</small></span>
            </button>
            <p v-if="!filteredGroups.length" class="key-empty">{{ text('没有匹配的可用分组', 'No matching groups') }}</p>
          </div>
          <div v-for="group in filteredGroups.filter(item => item.is_new || (item.recent_rate_change && item.subscription_type !== 'subscription'))" :key="'rate-' + group.id" class="key-rate-note"><span>{{ group.name }}</span> <GroupChangeIndicator :group="group" /></div>
          <p class="key-count">{{ text('显示 ', 'Showing ') }}{{ filteredGroups.length }} / {{ eligibleGroups.length }}<span v-if="form.smart">{{ text('已添加 ', 'Added ') }}{{ form.routing.length }} / 10</span></p>
        </section>
        <section class="key-configuration" :aria-label="text('配置密钥', 'Configure key')">
          <h3><span class="key-step">2</span>{{ text('配置 API Key', 'Configure API key') }}</h3>
          <p v-if="error" class="portal-error" role="alert">{{ error }}</p>
          <div v-if="form.smart" class="key-routing">
            <div class="key-routing-heading"><h4>{{ text('候选分组顺序', 'Routing order') }}</h4><span>{{ form.routing.length }} / 10</span></div>
            <ol v-if="form.routing.length" class="routing-order">
              <li v-for="(id, index) in form.routing" :key="id" :class="{ unavailable: !routingGroups.some(group => group.id === id) }">
                <span class="route-number">{{ index + 1 }}</span><div class="route-copy"><strong>{{ groups.find(group => group.id === id)?.name || `#${id}` }}</strong><small v-if="!routingGroups.some(group => group.id === id)">{{ text('不可用，请移除或替换', 'Unavailable — remove or replace') }}</small><small v-else-if="index === 0">{{ text('同分时首选', 'First when tied') }}</small><small v-else>{{ text('回退候选', 'Fallback group') }}</small></div>
                <div class="route-actions"><button type="button" :disabled="index === 0" :aria-label="text(`上移第${index + 1}个分组`, `Move group ${index + 1} up`)" @click="moveRouting(index, -1)">↑</button><button type="button" :disabled="index === form.routing.length - 1" :aria-label="text(`下移第${index + 1}个分组`, `Move group ${index + 1} down`)" @click="moveRouting(index, 1)">↓</button><button type="button" :aria-label="text(`移除第${index + 1}个分组`, `Remove group ${index + 1}`)" @click="removeRouting(index)">×</button></div>
              </li>
            </ol>
            <p v-else class="key-selected-empty">{{ text('从左侧添加路由候选分组。', 'Add routing candidates from the group list.') }}</p>
            <p class="key-hint">{{ text('候选会根据本站成功率、缓存率和首字延迟排序；同分时使用此处顺序。', 'Candidates are ranked by local success rate, cache rate and first-token latency. This order breaks ties.') }}</p>
          </div>
          <div v-else class="key-selected"><h4>{{ text('所选分组', 'Selected group') }}</h4><div v-if="selectedGroup" class="key-selected-summary"><strong>{{ selectedGroup.name }}</strong><span>{{ text('分组倍率 ', 'Group rate ') }}{{ rate(selectedGroup) }}×</span><small v-if="selectedGroup.description">{{ selectedGroup.description }}</small><small v-if="selectedGroup.peak_rate_enabled">{{ peakText(selectedGroup) }}</small></div><p v-else class="key-selected-empty">{{ text('从左侧选择分组，查看分组倍率与说明。', 'Choose a group to view its rate and description.') }}</p></div>
          <label class="portal-field">{{ text('名称', 'Name') }}<input v-model="form.name" required maxlength="100" :placeholder="text('为密钥起一个便于识别的名称', 'A memorable name for this key')" data-test="key-name" data-tour="key-form-name" /></label>
          <div class="key-common-fields">
            <label v-if="!editing" class="portal-field">{{ text('有效期', 'Expiry') }}<input v-model.number="form.days" type="number" min="1" step="1" :placeholder="text('永久有效', 'Never expires')" data-test="key-days" /><small>{{ text('有效天数，留空为永久', 'Days until expiry; empty means never') }}</small></label>
            <label v-else class="portal-field">{{ text('有效期', 'Expiry') }}<input v-model="form.expiry" type="datetime-local" /><small>{{ text('留空为永久有效', 'Empty means never expires') }}</small></label>
            <label class="portal-field">{{ text('额度（USD）', 'Quota (USD)') }}<input v-model.number="form.quota" type="number" min="0" step="0.01" required data-test="key-quota" /><small>{{ text('0 为不限额度', '0 means unlimited') }}</small></label>
          </div>
          <label class="portal-field">{{ text('Key 并发上限', 'Key concurrency limit') }}<input v-model.number="form.concurrency" type="number" min="0" step="1" required data-test="key-concurrency" /><small>{{ text('0 为不额外限制，仍受用户和账号并发限制。', '0 adds no key limit; user and account limits still apply.') }}</small></label>
          <details class="key-advanced"><summary>{{ text('高级设置', 'Advanced settings') }}</summary>
            <label class="portal-field">{{ text('Anthropic 缓存时长', 'Anthropic cache duration') }}<select v-model="form.cacheTtl" data-test="anthropic-cache-ttl"><option value="inherit">{{ text('跟随账号与全局设置', 'Follow account and global settings') }}</option><option value="off">{{ text('关闭覆盖', 'Disable override') }}</option><option value="5m">{{ text('强制 5 分钟', 'Force 5 minutes') }}</option><option value="1h">{{ text('强制 1 小时', 'Force 1 hour') }}</option></select><small>{{ text('仅对原生 Anthropic 生效（不包括 Bedrock）；关闭覆盖时按上游返回的用量计费。', 'Applies only to native Anthropic, excluding Bedrock. Disabling the override uses upstream usage.') }}</small></label>
            <div class="key-rate-fields"><label v-for="field in rateFields" :key="field.key" class="portal-field">{{ field.label }}<input v-model.number="form[field.key]" type="number" min="0" step="0.01" required /></label></div><p class="key-hint">{{ text('窗口限额按 USD 计算，0 为不限。', 'Window limits are in USD; 0 means unlimited.') }}</p>
            <div v-if="editing" class="key-resets"><label><input v-model="form.resetQuota" type="checkbox" />{{ text('重置已用额度', 'Reset used quota') }}</label><label><input v-model="form.resetRates" type="checkbox" />{{ text('重置限流计数', 'Reset rate-limit usage') }}</label><small>{{ text('仅勾选后保存才会重置。', 'Resets apply only when selected and saved.') }}</small></div>
            <label class="portal-field">{{ text('IP 白名单（每行一个 IP 或 CIDR）', 'IP allowlist (one IP/CIDR per line)') }}<textarea v-model="form.allow" rows="2" /></label>
            <label class="portal-field">{{ text('IP 黑名单（每行一个 IP 或 CIDR）', 'IP blocklist (one IP/CIDR per line)') }}<textarea v-model="form.block" rows="2" /></label>
            <label v-if="!editing" class="portal-field">{{ text('自定义密钥（可选）', 'Custom key (optional)') }}<input v-model="form.custom" type="password" autocomplete="off" /><small>{{ text('至少16位，仅支持字母、数字、下划线和连字符。', 'At least 16 characters: letters, numbers, underscores or hyphens.') }}</small></label>
          </details>
        </section>
      </div>
      <footer class="key-editor-footer"><span aria-live="polite">{{ form.smart ? text(`已选择 ${form.routing.length} 个路由分组`, `${form.routing.length} routing groups selected`) : selectedGroup?.name || text('请选择分组', 'Select a group') }}</span><button type="button" class="portal-button secondary" :disabled="saving" @click="$emit('close')">{{ text('取消', 'Cancel') }}</button><button class="portal-button" data-tour="key-form-submit" :disabled="saving">{{ saving ? text('保存中…', 'Saving…') : editing ? text('保存修改', 'Save changes') : text('确认并创建', 'Create API key') }}</button></footer>
    </form>
  </PortalDialog>
</template>
<script setup lang="ts">
import { userFacingError } from '@/utils/userFacingError'
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PortalDialog from './PortalDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupChangeIndicator from '@/components/common/GroupChangeIndicator.vue'
import { keysAPI } from '@/api/keys'
import { useOnboardingStore } from '@/stores/onboarding'
import { isSmartRoutingGroup } from '@/utils/smartRouting'
import { getKeyGroupProvider, type KeyGroupProvider } from '@/utils/keyGroupProviders'
import type { ApiKey, Group } from '@/types'
const props = withDefaults(defineProps<{ open: boolean; editing: ApiKey | null; groups: Group[]; selectedGroup?: number; groupRates?: Record<number, number> }>(), { groupRates: () => ({}) })
const emit = defineEmits<{ close: []; saved: [key: ApiKey] }>()
const { locale } = useI18n()
const onboardingStore = useOnboardingStore()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const saving = ref(false)
const error = ref('')
const search = ref('')
const category = ref<'all' | KeyGroupProvider>('all')
const routingGroups = computed(() => props.groups.filter(isSmartRoutingGroup))
const form = reactive({ name: '', group: '', smart: false, quota: 0, concurrency: 0, cacheTtl: 'inherit' as NonNullable<ApiKey['anthropic_cache_ttl_mode']>, days: '' as number | string, expiry: '', limit5h: 0, limit1d: 0, limit7d: 0, allow: '', block: '', custom: '', routing: [] as number[], resetQuota: false, resetRates: false })
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
  Object.assign(form, { name:key?.name || '', group:String(key?.group_id ?? props.selectedGroup ?? ''), smart:!!key?.routing_group_ids?.length, quota:key?.quota || 0, concurrency:key?.concurrency_limit ?? 0, cacheTtl:key?.anthropic_cache_ttl_mode ?? 'inherit', days:'', expiry:localDate(key?.expires_at ?? null), limit5h:key?.rate_limit_5h || 0, limit1d:key?.rate_limit_1d || 0, limit7d:key?.rate_limit_7d || 0, allow:key?.ip_whitelist?.join('\n') || '', block:key?.ip_blacklist?.join('\n') || '', custom:'', routing:[...(key?.routing_group_ids || [])], resetQuota:false, resetRates:false })
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
  if (!Number.isInteger(Number(form.concurrency)) || Number(form.concurrency) < 0) { error.value = text('并发上限必须是非负整数。', 'Concurrency limit must be a nonnegative integer.'); return }
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
      ? await keysAPI.update(props.editing.id, { name:form.name.trim(), group_id:groupId, quota:Number(form.quota), expires_at:form.expiry ? new Date(form.expiry).toISOString() : '', ip_whitelist:lines(form.allow), ip_blacklist:lines(form.block), routing_group_ids:routing, concurrency_limit:Number(form.concurrency), ...(form.cacheTtl !== (props.editing.anthropic_cache_ttl_mode ?? 'inherit') ? { anthropic_cache_ttl_mode:form.cacheTtl } : {}), ...limits, ...(form.resetQuota ? { reset_quota:true } : {}), ...(form.resetRates ? { reset_rate_limit_usage:true } : {}) })
      : await keysAPI.create(form.name.trim(), groupId, form.custom || undefined, lines(form.allow), lines(form.block), Number(form.quota), form.days ? Number(form.days) : undefined, limits, routing, Number(form.concurrency), form.cacheTtl)
    emit('saved', key)
    if (!props.editing && onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) void onboardingStore.nextStep(500)
  } catch (e) { error.value = userFacingError(e, locale.value, text('保存失败，请重试。', 'Could not save. Please retry.')) }
  finally { saving.value = false }
}
</script>
<style scoped>
:global(dialog.portal-dialog.wide:has(.key-editor)) { width: min(1120px, calc(100vw - 48px)); max-width:1120px; padding:0; overflow:hidden; }
:global(dialog.portal-dialog:has(.key-editor) > .portal-dialog-header) { margin:0; padding:22px 28px; border-bottom:1px solid rgb(var(--line)); }
.key-editor { color:rgb(var(--ink)); background:rgb(var(--surface-elevated)); }
.key-editor-body { display:grid; grid-template-columns:1fr 1fr; max-height:calc(90dvh - 160px); overflow-y:auto; }
.key-picker,.key-configuration { min-width:0; padding:24px 28px; }
.key-picker { border-right:1px solid rgb(var(--line)); background:rgb(var(--surface-sidebar) / .6); }
.key-editor h3 { display:flex; align-items:center; gap:10px; margin:0 0 20px; font-size:15.5px; font-weight:650; letter-spacing:-0.01em; }
.key-step { display:grid; place-items:center; width:25px; height:25px; border-radius:7px; background:rgb(var(--brand-moss) / .14); color:rgb(var(--brand-moss)); font-family:var(--font-mono); font-size:12.5px; font-weight:650; }
.key-mode { display:flex; padding:4px; margin-bottom:16px; border:1px solid rgb(var(--line)); border-radius:10px; background:rgb(var(--surface-2)); }
.key-mode button { flex:1; padding:8px 10px; border:0; border-radius:7px; font:inherit; font-size:13px; font-weight:500; color:rgb(var(--ink-2)); background:transparent; cursor:pointer; transition:background-color var(--motion-fast) ease, color var(--motion-fast) ease; }
.key-mode button[aria-pressed=true] { background:rgb(var(--surface)); color:rgb(var(--ink)); font-weight:600; box-shadow:0 1px 3px rgb(15 23 42 / .08); }
.key-search { display:flex; align-items:center; gap:9px; min-height:40px; padding:0 12px; border:1px solid rgb(var(--line)); border-radius:var(--radius-control); background:rgb(var(--surface)); color:rgb(var(--ink-3)); transition:border-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease; }
.key-search input { width:100%; min-width:0; padding:9px 0; border:0; background:transparent; color:rgb(var(--ink)); font:inherit; font-size:13.5px; outline:none; }
.key-search:focus-within { border-color:rgb(var(--focus-ring)); box-shadow:0 0 0 3px rgb(var(--focus-ring) / .14); }
.key-providers { display:flex; flex-wrap:wrap; gap:6px; margin:14px 0; }
.key-providers button { border:1px solid rgb(var(--line)); border-radius:8px; background:rgb(var(--surface)); padding:5px 10px; color:rgb(var(--ink-2)); font:inherit; font-size:12.5px; font-weight:500; cursor:pointer; transition:border-color var(--motion-fast) ease, background-color var(--motion-fast) ease; }
.key-providers button:hover { border-color:rgb(var(--line-strong)); color:rgb(var(--ink)); }
.key-providers button[aria-pressed=true] { background:rgb(var(--brand-moss) / .1); color:rgb(var(--ink)); border-color:rgb(var(--brand-moss)); font-weight:600; }
.key-providers span { margin-left:3px; opacity:.72; font-variant-numeric:tabular-nums; }
.key-group-list { display:grid; gap:9px; max-height:460px; overflow-y:auto; padding:2px 5px 2px 2px; }
.key-group-card { display:flex; align-items:flex-start; gap:11px; width:100%; padding:14px 13px; border:1px solid rgb(var(--line)); border-radius:11px; background:rgb(var(--surface)); text-align:left; font:inherit; cursor:pointer; transition:border-color var(--motion-fast) ease, background-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease; }
.key-group-card:hover:not(:disabled) { border-color:rgb(var(--line-strong)); background:rgb(var(--surface-2) / .5); }
.key-group-card.selected { border-color:rgb(var(--brand-moss)); background:rgb(var(--brand-moss) / .08); box-shadow:0 0 0 1px rgb(var(--brand-moss) / .22); }
.key-group-card:disabled { cursor:default; }
.key-group-card:disabled:not(.selected) { opacity:.45; }
.key-radio { display:grid; place-items:center; flex-shrink:0; width:18px; height:18px; margin-top:2px; border:1px solid rgb(var(--line-strong)); border-radius:50%; color:rgb(var(--ink-3)); font-size:11px; font-weight:700; }
.key-group-card.selected .key-radio { color:#fff; background:rgb(var(--brand-moss)); border-color:rgb(var(--brand-moss)); }
.key-group-copy { display:grid; gap:5px; min-width:0; flex:1; }
.key-group-copy strong { font-size:13.5px; font-weight:600; overflow-wrap:anywhere; }
.key-new { margin-left:5px; border-radius:999px; padding:2px 7px; background:rgb(var(--brand-moss) / .14); color:rgb(var(--brand-moss)); font-size:10.5px; font-weight:650; }
.key-rate-note { display:flex; justify-content:space-between; gap:12px; padding:6px 3px; color:rgb(var(--ink-3)); font-size:12px; }
.key-description { color:rgb(var(--ink-2)); font-size:12.5px; line-height:1.6; white-space:pre-line; overflow-wrap:anywhere; }
.key-group-copy small,.key-group-rate small { color:rgb(var(--ink-3)); font-size:12px; line-height:1.5; }
.key-group-rate { display:grid; flex-shrink:0; gap:4px; max-width:130px; text-align:right; font-size:12.5px; color:rgb(var(--ink-3)); font-variant-numeric:tabular-nums; }
.key-group-rate b { color:rgb(var(--ink)); font-weight:650; }
.key-count { display:flex; justify-content:space-between; margin:16px 0 0; padding-top:14px; border-top:1px solid rgb(var(--line)); color:rgb(var(--ink-3)); font-size:12.5px; font-variant-numeric:tabular-nums; }
.key-editor h4 { margin:0 0 10px; font-size:13px; font-weight:600; color:rgb(var(--ink-2)); }
.key-selected { margin-bottom:20px; }
.key-selected-summary { display:flex; flex-wrap:wrap; gap:6px 16px; padding:14px 16px; border:1px solid rgb(var(--brand-moss) / .28); border-radius:10px; background:rgb(var(--brand-moss) / .06); font-size:13px; }
.key-selected-summary strong { flex:1; overflow-wrap:anywhere; font-weight:650; }
.key-selected-summary small { flex-basis:100%; color:rgb(var(--ink-2)); line-height:1.6; white-space:pre-line; overflow-wrap:anywhere; }
.key-selected-empty { margin:0 0 20px; padding:20px 16px; border:1px dashed rgb(var(--line-strong) / .7); border-radius:10px; color:rgb(var(--ink-3)); font-size:13px; line-height:1.65; background:rgb(var(--surface-2) / .35); }
.key-common-fields { display:grid; grid-template-columns:1fr 1fr; gap:14px; }
.key-editor .portal-field { margin-bottom:16px; font-size:13px; gap:7px; }
.key-editor .portal-field input,.key-editor .portal-field textarea { width:100%; min-width:0; padding:9px 12px; border-radius:var(--radius-control); font-size:13.5px; }
.key-editor .portal-field small { color:rgb(var(--ink-3)); font-size:12.5px; line-height:1.5; }
.key-advanced { margin-top:4px; border-top:1px solid rgb(var(--line)); }
.key-advanced summary { cursor:pointer; padding:14px 0; color:rgb(var(--ink-2)); font-size:13px; font-weight:550; }
.key-rate-fields { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:10px; }
.key-rate-fields .portal-field { margin-bottom:0; }
.key-resets { display:grid; gap:10px; margin:10px 0 20px; font-size:13px; }
.key-resets label { display:flex; align-items:center; gap:8px; }
.key-resets input { accent-color:rgb(var(--brand-moss)); }
.key-resets small,.key-hint { color:rgb(var(--ink-3)); font-size:12.5px; line-height:1.65; }
.key-hint { margin:8px 0 16px; }
.key-empty { padding:30px 10px; color:rgb(var(--ink-3)); font-size:13px; text-align:center; }
.key-routing-heading { display:flex; align-items:baseline; justify-content:space-between; font-size:13px; color:rgb(var(--ink-3)); font-variant-numeric:tabular-nums; }
.key-routing-heading h4 { color:rgb(var(--ink)); }
.routing-order { display:grid; gap:8px; padding:0; margin:0; list-style:none; }
.routing-order li { display:flex; align-items:center; gap:10px; padding:10px 12px; border:1px solid rgb(var(--line)); border-radius:10px; background:rgb(var(--surface-2) / .45); }
.routing-order li.unavailable { border-color:rgb(var(--text-err) / .35); background:rgb(var(--status-err-bg) / .4); }
.route-number { display:grid; place-items:center; width:22px; height:22px; flex-shrink:0; border-radius:6px; background:rgb(var(--brand-moss) / .14); color:rgb(var(--brand-moss)); font-family:var(--font-mono); font-size:12px; font-weight:650; }
.route-copy { display:grid; gap:3px; flex:1; min-width:0; }
.route-copy strong { overflow-wrap:anywhere; font-size:13px; font-weight:600; }
.route-copy small { color:rgb(var(--ink-3)); font-size:12px; }
.route-actions { display:flex; flex-shrink:0; gap:2px; }
.route-actions button { width:27px; height:28px; border:0; border-radius:6px; background:transparent; color:rgb(var(--ink-2)); cursor:pointer; }
.route-actions button:hover:not(:disabled) { background:rgb(var(--surface)); color:rgb(var(--ink)); }
.route-actions button:disabled { opacity:.25; cursor:default; }
.key-editor-footer { display:flex; align-items:center; gap:10px; padding:16px 28px; border-top:1px solid rgb(var(--line)); background:rgb(var(--surface)); }
.key-editor-footer>span { flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:13px; color:rgb(var(--ink-2)); }
.key-editor-footer .portal-button { font-size:13.5px; }
.key-editor button:focus-visible { outline:2px solid rgb(var(--focus-ring)); outline-offset:2px; }
@media(max-width:767px) { :global(dialog.portal-dialog.wide:has(.key-editor)) { width:calc(100vw - 20px); } :global(dialog.portal-dialog:has(.key-editor) > .portal-dialog-header) { padding:20px; } .key-editor-body { grid-template-columns:1fr; max-height:calc(90dvh - 154px); } .key-picker,.key-configuration { padding:20px; } .key-picker { border-right:0; border-bottom:1px solid rgb(var(--line)); } .key-group-list { max-height:300px; } .key-editor-footer { padding:15px 20px; } .key-editor-footer>span { display:none; } .key-editor-footer .portal-button:first-of-type { margin-left:auto; } }
@media(max-width:420px) { .key-common-fields { grid-template-columns:1fr; gap:0; } .key-group-card { flex-wrap:wrap; } .key-group-rate { flex-basis:100%; max-width:none; text-align:left; padding-left:29px; } .key-rate-fields { grid-template-columns:1fr; } }
</style>
