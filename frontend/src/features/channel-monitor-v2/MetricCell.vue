<template>
  <div
    class="stat-card telemetry-stat-card rounded-3xl p-4 ring-1 ring-gray-900/5 dark:ring-white/5"
    :title="title || undefined"
  >
    <div class="min-w-0 flex-1 w-full">
      <div class="flex items-center justify-between gap-2">
        <span class="stat-label telemetry-stat-label text-[11px] font-semibold uppercase tracking-wider">{{ label }}</span>
        <span
          v-if="resolvedState"
          class="h-2 w-2 shrink-0 rounded-full"
          :class="dotClass"
          aria-hidden="true"
        ></span>
      </div>
      <strong
        class="stat-value telemetry-stat-value mt-2 block overflow-visible text-2xl tabular-nums leading-tight !text-clip !whitespace-normal"
        :class="stateClass"
      >{{ value }}</strong>
      <div
        v-if="detailParts.length > 1"
        class="telemetry-stat-detail mt-1.5 flex flex-wrap gap-x-2 gap-y-0.5 text-[11px] leading-snug tabular-nums"
      >
        <span
          v-for="(part, index) in detailParts"
          :key="`${index}:${part}`"
          class="whitespace-nowrap tabular-nums"
        >{{ part }}</span>
      </div>
      <small
        v-else-if="detail"
        class="telemetry-stat-detail mt-1.5 block text-[11px] leading-snug tabular-nums"
      >{{ detail }}</small>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { HealthState } from '@/api/channelMonitorV2'

const props = defineProps<{
  label: string
  value: string
  detail: string
  state?: HealthState
  /** Exact numeric tooltip (e.g. uncompacted RPM/TPM). */
  title?: string
}>()

/** Split "AVG 475ms · P90 800ms" into chips so nothing is ellipsized. */
const detailParts = computed(() => {
  const raw = (props.detail || '').trim()
  if (!raw || raw === '-') return []
  return raw
    .split(/\s*[·|]\s*/)
    .map((part) => part.trim())
    .filter(Boolean)
})

const missingValue = computed(() => {
  const value = (props.value || '').trim()
  return value === '' || value === '-' || value === '—'
})

const resolvedState = computed(() => (missingValue.value ? undefined : props.state))

const stateClass = computed(() => {
  if (!resolvedState.value) return missingValue.value ? 'text-gray-500 dark:text-dark-400' : 'text-gray-900 dark:text-white'
  if (resolvedState.value === 'healthy') return 'text-emerald-600 dark:text-emerald-400'
  if (resolvedState.value === 'warning') return 'text-amber-600 dark:text-amber-400'
  if (resolvedState.value === 'critical') return 'text-red-600 dark:text-red-400'
  return 'text-gray-500 dark:text-dark-400'
})

const dotClass = computed(() => {
  if (resolvedState.value === 'healthy') return 'bg-emerald-500 ring-2 ring-emerald-500/20'
  if (resolvedState.value === 'warning') return 'bg-amber-500 ring-2 ring-amber-500/20'
  if (resolvedState.value === 'critical') return 'bg-red-500 ring-2 ring-red-500/20'
  return 'bg-gray-300 dark:bg-dark-600'
})
</script>

<style scoped>
.telemetry-stat-card {
  min-height: 6.5rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  background: rgb(var(--surface));
  border: 1px solid rgb(var(--line));
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-panel);
  --tw-ring-shadow: 0 0 #0000;
  transition: border-color var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}
.telemetry-stat-card:hover {
  border-color: rgb(var(--line-strong));
  box-shadow: var(--shadow-card-hover);
}
.telemetry-stat-label {
  color: rgb(var(--ink-3));
}
.telemetry-stat-value {
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
.telemetry-stat-detail {
  color: rgb(var(--ink-3));
  font-variant-numeric: tabular-nums;
}
</style>
