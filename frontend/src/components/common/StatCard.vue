<template>
  <div class="stat-card">
    <div :class="['stat-icon', iconClass]">
      <component v-if="icon" :is="icon" class="h-5 w-5" aria-hidden="true" />
    </div>
    <div class="min-w-0 flex-1">
      <p class="stat-label truncate">{{ title }}</p>
      <div class="mt-1.5 flex flex-wrap items-baseline gap-2">
        <p class="stat-value tabular-nums" :title="String(formattedValue)">{{ formattedValue }}</p>
        <span v-if="change !== undefined" :class="['stat-trend tabular-nums', trendClass]">
          <Icon
            v-if="changeType !== 'neutral'"
            name="arrowUp"
            size="xs"
            :class="changeType === 'down' && 'rotate-180'"
          />
          {{ formattedChange }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import Icon from '@/components/icons/Icon.vue'

type ChangeType = 'up' | 'down' | 'neutral'
type IconVariant = 'primary' | 'success' | 'warning' | 'danger'

interface Props {
  title: string
  value: number | string
  icon?: Component
  iconVariant?: IconVariant
  change?: number
  changeType?: ChangeType
  formatValue?: (value: number | string) => string
}

const props = withDefaults(defineProps<Props>(), {
  changeType: 'neutral',
  iconVariant: 'primary'
})

const formattedValue = computed(() => {
  if (props.formatValue) {
    return props.formatValue(props.value)
  }
  if (typeof props.value === 'number') {
    return props.value.toLocaleString()
  }
  return props.value
})

const formattedChange = computed(() => {
  if (props.change === undefined) return ''
  const absChange = Math.abs(props.change)
  return `${absChange}%`
})

const iconClass = computed(() => {
  const classes: Record<IconVariant, string> = {
    primary: 'stat-icon-primary',
    success: 'stat-icon-success',
    warning: 'stat-icon-warning',
    danger: 'stat-icon-danger'
  }
  return classes[props.iconVariant]
})

const trendClass = computed(() => {
  const classes: Record<ChangeType, string> = {
    up: 'stat-trend-up',
    down: 'stat-trend-down',
    neutral: 'stat-trend-neutral'
  }
  return classes[props.changeType]
})
</script>

<style scoped>
.stat-card {
  align-items: center;
  transition: border-color var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}

.stat-card:hover {
  border-color: rgb(var(--line-strong));
  box-shadow: var(--shadow-card-hover);
}

.stat-icon {
  width: 2.5rem;
  height: 2.5rem;
  flex-shrink: 0;
  border-radius: var(--radius-control);
  border: 1px solid transparent;
}

.stat-icon-primary {
  color: rgb(var(--text-primary));
  background: rgb(var(--brand-moss) / 0.1);
  border-color: rgb(var(--brand-moss) / 0.2);
}

.stat-icon-success {
  color: rgb(var(--text-ok));
  background: rgb(var(--status-ok-bg));
  border-color: rgb(var(--text-ok) / 0.2);
}

.stat-icon-warning {
  color: rgb(var(--text-warn));
  background: rgb(var(--status-warn-bg));
  border-color: rgb(var(--text-warn) / 0.22);
}

.stat-icon-danger {
  color: rgb(var(--text-err));
  background: rgb(var(--status-err-bg));
  border-color: rgb(var(--text-err) / 0.22);
}

.stat-trend-neutral {
  color: rgb(var(--ink-3));
}
</style>
