<template>
  <div class="min-w-0 max-w-full bg-gray-50/80 dark:bg-dark-900/45">
    <div v-if="loading" class="flex items-center justify-center py-3">
      <LoadingSpinner />
    </div>
    <div v-else-if="items.length === 0" class="py-2 text-center text-xs text-gray-400 dark:text-dark-400">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
    <table v-else class="w-full text-xs">
      <tbody>
        <tr
          v-for="user in displayItems"
          :key="user.user_id"
          class="border-t border-gray-200/70 dark:border-dark-700/70"
        >
          <td class="distribution-name py-1.5 pl-6 text-gray-700 dark:text-dark-200" :title="user.email">
            {{ user.email || `User #${user.user_id}` }}
          </td>
          <td class="py-1.5 text-right font-mono tabular-nums text-gray-600 dark:text-dark-200">
            {{ user.requests.toLocaleString() }}
          </td>
          <td class="py-1.5 text-right font-mono tabular-nums text-gray-600 dark:text-dark-200">
            {{ formatTokens(user.total_tokens) }}
          </td>
          <td class="py-1.5 text-right font-mono font-medium tabular-nums text-primary-700 dark:text-primary-300">
            ${{ formatCost(user.actual_cost) }}
          </td>
          <td v-if="showAccountCost" class="py-1.5 text-right font-mono tabular-nums text-amber-600 dark:text-amber-400">
            ${{ formatCost(user.account_cost) }}
          </td>
          <td class="py-1.5 pr-1 text-right font-mono tabular-nums text-gray-400 dark:text-dark-400">
            ${{ formatCost(user.cost) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { UserBreakdownItem } from '@/types'
import { distributionValue, sortDescending, type DistributionMetric } from '@/utils/chartDistribution'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  items: UserBreakdownItem[]
  loading?: boolean
  showAccountCost?: boolean
  metric?: DistributionMetric
}>(), {
  loading: false,
  showAccountCost: true,
  metric: 'tokens',
})

const showAccountCost = computed(() => props.showAccountCost)
const displayItems = computed(() => sortDescending(props.items, item => distributionValue(item, props.metric)))

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

const formatCost = (value: number | undefined | null): string => {
  if (value == null) return '0.0000'
  if (value >= 1000) return (value / 1000).toFixed(2) + 'K'
  if (value >= 1) return value.toFixed(2)
  if (value >= 0.01) return value.toFixed(3)
  return value.toFixed(4)
}
</script>
