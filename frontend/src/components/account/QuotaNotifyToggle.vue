<script setup lang="ts">
import { QUOTA_THRESHOLD_TYPE_FIXED, QUOTA_THRESHOLD_TYPE_PERCENTAGE, type QuotaThresholdType } from '@/constants/account'
import { UiSelect, type UiSelectOption } from '@/ui'

defineProps<{
  enabled: boolean | null
  threshold: number | null
  thresholdType: QuotaThresholdType | null
}>()

const emit = defineEmits<{
  'update:enabled': [value: boolean | null]
  'update:threshold': [value: number | null]
  'update:thresholdType': [value: QuotaThresholdType | null]
}>()

const thresholdTypeOptions: UiSelectOption[] = [
  { value: QUOTA_THRESHOLD_TYPE_FIXED, label: '$' },
  { value: QUOTA_THRESHOLD_TYPE_PERCENTAGE, label: '%' }
]
</script>

<template>
  <div class="flex items-center gap-1.5">
    <button
      type="button"
      @click="emit('update:enabled', !enabled)"
      :class="[
        'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
        enabled ? 'bg-[var(--atelier-ink)]' : 'bg-[var(--atelier-border-strong)]'
      ]"
    >
      <span
        :class="[
          'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-[var(--atelier-surface)] shadow ring-0 transition duration-200 ease-in-out',
          enabled ? 'translate-x-4' : 'translate-x-0'
        ]"
      />
    </button>
    <template v-if="enabled">
      <input
        :value="threshold"
        @input="emit('update:threshold', parseFloat(($event.target as HTMLInputElement).value) || null)"
        type="number"
        min="0"
        :max="thresholdType === QUOTA_THRESHOLD_TYPE_PERCENTAGE ? 100 : undefined"
        :step="thresholdType === QUOTA_THRESHOLD_TYPE_PERCENTAGE ? 1 : 0.01"
        class="input py-1 font-mono text-sm tabular-nums flex-1 min-w-0"
      />
      <div class="w-[4.5rem] flex-shrink-0">
        <UiSelect
          :model-value="thresholdType || QUOTA_THRESHOLD_TYPE_FIXED"
          :options="thresholdTypeOptions"
          size="sm"
          @update:model-value="(val) => emit('update:thresholdType', val as QuotaThresholdType)"
        />
      </div>
    </template>
  </div>
</template>
