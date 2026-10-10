<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PortalUsageMonth } from '@/utils/portalUsage'
import { formatUsageMoney, formatUsageNumber } from '@/utils/portalUsage'
const props = defineProps<{ months: PortalUsageMonth[] }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const selected = ref(11)
const chartMax = computed(() => Math.max(1, ...props.months.map(month => month.cost)) * 1.15)
const points = computed(() => props.months.map((month, index) => ({ ...month, x: 56 + index * 76, y: 180 - month.cost / chartMax.value * 148 })))
const path = computed(() => points.value.map((point, index) => `${index ? 'L' : 'M'} ${point.x} ${point.y}`).join(' '))
const area = computed(() => points.value.length ? `${path.value} L ${points.value[points.value.length - 1]?.x} 180 L 56 180 Z` : '')
const active = computed(() => points.value[selected.value] || points.value[points.value.length - 1])
</script>

<template>
  <div class="portal-usage-chart">
    <div class="chart-meta"><span>{{ copy('实际消费 · USD', 'Billed cost · USD') }}</span><span v-if="active">{{ active.key }} <strong>{{ formatUsageMoney(active.cost, 2) }}</strong></span></div>
    <div class="chart-scroll"><svg viewBox="0 0 940 220" role="img" :aria-label="copy('过去十二个月的实际消费曲线', 'Billed cost over the last twelve months')">
      <title>{{ copy('过去十二个月的实际消费', 'Billed cost over the last twelve months') }}</title>
      <g v-for="tick in [0, 1, 2, 3]" :key="tick"><line x1="56" x2="915" :y1="180 - tick * 49.33" :y2="180 - tick * 49.33" stroke="#efede3" stroke-dasharray="3 4" /><text x="44" :y="184 - tick * 49.33" text-anchor="end">{{ (chartMax * tick / 3).toFixed(chartMax >= 100 ? 0 : 1) }}</text></g>
      <path :d="area" fill="#f3f0eb" opacity="0.65" />
      <path :d="path" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
      <g v-for="(point, index) in points" :key="point.key" class="chart-point" tabindex="0" :aria-label="`${point.key}, ${formatUsageMoney(point.cost, 2)}, ${point.requests} ${copy('次请求', 'requests')}`" @mouseenter="selected = index" @focus="selected = index">
        <rect :x="point.x - 30" y="20" width="60" height="165" fill="transparent" />
        <circle :cx="point.x" :cy="point.y" :r="selected === index ? 4 : 2.5" fill="#fffdf7" stroke="#000" stroke-width="1.4" />
        <text :x="point.x" y="209" text-anchor="middle">{{ zh ? point.label : new Date(`${point.key}-01T12:00:00`).toLocaleDateString('en', { month: 'short' }) }}</text>
        <title>{{ point.key }} · {{ formatUsageMoney(point.cost, 2) }} · {{ formatUsageNumber(point.requests) }} {{ copy('次请求', 'requests') }}</title>
      </g>
    </svg></div>
    <div class="chart-detail" aria-live="polite"><template v-if="active">{{ active.key }} · {{ formatUsageNumber(active.requests) }} {{ copy('次请求', 'requests') }} · {{ formatUsageNumber(active.tokens) }} Token</template></div>
  </div>
</template>

<style scoped>
.chart-meta { display: flex; justify-content: space-between; gap: 18px; color: #929088; font-size: 12px; padding: 0 6px 14px; }
.chart-meta strong { font-weight: 500; color: #000; margin-left: 8px; }
.chart-scroll { width: 100%; overflow-x: auto; }
svg { width: 100%; height: auto; min-width: 640px; display: block; overflow: visible; }
svg text { fill: #929088; font: 11px system-ui, sans-serif; }
.chart-point { outline: none; }
.chart-point:focus circle { stroke-width: 3; }
.chart-detail { text-align: right; min-height: 20px; font-size: 11px; color: #929088; margin-top: 8px; }
@media (max-width: 739px) { svg { min-width: 570px; } .chart-meta { flex-wrap: wrap; } }
</style>
