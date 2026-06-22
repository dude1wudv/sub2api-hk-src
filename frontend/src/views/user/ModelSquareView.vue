<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] px-4 py-6 sm:px-6 lg:px-8">
      <!-- Header -->
      <div class="mb-6">
        <div class="mb-4 rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 p-8 text-white shadow-lg">
          <h1 class="mb-2 text-3xl font-bold">{{ t('modelSquare.title') }}</h1>
          <p class="text-blue-100">{{ t('modelSquare.description') }}</p>
          <div class="mt-4 inline-flex items-center gap-2 rounded-lg bg-white/20 px-4 py-2 backdrop-blur-sm">
            <span class="text-sm font-medium">{{ t('modelSquare.totalModels') }}</span>
            <span class="rounded-full bg-white/30 px-3 py-1 text-lg font-bold">{{ totalModelCount }}</span>
          </div>
        </div>
      </div>

      <div class="flex gap-6">
        <!-- Left Sidebar -->
        <aside class="w-64 flex-shrink-0">
          <div class="sticky top-6 space-y-4">
            <!-- Search -->
            <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-900">
              <div class="relative">
                <Icon
                  name="search"
                  size="md"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
                />
                <input
                  v-model="searchQuery"
                  type="text"
                  :placeholder="t('modelSquare.searchPlaceholder')"
                  class="w-full rounded-lg border border-gray-300 bg-white py-2 pl-10 pr-3 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-dark-600 dark:bg-dark-800 dark:text-white"
                />
              </div>
            </div>

            <!-- Provider Filter -->
            <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-900">
              <h3 class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">{{ t('modelSquare.provider') }}</h3>
              <div class="space-y-2">
                <button
                  @click="selectedProvider = null"
                  :class="[
                    'flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    selectedProvider === null
                      ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                      : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
                  ]"
                >
                  <span>{{ t('modelSquare.allProviders') }}</span>
                  <span class="ml-auto text-xs text-gray-500 dark:text-gray-400">{{ totalModelCount }}</span>
                </button>
                <button
                  @click="selectedProvider = 'openai'"
                  :class="[
                    'flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    selectedProvider === 'openai'
                      ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                      : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
                  ]"
                >
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/>
                  </svg>
                  <span>OpenAI</span>
                  <span class="ml-auto text-xs text-gray-500 dark:text-gray-400">{{ TARGET_MODELS.openai.length }}</span>
                </button>
                <button
                  @click="selectedProvider = 'anthropic'"
                  :class="[
                    'flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    selectedProvider === 'anthropic'
                      ? 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                      : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
                  ]"
                >
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M17.5 12l-5 5-5-5m10-7l-5 5-5-5"/>
                  </svg>
                  <span>Anthropic</span>
                  <span class="ml-auto text-xs text-gray-500 dark:text-gray-400">{{ TARGET_MODELS.anthropic.length }}</span>
                </button>
                <button
                  @click="selectedProvider = 'image'"
                  :class="[
                    'flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    selectedProvider === 'image'
                      ? 'bg-purple-50 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                      : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
                  ]"
                >
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                  </svg>
                  <span>Image</span>
                  <span class="ml-auto text-xs text-gray-500 dark:text-gray-400">{{ TARGET_MODELS.image.length }}</span>
                </button>
              </div>
            </div>

            <!-- Group Filter -->
            <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-900">
              <h3 class="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">{{ t('modelSquare.groups') }}</h3>
              <div class="space-y-2">
                <button
                  @click="selectedGroupId = null"
                  :class="[
                    'flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    selectedGroupId === null
                      ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                      : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
                  ]"
                >
                  <span>{{ t('modelSquare.lowestRateLabel') }}</span>
                  <span class="text-xs font-semibold">{{ getLowestRate().toFixed(2) }}x</span>
                </button>
                <button
                  v-for="group in filteredGroups"
                  :key="group.id"
                  @click="selectedGroupId = group.id"
                  :class="[
                    'flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm transition-colors',
                    selectedGroupId === group.id
                      ? 'bg-blue-50 font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                      : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
                  ]"
                >
                  <span class="truncate">{{ group.name }}</span>
                  <span class="ml-2 text-xs font-semibold">{{ (userGroupRates[group.id] ?? group.rate_multiplier).toFixed(2) }}x</span>
                </button>
              </div>
            </div>
          </div>
        </aside>

        <!-- Main Content -->
        <main class="flex-1">
          <!-- Loading State -->
          <div v-if="loading" class="flex items-center justify-center py-20">
            <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
          </div>

          <!-- Empty State -->
          <div v-else-if="displayedModels.length === 0"
               class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-20 dark:border-dark-600">
            <Icon name="search" size="xl" class="mb-3 text-gray-400" />
            <p class="text-gray-500 dark:text-gray-400">{{ t('modelSquare.noResults') }}</p>
          </div>

          <!-- Model Cards Grid -->
          <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div
              v-for="model in displayedModels"
              :key="model.name"
              class="group relative overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition-all hover:shadow-lg dark:border-dark-600 dark:bg-dark-900"
            >
              <!-- Provider Icon -->
              <div :class="[
                'absolute right-3 top-3 flex h-10 w-10 items-center justify-center rounded-lg',
                model.provider === 'OpenAI'
                  ? 'bg-emerald-100 dark:bg-emerald-900/30'
                  : model.provider === 'Anthropic'
                  ? 'bg-amber-100 dark:bg-amber-900/30'
                  : 'bg-purple-100 dark:bg-purple-900/30'
              ]">
                <svg v-if="model.provider === 'OpenAI'" class="h-5 w-5 text-emerald-600 dark:text-emerald-400" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/>
                </svg>
                <svg v-else-if="model.provider === 'Anthropic'" class="h-5 w-5 text-amber-600 dark:text-amber-400" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M17.5 12l-5 5-5-5m10-7l-5 5-5-5"/>
                </svg>
                <svg v-else class="h-5 w-5 text-purple-600 dark:text-purple-400" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                </svg>
              </div>

              <div class="p-5">
                <!-- Model Name -->
                <h3 class="mb-3 pr-12 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ model.name }}
                </h3>

                <!-- Pricing -->
                <div class="space-y-2.5">
                  <div v-if="model.isPerCall" class="rounded-lg bg-green-50 p-3 dark:bg-green-900/20">
                    <div class="text-center">
                      <div class="text-sm font-medium text-green-700 dark:text-green-400">{{ t('modelSquare.perCallPrice') }}</div>
                      <div class="mt-1 text-2xl font-bold text-green-900 dark:text-green-300">¥0.01</div>
                    </div>
                  </div>
                  <template v-else-if="model.pricePerImage">
                    <!-- Image model: per-image pricing -->
                    <div class="rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
                      <div class="text-center">
                        <div class="text-sm font-medium text-purple-700 dark:text-purple-400">{{ t('modelSquare.perImagePrice') }}</div>
                        <div class="mt-1 text-2xl font-bold text-purple-900 dark:text-purple-300">{{ formatImagePrice(model.pricePerImage) }}</div>
                      </div>
                    </div>
                  </template>
                  <template v-else>
                    <div v-if="model.input > 0" class="flex items-baseline justify-between">
                      <span class="text-sm text-gray-600 dark:text-gray-400">{{ t('modelSquare.inputPrice') }}</span>
                      <span class="text-base font-bold text-gray-900 dark:text-white">{{ formatPrice(model.input) }}</span>
                    </div>
                    <div v-if="model.output > 0" class="flex items-baseline justify-between">
                      <span class="text-sm text-gray-600 dark:text-gray-400">{{ t('modelSquare.outputPrice') }}</span>
                      <span class="text-base font-bold text-gray-900 dark:text-white">{{ formatPrice(model.output) }}</span>
                    </div>
                    <div v-if="model.cacheWrite && model.cacheWrite > 0" class="flex items-baseline justify-between">
                      <span class="text-sm text-gray-600 dark:text-gray-400">{{ t('modelSquare.cacheWritePrice') }}</span>
                      <span class="text-base font-bold text-gray-900 dark:text-white">{{ formatPrice(model.cacheWrite) }}</span>
                    </div>
                    <div v-if="model.cacheRead > 0" class="flex items-baseline justify-between">
                      <span class="text-sm text-gray-600 dark:text-gray-400">{{ t('modelSquare.cacheReadPrice') }}</span>
                      <span class="text-base font-bold text-gray-900 dark:text-white">{{ formatPrice(model.cacheRead) }}</span>
                    </div>
                  </template>
                </div>

                <!-- Group Badge -->
                <div class="mt-4 flex items-center gap-2 border-t border-gray-100 pt-3 dark:border-dark-700">
                  <span class="inline-flex items-center gap-1.5 rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                    <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
                    </svg>
                    {{ model.groupLabel }}
                  </span>
                </div>
              </div>

              <!-- Hover Effect Border -->
              <div :class="[
                'absolute inset-0 rounded-xl opacity-0 transition-opacity group-hover:opacity-100',
                model.provider === 'OpenAI'
                  ? 'ring-2 ring-emerald-500/50'
                  : model.provider === 'Anthropic'
                  ? 'ring-2 ring-amber-500/50'
                  : 'ring-2 ring-purple-500/50'
              ]"></div>
            </div>
          </div>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Group } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

// Target models with correct base pricing (USD per million tokens)
const TARGET_MODELS = {
  openai: [
    { name: 'gpt-5.5', input: 5.0, output: 30.0, cacheRead: 0.5 },
    { name: 'gpt-5.4', input: 2.5, output: 15.0, cacheRead: 0.25 },
    { name: 'gpt-5.4-mini', input: 0.75, output: 4.5, cacheRead: 0.075 },
    { name: 'codex-auto-review', input: 5.0, output: 30.0, cacheRead: 0.5 }
  ],
  anthropic: [
    { name: 'claude-haiku-4-5', input: 1.0, output: 5.0, cacheWrite: 1.25, cacheRead: 0.1 },
    { name: 'claude-haiku-4-5-20251001', input: 1.0, output: 5.0, cacheWrite: 1.25, cacheRead: 0.1 },
    { name: 'claude-sonnet-4-6', input: 3.0, output: 15.0, cacheWrite: 3.75, cacheRead: 0.3 },
    { name: 'claude-opus-4-8', input: 5.0, output: 25.0, cacheWrite: 6.25, cacheRead: 1.0 }
  ],
  image: [
    { name: 'image2', pricePerImage: 0.05 },
    { name: 'image2 2k', pricePerImage: 0.1 },
    { name: 'image2 4k', pricePerImage: 0.2 }
  ]
}

interface ModelPricing {
  name: string
  provider: string
  input: number
  output: number
  cacheRead: number
  cacheWrite?: number
  pricePerImage?: number
  groupLabel: string
  isPerCall?: boolean
}

const availableGroups = ref<Group[]>([])
const userGroupRates = ref<Record<number, number>>({})
const selectedProvider = ref<string | null>(null)
const selectedGroupId = ref<number | null>(null)
const loading = ref(false)
const searchQuery = ref('')

const totalModelCount = computed(() =>
  TARGET_MODELS.openai.length + TARGET_MODELS.anthropic.length + TARGET_MODELS.image.length
)

// Group name patterns for platform binding
const OPENAI_GROUP_PATTERNS = ['openai', 'gpt', 'plus', 'pro', '自用', '高缓存']
const ANTHROPIC_GROUP_PATTERNS = ['anthropic', 'claude', 'kiro', '满血', 'ccmax']
const IMAGE_GROUP_PATTERNS = ['image']
const PER_CALL_PATTERNS = ['按次', '0.01']

// Classify groups by platform
const openAIGroups = computed(() => {
  return availableGroups.value.filter(g =>
    OPENAI_GROUP_PATTERNS.some(p => g.name.toLowerCase().includes(p.toLowerCase()))
  )
})

const anthropicGroups = computed(() => {
  return availableGroups.value.filter(g =>
    ANTHROPIC_GROUP_PATTERNS.some(p => g.name.toLowerCase().includes(p.toLowerCase()))
  )
})

const imageGroups = computed(() => {
  return availableGroups.value.filter(g =>
    IMAGE_GROUP_PATTERNS.some(p => g.name.toLowerCase().includes(p.toLowerCase()))
  )
})

const filteredGroups = computed(() => {
  if (!selectedProvider.value) return availableGroups.value
  if (selectedProvider.value === 'openai') return openAIGroups.value
  if (selectedProvider.value === 'anthropic') return anthropicGroups.value
  if (selectedProvider.value === 'image') return imageGroups.value
  return availableGroups.value
})

// Check if a group is per-call billing
function isPerCallGroup(groupId: number | null): boolean {
  if (groupId === null) return false
  const group = availableGroups.value.find(g => g.id === groupId)
  if (!group) return false
  return PER_CALL_PATTERNS.some(p => group.name.toLowerCase().includes(p.toLowerCase()))
}

// Determine which platform(s) to show based on selected group
function getPlatformsForGroup(groupId: number | null): string[] {
  if (groupId === null) {
    // No group selected: show all platforms
    return ['openai', 'anthropic', 'image']
  }

  const group = availableGroups.value.find(g => g.id === groupId)
  if (!group) return ['openai', 'anthropic', 'image']

  const name = group.name.toLowerCase()

  if (OPENAI_GROUP_PATTERNS.some(p => name.includes(p.toLowerCase()))) {
    return ['openai']
  }
  if (ANTHROPIC_GROUP_PATTERNS.some(p => name.includes(p.toLowerCase()))) {
    return ['anthropic']
  }
  if (IMAGE_GROUP_PATTERNS.some(p => name.includes(p.toLowerCase()))) {
    return ['image']
  }

  return ['openai', 'anthropic', 'image']
}

// Get lowest rate for each platform
const lowestOpenAIRate = computed(() => {
  if (openAIGroups.value.length === 0) return 1.0
  const rates = openAIGroups.value.map(g => userGroupRates.value[g.id] ?? g.rate_multiplier)
  return Math.min(...rates)
})

const lowestAnthropicRate = computed(() => {
  if (anthropicGroups.value.length === 0) return 1.0
  const rates = anthropicGroups.value.map(g => userGroupRates.value[g.id] ?? g.rate_multiplier)
  return Math.min(...rates)
})

const lowestImageRate = computed(() => {
  if (imageGroups.value.length === 0) return 1.0
  const rates = imageGroups.value.map(g => userGroupRates.value[g.id] ?? g.rate_multiplier)
  return Math.min(...rates)
})

// Get lowest rate for current filter
function getLowestRate(): number {
  if (selectedProvider.value === 'openai') return lowestOpenAIRate.value
  if (selectedProvider.value === 'anthropic') return lowestAnthropicRate.value
  if (selectedProvider.value === 'image') return lowestImageRate.value
  return Math.min(lowestOpenAIRate.value, lowestAnthropicRate.value, lowestImageRate.value)
}

// Get effective multiplier and label for a specific provider
function getMultiplierInfo(provider: 'openai' | 'anthropic' | 'image') {
  if (selectedGroupId.value !== null) {
    const group = availableGroups.value.find(g => g.id === selectedGroupId.value)
    if (group) {
      const rate = userGroupRates.value[group.id] ?? group.rate_multiplier
      const isPerCall = isPerCallGroup(selectedGroupId.value)
      return {
        multiplier: rate,
        label: `${group.name} - ${rate.toFixed(2)}x`,
        isPerCall
      }
    }
  }

  let rate = 1.0
  if (provider === 'openai') rate = lowestOpenAIRate.value
  else if (provider === 'anthropic') rate = lowestAnthropicRate.value
  else if (provider === 'image') rate = lowestImageRate.value

  return { multiplier: rate, label: `${t('modelSquare.lowestRateLabel')} ${rate.toFixed(2)}x`, isPerCall: false }
}

// All models with pricing
const allModels = computed<ModelPricing[]>(() => {
  const models: ModelPricing[] = []
  const visiblePlatforms = getPlatformsForGroup(selectedGroupId.value)

  // Add OpenAI models
  if (visiblePlatforms.includes('openai') && (!selectedProvider.value || selectedProvider.value === 'openai')) {
    const { multiplier, label, isPerCall } = getMultiplierInfo('openai')
    TARGET_MODELS.openai.forEach(model => {
      if (isPerCall) {
        // Per-call billing: fixed 0.01 for all prices
        models.push({
          name: model.name,
          provider: 'OpenAI',
          input: 0.01,
          output: 0.01,
          cacheRead: 0.01,
          groupLabel: label,
          isPerCall: true
        })
      } else {
        models.push({
          name: model.name,
          provider: 'OpenAI',
          input: model.input * multiplier,
          output: model.output * multiplier,
          cacheRead: model.cacheRead * multiplier,
          groupLabel: label,
          isPerCall: false
        })
      }
    })
  }

  // Add Anthropic models
  if (visiblePlatforms.includes('anthropic') && (!selectedProvider.value || selectedProvider.value === 'anthropic')) {
    const { multiplier, label, isPerCall } = getMultiplierInfo('anthropic')
    TARGET_MODELS.anthropic.forEach(model => {
      if (isPerCall) {
        // Per-call billing: fixed 0.01 for all prices
        models.push({
          name: model.name,
          provider: 'Anthropic',
          input: 0.01,
          output: 0.01,
          cacheRead: 0.01,
          cacheWrite: 0.01,
          groupLabel: label,
          isPerCall: true
        })
      } else {
        models.push({
          name: model.name,
          provider: 'Anthropic',
          input: model.input * multiplier,
          output: model.output * multiplier,
          cacheRead: model.cacheRead * multiplier,
          cacheWrite: model.cacheWrite * multiplier,
          groupLabel: label,
          isPerCall: false
        })
      }
    })
  }

  // Add Image models
  if (visiblePlatforms.includes('image') && (!selectedProvider.value || selectedProvider.value === 'image')) {
    const { multiplier, label } = getMultiplierInfo('image')
    TARGET_MODELS.image.forEach(model => {
      models.push({
        name: model.name,
        provider: 'Image',
        input: 0,
        output: 0,
        cacheRead: 0,
        pricePerImage: model.pricePerImage * multiplier,
        groupLabel: label,
        isPerCall: false
      })
    })
  }

  return models
})

// Filter by search query
const displayedModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return allModels.value
  return allModels.value.filter(m => m.name.toLowerCase().includes(q))
})

// Format price as ¥X.XX / 1M tokens
function formatPrice(price: number): string {
  return `¥${price.toFixed(4)} / 1M tokens`
}

// Format image price as ¥X.XX / 张
function formatImagePrice(price: number): string {
  return `¥${price.toFixed(2)} / 张`
}

async function loadModels() {
  loading.value = true
  try {
    const [groups, rates] = await Promise.all([
      userGroupsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    availableGroups.value = groups
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadModels)
</script>
