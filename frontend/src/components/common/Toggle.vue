<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    class="relative inline-flex h-[22px] w-[40px] flex-shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-150 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--atelier-focus)] focus-visible:ring-offset-2"
    :class="modelValue ? 'bg-[var(--atelier-ink)]' : 'bg-[#d7d7d0] dark:bg-[#2f333a]'"
    @click="toggle"
  >
    <span
      class="pointer-events-none inline-block h-4 w-4 transform rounded-full shadow-sm transition-transform duration-150 ease-out"
      :class="modelValue ? 'translate-x-[18px] bg-[var(--atelier-ink-inverse)]' : 'translate-x-[1px] bg-white dark:bg-[#d4d7de]'"
    />
  </button>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  emit('update:modelValue', !props.modelValue)
}
</script>
