<template>
  <div class="chart-panel card min-w-0 max-w-full p-4 sm:p-5">
    <div class="mb-4 flex min-w-0 flex-wrap items-center justify-between gap-3">
      <h3 class="min-w-0 break-words text-sm font-semibold text-gray-900 dark:text-white">
        {{ title || t('usage.endpointDistribution') }}
      </h3>
      <div class="flex min-w-0 max-w-full flex-wrap items-center justify-end gap-2">
        <div
          v-if="showSourceToggle"
          class="inline-flex max-w-full flex-wrap rounded-lg border border-gray-200 bg-gray-50/80 p-0.5 [&>button]:min-h-11 sm:[&>button]:min-h-9 dark:border-dark-700 dark:bg-dark-800/80"
        >
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="source === 'inbound'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:source', 'inbound')"
          >
            {{ t('usage.inbound') }}
          </button>
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="source === 'upstream'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:source', 'upstream')"
          >
            {{ t('usage.upstream') }}
          </button>
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="source === 'path'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:source', 'path')"
          >
            {{ t('usage.path') }}
          </button>
        </div>

        <div
          v-if="showMetricToggle"
          class="inline-flex max-w-full flex-wrap rounded-lg border border-gray-200 bg-gray-50/80 p-0.5 [&>button]:min-h-11 sm:[&>button]:min-h-9 dark:border-dark-700 dark:bg-dark-800/80"
        >
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="metric === 'tokens'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:metric', 'tokens')"
          >
            {{ t('admin.dashboard.metricTokens') }}
          </button>
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="metric === 'actual_cost'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="emit('update:metric', 'actual_cost')"
          >
            {{ t('admin.dashboard.metricActualCost') }}
          </button>
        </div>
      </div>
    </div>
    <div v-if="loading" class="flex h-48 items-center justify-center">
      <LoadingSpinner />
    </div>
    <DistributionChartLayout v-else-if="displayEndpointStats.length > 0 && chartData"
      :table-label="title || t('usage.endpointDistribution')" :columns="5"
    >
      <template #chart>
        <Doughnut :data="chartData" :options="doughnutOptions" />
      </template>
        <table class="w-full text-xs">
          <thead>
            <tr class="text-gray-400 dark:text-dark-400">
              <th class="pb-2 text-left font-medium">{{ t('usage.endpoint') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('admin.dashboard.requests') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('admin.dashboard.tokens') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('admin.dashboard.actual') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('admin.dashboard.standard') }}</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="item in displayEndpointStats" :key="item.endpoint">
              <tr
                class="border-t border-gray-100 transition-colors dark:border-dark-700"
                :class="enableBreakdown ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40' : ''"
                @click="enableBreakdown && toggleBreakdown(item.endpoint)"
              >
                <td class="distribution-name py-1.5 font-medium" :class="enableBreakdown ? 'text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300' : 'text-gray-900 dark:text-white'" :title="item.endpoint || t('common.unknown')">
                  <span class="flex min-w-0 items-start gap-1">
                    <svg v-if="enableBreakdown && expandedKey === item.endpoint" class="h-3 w-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/></svg>
                    <svg v-else-if="enableBreakdown" class="h-3 w-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
                    <span class="min-w-0 [overflow-wrap:anywhere]">{{ item.endpoint || t('common.unknown') }}</span>
                  </span>
                </td>
                <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                  {{ formatNumber(item.requests) }}
                </td>
                <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                  {{ formatTokens(item.total_tokens) }}
                </td>
                <td class="py-1.5 text-right font-mono font-medium tabular-nums text-primary-700 dark:text-primary-300">
                  ${{ formatCost(item.actual_cost) }}
                </td>
                <td class="py-1.5 text-right font-mono tabular-nums text-gray-400 dark:text-dark-400">
                  ${{ formatCost(item.cost) }}
                </td>
              </tr>
              <tr v-if="expandedKey === item.endpoint">
                <td colspan="5" class="p-0">
                  <UserBreakdownSubTable
                    :items="breakdownItems"
                    :loading="breakdownLoading"
                    :metric="metric"
                    :show-account-cost="false"
                  />
                </td>
              </tr>
            </template>
          </tbody>
        </table>
    </DistributionChartLayout>
    <div v-else class="flex h-48 items-center justify-center text-xs text-gray-400 dark:text-dark-400">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js'
import { Doughnut } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import UserBreakdownSubTable from './UserBreakdownSubTable.vue'
import DistributionChartLayout from './DistributionChartLayout.vue'
import type { EndpointStat, UserBreakdownItem } from '@/types'
import { getUserBreakdown } from '@/api/admin/dashboard'
import { distributionValue, rankChartRows } from '@/utils/chartDistribution'

ChartJS.register(ArcElement, Tooltip, Legend)

const chartTheme = useChartTheme()
const { t } = useI18n()

type DistributionMetric = 'tokens' | 'actual_cost'
type EndpointSource = 'inbound' | 'upstream' | 'path'

const props = withDefaults(
  defineProps<{
    endpointStats: EndpointStat[]
    upstreamEndpointStats?: EndpointStat[]
    endpointPathStats?: EndpointStat[]
    loading?: boolean
    title?: string
    metric?: DistributionMetric
    source?: EndpointSource
    showMetricToggle?: boolean
    showSourceToggle?: boolean
    enableBreakdown?: boolean
    startDate?: string
    endDate?: string
    filters?: Record<string, any>
  }>(),
  {
    upstreamEndpointStats: () => [],
    endpointPathStats: () => [],
    loading: false,
    title: '',
    metric: 'tokens',
    source: 'inbound',
    showMetricToggle: false,
    showSourceToggle: false,
    enableBreakdown: true
  }
)

const emit = defineEmits<{
  'update:metric': [value: DistributionMetric]
  'update:source': [value: EndpointSource]
}>()

const expandedKey = ref<string | null>(null)
const breakdownItems = ref<UserBreakdownItem[]>([])
const breakdownLoading = ref(false)

const toggleBreakdown = async (endpoint: string) => {
  if (expandedKey.value === endpoint) {
    expandedKey.value = null
    return
  }
  expandedKey.value = endpoint
  breakdownLoading.value = true
  breakdownItems.value = []
  try {
    const res = await getUserBreakdown({
      ...props.filters,
      start_date: props.startDate,
      end_date: props.endDate,
      endpoint,
      endpoint_type: props.source,
    })
    breakdownItems.value = res.users || []
  } catch {
    breakdownItems.value = []
  } finally {
    breakdownLoading.value = false
  }
}

const chartColors = useChartPalette()

const displayEndpointStats = computed(() => {
  const sourceStats = props.source === 'upstream'
    ? props.upstreamEndpointStats
    : props.source === 'path'
      ? props.endpointPathStats
      : props.endpointStats
  if (!sourceStats?.length) return []

  return rankChartRows(sourceStats, row => distributionValue(row, props.metric), chartColors.value)
})

const chartData = computed(() => {
  if (!displayEndpointStats.value?.length) return null

  return {
    labels: displayEndpointStats.value.map((item) => item.endpoint || t('common.unknown')),
    datasets: [
      {
        data: displayEndpointStats.value.map((item) => distributionValue(item, props.metric)),
        backgroundColor: displayEndpointStats.value.map((item) => item.chartColor),
        borderWidth: 0
      }
    ]
  }
})

const doughnutOptions = computed(() => ({
  responsive: true,
  animation: false as const,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    },
    tooltip: {
      ...chartTheme.value.tooltip,
      callbacks: {
        label: (context: any) => {
          const value = context.raw as number
          const total = context.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          const formattedValue = props.metric === 'actual_cost'
            ? `$${formatCost(value)}`
            : formatTokens(value)
          return `${context.label}: ${formattedValue} (${percentage}%)`
        }
      }
    }
  }
}))

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatNumber = (value: number): string => {
  return value.toLocaleString()
}

const formatCost = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}
</script>
