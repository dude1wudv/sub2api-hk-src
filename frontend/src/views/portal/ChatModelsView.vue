<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { chatAPI, type ChatCatalog, type ChatCatalogModel, type ChatPresetView } from '@/api/chat'
import { getAvailable } from '@/api/groups'
import type { Group } from '@/types'
import { useAppStore } from '@/stores'
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'

const { locale } = useI18n()
const app = useAppStore()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const groups = ref<Group[]>([])
const presets = ref<ChatPresetView[]>([])
const catalog = ref<ChatCatalog | null>(null)
const selectedGroup = ref<number | null>(null)
const search = ref('')
const view = ref<'saved' | 'catalog'>('saved')
const groupPickerOpen = ref(true)
const groupSearch = ref('')
const filteredGroups = computed(() => groups.value.filter(group => `${group.name} ${group.platform} ${group.id}`.toLocaleLowerCase().includes(groupSearch.value.trim().toLocaleLowerCase())))
const activeGroup = computed(() => groups.value.find(group => group.id === selectedGroup.value))
const loading = ref(false)
const catalogLoading = ref(false)
const saving = ref(false)
const error = ref('')
const catalogError = ref('')
const editingId = ref<string | null>(null)
const names = ref<Record<string, string>>({})
const editing = computed(() => presets.value.find(preset => preset.id === editingId.value))
let request: AbortController | undefined
const initialRequest = new AbortController()
let searchTimer: ReturnType<typeof setTimeout> | undefined
let disposed = false

const message = (cause: unknown) => cause instanceof Error ? cause.message : text('操作失败，请重试。', 'The operation failed. Please retry.')
async function loadPresets() {
  const value = await chatAPI.presets(initialRequest.signal)
  if (disposed) return
  presets.value = value
  names.value = Object.fromEntries(value.map(preset => [preset.id, preset.display_name]))
}
async function load() {
  loading.value = true
  error.value = ''
  try {
    const [available] = await Promise.all([getAvailable(), loadPresets()])
    if (!disposed) {
      groups.value = available
      if (!presets.value.length) view.value = 'catalog'
    }
  } catch (cause) { if (!disposed) error.value = message(cause) }
  finally { if (!disposed) loading.value = false }
}
async function loadCatalog() {
  request?.abort()
  catalog.value = null
  catalogError.value = ''
  catalogLoading.value = false
  const group = selectedGroup.value
  if (!group) return
  const current = new AbortController()
  request = current
  catalogLoading.value = true
  try {
    const result = await chatAPI.catalog(group, search.value, current.signal)
    if (!current.signal.aborted && !disposed) catalog.value = result
  } catch (cause) { if (!current.signal.aborted && !disposed) catalogError.value = message(cause) }
  finally { if (!current.signal.aborted && !disposed) catalogLoading.value = false }
}
function chooseGroup(id: number) {
  selectedGroup.value = id
  groupPickerOpen.value = false
}
watch(selectedGroup, () => { clearTimeout(searchTimer); void loadCatalog() })
watch(search, () => {
  clearTimeout(searchTimer)
  request?.abort()
  catalog.value = null
  searchTimer = setTimeout(() => void loadCatalog(), 250)
})
async function mutate(operation: () => Promise<unknown>) {
  if (saving.value) return
  saving.value = true
  try {
    await operation()
    await loadPresets()
    if (!disposed) app.showSuccess(text('聊天模型已更新', 'Chat models updated'))
  } catch (cause) { if (!disposed) app.showError(message(cause)) }
  finally { if (!disposed) saving.value = false }
}
async function add(model: ChatCatalogModel) {
  const groupId = catalog.value?.group.id
  if (!groupId || groupId !== selectedGroup.value || !model.available) return
  const id = editingId.value
  await mutate(async () => {
    if (id) await chatAPI.updatePreset(id, { group_id: groupId, model: model.id })
    else await chatAPI.createPreset({ group_id: groupId, model: model.id })
    editingId.value = null
  })
}
function editBinding(preset: ChatPresetView) {
  editingId.value = preset.id
  view.value = 'catalog'
  groupPickerOpen.value = true
  search.value = ''
  selectedGroup.value = groups.value.some(group => group.id === preset.group_id) ? preset.group_id : null
  void loadCatalog()
}
function move(index: number, direction: -1 | 1) {
  const ids = presets.value.map(preset => preset.id)
  const target = index + direction
  if (target < 0 || target >= ids.length) return
  ;[ids[index], ids[target]] = [ids[target]!, ids[index]!]
  void mutate(() => chatAPI.orderPresets(ids))
}
const alreadyAdded = (model: string) => presets.value.some(preset => preset.group_id === catalog.value?.group.id && preset.model === model)
const money = (value: number | null | undefined) => value == null ? '—' : new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD', maximumFractionDigits: 4 }).format(value)
function price(model: ChatCatalogModel) {
  const pricing = model.pricing
  const rate = catalog.value?.group.user_rate_multiplier ?? catalog.value?.group.rate_multiplier ?? 1
  if (!pricing) return text('价格暂不可用，以实际账单为准', 'Pricing unavailable; actual billing applies')
  if (pricing.billing_mode === 'per_request') return money(pricing.per_request_price == null ? null : pricing.per_request_price * rate) + text(' / 次', ' / request')
  const input = pricing.input_price == null ? null : pricing.input_price * 1_000_000 * rate
  const output = pricing.output_price == null ? null : pricing.output_price * 1_000_000 * rate
  return `${text('输入', 'Input')} ${money(input)} · ${text('输出', 'Output')} ${money(output)} ${text('/ 百万 token 起', '/ 1M tokens, from')}`
}
onMounted(load)
onBeforeUnmount(() => { disposed = true; clearTimeout(searchTimer); request?.abort(); initialRequest.abort() })
</script>

<template>
  <div class="portal-page chat-models-page">
    <PortalPageHeader :title="text('聊天模型', 'Chat models')">
      <RouterLink to="/chat" target="_blank" rel="noopener noreferrer" class="portal-button">{{ text('打开聊天', 'Open chat') }} <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 4h6v6M20 4l-9 9M10 4H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-5" /></svg></RouterLink>
    </PortalPageHeader>
    <p class="chat-description">{{ text('保存常用模型，在聊天中随时切换。每个模型固定使用你选择的分组。', 'Save models to switch between them in chat. Each model uses the group you choose.') }}</p>
    <nav class="portal-tabs chat-view-switch" :aria-label="text('模型配置视图', 'Model settings views')">
      <button type="button" data-view="saved" :class="{ active: view === 'saved' }" :aria-pressed="view === 'saved'" @click="view = 'saved'">{{ text('常用模型', 'Saved models') }}<span v-if="presets.length">{{ presets.length }}</span></button>
      <button type="button" data-view="catalog" :class="{ active: view === 'catalog' }" :aria-pressed="view === 'catalog'" @click="view = 'catalog'">{{ text('添加模型', 'Add models') }}</button>
    </nav>
    <p v-if="error" class="portal-error" role="alert">{{ error }} <button class="portal-button secondary" @click="load">{{ text('重试', 'Retry') }}</button></p>
    <p v-if="loading" role="status">{{ text('正在加载…', 'Loading…') }}</p>

    <section v-show="view === 'saved'" class="chat-saved" :aria-label="text('我的常用模型', 'My saved models')">
      <div v-if="!loading && !presets.length" class="chat-empty">
        <p>{{ text('还没有常用模型', 'No saved models yet') }}</p>
        <button class="portal-button secondary" @click="view = 'catalog'">{{ text('添加模型', 'Add models') }}</button>
      </div>
      <ol class="chat-model-list">
        <li v-for="(preset, index) in presets" :key="preset.id" :data-preset-id="preset.id" class="chat-saved-row">
          <details class="chat-preset-settings">
            <summary>
              <span class="chat-model-identity">
                <span class="chat-model-title"><strong>{{ preset.display_name || preset.model }}</strong><span v-if="preset.is_default" class="chat-default">{{ text('默认', 'Default') }}</span></span>
                <span v-if="preset.display_name && preset.display_name !== preset.model" class="chat-saved-code">{{ preset.model }}</span>
                <span class="chat-saved-group">{{ preset.group.name || text('不可访问的分组', 'Unavailable group') }} <span>#{{ preset.group_id }}</span></span>
                <span v-if="!preset.available" class="chat-unavailable" role="status">{{ preset.unavailable_reason }}</span>
              </span>
              <span class="chat-manage-label">{{ text('管理', 'Manage') }}<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></span>
            </summary>
            <div class="chat-preset-editor">
              <form class="chat-rename" @submit.prevent="mutate(() => chatAPI.updatePreset(preset.id, { display_name: names[preset.id] }))">
                <label class="portal-field" :for="`name-${preset.id}`">{{ text('显示名', 'Display name') }}<input :id="`name-${preset.id}`" v-model="names[preset.id]" maxlength="200" :disabled="saving" :placeholder="preset.model" /></label>
                <button class="portal-button" :disabled="saving || names[preset.id] === preset.display_name">{{ text('保存', 'Save') }}</button>
              </form>
              <div class="chat-model-actions">
                <div class="chat-binding-actions">
                  <button class="portal-button secondary" :disabled="saving || !preset.available || preset.is_default" @click="mutate(() => chatAPI.updatePreset(preset.id, { is_default: true }))">{{ text('设为默认', 'Set default') }}</button>
                  <button class="portal-button secondary" :disabled="saving" @click="editBinding(preset)">{{ text('修改绑定', 'Change binding') }}</button>
                  <button class="portal-button secondary" :disabled="saving || !preset.available" @click="mutate(() => chatAPI.reconnectPreset(preset.id))" :title="text('仅重建失效的聊天凭据，仍受普通密钥创建规则限制', 'Replace only an invalid chat credential, subject to normal key creation limits')">{{ text('重新连接', 'Reconnect') }}</button>
                </div>
                <div class="chat-order-actions">
                  <button class="portal-button secondary chat-icon-button" :disabled="saving || index === 0" :aria-label="text('上移', 'Move up') + ' ' + preset.model" @click="move(index, -1)"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 12 6-6 6 6M12 6v12" /></svg></button>
                  <button class="portal-button secondary chat-icon-button" :disabled="saving || index === presets.length - 1" :aria-label="text('下移', 'Move down') + ' ' + preset.model" @click="move(index, 1)"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 12 6 6 6-6M12 6v12" /></svg></button>
                  <button class="portal-button danger" :disabled="saving" @click="mutate(() => chatAPI.deletePreset(preset.id))">{{ text('移除', 'Remove') }}</button>
                </div>
              </div>
            </div>
          </details>
        </li>
      </ol>
      <p v-if="presets.length" class="chat-footnote">{{ text('修改或移除配置，不影响历史回答。', 'Changing or removing a model does not affect past replies.') }}</p>
    </section>

    <section v-show="view === 'catalog'" class="chat-browser" :aria-label="text('添加模型', 'Add models')">
      <aside class="chat-groups" aria-labelledby="chat-groups-title">
        <header class="chat-section-head"><h2 id="chat-groups-title">{{ text('分组', 'Groups') }}</h2><span>{{ groups.length }}</span></header>
        <button type="button" class="chat-mobile-group" :aria-expanded="groupPickerOpen" aria-controls="chat-group-options" @click="groupPickerOpen = !groupPickerOpen"><span>{{ activeGroup?.name || text('选择分组', 'Choose a group') }}</span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></button>
        <div id="chat-group-options" class="chat-group-options" :class="{ 'is-collapsed': !groupPickerOpen }">
          <label class="chat-search"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 4 4" /></svg><input v-model="groupSearch" type="search" :aria-label="text('搜索分组', 'Search groups')" :placeholder="text('搜索名称或 ID', 'Search name or ID')" maxlength="256" /></label>
          <div class="chat-group-list" :aria-label="text('可选分组', 'Available groups')">
            <button v-for="group in filteredGroups" :key="group.id" type="button" :data-group-id="group.id" :aria-pressed="selectedGroup === group.id" :disabled="loading || saving" :title="group.name" @click="chooseGroup(group.id)">
              <span class="chat-group-name">{{ group.name }}</span><span class="chat-group-meta">#{{ group.id }} · {{ group.platform }}<svg v-if="selectedGroup === group.id" viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6" /></svg></span>
            </button>
            <p v-if="!loading && !filteredGroups.length" class="chat-empty">{{ text('没有匹配的分组', 'No matching groups') }}</p>
          </div>
        </div>
      </aside>
      <div class="chat-catalog" aria-labelledby="model-catalog-title">
        <header class="chat-section-head"><h2 id="model-catalog-title">{{ activeGroup?.name || text('模型目录', 'Model catalog') }}</h2><span v-if="activeGroup">#{{ activeGroup.id }}</span></header>
        <div v-if="editing" class="chat-editing" role="status"><span>{{ text('修改绑定：', 'Rebind: ') }}<strong>{{ editing.display_name || editing.model }}</strong></span><button class="portal-button secondary" @click="editingId = null">{{ text('取消', 'Cancel') }}</button></div>
        <label class="chat-search"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 4 4" /></svg><input id="chat-model-search" v-model="search" type="search" :aria-label="text('搜索模型', 'Search models')" :placeholder="text('搜索此分组的模型', 'Search models in this group')" :disabled="!selectedGroup" maxlength="256" /></label>
        <div v-if="catalog" class="chat-catalog-meta"><span>{{ text('你的倍率', 'Your rate') }} ×{{ catalog.group.user_rate_multiplier }}</span><span>{{ catalog.group.subscription_type === 'subscription' ? text('套餐计费', 'Subscription billing') : text('余额计费', 'Balance billing') }} · {{ catalog.models.length }} {{ text('个模型', 'models') }}</span></div>
        <div class="chat-catalog-body" :aria-busy="catalogLoading">
          <p v-if="!selectedGroup" class="chat-empty">{{ text('选择分组，查看模型和价格。', 'Choose a group to see its models and prices.') }}</p>
          <p v-if="catalogLoading" class="chat-empty" role="status">{{ text('加载目录…', 'Loading catalog…') }}</p>
          <p v-if="catalogError" class="chat-empty" role="alert">{{ catalogError }} <button class="portal-button secondary" @click="loadCatalog">{{ text('重试', 'Retry') }}</button></p>
          <p v-if="catalog && !catalog.models.length" class="chat-empty">{{ text('没有符合条件的聊天模型。可调整搜索或选择其他分组。', 'No compatible chat models. Change the search or select another group.') }}</p>
          <ul class="chat-model-list">
            <li v-for="model in catalog?.models || []" :key="model.id" class="chat-catalog-row">
              <div class="chat-model-info">
                <strong class="chat-model-name">{{ model.id }}</strong>
                <p class="chat-price">{{ price(model) }}</p>
                <div class="chat-model-tags"><span>{{ model.capabilities.transport === 'messages' ? 'Messages' : model.capabilities.transport === 'gemini' ? 'Gemini v1beta' : model.capabilities.transport === 'responses' ? 'Responses' : 'Chat Completions' }}</span><span v-if="model.capabilities.image_input">{{ text('图片', 'Images') }}</span><span v-if="model.capabilities.document_input">{{ text('文档', 'Files') }}</span><span v-if="model.capabilities.reasoning">{{ text('思考', 'Reasoning') }}</span><span v-if="model.capabilities.search">{{ text('搜索', 'Search') }}</span></div>
                <p v-if="!model.available" class="chat-unavailable">{{ model.unavailable_reason }}</p>
              </div>
              <button class="portal-button secondary chat-add" :disabled="saving || !model.available || (!editingId && alreadyAdded(model.id))" @click="add(model)">{{ editingId ? text('绑定', 'Bind') : alreadyAdded(model.id) ? text('已添加', 'Added') : text('添加常用', 'Add model') }}</button>
            </li>
          </ul>
        </div>
        <details class="chat-pricing-note"><summary>{{ text('价格与可用性', 'Pricing and availability') }}</summary><p>{{ text('价格为标准时段最低档，已应用你的倍率。阶梯、时段、缓存及工具费用以实际账单为准。目录表示配置兼容性，不是实时健康保证。', 'Lowest standard-period prices with your rate applied. Tiers, time periods, cache and tools follow actual billing. Compatibility is not a live health guarantee.') }}</p></details>
      </div>
    </section>
  </div>
</template>

<style scoped>
.chat-models-page { min-width: 0; font-family: var(--font-sans); }
.chat-models-page svg { width: 18px; height: 18px; flex: none; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
.chat-description { margin: 8px 0 24px; color: rgb(var(--ink-2)); font-size: 14px; line-height: 1.6; }
.chat-view-switch button { display: flex; align-items: center; gap: 8px; font-family: inherit; font-size: 14px; }
.chat-view-switch span { color: rgb(var(--ink-3)); font-variant-numeric: tabular-nums; }
.chat-saved, .chat-browser { border: 1px solid rgb(var(--line)); border-radius: var(--radius-card); background: rgb(var(--surface)); }
.chat-model-list { list-style: none; margin: 0; padding: 0; }
.chat-saved-row + .chat-saved-row { border-top: 1px solid rgb(var(--line)); }
.chat-preset-settings > summary { display: flex; align-items: center; gap: 24px; padding: 18px 24px; cursor: pointer; list-style: none; }
.chat-preset-settings > summary::-webkit-details-marker { display: none; }
.chat-preset-settings > summary:hover { background: rgb(var(--table-hover)); }
.chat-model-identity { flex: 1; min-width: 0; display: grid; gap: 4px; overflow-wrap: anywhere; }
.chat-model-title { display: flex; align-items: baseline; flex-wrap: wrap; gap: 10px; }
.chat-model-title strong, .chat-model-name { font-size: 15px; font-weight: 600; line-height: 1.5; overflow-wrap: anywhere; }
.chat-default { font-size: 13px; color: rgb(var(--text-primary)); }
.chat-saved-group, .chat-saved-code { font-size: 14px; color: rgb(var(--ink-2)); }
.chat-saved-group > span { margin-left: 6px; color: rgb(var(--ink-3)); }
.chat-manage-label { display: flex; align-items: center; gap: 6px; flex: none; color: rgb(var(--ink-2)); font-size: 14px; }
.chat-preset-settings[open] .chat-manage-label svg { transform: rotate(180deg); }
.chat-preset-editor { padding: 0 24px 20px; }
.chat-rename { display: grid; grid-template-columns: minmax(0, 440px) auto; justify-content: start; align-items: end; gap: 12px; padding: 8px 0 20px; }
.chat-rename .portal-field { font-size: 14px; }
.chat-rename input { min-width: 0; font-family: inherit; }
.chat-model-actions, .chat-binding-actions, .chat-order-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.chat-model-actions { justify-content: space-between; gap: 16px; }
.chat-models-page .portal-button { font-size: 14px; box-shadow: none; }
.chat-icon-button { width: 40px; padding-inline: 0; }
.chat-footnote { margin: 0; padding: 14px 24px; border-top: 1px solid rgb(var(--line)); color: rgb(var(--ink-3)); font-size: 13px; }
.chat-browser { display: grid; grid-template-columns: 260px minmax(0, 1fr); align-items: start; }
.chat-groups { min-width: 0; padding: 20px 12px; }
.chat-catalog { min-width: 0; min-height: 440px; padding: 20px 24px; border-left: 1px solid rgb(var(--line)); }
.chat-section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
.chat-section-head h2 { margin: 0; font-size: 15px; font-weight: 600; line-height: 1.5; overflow-wrap: anywhere; }
.chat-section-head > span { flex: none; font-size: 13px; color: rgb(var(--ink-3)); }
.chat-groups .chat-section-head { padding-inline: 8px; }
.chat-search { display: flex; align-items: center; gap: 8px; min-height: 40px; padding: 8px 12px; border: 1px solid rgb(var(--line)); border-radius: var(--radius-control); color: rgb(var(--ink-3)); }
.chat-search:focus-within { border-color: rgb(var(--focus-ring)); box-shadow: 0 0 0 3px rgb(var(--focus-ring) / .14); }
.chat-search input { width: 100%; min-width: 0; border: 0; outline: 0; background: transparent; color: rgb(var(--ink)); font: inherit; font-size: 14px; }
.chat-group-list { max-height: min(620px, 65dvh); margin-top: 12px; overflow-y: auto; overscroll-behavior: contain; scrollbar-width: thin; }
.chat-group-list button { display: grid; gap: 4px; width: 100%; margin: 2px 0; padding: 10px 12px; border: 0; border-radius: 8px; background: transparent; color: rgb(var(--ink)); font: inherit; text-align: left; cursor: pointer; }
.chat-group-list button:hover { background: rgb(var(--surface-2)); }
.chat-group-list button[aria-pressed=true] { background: rgb(var(--selection-bg)); }
.chat-group-name { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; font-size: 14px; font-weight: 500; line-height: 1.5; overflow-wrap: anywhere; }
.chat-group-meta { display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 13px; color: rgb(var(--ink-3)); }
.chat-group-meta svg { color: rgb(var(--text-primary)); }
.chat-mobile-group { display: none; }
.chat-catalog-meta { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 8px; margin: 14px 0; color: rgb(var(--ink-2)); font-size: 13px; }
.chat-catalog-row { display: flex; align-items: center; gap: 20px; padding: 18px 0; border-top: 1px solid rgb(var(--line)); }
.chat-model-info { flex: 1; min-width: 0; }
.chat-price { margin: 5px 0 0; color: rgb(var(--ink-2)); font-size: 14px; font-variant-numeric: tabular-nums; line-height: 1.6; }
.chat-model-tags { display: flex; flex-wrap: wrap; gap: 4px 12px; margin-top: 6px; color: rgb(var(--ink-3)); font-size: 13px; }
.chat-add { flex: none; }
.chat-pricing-note { padding-top: 16px; border-top: 1px solid rgb(var(--line)); color: rgb(var(--ink-3)); font-size: 13px; }
.chat-pricing-note summary { cursor: pointer; width: fit-content; }
.chat-pricing-note p { margin: 8px 0 0; line-height: 1.7; }
.chat-unavailable { display: block; margin: 6px 0 0; color: rgb(var(--text-err)); font-size: 14px; }
.chat-empty { margin: 0; padding: 40px 20px; color: rgb(var(--ink-2)); font-size: 14px; line-height: 1.7; text-align: center; }
.chat-empty p { margin: 0 0 16px; }
.chat-editing { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-bottom: 16px; font-size: 14px; overflow-wrap: anywhere; }
@media (max-width: 1023px) {
  .chat-browser { grid-template-columns: 220px minmax(0, 1fr); }
  .chat-catalog { padding: 20px; }
}
@media (max-width: 767px) {
  .chat-description { margin-bottom: 20px; }
  .chat-view-switch { width: 100%; }
  .chat-view-switch button { flex: 1; justify-content: center; min-height: 44px; }
  .chat-browser { grid-template-columns: minmax(0, 1fr); }
  .chat-groups { padding: 16px; border-bottom: 1px solid rgb(var(--line)); }
  .chat-groups .chat-section-head { display: none; }
  .chat-mobile-group { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; min-height: 44px; padding: 10px 12px; border: 1px solid rgb(var(--line)); border-radius: var(--radius-control); background: rgb(var(--surface)); color: rgb(var(--ink)); font: inherit; text-align: left; }
  .chat-mobile-group > span { min-width: 0; overflow-wrap: anywhere; }
  .chat-group-options { margin-top: 12px; }
  .chat-group-options.is-collapsed { display: none; }
  .chat-group-list { max-height: 280px; }
  .chat-catalog { min-height: 0; padding: 16px; border-left: 0; }
  .chat-catalog-row { flex-wrap: wrap; gap: 12px; padding: 16px 0; }
  .chat-model-info { flex-basis: 100%; }
  .chat-add { margin-left: auto; }
  .chat-preset-settings > summary { gap: 12px; padding: 16px; }
  .chat-manage-label { gap: 2px; }
  .chat-preset-editor { padding: 0 16px 16px; }
  .chat-rename { grid-template-columns: minmax(0, 1fr) auto; gap: 8px; }
  .chat-binding-actions { width: 100%; }
  .chat-footnote { padding: 14px 16px; }
  .chat-models-page .portal-button { min-height: 44px; }
  .chat-icon-button { width: 44px; }
  .chat-search input { font-size: 16px; }
}
@media (pointer: coarse) {
  .chat-models-page .portal-button, .chat-pricing-note summary { min-height: 44px; }
  .chat-icon-button { width: 44px; }
}
</style>
