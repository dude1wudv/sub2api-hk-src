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
    if (!disposed) groups.value = available
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
      <RouterLink to="/chat" class="portal-button">{{ text('进入聊天', 'Open chat') }} →</RouterLink>
    </PortalPageHeader>
    <p class="chat-description">{{ text('先明确选择分组和模型。聊天页只显示你的常用项，不会自动跨组切换；删除或修改配置不会改写历史回答。', 'Choose a group and model explicitly. Chat shows only your saved models and never switches groups automatically. Changes do not rewrite historical replies.') }}</p>
    <p v-if="error" role="alert">{{ error }} <button class="portal-button" @click="load">{{ text('重试', 'Retry') }}</button></p>
    <p v-if="loading" role="status">{{ text('正在加载…', 'Loading…') }}</p>
    <div class="chat-models-grid">
      <section aria-labelledby="saved-models-title" class="chat-model-panel">
        <h2 id="saved-models-title">{{ text('我的常用模型', 'My saved models') }} <span>{{ presets.length }}</span></h2>
        <p v-if="!loading && !presets.length" class="chat-description">{{ text('还没有常用模型。请在目录中选择分组并添加模型。', 'No saved models. Select a group in the catalog and add a model.') }}</p>
        <ol class="chat-model-list">
          <li v-for="(preset, index) in presets" :key="preset.id" :data-preset-id="preset.id">
            <div class="chat-model-title"><strong>{{ preset.display_name || preset.model }}</strong><span v-if="preset.is_default" class="chat-badge">{{ text('默认', 'Default') }}</span></div>
            <p class="chat-model-code">{{ preset.model }}</p>
            <p>{{ preset.group.name || text('不可访问的分组', 'Unavailable group') }} · #{{ preset.group_id }}</p>
            <p v-if="!preset.available" class="chat-unavailable" role="status">{{ preset.unavailable_reason }}</p>
            <form class="chat-rename" @submit.prevent="mutate(() => chatAPI.updatePreset(preset.id, { display_name: names[preset.id] }))">
              <label :for="`name-${preset.id}`">{{ text('显示名', 'Display name') }}</label>
              <input :id="`name-${preset.id}`" v-model="names[preset.id]" class="portal-input" maxlength="200" :disabled="saving" :placeholder="preset.model" />
              <button class="portal-button" :disabled="saving || names[preset.id] === preset.display_name">{{ text('保存', 'Save') }}</button>
            </form>
            <div class="chat-model-actions">
              <button class="portal-button" :disabled="saving || !preset.available || preset.is_default" @click="mutate(() => chatAPI.updatePreset(preset.id, { is_default: true }))">{{ text('设为默认', 'Set default') }}</button>
              <button class="portal-button" :disabled="saving" @click="editBinding(preset)">{{ text('修改绑定', 'Change binding') }}</button>
              <button class="portal-button" :disabled="saving || !preset.available" @click="mutate(() => chatAPI.reconnectPreset(preset.id))" :title="text('仅重建失效的聊天凭据，仍受普通密钥创建规则限制', 'Replace only an invalid chat credential, subject to normal key creation limits')">{{ text('重新连接', 'Reconnect') }}</button>
              <button class="portal-button" :disabled="saving || index === 0" :aria-label="text('上移', 'Move up') + ' ' + preset.model" @click="move(index, -1)">↑</button>
              <button class="portal-button" :disabled="saving || index === presets.length - 1" :aria-label="text('下移', 'Move down') + ' ' + preset.model" @click="move(index, 1)">↓</button>
              <button class="portal-button" :disabled="saving" @click="mutate(() => chatAPI.deletePreset(preset.id))">{{ text('移除', 'Remove') }}</button>
            </div>
          </li>
        </ol>
      </section>
      <section aria-labelledby="model-catalog-title" class="chat-model-panel">
        <h2 id="model-catalog-title">{{ text('选择分组与模型', 'Choose group and model') }}</h2>
        <p v-if="editing" role="status">{{ text('正在修改：', 'Changing: ') }}{{ editing.display_name || editing.model }} <button class="portal-button" @click="editingId = null">{{ text('取消', 'Cancel') }}</button></p>
        <label for="chat-group">{{ text('分组 / 渠道', 'Group / channel') }}</label>
        <select id="chat-group" v-model="selectedGroup" class="portal-input" :disabled="loading || saving">
          <option :value="null" disabled>{{ text('请选择分组', 'Select a group') }}</option>
          <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }} · #{{ group.id }} · {{ group.subscription_type === 'subscription' ? text('套餐', 'Subscription') : text('余额', 'Balance') }}</option>
        </select>
        <label for="chat-model-search">{{ text('搜索模型', 'Search models') }}</label>
        <input id="chat-model-search" v-model="search" type="search" class="portal-input" :disabled="!selectedGroup" maxlength="256" />
        <p v-if="catalog" class="chat-description">{{ text('分组倍率', 'Group rate') }} ×{{ catalog.group.rate_multiplier }} · {{ text('你的倍率', 'Your rate') }} ×{{ catalog.group.user_rate_multiplier }} · {{ catalog.group.subscription_type === 'subscription' ? text('套餐计费', 'Subscription billing') : text('余额计费', 'Balance billing') }}</p>
        <p class="chat-description">{{ text('价格显示标准时段最低档，已应用你的倍率。阶梯、时段、缓存及工具费用以实际用量账单为准。目录表示配置兼容性，不是实时健康保证。', 'Prices show the lowest standard-period tier with your rate applied. Context tiers, time periods, cache and tools follow actual usage billing. Catalog compatibility is not a live health guarantee.') }}</p>
        <p v-if="catalogLoading" role="status">{{ text('加载目录…', 'Loading catalog…') }}</p>
        <p v-if="catalogError" role="alert">{{ catalogError }} <button class="portal-button" @click="loadCatalog">{{ text('重试', 'Retry') }}</button></p>
        <p v-if="catalog && !catalog.models.length">{{ text('没有符合条件的聊天模型。可调整搜索或选择其他分组。', 'No compatible chat models. Change the search or select another group.') }}</p>
        <ul class="chat-model-list">
          <li v-for="model in catalog?.models || []" :key="model.id">
            <strong class="chat-model-code">{{ model.id }}</strong>
            <p>{{ price(model) }}</p>
            <p class="chat-description">{{ model.capabilities.image_input ? text('图片 · ', 'Images · ') : '' }}{{ model.capabilities.document_input ? text('文档 · ', 'Documents · ') : '' }}{{ model.capabilities.search ? text('搜索 · ', 'Search · ') : '' }}{{ text('文字聊天', 'Text chat') }}</p>
            <button class="portal-button" :disabled="saving || !model.available || (!editingId && alreadyAdded(model.id))" @click="add(model)">{{ editingId ? text('绑定此模型', 'Use this binding') : alreadyAdded(model.id) ? text('已添加', 'Added') : text('添加常用', 'Add model') }}</button>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<style scoped>
.chat-models-page { min-width: 0; }
.chat-description { color: rgb(var(--muted)); font-size: 13px; line-height: 1.7; }
.chat-models-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 24px; margin-top: 24px; }
.chat-model-panel { min-width: 0; border: 1px solid rgb(var(--line)); border-radius: 12px; padding: 20px; }
.chat-model-panel h2 { font-size: 18px; margin: 0 0 20px; }
.chat-model-panel h2 span { font-size: 13px; color: rgb(var(--muted)); }
.chat-model-panel > label { display: block; font-size: 13px; margin: 16px 0 6px; }
.chat-model-panel > .portal-input { width: 100%; min-width: 0; }
.chat-model-list { list-style: none; margin: 20px 0 0; padding: 0; display: grid; gap: 16px; }
.chat-model-list > li { min-width: 0; border-top: 1px solid rgb(var(--line)); padding-top: 16px; overflow-wrap: anywhere; }
.chat-model-list p { margin: 6px 0; font-size: 13px; }
.chat-model-title { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.chat-model-code { font-family: var(--font-mono, monospace); overflow-wrap: anywhere; }
.chat-badge { border: 1px solid rgb(var(--line)); border-radius: 6px; padding: 2px 6px; font-size: 11px; }
.chat-rename { display: grid; grid-template-columns: 1fr auto; gap: 6px; margin: 12px 0; }
.chat-rename label { grid-column: 1 / -1; font-size: 12px; }
.chat-rename input { min-width: 0; width: 100%; }
.chat-model-actions { display: flex; gap: 6px; flex-wrap: wrap; }
.chat-unavailable { color: rgb(var(--danger, 185 28 28)); }
@media (max-width: 767px) { .chat-models-grid { grid-template-columns: minmax(0, 1fr); gap: 16px; } .chat-model-panel { padding: 16px; } .portal-button { min-height: 40px; } }
</style>
