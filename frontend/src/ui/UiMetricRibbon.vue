<template>
  <section
    class="relative overflow-hidden border-b border-[var(--atelier-border)] pb-6 pt-1"
  >
    <div
      v-if="showDots"
      class="atelier-dot-matrix atelier-dot-mask-right pointer-events-none absolute inset-y-0 right-0 w-72 opacity-80"
      aria-hidden="true"
    />
    <div class="relative z-10 flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
      <div class="flex flex-wrap items-end gap-4">
        <div>
          <p class="text-xs font-medium text-[var(--atelier-ink-muted)]">
            {{ primaryLabel }}
          </p>
          <div class="mt-1 flex items-baseline gap-3">
            <span class="font-mono text-3xl font-semibold tracking-tight tabular-nums text-[var(--atelier-ink)] sm:text-[34px]">
              {{ primaryValue }}
            </span>
            <span v-if="primaryMeta" class="text-xs text-[var(--atelier-ink-secondary)]">
              {{ primaryMeta }}
            </span>
          </div>
        </div>
        <div v-if="$slots.primaryAction" class="pb-1">
          <slot name="primaryAction" />
        </div>
      </div>

      <div
        v-if="metrics.length"
        class="flex flex-wrap items-center gap-x-6 gap-y-3 lg:flex-nowrap"
      >
        <template v-for="(item, idx) in metrics" :key="item.label">
          <div
            v-if="idx > 0"
            class="hidden h-8 w-px bg-[var(--atelier-border)] sm:block"
            aria-hidden="true"
          />
          <div class="min-w-[110px]">
            <p class="text-xs text-[var(--atelier-ink-muted)]">{{ item.label }}</p>
            <div class="mt-1 flex items-center gap-1.5">
              <UiStatusDot v-if="item.dot" :tone="item.dot" />
              <span class="font-mono text-base font-semibold tabular-nums text-[var(--atelier-ink)]">
                {{ item.value }}
              </span>
              <span
                v-if="item.delta"
                class="font-mono text-xs tabular-nums"
                :class="item.deltaTone === 'ok' ? 'text-[var(--atelier-ok)]' : item.deltaTone === 'err' ? 'text-[var(--atelier-err)]' : 'text-[var(--atelier-ink-muted)]'"
              >
                {{ item.delta }}
              </span>
            </div>
            <p v-if="item.sub" class="mt-0.5 text-xs tabular-nums text-[var(--atelier-ink-muted)]">
              {{ item.sub }}
            </p>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import UiStatusDot from './UiStatusDot.vue'

export interface RibbonMetric {
  label: string
  value: string
  sub?: string
  delta?: string
  deltaTone?: 'ok' | 'err' | 'neutral'
  dot?: 'ok' | 'warn' | 'err' | 'info' | 'neutral'
}

withDefaults(
  defineProps<{
    primaryLabel: string
    primaryValue: string
    primaryMeta?: string
    metrics?: RibbonMetric[]
    showDots?: boolean
  }>(),
  {
    primaryMeta: '',
    metrics: () => [],
    showDots: true
  }
)
</script>
