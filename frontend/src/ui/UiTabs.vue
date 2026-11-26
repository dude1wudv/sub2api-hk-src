<template>
  <TabsRoot
    :model-value="modelValue"
    @update:model-value="(val) => $emit('update:modelValue', String(val))"
  >
    <TabsList class="atelier-tabs-list" :aria-label="ariaLabel">
      <TabsTrigger
        v-for="tab in tabs"
        :key="tab.value"
        :value="tab.value"
        :disabled="tab.disabled"
        class="atelier-tab-trigger"
      >
        <slot name="tab" :tab="tab">
          {{ tab.label }}
        </slot>
      </TabsTrigger>
    </TabsList>
    <slot />
  </TabsRoot>
</template>

<script setup lang="ts">
import { TabsList, TabsRoot, TabsTrigger } from 'reka-ui'

export interface UnderlineTab {
  value: string
  label: string
  disabled?: boolean
}

defineProps<{
  modelValue: string
  tabs: UnderlineTab[]
  ariaLabel?: string
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>
