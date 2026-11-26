<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  UiButton,
  UiSelect,
  UiStatusDot,
  BrandModelIcon,
  BrandPlatformIcon,
  DotMatrixBackdrop
} from '@/ui'
import {
  chatAPI,
  type ChatCatalog,
  type ChatCatalogModel,
  type ChatPresetView,
  type ChatReasoningEffort
} from '@/api/chat'
import { getAvailable } from '@/api/groups'
import type { Group } from '@/types'
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'

const { locale } = useI18n()
const app = useAppStore()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)

const groups = ref<Group[]>([])
const presets = ref<ChatPresetView[]>([])
const catalog = ref<ChatCatalog | null>(null)
const selectedGroup = ref<number | null>(null)
const search = ref('')
const view = ref<'saved' | 'catalog'>('saved')
const groupPickerOpen = ref(true)
const groupSearch = ref('')

const filteredGroups = computed(() =>
  groups.value.filter((group) =>
    `${group.name} ${group.platform} ${group.id}`
      .toLocaleLowerCase()
      .includes(groupSearch.value.trim().toLocaleLowerCase())
  )
)
const activeGroup = computed(() => groups.value.find((group) => group.id === selectedGroup.value))
const loading = ref(false)
const catalogLoading = ref(false)
const saving = ref(false)
const error = ref('')
const catalogError = ref('')
const editingId = ref<string | null>(null)
const names = ref<Record<string, string>>({})

type ReasoningProfile = {
  normal_reasoning_effort: ChatReasoningEffort
  thinking_reasoning_effort: ChatReasoningEffort
}
const reasoning = ref<Record<string, ReasoningProfile>>({})

const effortOptions = (preset: ChatPresetView): ChatReasoningEffort[] =>
  preset.capabilities.reasoning_efforts || ['low', 'medium', 'high']

const effortLabel = (effort: ChatReasoningEffort) =>
  ({
    none: text('关闭', 'Off'),
    low: text('低', 'Low'),
    medium: text('中', 'Medium'),
    high: text('高', 'High')
  })[effort] + ` · ${effort}`

const normalEffortSelectOptions = (preset: ChatPresetView) =>
  effortOptions(preset).map((effort) => ({ value: effort, label: effortLabel(effort) }))

const thinkingEffortSelectOptions = (preset: ChatPresetView) =>
  effortOptions(preset)
    .filter((value) => value !== 'none')
    .map((effort) => ({ value: effort, label: effortLabel(effort) }))

const reasoningChanged = (preset: ChatPresetView) =>
  reasoning.value[preset.id]?.normal_reasoning_effort !== preset.normal_reasoning_effort ||
  reasoning.value[preset.id]?.thinking_reasoning_effort !== preset.thinking_reasoning_effort

const editing = computed(() => presets.value.find((preset) => preset.id === editingId.value))
let request: AbortController | undefined
const initialRequest = new AbortController()
let searchTimer: ReturnType<typeof setTimeout> | undefined
let disposed = false

const message = (cause: unknown) =>
  cause instanceof Error ? cause.message : text('操作失败，请重试。', 'The operation failed. Please retry.')

async function loadPresets() {
  const value = await chatAPI.presets(initialRequest.signal)
  if (disposed) return
  presets.value = value
  names.value = Object.fromEntries(value.map((preset) => [preset.id, preset.display_name]))
  reasoning.value = Object.fromEntries(
    value.map((preset) => [
      preset.id,
      {
        normal_reasoning_effort: preset.normal_reasoning_effort,
        thinking_reasoning_effort: preset.thinking_reasoning_effort
      }
    ])
  )
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
  } catch (cause) {
    if (!disposed) error.value = message(cause)
  } finally {
    if (!disposed) loading.value = false
  }
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
  } catch (cause) {
    if (!current.signal.aborted && !disposed) catalogError.value = message(cause)
  } finally {
    if (!current.signal.aborted && !disposed) catalogLoading.value = false
  }
}

function chooseGroup(id: number) {
  selectedGroup.value = id
  groupPickerOpen.value = false
}

watch(selectedGroup, () => {
  clearTimeout(searchTimer)
  void loadCatalog()
})
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
  } catch (cause) {
    if (!disposed) app.showError(message(cause))
  } finally {
    if (!disposed) saving.value = false
  }
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
  selectedGroup.value = groups.value.some((group) => group.id === preset.group_id) ? preset.group_id : null
  void loadCatalog()
}

function move(index: number, direction: -1 | 1) {
  const ids = presets.value.map((preset) => preset.id)
  const target = index + direction
  if (target < 0 || target >= ids.length) return
  ;[ids[index], ids[target]] = [ids[target]!, ids[index]!]
  void mutate(() => chatAPI.orderPresets(ids))
}

const alreadyAdded = (model: string) =>
  presets.value.some((preset) => preset.group_id === catalog.value?.group.id && preset.model === model)

const money = (value: number | null | undefined) =>
  value == null
    ? '—'
    : new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD', maximumFractionDigits: 4 }).format(value)

function price(model: ChatCatalogModel) {
  const pricing = model.pricing
  const rate = catalog.value?.group.user_rate_multiplier ?? catalog.value?.group.rate_multiplier ?? 1
  if (!pricing) return text('价格暂不可用，以实际账单为准', 'Pricing unavailable; actual billing applies')
  if (pricing.billing_mode === 'per_request') {
    return money(pricing.per_request_price == null ? null : pricing.per_request_price * rate) + text(' / 次', ' / request')
  }
  const input = pricing.input_price == null ? null : pricing.input_price * 1_000_000 * rate
  const output = pricing.output_price == null ? null : pricing.output_price * 1_000_000 * rate
  return `${text('输入', 'Input')} ${money(input)} · ${text('输出', 'Output')} ${money(output)} ${text('/ 百万 token 起', '/ 1M tokens, from')}`
}

onMounted(load)
onBeforeUnmount(() => {
  disposed = true
  clearTimeout(searchTimer)
  request?.abort()
  initialRequest.abort()
})
</script>

<template>
  <div class="chat-models-page space-y-6">
    <!-- Minimal Page Header -->
    <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
          {{ text('聊天模型', 'Chat models') }}
        </h1>
        <p class="mt-0.5 text-sm text-[var(--atelier-ink-secondary)]">
          {{ text('保存常用模型，在聊天中随时切换。每个模型固定使用你选择的分组与推理强度配置。', 'Save models to switch between them in chat. Each model uses the group and reasoning profile you choose.') }}
        </p>
      </div>
      <div class="shrink-0">
        <RouterLink
          to="/chat"
          target="_blank"
          rel="noopener noreferrer"
          class="atelier-btn atelier-btn-primary h-9 px-4 text-sm"
        >
          {{ text('打开聊天', 'Open chat') }}
          <Icon name="externalLink" size="xs" aria-hidden="true" />
        </RouterLink>
      </div>
    </header>

    <!-- Primary In-Page Underline Tabs -->
    <nav class="atelier-tabs-list" :aria-label="text('模型配置视图', 'Model settings views')">
      <button
        type="button"
        data-view="saved"
        class="atelier-tab-trigger"
        :data-active="view === 'saved' ? 'true' : 'false'"
        :aria-pressed="view === 'saved'"
        @click="view = 'saved'"
      >
        <span>{{ text('常用模型', 'Saved models') }}</span>
        <span
          v-if="presets.length"
          class="rounded-full bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-secondary)]"
        >
          {{ presets.length }}
        </span>
      </button>
      <button
        type="button"
        data-view="catalog"
        class="atelier-tab-trigger"
        :data-active="view === 'catalog' ? 'true' : 'false'"
        :aria-pressed="view === 'catalog'"
        @click="view = 'catalog'"
      >
        {{ text('添加模型', 'Add models') }}
      </button>
    </nav>

    <!-- Error / Loading States -->
    <div
      v-if="error"
      class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-xs text-[var(--atelier-err)]"
      role="alert"
    >
      <span>{{ error }}</span>
      <UiButton variant="secondary" size="sm" @click="load">{{ text('重试', 'Retry') }}</UiButton>
    </div>
    <div v-if="loading" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]" role="status">
      {{ text('正在加载…', 'Loading…') }}
    </div>

    <!-- Saved Models List -->
    <section v-show="view === 'saved'" class="space-y-3" :aria-label="text('我的常用模型', 'My saved models')">
      <div
        v-if="!loading && !presets.length"
        class="atelier-surface relative overflow-hidden px-6 py-12 text-center"
      >
        <DotMatrixBackdrop mask="edges" />
        <div class="relative z-10 flex flex-col items-center gap-3">
          <p class="text-sm font-medium text-[var(--atelier-ink)]">{{ text('还没有常用模型', 'No saved models yet') }}</p>
          <UiButton variant="primary" size="sm" @click="view = 'catalog'">
            {{ text('添加模型', 'Add models') }}
          </UiButton>
        </div>
      </div>

      <ol v-else class="space-y-2.5">
        <li
          v-for="(preset, index) in presets"
          :key="preset.id"
          :data-preset-id="preset.id"
          class="atelier-surface overflow-hidden transition-colors hover:border-[var(--atelier-border-strong)]"
        >
          <details class="group">
            <summary class="flex cursor-pointer list-none items-center justify-between gap-3 px-4 py-3.5 transition-colors hover:bg-[var(--atelier-hover)]">
              <div class="flex min-w-0 items-center gap-3">
                <BrandModelIcon :model="preset.model" size="md" />
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <strong class="truncate text-sm font-semibold text-[var(--atelier-ink)]">
                      {{ preset.display_name || preset.model }}
                    </strong>
                    <span
                      v-if="preset.is_default"
                      class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 text-[11px] font-medium text-[var(--atelier-ink)]"
                    >
                      {{ text('默认', 'Default') }}
                    </span>
                    <span
                      v-if="!preset.available"
                      class="inline-flex items-center gap-1 rounded-full bg-[var(--atelier-err-bg)] px-2 py-0.5 text-[11px] font-medium text-[var(--atelier-err)]"
                      role="status"
                    >
                      <UiStatusDot tone="err" />
                      {{ preset.unavailable_reason }}
                    </span>
                  </div>
                  <div class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-[var(--atelier-ink-muted)]">
                    <code v-if="preset.display_name && preset.display_name !== preset.model" class="font-mono text-[11.5px] text-[var(--atelier-ink-secondary)]">
                      {{ preset.model }}
                    </code>
                    <span class="inline-flex items-center gap-1">
                      <span>{{ preset.group.name || text('不可访问的分组', 'Unavailable group') }}</span>
                      <span class="font-mono">#{{ preset.group_id }}</span>
                    </span>
                  </div>
                </div>
              </div>
              <span class="inline-flex shrink-0 items-center gap-1 text-xs font-medium text-[var(--atelier-ink-secondary)]">
                <span>{{ text('管理', 'Manage') }}</span>
                <Icon name="chevronDown" size="xs" class="transition-transform group-open:rotate-180" aria-hidden="true" />
              </span>
            </summary>

            <div class="space-y-4 border-t border-[var(--atelier-border)] bg-[var(--atelier-subtle)]/50 px-4 py-4">
              <!-- Rename Form -->
              <form
                class="flex flex-col gap-2.5 sm:flex-row sm:items-end"
                @submit.prevent="mutate(() => chatAPI.updatePreset(preset.id, { display_name: names[preset.id] }))"
              >
                <label class="block flex-1 space-y-1" :for="`name-${preset.id}`">
                  <span class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ text('显示名', 'Display name') }}</span>
                  <input
                    :id="`name-${preset.id}`"
                    v-model="names[preset.id]"
                    class="atelier-input"
                    maxlength="200"
                    :disabled="saving"
                    :placeholder="preset.model"
                  />
                </label>
                <UiButton
                  type="submit"
                  variant="primary"
                  size="md"
                  :disabled="saving || names[preset.id] === preset.display_name"
                >
                  {{ text('保存', 'Save') }}
                </UiButton>
              </form>

              <!-- Reasoning Effort Form (Uses UiSelect — zero native select) -->
              <form
                v-if="preset.capabilities.reasoning"
                class="chat-reasoning space-y-2.5 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-3.5"
                :aria-label="text('推理强度', 'Reasoning effort')"
                @submit.prevent="mutate(() => chatAPI.updatePreset(preset.id, reasoning[preset.id]!))"
              >
                <fieldset :disabled="saving" :aria-describedby="`reasoning-note-${preset.id}`" class="space-y-3">
                  <legend class="text-xs font-semibold text-[var(--atelier-ink)]">{{ text('推理强度', 'Reasoning effort') }}</legend>
                  <div class="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
                    <div class="space-y-1">
                      <label :for="`normal-effort-${preset.id}`" class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
                        {{ text('普通模式', 'Normal mode') }}
                      </label>
                      <UiSelect
                        :id="`normal-effort-${preset.id}`"
                        :model-value="reasoning[preset.id]!.normal_reasoning_effort"
                        :options="normalEffortSelectOptions(preset)"
                        :disabled="saving"
                        :aria-label="text('普通模式', 'Normal mode')"
                        @update:model-value="(val) => { reasoning[preset.id]!.normal_reasoning_effort = String(val) as ChatReasoningEffort }"
                      />
                    </div>
                    <div class="space-y-1">
                      <label :for="`thinking-effort-${preset.id}`" class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
                        {{ text('深度思考', 'Deep thinking') }}
                      </label>
                      <UiSelect
                        :id="`thinking-effort-${preset.id}`"
                        :model-value="reasoning[preset.id]!.thinking_reasoning_effort"
                        :options="thinkingEffortSelectOptions(preset)"
                        :disabled="saving"
                        :aria-label="text('深度思考', 'Deep thinking')"
                        @update:model-value="(val) => { reasoning[preset.id]!.thinking_reasoning_effort = String(val) as ChatReasoningEffort }"
                      />
                    </div>
                    <UiButton
                      type="submit"
                      variant="secondary"
                      size="md"
                      :disabled="saving || !reasoningChanged(preset)"
                    >
                      {{ text('保存强度', 'Save effort') }}
                    </UiButton>
                  </div>
                </fieldset>
                <p :id="`reasoning-note-${preset.id}`" class="text-xs leading-relaxed text-[var(--atelier-ink-muted)]">
                  {{ text('默认普通 low、深度 high。保存后用于新一轮回答；强度越高可能越慢、费用越高。不同模型按支持的档位或预算映射，固定强度模型以渠道规则为准。', 'Defaults: normal low, deep high. Applies to new replies; higher effort may increase latency and cost. Levels map to each model’s supported tiers or budgets; fixed-effort models follow channel rules.') }}
                </p>
              </form>

              <!-- Actions Bar -->
              <div class="flex flex-wrap items-center justify-between gap-2 border-t border-[var(--atelier-border-subtle)] pt-3">
                <div class="flex flex-wrap items-center gap-2">
                  <UiButton
                    variant="secondary"
                    size="sm"
                    :disabled="saving || !preset.available || preset.is_default"
                    @click="mutate(() => chatAPI.updatePreset(preset.id, { is_default: true }))"
                  >
                    {{ text('设为默认', 'Set default') }}
                  </UiButton>
                  <UiButton variant="secondary" size="sm" :disabled="saving" @click="editBinding(preset)">
                    {{ text('修改绑定', 'Change binding') }}
                  </UiButton>
                  <UiButton
                    variant="secondary"
                    size="sm"
                    :disabled="saving || !preset.available"
                    :title="text('仅重建失效的聊天凭据，仍受普通密钥创建规则限制', 'Replace only an invalid chat credential, subject to normal key creation limits')"
                    @click="mutate(() => chatAPI.reconnectPreset(preset.id))"
                  >
                    {{ text('重新连接', 'Reconnect') }}
                  </UiButton>
                </div>
                <div class="flex items-center gap-1.5">
                  <UiButton
                    variant="secondary"
                    size="sm"
                    class="!w-8 !px-0"
                    :disabled="saving || index === 0"
                    :aria-label="text('上移', 'Move up') + ' ' + preset.model"
                    @click="move(index, -1)"
                  >
                    <Icon name="arrowUp" size="xs" aria-hidden="true" />
                  </UiButton>
                  <UiButton
                    variant="secondary"
                    size="sm"
                    class="!w-8 !px-0"
                    :disabled="saving || index === presets.length - 1"
                    :aria-label="text('下移', 'Move down') + ' ' + preset.model"
                    @click="move(index, 1)"
                  >
                    <Icon name="arrowDown" size="xs" aria-hidden="true" />
                  </UiButton>
                  <UiButton
                    variant="danger"
                    size="sm"
                    :disabled="saving"
                    @click="mutate(() => chatAPI.deletePreset(preset.id))"
                  >
                    {{ text('移除', 'Remove') }}
                  </UiButton>
                </div>
              </div>
            </div>
          </details>
        </li>
      </ol>

      <p v-if="presets.length" class="text-xs text-[var(--atelier-ink-muted)]">
        {{ text('修改或移除配置，不影响历史回答。', 'Changing or removing a model does not affect past replies.') }}
      </p>
    </section>

    <!-- Catalog Browser -->
    <section
      v-show="view === 'catalog'"
      class="chat-browser grid min-w-0 grid-cols-1 items-start gap-6 lg:grid-cols-[248px_minmax(0,1fr)]"
      :aria-label="text('添加模型', 'Add models')"
    >
      <!-- Groups Rail -->
      <aside class="atelier-surface min-w-0 p-3.5" aria-labelledby="chat-groups-title">
        <header class="mb-2.5 flex items-center justify-between gap-2 px-1">
          <h2 id="chat-groups-title" class="text-xs font-semibold uppercase tracking-wider text-[var(--atelier-ink-muted)]">
            {{ text('可选分组', 'Groups') }}
          </h2>
          <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">{{ groups.length }}</span>
        </header>

        <button
          type="button"
          class="mb-2 flex w-full items-center justify-between rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-3 py-2 text-xs font-medium text-[var(--atelier-ink)] lg:hidden"
          :aria-expanded="groupPickerOpen"
          aria-controls="chat-group-options"
          @click="groupPickerOpen = !groupPickerOpen"
        >
          <span class="truncate">{{ activeGroup?.name || text('选择分组', 'Choose a group') }}</span>
          <Icon name="chevronDown" size="xs" aria-hidden="true" />
        </button>

        <div id="chat-group-options" class="chat-group-options space-y-2.5" :class="{ 'is-collapsed': !groupPickerOpen }">
          <div class="relative">
            <Icon name="search" size="xs" class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
            <input
              v-model="groupSearch"
              type="search"
              class="atelier-input !min-h-[32px] !pl-8 text-xs"
              :aria-label="text('搜索分组', 'Search groups')"
              :placeholder="text('搜索名称或 ID', 'Search name or ID')"
              maxlength="256"
            />
          </div>

          <div class="max-h-[460px] space-y-1 overflow-y-auto pr-0.5" :aria-label="text('可选分组', 'Available groups')">
            <button
              v-for="group in filteredGroups"
              :key="group.id"
              type="button"
              class="flex w-full flex-col gap-1 rounded-lg border px-2.5 py-2 text-left transition-colors disabled:opacity-50"
              :class="selectedGroup === group.id ? 'border-[var(--atelier-border-strong)] bg-[var(--atelier-selected-row)] text-[var(--atelier-ink)]' : 'border-transparent text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]'"
              :data-group-id="group.id"
              :aria-pressed="selectedGroup === group.id"
              :disabled="loading || saving"
              :title="group.name"
              @click="chooseGroup(group.id)"
            >
              <span class="flex min-w-0 items-center gap-2">
                <BrandPlatformIcon :platform="group.platform" size="sm" />
                <span class="truncate text-xs font-medium text-[var(--atelier-ink)]">{{ group.name }}</span>
              </span>
              <span class="flex items-center justify-between pl-6 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
                <span>#{{ group.id }} · {{ group.platform }}</span>
                <span>×{{ group.rate_multiplier ?? 1 }}</span>
              </span>
            </button>
            <p v-if="!loading && !filteredGroups.length" class="py-6 text-center text-xs text-[var(--atelier-ink-muted)]">
              {{ text('没有匹配的分组', 'No matching groups') }}
            </p>
          </div>
        </div>
      </aside>

      <!-- Models Catalog -->
      <div class="min-w-0 space-y-4" aria-labelledby="model-catalog-title">
        <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h2 id="model-catalog-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
                {{ activeGroup?.name || text('模型目录', 'Model catalog') }}
              </h2>
              <span v-if="activeGroup" class="font-mono text-xs text-[var(--atelier-ink-muted)]">#{{ activeGroup.id }}</span>
            </div>
            <div v-if="catalog" class="flex flex-wrap items-center gap-2 text-xs text-[var(--atelier-ink-secondary)]">
              <span class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 font-mono font-medium tabular-nums text-[var(--atelier-ink)]">
                {{ text('你的倍率', 'Your rate') }} ×{{ catalog.group.user_rate_multiplier }}
              </span>
              <span class="rounded-full bg-[var(--atelier-subtle)] px-2 py-0.5 text-[var(--atelier-ink-muted)]">
                {{ catalog.group.subscription_type === 'subscription' ? text('套餐计费', 'Subscription billing') : text('余额计费', 'Balance billing') }}
              </span>
              <span class="font-mono tabular-nums text-[var(--atelier-ink-muted)]">
                {{ catalog.models.length }} {{ text('个模型', 'models') }}
              </span>
            </div>
          </div>

          <div class="relative w-full shrink-0 sm:w-64">
            <Icon name="search" size="xs" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" aria-hidden="true" />
            <input
              id="chat-model-search"
              v-model="search"
              type="search"
              class="atelier-input !min-h-[36px] !pl-8 text-xs"
              :aria-label="text('搜索模型', 'Search models')"
              :placeholder="text('搜索此分组的模型', 'Search models in this group')"
              :disabled="!selectedGroup"
              maxlength="256"
            />
          </div>
        </header>

        <div
          v-if="editing"
          class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] px-4 py-2.5 text-xs"
          role="status"
        >
          <span>{{ text('修改绑定：', 'Rebind: ') }}<strong class="font-semibold text-[var(--atelier-ink)]">{{ editing.display_name || editing.model }}</strong></span>
          <UiButton variant="secondary" size="sm" @click="editingId = null">{{ text('取消', 'Cancel') }}</UiButton>
        </div>

        <div :aria-busy="catalogLoading">
          <div v-if="!selectedGroup" class="atelier-surface py-14 text-center text-sm text-[var(--atelier-ink-muted)]">
            {{ text('选择分组，查看模型和价格。', 'Choose a group to see its models and prices.') }}
          </div>
          <p v-else-if="catalogLoading" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]" role="status">
            {{ text('加载目录…', 'Loading catalog…') }}
          </p>
          <div
            v-else-if="catalogError"
            class="flex items-center justify-between gap-2 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-xs text-[var(--atelier-err)]"
            role="alert"
          >
            <span>{{ catalogError }}</span>
            <UiButton variant="secondary" size="sm" @click="loadCatalog">{{ text('重试', 'Retry') }}</UiButton>
          </div>
          <p v-else-if="catalog && !catalog.models.length" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]">
            {{ text('没有符合条件的聊天模型。可调整搜索或选择其他分组。', 'No compatible chat models. Change the search or select another group.') }}
          </p>

          <ul v-else-if="catalog?.models?.length" class="atelier-surface divide-y divide-[var(--atelier-border-subtle)] overflow-hidden">
            <li
              v-for="model in catalog.models"
              :key="model.id"
              class="flex flex-col gap-3 px-4 py-3.5 transition-colors hover:bg-[var(--atelier-hover)] sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="flex min-w-0 items-start gap-3">
                <BrandModelIcon :model="model.id" size="md" class="mt-0.5" />
                <div class="min-w-0 space-y-1">
                  <strong class="block truncate text-sm font-semibold text-[var(--atelier-ink)]">{{ model.id }}</strong>
                  <p class="font-mono text-xs tabular-nums text-[var(--atelier-ink-secondary)]">{{ price(model) }}</p>
                  <div class="flex flex-wrap items-center gap-1.5 text-[11px]">
                    <span class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[var(--atelier-ink-secondary)]">
                      {{ model.capabilities.transport === 'messages' ? 'Messages' : model.capabilities.transport === 'gemini' ? 'Gemini v1beta' : model.capabilities.transport === 'responses' ? 'Responses' : 'Chat Completions' }}
                    </span>
                    <span v-if="model.capabilities.image_input" class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[var(--atelier-ink-muted)]">{{ text('图片', 'Images') }}</span>
                    <span v-if="model.capabilities.document_input" class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[var(--atelier-ink-muted)]">{{ text('文档', 'Files') }}</span>
                    <span v-if="model.capabilities.reasoning" class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-medium text-[var(--atelier-ink)]">{{ text('思考', 'Reasoning') }}</span>
                    <span v-if="model.capabilities.search" class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 text-[var(--atelier-ink-muted)]">{{ text('搜索', 'Search') }}</span>
                  </div>
                  <p v-if="!model.available" class="text-xs text-[var(--atelier-err)]">{{ model.unavailable_reason }}</p>
                </div>
              </div>

              <UiButton
                :variant="!editingId && alreadyAdded(model.id) ? 'secondary' : 'primary'"
                size="sm"
                class="shrink-0"
                :disabled="saving || !model.available || (!editingId && alreadyAdded(model.id))"
                @click="add(model)"
              >
                {{ editingId ? text('绑定', 'Bind') : alreadyAdded(model.id) ? text('已添加', 'Added') : text('添加常用', 'Add model') }}
              </UiButton>
            </li>
          </ul>
        </div>

        <details class="rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-4 py-2.5 text-xs">
          <summary class="cursor-pointer font-medium text-[var(--atelier-ink)]">{{ text('价格与可用性说明', 'Pricing and availability') }}</summary>
          <p class="mt-2 leading-relaxed text-[var(--atelier-ink-secondary)]">
            {{ text('价格为标准时段最低档，已应用你的倍率。阶梯、时段、缓存及工具费用以实际账单为准。目录表示配置兼容性，不是实时健康保证。', 'Lowest standard-period prices with your rate applied. Tiers, time periods, cache and tools follow actual billing. Compatibility is not a live health guarantee.') }}
          </p>
        </details>
      </div>
    </section>
  </div>
</template>

<style scoped>
@media (max-width: 1023px) {
  .chat-group-options.is-collapsed {
    display: none;
  }
}
</style>
