<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-80">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('modelSquare.searchPlaceholder')"
                class="input pl-10"
              />
            </div>

            <select
              v-model="selectedGroupId"
              class="input w-full sm:w-64"
              :disabled="loading || availableGroups.length === 0"
            >
              <option :value="null">{{ t('modelSquare.lowestRate') }}</option>
              <option v-for="group in availableGroups" :key="group.id" :value="group.id">
                {{ group.name }} ({{ group.rate_multiplier }}x)
              </option>
            </select>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              @click="loadModels"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh', 'Refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="loading" class="flex items-center justify-center py-12">
          <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
        </div>

        <div v-else-if="filteredModels.length === 0" class="py-12 text-center text-gray-500 dark:text-gray-400">
          {{ searchQuery ? t('modelSquare.noResults') : t('modelSquare.empty') }}
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelSquare.columns.model') }}
                </th>
                <th class="px-4 py-3 text-left text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelSquare.columns.provider') }}
                </th>
                <th class="px-4 py-3 text-right text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelSquare.columns.inputPrice') }}
                </th>
                <th class="px-4 py-3 text-right text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelSquare.columns.outputPrice') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-600 dark:bg-dark-900">
              <template v-for="group in groupedModels" :key="group.provider">
                <tr v-for="(model, idx) in group.models" :key="model.name"
                    class="hover:bg-gray-50 dark:hover:bg-dark-800">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <code class="rounded bg-gray-100 px-2 py-0.5 text-sm font-mono text-gray-800 dark:bg-dark-700 dark:text-gray-200">
                        {{ model.name }}
                      </code>
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <span v-if="idx === 0" class="text-sm text-gray-700 dark:text-gray-300">
                      {{ group.provider }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-right font-mono text-sm">
                    <div class="flex flex-col items-end gap-0.5">
                      <span class="text-gray-700 dark:text-gray-300">
                        {{ formatPrice(model.input) }}
                      </span>
                      <span v-if="selectedGroupName" class="text-xs text-blue-600 dark:text-blue-400">
                        {{ t('modelSquare.groupRate', { group: selectedGroupName, rate: effectiveMultiplier.toFixed(2) }) }}
                      </span>
                      <span v-else-if="hasCustomRate" class="text-xs text-green-600 dark:text-green-400">
                        {{ t('modelSquare.lowestRateApplied', { rate: effectiveMultiplier.toFixed(2) }) }}
                      </span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-right font-mono text-sm">
                    <div class="flex flex-col items-end gap-0.5">
                      <span class="text-gray-700 dark:text-gray-300">
                        {{ formatPrice(model.output) }}
                      </span>
                      <span v-if="selectedGroupName" class="text-xs text-blue-600 dark:text-blue-400">
                        {{ t('modelSquare.groupRate', { group: selectedGroupName, rate: effectiveMultiplier.toFixed(2) }) }}
                      </span>
                      <span v-else-if="hasCustomRate" class="text-xs text-green-600 dark:text-green-400">
                        {{ t('modelSquare.lowestRateApplied', { rate: effectiveMultiplier.toFixed(2) }) }}
                      </span>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Group } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

// Target models with fallback pricing (USD per million tokens)
const TARGET_MODELS = {
  openai: [
    { name: 'gpt-5.5', input: 5.0, output: 15.0 },
    { name: 'gpt-5.4', input: 2.5, output: 10.0 },
    { name: 'gpt-5.4-mini', input: 0.15, output: 0.6 },
    { name: 'codex-auto-review', input: 5.0, output: 15.0 }
  ],
  anthropic: [
    { name: 'claude-haiku-4-5', input: 1.0, output: 5.0 },
    { name: 'claude-haiku-4-5-20251001', input: 1.0, output: 5.0 },
    { name: 'claude-sonnet-4-6', input: 3.0, output: 15.0 },
    { name: 'claude-opus-4-8', input: 5.0, output: 25.0 }
  ]
}

interface ModelPricing {
  name: string
  provider: string
  input: number
  output: number
}

const availableGroups = ref<Group[]>([])
const userGroupRates = ref<Record<number, number>>({})
const selectedGroupId = ref<number | null>(null)
const loading = ref(false)
const searchQuery = ref('')

// Compute effective multiplier based on selection
const effectiveMultiplier = computed(() => {
  if (selectedGroupId.value !== null) {
    // Specific group selected
    const group = availableGroups.value.find(g => g.id === selectedGroupId.value)
    if (group) {
      return userGroupRates.value[group.id] ?? group.rate_multiplier
    }
  }

  // No selection: use lowest rate among user's groups
  const rates = availableGroups.value.map(g =>
    userGroupRates.value[g.id] ?? g.rate_multiplier
  )
  return rates.length > 0 ? Math.min(...rates) : 1.0
})

const hasCustomRate = computed(() => {
  return Object.keys(userGroupRates.value).length > 0
})

const selectedGroupName = computed(() => {
  if (selectedGroupId.value === null) return null
  const group = availableGroups.value.find(g => g.id === selectedGroupId.value)
  return group?.name || null
})

// Extract pricing for target models (using fallback values)
const allModels = computed<ModelPricing[]>(() => {
  const models: ModelPricing[] = []

  // OpenAI models
  TARGET_MODELS.openai.forEach(model => {
    models.push({
      name: model.name,
      provider: 'OpenAI',
      input: model.input * effectiveMultiplier.value,
      output: model.output * effectiveMultiplier.value
    })
  })

  // Anthropic models
  TARGET_MODELS.anthropic.forEach(model => {
    models.push({
      name: model.name,
      provider: 'Anthropic',
      input: model.input * effectiveMultiplier.value,
      output: model.output * effectiveMultiplier.value
    })
  })

  return models
})

// Filter by search query
const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return allModels.value
  return allModels.value.filter(m =>
    m.name.toLowerCase().includes(q) ||
    m.provider.toLowerCase().includes(q)
  )
})

// Group by provider
const groupedModels = computed(() => {
  const groups: Record<string, ModelPricing[]> = {}
  filteredModels.value.forEach(model => {
    if (!groups[model.provider]) {
      groups[model.provider] = []
    }
    groups[model.provider].push(model)
  })
  return Object.entries(groups).map(([provider, models]) => ({
    provider,
    models
  }))
})

// Format price as ¥X.XX / 1M tokens
function formatPrice(price: number): string {
  return `¥${price.toFixed(2)}`
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
