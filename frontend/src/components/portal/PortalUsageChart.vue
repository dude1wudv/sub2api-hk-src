<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery, useResizeObserver } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
const container = ref<HTMLElement | null>(null)
const width = ref(320)
useResizeObserver(container, entries => {
  const measured = entries[0]?.contentRect.width
  if (measured && Number.isFinite(measured)) width.value = measured
})
import type { PortalUsageMonth } from '@/utils/portalUsage'
import { formatUsageMoney, formatUsageNumber } from '@/utils/portalUsage'
const props = defineProps<{ months: PortalUsageMonth[] }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const selected = ref(11)
const chartMax = computed(() => Math.max(1, ...props.months.map(month => month.cost)) * 1.15)
const points = computed(() => props.months.map((month, index) => ({ ...month, x: 56 + (props.months.length > 1 ? index / (props.months.length - 1) : 0.5) * Math.max(0, width.value - 68), y: 180 - month.cost / chartMax.value * 148 })))
const path = computed(() => points.value.map((point, index) => `${index ? 'L' : 'M'} ${point.x} ${point.y}`).join(' '))
const area = computed(() => points.value.length ? `${path.value} L ${points.value[points.value.length - 1]?.x} 180 L 56 180 Z` : '')
const active = computed(() => points.value[selected.value] || points.value[points.value.length - 1])
</script>

<template>
  <div class="portal-usage-chart">
    <div class="chart-meta"><span>{{ copy('实际消费 · USD', 'Billed cost · USD') }}</span><span v-if="active">{{ active.key }} <strong>{{ formatUsageMoney(active.cost, 2) }}</strong></span></div>
    <label v-if="mobile && months.length" class="chart-month-picker">{{ copy('月份', 'Month') }} <select :value="active?.key" @change="selected = months.findIndex(month => month.key === ($event.target as HTMLSelectElement).value)"><option v-for="month in months" :key="month.key" :value="month.key">{{ month.key }}</option></select></label>
    <div ref="container" class="chart-scroll"><svg :viewBox="`0 0 ${width} 220`" role="img" :aria-label="copy('过去十二个月的实际消费曲线', 'Billed cost over the last twelve months')">
      <title>{{ copy('过去十二个月的实际消费', 'Billed cost over the last twelve months') }}</title>
      <defs>
        <linearGradient id="portal-usage-area-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--chart-1)" stop-opacity="0.18" />
          <stop offset="100%" stop-color="var(--chart-1)" stop-opacity="0.01" />
        </linearGradient>
      </defs>
      <g v-for="tick in [0, 1, 2, 3]" :key="tick"><line x1="56" :x2="width - 12" :y1="180 - tick * 49.33" :y2="180 - tick * 49.33" stroke="var(--chart-grid)" stroke-width="1" /><text x="44" :y="184 - tick * 49.33" text-anchor="end">{{ (chartMax * tick / 3).toFixed(chartMax >= 100 ? 0 : 1) }}</text></g>
      <path :d="area" fill="url(#portal-usage-area-fill)" />
      <path :d="path" fill="none" stroke="var(--chart-1)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
      <g v-for="(point, index) in points" :key="point.key" class="chart-point" tabindex="0" :aria-label="`${point.key}, ${formatUsageMoney(point.cost, 2)}, ${point.requests} ${copy('次请求', 'requests')}`" @mouseenter="selected = index" @focus="selected = index" @click="selected = index">
        <rect :x="Math.max(56, point.x - 30)" y="20" :width="Math.max(0, Math.min(60, width - 12 - Math.max(56, point.x - 30)))" height="165" fill="transparent" />
        <circle :cx="point.x" :cy="point.y" :r="selected === index ? 4.5 : 3" fill="rgb(var(--surface))" stroke="var(--chart-1)" stroke-width="2" />
        <text v-if="width >= 480 || index % 3 === 0 || index === points.length - 1" :x="point.x" y="209" :text-anchor="index === points.length - 1 ? 'end' : 'middle'">{{ zh ? point.label : new Date(`${point.key}-01T12:00:00`).toLocaleDateString('en', { month: 'short' }) }}</text>
        <title>{{ point.key }} · {{ formatUsageMoney(point.cost, 2) }} · {{ formatUsageNumber(point.requests) }} {{ copy('次请求', 'requests') }}</title>
      </g>
    </svg></div>
    <div class="chart-detail" aria-live="polite"><template v-if="active">{{ active.key }} · {{ formatUsageNumber(active.requests) }} {{ copy('次请求', 'requests') }} · {{ formatUsageNumber(active.tokens) }} Token</template></div>
  </div>
</template>

<style scoped>
.chart-meta { display: flex; justify-content: space-between; gap: 18px; color: rgb(var(--ink-3)); font-size: 13px; padding: 0 6px 14px; }
.chart-meta strong { font-weight: 500; color: rgb(var(--ink)); margin-left: 8px; }
.chart-scroll { width: 100%; min-width: 0; }
svg { width: 100%; height: 220px; display: block; }
svg text { fill: var(--chart-text); font: 13px var(--font-sans); }
.chart-point { outline: none; }
.chart-point:focus circle { stroke-width: 3; }
.chart-detail { text-align: right; min-height: 20px; font-size: 13px; color: rgb(var(--ink-3)); margin-top: 8px; }
.chart-meta { flex-wrap: wrap; }
.chart-month-picker { display:flex; flex-wrap:wrap; align-items:center; gap:12px; margin-bottom:12px; }
.chart-month-picker select { min-height:44px; max-width:100%; background:rgb(var(--surface)); color:rgb(var(--ink)); border:1px solid rgb(var(--line)); border-radius:var(--radius-control); padding:8px 12px; }
</style>
