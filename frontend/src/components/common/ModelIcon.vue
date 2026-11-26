<template>
  <BrandModelIcon v-if="resolvedKey" :model="resolvedKey" :size="size" class="model-icon" />
  <span
    v-else
    class="inline-flex shrink-0 items-center justify-center rounded-md bg-[var(--atelier-subtle)] font-semibold uppercase text-[var(--atelier-ink-secondary)]"
    :style="{ width: size, height: size, fontSize: `calc(${size} * 0.48)` }"
  >
    {{ fallbackText }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BrandModelIcon from '@/ui/icons/BrandModelIcon.vue'
import { resolveBrandKey } from '@/ui/icons/resolveBrand'

const props = withDefaults(
  defineProps<{
    model: string
    size?: string
  }>(),
  {
    size: '18px'
  }
)

const resolvedKey = computed(() => resolveBrandKey(props.model))
const fallbackText = computed(() => (props.model || '?').trim().slice(0, 1).toUpperCase())
</script>
