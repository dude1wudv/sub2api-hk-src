<template>
  <svg
    v-if="spec"
    :viewBox="spec.viewBox"
    :class="sizeClass"
    :style="customSizeStyle"
    fill="currentColor"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    class="inline-block shrink-0"
    v-html="resolvedBody"
  />
  <svg
    v-else
    viewBox="0 0 24 24"
    :class="sizeClass"
    :style="customSizeStyle"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    aria-hidden="true"
    class="inline-block shrink-0 text-[var(--atelier-ink-secondary)]"
  >
    <circle cx="12" cy="12" r="8.5" />
    <path d="M8.5 12h7M12 8.5v7" stroke-linecap="round" />
  </svg>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { BRAND_ICON_DATA, type BrandIconSpec } from './brandIconData'
import { resolveBrandKey } from './resolveBrand'

const props = withDefaults(
  defineProps<{
    model?: string | null
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | string
  }>(),
  {
    model: '',
    size: 'sm'
  }
)

const uid = useId().replace(/[^a-zA-Z0-9_-]/g, '')

const presetSizes = new Set(['xs', 'sm', 'md', 'lg', 'xl'])

const sizeClass = computed(() => {
  switch (props.size) {
    case 'xs':
      return 'h-3.5 w-3.5'
    case 'sm':
      return 'h-4 w-4'
    case 'md':
      return 'h-5 w-5'
    case 'lg':
      return 'h-6 w-6'
    case 'xl':
      return 'h-8 w-8'
    default:
      return presetSizes.has(props.size) ? 'h-4 w-4' : ''
  }
})

const customSizeStyle = computed(() => {
  if (!props.size || presetSizes.has(props.size)) return undefined
  return { width: props.size, height: props.size }
})

const spec = computed<BrandIconSpec | null>(() => {
  const key = resolveBrandKey(props.model)
  return key ? BRAND_ICON_DATA[key] ?? null : null
})

const resolvedBody = computed(() => {
  if (!spec.value) return ''
  return spec.value.body.replace(/__UID__/g, `bi_${uid}_`)
})
</script>
