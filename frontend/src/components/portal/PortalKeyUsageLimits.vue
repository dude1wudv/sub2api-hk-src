<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ApiKey } from '@/types'
const props = defineProps<{ apiKey: ApiKey }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const rows = computed(() => [
  { label: text('额度', 'Quota'), used: props.apiKey.quota_used, limit: props.apiKey.quota, reset: null },
  { label: text('5 小时', '5 hours'), used: props.apiKey.usage_5h, limit: props.apiKey.rate_limit_5h, reset: props.apiKey.reset_5h_at },
  { label: text('每日', 'Daily'), used: props.apiKey.usage_1d, limit: props.apiKey.rate_limit_1d, reset: props.apiKey.reset_1d_at },
  { label: text('7 日', '7 days'), used: props.apiKey.usage_7d, limit: props.apiKey.rate_limit_7d, reset: props.apiKey.reset_7d_at },
].filter(row => row.limit > 0))
const ratio = (used: number, limit: number) => Math.min(100, Math.max(0, Number.isFinite(used) ? used / limit * 100 : 0))
const money = (value: number) => '$' + (Number.isFinite(value) ? value : 0).toFixed(2)
const resetTime = (value: string) => new Date(value).toLocaleString(locale.value)
</script>
<template>
  <div v-if="rows.length" class="key-usage-limits">
    <div v-for="row in rows" :key="row.label" class="key-limit" :data-level="ratio(row.used, row.limit) >= 100 ? 'exhausted' : ratio(row.used, row.limit) >= 80 ? 'warning' : 'normal'">
      <div class="key-limit-label"><span>{{ row.label }}</span><span>{{ money(row.used) }} / {{ money(row.limit) }}</span></div>
      <div role="progressbar" :aria-label="row.label" :aria-valuenow="ratio(row.used, row.limit)" :aria-valuemin="0" :aria-valuemax="100" class="key-limit-track"><span :style="{ width: ratio(row.used, row.limit) + '%' }" /></div>
      <small v-if="row.reset">{{ text('重置 ', 'Resets ') }}{{ resetTime(row.reset) }}</small>
    </div>
  </div>
</template>
<style scoped>
.key-usage-limits { display:grid; gap:8px; min-width:156px; margin-top:9px; font-size:12px; }
.key-limit { --meter-color:var(--text-ok); }
.key-limit[data-level=warning] { --meter-color:var(--text-warn); }
.key-limit[data-level=exhausted] { --meter-color:var(--text-err); }
.key-limit-label { display:flex; justify-content:space-between; gap:12px; color:rgb(var(--ink-2)); font-variant-numeric:tabular-nums; }
.key-limit-track { height:5px; margin-top:4px; border-radius:999px; background:rgb(var(--surface-2)); overflow:hidden; }
.key-limit-track span { display:block; height:100%; border-radius:inherit; background:rgb(var(--meter-color)); transition:width 180ms ease; }
.key-limit small { display:block; margin-top:3px; color:rgb(var(--ink-3)); font-size:11px; }
@media(prefers-reduced-motion:reduce) { .key-limit-track span { transition:none; } }
</style>
