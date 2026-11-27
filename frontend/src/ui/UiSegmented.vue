<template>
  <div
    class="atelier-segmented"
    role="radiogroup"
    :aria-label="ariaLabel"
  >
    <button
      v-for="item in options"
      :key="String(item.value)"
      type="button"
      role="radio"
      :aria-checked="modelValue === item.value"
      :data-active="modelValue === item.value ? 'true' : 'false'"
      :disabled="item.disabled"
      class="atelier-segmented-item"
      :class="size === 'sm' ? '!px-2.5 !py-1 !text-xs' : ''"
      @click="$emit('update:modelValue', item.value)"
    >
      <slot name="option" :option="item">
        {{ item.label }}
      </slot>
    </button>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number">
export interface SegmentedOption<V = string> {
  value: V
  label: string
  disabled?: boolean
}

withDefaults(
  defineProps<{
    modelValue: T
    options: SegmentedOption<T>[]
    size?: 'sm' | 'md'
    ariaLabel?: string
  }>(),
  {
    size: 'md',
    ariaLabel: undefined
  }
)

defineEmits<{
  'update:modelValue': [value: T]
}>()
</script>
