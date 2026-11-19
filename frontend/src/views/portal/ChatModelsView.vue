<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { chatAPI, type ChatCatalog, type ChatCatalogModel, type ChatPresetView, type ChatReasoningEffort } from '@/api/chat'
import { getAvailable } from '@/api/groups'
import type { Group } from '@/types'
import { useAppStore } from '@/stores'
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'

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
type ReasoningProfile = { normal_reasoning_effort: ChatReasoningEffort; thinking_reasoning_effort: ChatReasoningEffort }
const reasoning = ref<Record<string, ReasoningProfile>>({})
const effortOptions = (preset: ChatPresetView): ChatReasoningEffort[] => preset.capabilities.reasoning_efforts || ['low', 'medium', 'high']
const effortLabel = (effort: ChatReasoningEffort) => ({ none: text('关闭', 'Off'), low: text('低', 'Low'), medium: text('中', 'Medium'), high: text('高', 'High') })[effort] + ` · ${effort}`
const reasoningChanged = (preset: ChatPresetView) => reasoning.value[preset.id]?.normal_reasoning_effort !== preset.normal_reasoning_effort || reasoning.value[preset.id]?.thinking_reasoning_effort !== preset.thinking_reasoning_effort
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
  reasoning.value = Object.fromEntries(value.map(preset => [preset.id, { normal_reasoning_effort: preset.normal_reasoning_effort, thinking_reasoning_effort: preset.thinking_reasoning_effort }]))
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
    <PortalPageHeader
      :title="text('聊天模型', 'Chat models')"
      :description="text('保存常用模型，在聊天中随时切换。每个模型固定使用你选择的分组与推理强度配置。', 'Save models to switch between them in chat. Each model uses the group and reasoning profile you choose.')"
    >
      <RouterLink to="/chat" target="_blank" rel="noopener noreferrer" class="portal-button">
        {{ text('打开聊天', 'Open chat') }}
        <Icon name="externalLink" size="sm" aria-hidden="true" />
      </RouterLink>
    </PortalPageHeader>

    <nav class="portal-tabs chat-view-switch" :aria-label="text('模型配置视图', 'Model settings views')">
      <button type="button" data-view="saved" :class="{ active: view === 'saved' }" :aria-pressed="view === 'saved'" @click="view = 'saved'">
        {{ text('常用模型', 'Saved models') }}
        <span v-if="presets.length" class="chat-tab-badge">{{ presets.length }}</span>
      </button>
      <button type="button" data-view="catalog" :class="{ active: view === 'catalog' }" :aria-pressed="view === 'catalog'" @click="view = 'catalog'">
        {{ text('添加模型', 'Add models') }}
      </button>
    </nav>

    <p v-if="error" class="portal-error" role="alert">{{ error }} <button class="portal-button secondary" @click="load">{{ text('重试', 'Retry') }}</button></p>
    <div v-if="loading" class="chat-status-card" role="status">{{ text('正在加载…', 'Loading…') }}</div>

    <section v-show="view === 'saved'" class="chat-saved" :aria-label="text('我的常用模型', 'My saved models')">
      <div v-if="!loading && !presets.length" class="portal-panel chat-empty-panel">
        <div class="chat-empty-icon" aria-hidden="true"><Icon name="cube" size="lg" /></div>
        <p>{{ text('还没有常用模型', 'No saved models yet') }}</p>
        <button class="portal-button" @click="view = 'catalog'">{{ text('添加模型', 'Add models') }}</button>
      </div>
      <ol v-else class="chat-model-list">
        <li v-for="(preset, index) in presets" :key="preset.id" :data-preset-id="preset.id" class="chat-saved-row">
          <details class="chat-preset-settings">
            <summary>
              <span class="chat-model-avatar" aria-hidden="true">
                <ModelIcon :model="preset.model" size="22px" />
              </span>
              <span class="chat-model-identity">
                <span class="chat-model-title">
                  <strong>{{ preset.display_name || preset.model }}</strong>
                  <span v-if="preset.is_default" class="chat-default">{{ text('默认', 'Default') }}</span>
                  <span v-if="!preset.available" class="chat-unavailable-badge" role="status">{{ preset.unavailable_reason }}</span>
                </span>
                <span class="chat-saved-meta">
                  <code v-if="preset.display_name && preset.display_name !== preset.model" class="chat-saved-code">{{ preset.model }}</code>
                  <span class="chat-saved-group">
                    {{ preset.group.name || text('不可访问的分组', 'Unavailable group') }}
                    <span>#{{ preset.group_id }}</span>
                  </span>
                </span>
              </span>
              <span class="chat-manage-label">
                <span>{{ text('管理', 'Manage') }}</span>
                <Icon name="chevronDown" size="sm" aria-hidden="true" />
              </span>
            </summary>
            <div class="chat-preset-editor">
              <form class="chat-rename" @submit.prevent="mutate(() => chatAPI.updatePreset(preset.id, { display_name: names[preset.id] }))">
                <label class="portal-field" :for="`name-${preset.id}`">
                  <span>{{ text('显示名', 'Display name') }}</span>
                  <input :id="`name-${preset.id}`" v-model="names[preset.id]" maxlength="200" :disabled="saving" :placeholder="preset.model" />
                </label>
                <button class="portal-button" :disabled="saving || names[preset.id] === preset.display_name">{{ text('保存', 'Save') }}</button>
              </form>

              <form v-if="preset.capabilities.reasoning" class="chat-reasoning" :aria-label="text('推理强度', 'Reasoning effort')" @submit.prevent="mutate(() => chatAPI.updatePreset(preset.id, reasoning[preset.id]!))">
                <fieldset :disabled="saving" :aria-describedby="`reasoning-note-${preset.id}`">
                  <legend>{{ text('推理强度', 'Reasoning effort') }}</legend>
                  <div class="chat-reasoning-fields">
                    <label class="portal-field" :for="`normal-effort-${preset.id}`">
                      <span>{{ text('普通模式', 'Normal mode') }}</span>
                      <span class="chat-select-wrap">
                        <select :id="`normal-effort-${preset.id}`" v-model="reasoning[preset.id]!.normal_reasoning_effort">
                          <option v-for="effort in effortOptions(preset)" :key="effort" :value="effort">{{ effortLabel(effort) }}</option>
                        </select>
                        <Icon name="chevronDown" size="xs" class="chat-select-chevron" aria-hidden="true" />
                      </span>
                    </label>
                    <label class="portal-field" :for="`thinking-effort-${preset.id}`">
                      <span>{{ text('深度思考', 'Deep thinking') }}</span>
                      <span class="chat-select-wrap">
                        <select :id="`thinking-effort-${preset.id}`" v-model="reasoning[preset.id]!.thinking_reasoning_effort">
                          <option v-for="effort in effortOptions(preset).filter(value => value !== 'none')" :key="effort" :value="effort">{{ effortLabel(effort) }}</option>
                        </select>
                        <Icon name="chevronDown" size="xs" class="chat-select-chevron" aria-hidden="true" />
                      </span>
                    </label>
                    <button class="portal-button secondary" :disabled="saving || !reasoningChanged(preset)">{{ text('保存强度', 'Save effort') }}</button>
                  </div>
                </fieldset>
                <p :id="`reasoning-note-${preset.id}`">{{ text('默认普通 low、深度 high。保存后用于新一轮回答；强度越高可能越慢、费用越高。不同模型按支持的档位或预算映射，固定强度模型以渠道规则为准。', 'Defaults: normal low, deep high. Applies to new replies; higher effort may increase latency and cost. Levels map to each model’s supported tiers or budgets; fixed-effort models follow channel rules.') }}</p>
              </form>

              <div class="chat-model-actions">
                <div class="chat-binding-actions">
                  <button class="portal-button secondary" :disabled="saving || !preset.available || preset.is_default" @click="mutate(() => chatAPI.updatePreset(preset.id, { is_default: true }))">{{ text('设为默认', 'Set default') }}</button>
                  <button class="portal-button secondary" :disabled="saving" @click="editBinding(preset)">{{ text('修改绑定', 'Change binding') }}</button>
                  <button class="portal-button secondary" :disabled="saving || !preset.available" @click="mutate(() => chatAPI.reconnectPreset(preset.id))" :title="text('仅重建失效的聊天凭据，仍受普通密钥创建规则限制', 'Replace only an invalid chat credential, subject to normal key creation limits')">{{ text('重新连接', 'Reconnect') }}</button>
                </div>
                <div class="chat-order-actions">
                  <button class="portal-button secondary chat-icon-button" :disabled="saving || index === 0" :aria-label="text('上移', 'Move up') + ' ' + preset.model" @click="move(index, -1)">
                    <Icon name="arrowUp" size="sm" aria-hidden="true" />
                  </button>
                  <button class="portal-button secondary chat-icon-button" :disabled="saving || index === presets.length - 1" :aria-label="text('下移', 'Move down') + ' ' + preset.model" @click="move(index, 1)">
                    <Icon name="arrowDown" size="sm" aria-hidden="true" />
                  </button>
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
        <header class="chat-section-head">
          <h2 id="chat-groups-title">{{ text('可选分组', 'Groups') }}</h2>
          <span class="chat-count-pill">{{ groups.length }}</span>
        </header>
        <button type="button" class="chat-mobile-group" :aria-expanded="groupPickerOpen" aria-controls="chat-group-options" @click="groupPickerOpen = !groupPickerOpen">
          <span>{{ activeGroup?.name || text('选择分组', 'Choose a group') }}</span>
          <Icon name="chevronDown" size="sm" aria-hidden="true" />
        </button>
        <div id="chat-group-options" class="chat-group-options" :class="{ 'is-collapsed': !groupPickerOpen }">
          <label class="chat-search">
            <Icon name="search" size="sm" aria-hidden="true" />
            <input v-model="groupSearch" type="search" :aria-label="text('搜索分组', 'Search groups')" :placeholder="text('搜索名称或 ID', 'Search name or ID')" maxlength="256" />
          </label>
          <div class="chat-group-list" :aria-label="text('可选分组', 'Available groups')">
            <button
              v-for="group in filteredGroups"
              :key="group.id"
              type="button"
              :data-group-id="group.id"
              :aria-pressed="selectedGroup === group.id"
              :disabled="loading || saving"
              :title="group.name"
              @click="chooseGroup(group.id)"
            >
              <span class="chat-group-name">{{ group.name }}</span>
              <span class="chat-group-meta">
                <span>#{{ group.id }} · {{ group.platform }}</span>
                <span class="chat-group-rate">×{{ group.rate_multiplier ?? 1 }}</span>
              </span>
            </button>
            <p v-if="!loading && !filteredGroups.length" class="chat-empty">{{ text('没有匹配的分组', 'No matching groups') }}</p>
          </div>
        </div>
      </aside>

      <div class="chat-catalog" aria-labelledby="model-catalog-title">
        <header class="chat-catalog-head">
          <div class="chat-catalog-title-block">
            <div class="chat-section-head">
              <h2 id="model-catalog-title">{{ activeGroup?.name || text('模型目录', 'Model catalog') }}</h2>
              <span v-if="activeGroup" class="chat-count-pill">#{{ activeGroup.id }}</span>
            </div>
            <div v-if="catalog" class="chat-catalog-meta">
              <span class="chat-meta-pill is-rate">{{ text('你的倍率', 'Your rate') }} ×{{ catalog.group.user_rate_multiplier }}</span>
              <span class="chat-meta-pill">{{ catalog.group.subscription_type === 'subscription' ? text('套餐计费', 'Subscription billing') : text('余额计费', 'Balance billing') }}</span>
              <span class="chat-meta-pill">{{ catalog.models.length }} {{ text('个模型', 'models') }}</span>
            </div>
          </div>
          <label class="chat-search chat-model-search">
            <Icon name="search" size="sm" aria-hidden="true" />
            <input id="chat-model-search" v-model="search" type="search" :aria-label="text('搜索模型', 'Search models')" :placeholder="text('搜索此分组的模型', 'Search models in this group')" :disabled="!selectedGroup" maxlength="256" />
          </label>
        </header>

        <div v-if="editing" class="chat-editing" role="status">
          <span>{{ text('修改绑定：', 'Rebind: ') }}<strong>{{ editing.display_name || editing.model }}</strong></span>
          <button class="portal-button secondary" @click="editingId = null">{{ text('取消', 'Cancel') }}</button>
        </div>

        <div class="chat-catalog-body" :aria-busy="catalogLoading">
          <div v-if="!selectedGroup" class="chat-empty-box">
            <Icon name="cube" size="lg" aria-hidden="true" />
            <p>{{ text('选择分组，查看模型和价格。', 'Choose a group to see its models and prices.') }}</p>
          </div>
          <p v-if="catalogLoading" class="chat-empty" role="status">{{ text('加载目录…', 'Loading catalog…') }}</p>
          <p v-if="catalogError" class="chat-empty" role="alert">{{ catalogError }} <button class="portal-button secondary" @click="loadCatalog">{{ text('重试', 'Retry') }}</button></p>
          <p v-if="catalog && !catalog.models.length" class="chat-empty">{{ text('没有符合条件的聊天模型。可调整搜索或选择其他分组。', 'No compatible chat models. Change the search or select another group.') }}</p>
          <ul v-if="catalog?.models?.length" class="chat-catalog-list">
            <li v-for="model in catalog.models" :key="model.id" class="chat-catalog-row">
              <span class="chat-model-avatar" aria-hidden="true">
                <ModelIcon :model="model.id" size="20px" />
              </span>
              <div class="chat-model-info">
                <strong class="chat-model-name">{{ model.id }}</strong>
                <p class="chat-price">{{ price(model) }}</p>
                <div class="chat-model-tags">
                  <span class="chat-cap-tag is-transport">{{ model.capabilities.transport === 'messages' ? 'Messages' : model.capabilities.transport === 'gemini' ? 'Gemini v1beta' : model.capabilities.transport === 'responses' ? 'Responses' : 'Chat Completions' }}</span>
                  <span v-if="model.capabilities.image_input" class="chat-cap-tag">{{ text('图片', 'Images') }}</span>
                  <span v-if="model.capabilities.document_input" class="chat-cap-tag">{{ text('文档', 'Files') }}</span>
                  <span v-if="model.capabilities.reasoning" class="chat-cap-tag is-accent">{{ text('思考', 'Reasoning') }}</span>
                  <span v-if="model.capabilities.search" class="chat-cap-tag">{{ text('搜索', 'Search') }}</span>
                </div>
                <p v-if="!model.available" class="chat-unavailable">{{ model.unavailable_reason }}</p>
              </div>
              <button
                class="portal-button chat-add"
                :class="{ secondary: !editingId && alreadyAdded(model.id) }"
                :disabled="saving || !model.available || (!editingId && alreadyAdded(model.id))"
                @click="add(model)"
              >
                {{ editingId ? text('绑定', 'Bind') : alreadyAdded(model.id) ? text('已添加', 'Added') : text('添加常用', 'Add model') }}
              </button>
            </li>
          </ul>
        </div>
        <details class="chat-pricing-note">
          <summary>{{ text('价格与可用性说明', 'Pricing and availability') }}</summary>
          <p>{{ text('价格为标准时段最低档，已应用你的倍率。阶梯、时段、缓存及工具费用以实际账单为准。目录表示配置兼容性，不是实时健康保证。', 'Lowest standard-period prices with your rate applied. Tiers, time periods, cache and tools follow actual billing. Compatibility is not a live health guarantee.') }}</p>
        </details>
      </div>
    </section>
  </div>
</template>

<style scoped>
.chat-models-page {
  min-width: 0;
  font-family: var(--font-sans);
}

.chat-view-switch button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-family: inherit;
  font-size: 13.5px;
}

.chat-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  background: rgb(var(--brand-moss) / 0.14);
  color: rgb(var(--brand-moss));
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.chat-status-card {
  padding: 36px 20px;
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-card);
  background: rgb(var(--surface));
  color: rgb(var(--ink-3));
  text-align: center;
  font-size: 13.5px;
}

.chat-empty-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 48px 24px;
  text-align: center;
}

.chat-empty-icon {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgb(var(--surface-2));
  border: 1px solid rgb(var(--line));
  color: rgb(var(--ink-3));
}

.chat-empty-panel p {
  margin: 0;
  color: rgb(var(--ink-2));
  font-size: 14px;
}

.chat-model-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 12px;
}

.chat-saved-row {
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-card);
  background: rgb(var(--surface));
  box-shadow: var(--shadow-panel);
  overflow: hidden;
  transition: border-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease;
}

.chat-saved-row:hover {
  border-color: rgb(var(--line-strong));
}

.chat-preset-settings[open] {
  background: rgb(var(--surface));
}

.chat-preset-settings > summary {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 20px;
  cursor: pointer;
  list-style: none;
  outline: none;
  transition: background-color var(--motion-fast) ease;
}

.chat-preset-settings > summary::-webkit-details-marker {
  display: none;
}

.chat-preset-settings > summary:hover {
  background: rgb(var(--table-hover));
}

.chat-preset-settings > summary:focus-visible {
  outline: 2px solid rgb(var(--focus-ring));
  outline-offset: -2px;
}

.chat-model-avatar {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  border-radius: 10px;
  background: rgb(var(--surface-2));
  border: 1px solid rgb(var(--line));
}

.chat-models-page :deep(.model-icon path[fill='#000000']),
.chat-models-page :deep(.model-icon path[fill='#16191E']) {
  fill: rgb(var(--ink));
}

.chat-model-identity {
  flex: 1;
  min-width: 0;
  display: grid;
  gap: 4px;
  overflow-wrap: anywhere;
}

.chat-model-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.chat-model-title strong,
.chat-model-name {
  font-family: var(--font-mono);
  font-size: 14.5px;
  font-weight: 650;
  line-height: 1.4;
  color: rgb(var(--ink));
  overflow-wrap: anywhere;
}

.chat-default {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgb(var(--brand-moss) / 0.14);
  border: 1px solid rgb(var(--brand-moss) / 0.3);
  color: rgb(var(--brand-moss));
  font-size: 11.5px;
  font-weight: 650;
}

.chat-unavailable-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgb(var(--status-err-bg) / 0.75);
  border: 1px solid rgb(var(--text-err) / 0.3);
  color: rgb(var(--text-err));
  font-size: 11.5px;
  font-weight: 600;
}

.chat-saved-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  font-size: 13px;
  color: rgb(var(--ink-2));
}

.chat-saved-code {
  padding: 1px 6px;
  border-radius: 5px;
  background: rgb(var(--surface-2));
  border: 1px solid rgb(var(--line));
  font-family: var(--font-mono);
  font-size: 12px;
  color: rgb(var(--ink-2));
}

.chat-saved-group > span {
  margin-left: 4px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: rgb(var(--ink-3));
}

.chat-manage-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex: none;
  padding: 6px 11px;
  border-radius: 8px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.55);
  color: rgb(var(--ink-2));
  font-size: 12.5px;
  font-weight: 550;
  transition: border-color var(--motion-fast) ease, color var(--motion-fast) ease;
}

.chat-preset-settings > summary:hover .chat-manage-label {
  border-color: rgb(var(--line-strong));
  color: rgb(var(--ink));
}

.chat-preset-settings[open] .chat-manage-label :deep(svg) {
  transform: rotate(180deg);
}

.chat-preset-editor {
  padding: 18px 20px 20px;
  border-top: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.3);
}

.chat-rename {
  display: grid;
  grid-template-columns: minmax(0, 420px) auto;
  justify-content: start;
  align-items: end;
  gap: 10px;
  padding-bottom: 18px;
}

.chat-rename .portal-field {
  font-size: 13px;
}

.chat-rename input {
  min-width: 0;
  font-family: inherit;
}

.chat-reasoning {
  margin-bottom: 18px;
  padding: 16px 0;
  border-top: 1px solid rgb(var(--line));
  border-bottom: 1px solid rgb(var(--line));
}

.chat-reasoning fieldset {
  min-width: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.chat-reasoning legend {
  margin-bottom: 12px;
  padding: 0;
  font-size: 13.5px;
  font-weight: 650;
  color: rgb(var(--ink));
}

.chat-reasoning-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 220px)) auto;
  align-items: end;
  justify-content: start;
  gap: 12px;
}

.chat-reasoning .portal-field {
  font-size: 13px;
}

.chat-select-wrap {
  position: relative;
  display: block;
}

.chat-select-wrap select {
  width: 100%;
  min-height: 40px;
  padding: 8px 34px 8px 12px;
  appearance: none;
  -webkit-appearance: none;
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-control);
  background: rgb(var(--surface));
  color: rgb(var(--ink));
  font: inherit;
  font-size: 13.5px;
  cursor: pointer;
  transition: border-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease;
}

.chat-select-wrap select:hover:not(:disabled) {
  border-color: rgb(var(--line-strong));
}

.chat-select-wrap select:focus-visible {
  outline: none;
  border-color: rgb(var(--focus-ring));
  box-shadow: 0 0 0 3px rgb(var(--focus-ring) / 0.14);
}

.chat-select-chevron {
  position: absolute;
  right: 11px;
  top: 50%;
  transform: translateY(-50%);
  pointer-events: none;
  color: rgb(var(--ink-3));
}

.chat-reasoning p {
  max-width: 660px;
  margin: 10px 0 0;
  color: rgb(var(--ink-3));
  font-size: 12.5px;
  line-height: 1.6;
}

.chat-model-actions,
.chat-binding-actions,
.chat-order-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.chat-model-actions {
  justify-content: space-between;
  gap: 16px;
}

.chat-models-page .portal-button {
  font-size: 13.5px;
}

.chat-icon-button {
  width: 38px;
  padding-inline: 0;
}

.chat-footnote {
  margin: 14px 4px 0;
  color: rgb(var(--ink-3));
  font-size: 12.5px;
}

.chat-browser {
  display: grid;
  grid-template-columns: 276px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.chat-groups {
  position: sticky;
  top: 84px;
  min-width: 0;
  padding: 18px 14px;
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-card);
  background: rgb(var(--surface));
  box-shadow: var(--shadow-panel);
}

.chat-catalog {
  min-width: 0;
  min-height: 460px;
  padding: 22px 24px;
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-card);
  background: rgb(var(--surface));
  box-shadow: var(--shadow-panel);
}

.chat-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.chat-section-head h2 {
  margin: 0;
  font-size: 15.5px;
  font-weight: 650;
  line-height: 1.4;
  color: rgb(var(--ink));
  overflow-wrap: anywhere;
}

.chat-count-pill {
  flex: none;
  padding: 1px 8px;
  border-radius: 999px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2));
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 600;
  color: rgb(var(--ink-2));
}

.chat-groups .chat-section-head {
  padding-inline: 4px;
  margin-bottom: 12px;
}

.chat-search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 12px;
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-control);
  background: rgb(var(--surface-2) / 0.45);
  color: rgb(var(--ink-3));
  transition: border-color var(--motion-fast) ease, box-shadow var(--motion-fast) ease, background-color var(--motion-fast) ease;
}

.chat-search:focus-within {
  background: rgb(var(--surface));
  border-color: rgb(var(--focus-ring));
  box-shadow: 0 0 0 3px rgb(var(--focus-ring) / 0.14);
}

.chat-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: rgb(var(--ink));
  font: inherit;
  font-size: 13px;
}

.chat-group-list {
  max-height: min(600px, 65dvh);
  margin-top: 10px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.chat-group-list button {
  display: grid;
  gap: 5px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: rgb(var(--ink-2));
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color var(--motion-fast) ease, border-color var(--motion-fast) ease, color var(--motion-fast) ease;
}

.chat-group-list button:hover {
  background: rgb(var(--surface-2) / 0.75);
  color: rgb(var(--ink));
}

.chat-group-list button[aria-pressed='true'] {
  background: rgb(var(--brand-moss) / 0.1);
  border-color: rgb(var(--brand-moss) / 0.32);
  color: rgb(var(--ink));
}

.chat-group-name {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.chat-group-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 11.5px;
  color: rgb(var(--ink-3));
  font-variant-numeric: tabular-nums;
}

.chat-group-rate {
  padding: 1px 6px;
  border-radius: 999px;
  background: rgb(var(--surface-2));
  color: rgb(var(--ink-2));
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
}

.chat-group-list button[aria-pressed='true'] .chat-group-rate {
  background: rgb(var(--brand-moss) / 0.16);
  color: rgb(var(--brand-moss));
}

.chat-mobile-group {
  display: none;
}

.chat-catalog-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px solid rgb(var(--line));
}

.chat-catalog-title-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.chat-model-search {
  width: 240px;
  background: rgb(var(--surface));
}

.chat-catalog-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.chat-meta-pill {
  display: inline-flex;
  align-items: center;
  padding: 3px 9px;
  border-radius: 999px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface-2) / 0.6);
  font-size: 12px;
  color: rgb(var(--ink-2));
  font-variant-numeric: tabular-nums;
}

.chat-meta-pill.is-rate {
  border-color: rgb(var(--brand-moss) / 0.3);
  background: rgb(var(--brand-moss) / 0.1);
  color: rgb(var(--brand-moss));
  font-weight: 600;
}

.chat-editing {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  padding: 10px 14px;
  border: 1px solid rgb(var(--brand-moss) / 0.3);
  border-radius: 10px;
  background: rgb(var(--brand-moss) / 0.08);
  font-size: 13.5px;
  overflow-wrap: anywhere;
}

.chat-catalog-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 10px;
}

.chat-catalog-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid rgb(var(--line));
  border-radius: 12px;
  background: rgb(var(--surface-2) / 0.3);
  transition: border-color var(--motion-fast) ease, background-color var(--motion-fast) ease;
}

.chat-catalog-row:hover {
  border-color: rgb(var(--line-strong));
  background: rgb(var(--surface-2) / 0.55);
}

.chat-model-info {
  flex: 1;
  min-width: 0;
}

.chat-price {
  margin: 4px 0 0;
  color: rgb(var(--ink-2));
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  line-height: 1.5;
}

.chat-model-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.chat-cap-tag {
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid rgb(var(--line));
  background: rgb(var(--surface));
  color: rgb(var(--ink-2));
  font-size: 11px;
  font-weight: 550;
}

.chat-cap-tag.is-transport {
  font-family: var(--font-mono);
  color: rgb(var(--ink-3));
}

.chat-cap-tag.is-accent {
  border-color: rgb(var(--brand-moss) / 0.3);
  background: rgb(var(--brand-moss) / 0.1);
  color: rgb(var(--brand-moss));
}

.chat-add {
  flex: none;
}

.chat-pricing-note {
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid rgb(var(--line));
  color: rgb(var(--ink-3));
  font-size: 12.5px;
}

.chat-pricing-note summary {
  cursor: pointer;
  width: fit-content;
  font-weight: 550;
}

.chat-pricing-note p {
  margin: 8px 0 0;
  line-height: 1.65;
}

.chat-unavailable {
  display: block;
  margin: 6px 0 0;
  color: rgb(var(--text-err));
  font-size: 12.5px;
  font-weight: 550;
}

.chat-empty-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 52px 20px;
  border: 1px dashed rgb(var(--line));
  border-radius: 12px;
  background: rgb(var(--surface-2) / 0.25);
  color: rgb(var(--ink-3));
  text-align: center;
}

.chat-empty-box p {
  margin: 0;
  font-size: 13.5px;
}

.chat-empty {
  margin: 0;
  padding: 36px 20px;
  color: rgb(var(--ink-3));
  font-size: 13.5px;
  line-height: 1.65;
  text-align: center;
}

@media (max-width: 1023px) {
  .chat-browser {
    grid-template-columns: 230px minmax(0, 1fr);
    gap: 16px;
  }

  .chat-catalog {
    padding: 18px;
  }

  .chat-model-search {
    width: 100%;
  }
}

@media (max-width: 767px) {
  .chat-view-switch {
    width: 100%;
  }

  .chat-view-switch button {
    flex: 1;
    justify-content: center;
    min-height: 44px;
  }

  .chat-browser {
    grid-template-columns: minmax(0, 1fr);
  }

  .chat-groups {
    position: static;
    padding: 16px;
  }

  .chat-groups .chat-section-head {
    display: none;
  }

  .chat-mobile-group {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    width: 100%;
    min-height: 44px;
    padding: 10px 12px;
    border: 1px solid rgb(var(--line));
    border-radius: var(--radius-control);
    background: rgb(var(--surface));
    color: rgb(var(--ink));
    font: inherit;
    text-align: left;
  }

  .chat-mobile-group > span {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .chat-group-options {
    margin-top: 12px;
  }

  .chat-group-options.is-collapsed {
    display: none;
  }

  .chat-group-list {
    max-height: 280px;
  }

  .chat-catalog {
    min-height: 0;
    padding: 16px;
  }

  .chat-catalog-row {
    flex-wrap: wrap;
    gap: 12px;
  }

  .chat-model-info {
    flex-basis: calc(100% - 54px);
  }

  .chat-add {
    margin-left: auto;
  }

  .chat-preset-settings > summary {
    gap: 12px;
    padding: 14px 16px;
  }

  .chat-preset-editor {
    padding: 16px;
  }

  .chat-rename {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
  }

  .chat-reasoning-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .chat-reasoning-fields .portal-button {
    grid-column: 1 / -1;
    justify-self: start;
  }

  .chat-select-wrap select,
  .chat-rename input {
    min-height: 44px;
    font-size: 16px;
  }

  .chat-binding-actions {
    width: 100%;
  }

  .chat-models-page .portal-button {
    min-height: 44px;
  }

  .chat-icon-button {
    width: 44px;
  }

  .chat-search input {
    font-size: 16px;
  }
}

@media (pointer: coarse) {
  .chat-models-page .portal-button,
  .chat-pricing-note summary {
    min-height: 44px;
  }

  .chat-icon-button {
    width: 44px;
  }
}
</style>
