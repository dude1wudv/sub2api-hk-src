<template>
  <div class="card p-4">
    <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
      {{ t('payment.admin.dailyRevenue') }}
    </h3>
    <div class="h-64">
      <div v-if="loading" class="flex h-full items-center justify-center">
        <LoadingSpinner size="md" />
      </div>
      <Line v-else-if="chartData" :data="chartData" :options="chartOptions" />
      <div
        v-else
        class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('payment.admin.noData') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { DailyPaymentStats } from '@/types/payment'
import { useChartPalette, useChartTheme } from '@/composables/useChartPalette'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = defineProps<{
  data: DailyPaymentStats[]
  loading?: boolean
}>()

const palette = useChartPalette()
const chartTheme = useChartTheme()

const chartData = computed(() => {
  if (!props.data || props.data.length === 0) return null
  const currencies = [...new Set(props.data.flatMap(day => Object.keys(day.amount)))].sort()
  const pointBorder = chartTheme.value.pointBorder
  const pointRadius = props.data.length > 48 ? 0 : 2.5
  const orderCountColor = palette.value[Math.max(1, currencies.length) % palette.value.length]
  return {
    labels: props.data.map(d => d.date),
    datasets: [
      ...currencies.map((currency, index) => {
        const borderColor = palette.value[index % palette.value.length]
        const backgroundColor = `${borderColor}1a`
        return {
          label: `${currency} ${t('payment.admin.revenue')}`,
          data: props.data.map(day => day.amount[currency] || 0),
          borderColor,
          backgroundColor,
          borderWidth: 2,
          fill: true,
          tension: 0.3,
          pointRadius,
          pointHoverRadius: 5,
          pointBackgroundColor: pointBorder,
          pointBorderColor: borderColor,
          pointBorderWidth: 2,
        }
      }),
      {
        label: t('payment.admin.orderCount'),
        data: props.data.map(d => d.count),
        borderColor: orderCountColor,
        backgroundColor: `${orderCountColor}1a`,
        borderWidth: 2,
        fill: false,
        tension: 0.3,
        pointRadius,
        pointHoverRadius: 5,
        pointBackgroundColor: pointBorder,
        pointBorderColor: orderCountColor,
        pointBorderWidth: 2,
        yAxisID: 'y1',
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  animation: false as const,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  elements: {
    point: {
      radius: 2.5,
      hoverRadius: 5,
      borderWidth: 2,
      backgroundColor: chartTheme.value.pointBorder,
    },
    line: { borderWidth: 2 },
  },
  scales: {
    x: {
      border: { display: false },
      grid: { display: false },
      ticks: { color: chartTheme.value.text, maxRotation: 0, maxTicksLimit: 8, font: { size: 12 } },
    },
    y: {
      type: 'linear' as const,
      display: true,
      position: 'left' as const,
      beginAtZero: true,
      border: { display: false },
      title: { display: true, text: t('payment.admin.revenue'), color: chartTheme.value.text, font: { size: 12 } },
      ticks: { color: chartTheme.value.text, font: { size: 12 } },
      grid: { color: chartTheme.value.grid },
    },
    y1: {
      type: 'linear' as const,
      display: true,
      position: 'right' as const,
      beginAtZero: true,
      border: { display: false },
      title: { display: true, text: t('payment.admin.orderCount'), color: chartTheme.value.text, font: { size: 12 } },
      ticks: { color: chartTheme.value.text, font: { size: 12 } },
      grid: { drawOnChartArea: false },
    }
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartTheme.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        boxWidth: 8,
        padding: 15,
        font: { size: 12.5 },
      },
    },
    tooltip: { ...chartTheme.value.tooltip },
  }
}))
</script>
