<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { UiProgressBar } from '@/ui'
import type { ApiKey } from '@/types'

const props = defineProps<{ apiKey: ApiKey }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en)

const rows = computed(() =>
  [
    { label: text('额度', 'Quota'), used: props.apiKey.quota_used, limit: props.apiKey.quota, reset: null },
    { label: text('5 小时', '5 hours'), used: props.apiKey.usage_5h, limit: props.apiKey.rate_limit_5h, reset: props.apiKey.reset_5h_at },
    { label: text('每日', 'Daily'), used: props.apiKey.usage_1d, limit: props.apiKey.rate_limit_1d, reset: props.apiKey.reset_1d_at },
    { label: text('7 日', '7 days'), used: props.apiKey.usage_7d, limit: props.apiKey.rate_limit_7d, reset: props.apiKey.reset_7d_at }
  ].filter((row) => row.limit > 0)
)

const ratio = (used: number, limit: number) =>
  Math.min(100, Math.max(0, Number.isFinite(used) && limit > 0 ? (used / limit) * 100 : 0))

const levelOf = (used: number, limit: number): 'exhausted' | 'warning' | 'normal' => {
  const pct = ratio(used, limit)
  if (pct >= 100) return 'exhausted'
  if (pct >= 80) return 'warning'
  return 'normal'
}

const toneOf = (used: number, limit: number): 'err' | 'warn' | 'ink' => {
  const pct = ratio(used, limit)
  if (pct >= 100) return 'err'
  if (pct >= 80) return 'warn'
  return 'ink'
}

const money = (value: number) => {
  const n = Number.isFinite(value) ? value : 0
  return `$${n % 1 === 0 ? n.toFixed(0) : n.toFixed(2)}`
}

const resetTime = (value: string) => new Date(value).toLocaleString(locale.value)
</script>

<template>
  <div v-if="rows.length" class="mt-2 grid min-w-[152px] gap-2">
    <div
      v-for="row in rows"
      :key="row.label"
      class="key-limit space-y-1"
      :data-level="levelOf(row.used, row.limit)"
    >
      <div class="flex items-baseline justify-between gap-2 text-[11.5px] leading-none">
        <span class="text-[var(--atelier-ink-muted)]">{{ row.label }}</span>
        <span class="font-mono tabular-nums">
          <span class="font-medium text-[var(--atelier-ink)]">{{ money(row.used) }}</span>
          <span class="text-[var(--atelier-ink-muted)]"> / {{ money(row.limit) }}</span>
        </span>
      </div>
      <UiProgressBar
        :value="ratio(row.used, row.limit)"
        :tone="toneOf(row.used, row.limit)"
        :aria-label="row.label"
      />
      <small v-if="row.reset" class="block font-mono text-[10.5px] tabular-nums text-[var(--atelier-ink-muted)]">
        {{ text('重置 ', 'Resets ') }}{{ resetTime(row.reset) }}
      </small>
    </div>
  </div>
</template>
