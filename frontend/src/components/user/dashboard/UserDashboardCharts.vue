<template>
  <div class="space-y-6">
    <!-- Date Range Filter -->
    <div class="card p-4">
      <div class="flex flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.timeRange') }}:</span>
          <DateRangePicker :start-date="startDate" :end-date="endDate" @update:startDate="$emit('update:startDate', $event)" @update:endDate="$emit('update:endDate', $event)" @change="$emit('dateRangeChange', $event)" />
        </div>
        <div class="flex items-center gap-2">
          <span class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('dashboard.granularity') }}:</span>
          <div class="w-24 sm:w-28">
            <Select :model-value="granularity" :options="[{value:'day', label:t('dashboard.day')}, {value:'hour', label:t('dashboard.hour')}]" @update:model-value="$emit('update:granularity', $event)" @change="$emit('granularityChange')" />
          </div>
          <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary px-3 py-1.5 text-xs">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            <span class="hidden sm:inline">{{ t('common.refresh') }}</span>
          </button>
        </div>
      </div>
    </div>
    <UsageQueryPresets route-id="user.dashboard" :start-date="startDate" :end-date="endDate" @change="setRange" />

    <!-- Charts Grid -->
    <TokenUsageTrend :trend-data="trend" :loading="loading" />
    <div class="grid grid-cols-1 gap-4">
      <!-- Model Distribution Chart -->
      <div class="card relative overflow-hidden p-4 sm:p-5">
        <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.modelDistribution') }}</h3>
        <Skeleton v-if="loading" height="224px" />
        <div v-else class="flex flex-col items-center gap-4 sm:flex-row sm:gap-6">
          <div class="h-24 w-24 shrink-0">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div v-else class="flex h-full items-center justify-center text-xs text-gray-400 dark:text-dark-400">{{ t('dashboard.noDataAvailable') }}</div>
          </div>
          <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
            <table class="w-full text-xs">
              <thead>
                <tr class="text-gray-400 dark:text-dark-400">
                  <th class="pb-2 text-left font-medium">{{ t('dashboard.model') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.requests') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.tokens') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.actual') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.model" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="max-w-[160px] truncate py-1.5 font-mono font-medium text-gray-900 dark:text-white" :title="model.model"><router-link class="hover:underline" :to="{ path: '/usage', query: { model: model.model, start_date: startDate, end_date: endDate } }">{{ model.model }}</router-link></td>
                  <td class="py-1.5 text-right font-mono tabular-nums text-gray-600 dark:text-dark-200">{{ formatNumber(model.requests) }}</td>
                  <td class="py-1.5 text-right font-mono tabular-nums text-gray-600 dark:text-dark-200">{{ formatTokens(model.total_tokens) }}</td>
                  <td class="py-1.5 text-right font-mono font-medium tabular-nums text-primary-700 dark:text-primary-300">${{ formatCost(model.actual_cost) }}</td>
                  <td class="py-1.5 text-right font-mono tabular-nums text-gray-400 dark:text-dark-400">${{ formatCost(model.cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Skeleton from '@/components/common/Skeleton.vue'
import UsageQueryPresets from '@/components/admin/usage/UsageQueryPresets.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import { Doughnut } from 'vue-chartjs'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
const emit = defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
function setRange(range: { startDate: string; endDate: string }) {
  emit('update:startDate', range.startDate)
  emit('update:endDate', range.endDate)
  emit('update:granularity', range.startDate === range.endDate ? 'hour' : 'day')
  emit('dateRangeChange', range)
}
const { t } = useI18n()

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: ['#6366F1', '#818CF8', '#A1A1AA', '#64748B', '#C7D2FE', '#71717A']
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}
</script>
