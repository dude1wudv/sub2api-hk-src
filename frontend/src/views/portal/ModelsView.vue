<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { BrandMark, UiButton, UiTabs, type UnderlineTab } from '@/ui'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { useAuthStore, useAppStore } from '@/stores'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { matchesCatalogModel } from '@/utils/modelCatalog'
import { useColorMode } from '@/composables/useColorMode'
import PortalApiModels from '@/components/portal/PortalApiModels.vue'
import ModelCostCalculator from '@/components/portal/ModelCostCalculator.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const AvailableChannelsView = defineAsyncComponent(() => import('@/views/user/AvailableChannelsView.vue'))
const ChatModelsView = defineAsyncComponent(() => import('@/views/portal/ChatModelsView.vue'))

const route = useRoute()
const auth = useAuthStore()
const app = useAppStore()
const { locale } = useI18n()
const { isDark, toggleMode } = useColorMode()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)

const embedded = computed(() => route.query.embedded === '1' && auth.isAuthenticated)
const enabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.modelPlaza))
const channelsEnabled = computed(() => auth.isAuthenticated && resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.availableChannels))

const data = ref<ModelPlazaResponse | null>(null)
const loading = ref(false)
const error = ref('')
const groupId = ref<number | null>(null)
const search = ref(typeof route.query.model === 'string' ? route.query.model : '')
const activeTab = ref(
  route.query.tab === 'channels' && channelsEnabled.value
    ? 'channels'
    : route.query.tab === 'chat' && auth.isAuthenticated
      ? 'chat'
      : 'catalog'
)

const estimatorOpen = ref(route.query.tab === 'calculator')
const comparedKeys = ref<string[]>([])

const tabs = computed<UnderlineTab[]>(() => {
  const list: UnderlineTab[] = [
    { value: 'catalog', label: text('定价目录', 'Pricing catalog') }
  ]
  if (auth.isAuthenticated) {
    list.push({ value: 'chat', label: text('对话模型', 'Chat models') })
  }
  if (channelsEnabled.value) {
    list.push({ value: 'channels', label: text('可用渠道与倍率', 'Available channels') })
  }
  return list
})

function toggleCompare(key: string) {
  if (comparedKeys.value.includes(key)) {
    comparedKeys.value = comparedKeys.value.filter((k) => k !== key)
  } else {
    comparedKeys.value = [...comparedKeys.value, key]
    estimatorOpen.value = true
  }
}

function addCompare(key: string) {
  if (!comparedKeys.value.includes(key)) {
    comparedKeys.value = [...comparedKeys.value, key]
  }
}

function removeCompare(key: string) {
  comparedKeys.value = comparedKeys.value.filter((k) => k !== key)
}

function clearCompare() {
  comparedKeys.value = []
}

let request: AbortController | undefined
async function load() {
  request?.abort()
  const current = new AbortController()
  request = current
  loading.value = true
  error.value = ''
  try {
    if (!app.publicSettingsLoaded) await app.fetchPublicSettings()
    if (current.signal.aborted || !enabled.value) return
    const result = await getModelPlaza({ signal: current.signal })
    if (!current.signal.aborted) data.value = result
  } catch {
    if (!current.signal.aborted) error.value = text('模型价格加载失败，请重试。', 'Could not load model pricing. Please retry.')
  } finally {
    if (!current.signal.aborted) loading.value = false
  }
}

watch(() => route.query.model, (value) => { search.value = typeof value === 'string' ? value : '' })
watch(() => route.query.tab, (value) => {
  if (value === 'calculator') estimatorOpen.value = true
  else if (value === 'channels' && channelsEnabled.value) activeTab.value = 'channels'
  else if (value === 'chat' && auth.isAuthenticated) activeTab.value = 'chat'
})
watch([search, data], () => {
  const query = search.value.trim().toLowerCase()
  if (!query) return
  const matching = (data.value?.groups || []).filter((group) => group.models.some((model) => matchesCatalogModel(model, query)))
  if (matching.length && !matching.some((group) => group.id === groupId.value)) groupId.value = matching[0].id
})

onMounted(load)
onBeforeUnmount(() => request?.abort())
</script>

<template>
  <div :class="embedded ? 'space-y-6' : 'min-h-screen bg-[var(--atelier-canvas)]'">
    <!-- Standalone Public Header (only when not embedded in UserShell) -->
    <header
      v-if="!embedded"
      class="border-b border-[var(--atelier-border)] bg-[var(--atelier-surface)]"
    >
      <div class="mx-auto flex h-[52px] max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
        <RouterLink to="/home" class="inline-flex items-center gap-2 text-sm font-semibold tracking-tight text-[var(--atelier-ink)]" aria-label="Home">
          <BrandMark :size="20" />
          <span>{{ app.siteName || 'Sub2API' }}</span>
        </RouterLink>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="atelier-btn atelier-btn-ghost h-8 w-8 p-0"
            :aria-label="isDark ? text('切换至浅色', 'Switch to light mode') : text('切换至深色', 'Switch to dark mode')"
            @click="toggleMode"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="xs" aria-hidden="true" />
          </button>
          <LocaleSwitcher />
          <RouterLink to="/keys" class="atelier-btn atelier-btn-primary h-8 px-3 text-xs">
            {{ text('API 密钥', 'API keys') }} →
          </RouterLink>
        </div>
      </div>
    </header>

    <div :class="embedded ? 'space-y-6' : 'mx-auto max-w-6xl space-y-6 px-4 py-7 sm:px-6'">
      <!-- Page Header -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
            {{ text('模型与价格', 'Models & pricing') }}
          </h1>
          <p class="mt-0.5 text-sm text-[var(--atelier-ink-secondary)]">
            {{ text('浏览所有接入分组的模型单价、能力矩阵，并随时打开费用估算抽屉横向对比。', 'Explore per-group token pricing, capabilities, and interactive cost estimation.') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UiButton variant="secondary" size="sm" @click="estimatorOpen = true">
            <Icon name="calculator" size="xs" aria-hidden="true" />
            <span>{{ text('费用估算', 'Cost estimator') }}</span>
            <span
              v-if="comparedKeys.length"
              class="ml-0.5 rounded-full bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-[var(--atelier-ink)]"
            >
              {{ comparedKeys.length }}
            </span>
          </UiButton>
          <RouterLink v-if="auth.isAuthenticated" to="/playground" class="atelier-btn atelier-btn-secondary h-8 px-3 text-xs">
            {{ text('在线调试', 'Playground') }}
          </RouterLink>
          <RouterLink v-if="auth.isAuthenticated" to="/keys?create=1" class="atelier-btn atelier-btn-primary h-8 px-3 text-xs">
            {{ text('创建密钥', 'Create a key') }} ↗
          </RouterLink>
        </div>
      </div>

      <!-- Primary In-Page Underline Tabs (UiTabs — NO 3-layer pill stacking) -->
      <UiTabs
        v-model="activeTab"
        :tabs="tabs"
        :aria-label="text('模型广场视图', 'Model plaza views')"
      />

      <!-- Tab Content -->
      <PortalApiModels
        v-if="activeTab === 'catalog'"
        v-model:selected-group-id="groupId"
        v-model:search="search"
        :groups="data?.groups || []"
        :description="data?.description || ''"
        :pricing-enabled="enabled"
        :loading="loading"
        :error="error"
        :server-utc-offset="app.cachedPublicSettings?.server_utc_offset"
        :compared-keys="comparedKeys"
        @retry="load"
        @toggle-compare="toggleCompare"
      />
      <ChatModelsView v-else-if="activeTab === 'chat' && auth.isAuthenticated" />
      <AvailableChannelsView v-else-if="activeTab === 'channels' && channelsEnabled" />
    </div>

    <!-- On-Demand Slide-Over Interactive Cost Estimator (UiDrawer — NO dot-matrix texture) -->
    <ModelCostCalculator
      :open="estimatorOpen"
      :groups="data?.groups || []"
      :compared-keys="comparedKeys"
      @close="estimatorOpen = false"
      @add-compare="addCompare"
      @remove-compare="removeCompare"
      @clear-compare="clearCompare"
    />
  </div>
</template>
