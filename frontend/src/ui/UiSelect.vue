<template>
  <SelectRoot
    :model-value="internalValue"
    :disabled="disabled"
    @update:model-value="handleSelect"
  >
    <SelectTrigger
      :id="id"
      :aria-label="ariaLabel"
      :class="[
        'inline-flex items-center justify-between gap-2 rounded-lg border text-left transition-colors',
        'border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] text-[var(--atelier-ink)]',
        'hover:border-[var(--atelier-ink-muted)] focus:outline-none focus:border-[var(--atelier-focus)] focus:ring-2 focus:ring-blue-500/15',
        'disabled:cursor-not-allowed disabled:opacity-50',
        sizeClass,
        block ? 'w-full' : ''
      ]"
    >
      <span class="flex min-w-0 flex-1 items-center gap-2 truncate">
        <slot name="prefix" />
        <SelectValue :placeholder="placeholder || ''" class="truncate" />
      </span>
      <SelectIcon as-child>
        <svg
          class="h-3.5 w-3.5 shrink-0 text-[var(--atelier-ink-muted)]"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          aria-hidden="true"
        >
          <path d="m4.5 6.25 3.5 3.5 3.5-3.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </SelectIcon>
    </SelectTrigger>

    <SelectPortal :to="portalTarget">
      <SelectContent
        position="popper"
        :side-offset="4"
        class="z-[120] max-h-72 min-w-[var(--reka-select-trigger-width)] overflow-hidden rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-1 shadow-[var(--atelier-shadow-md)]"
      >
        <SelectViewport class="max-h-64 overflow-y-auto">
          <SelectItem
            v-for="opt in normalizedOptions"
            :key="opt.key"
            :value="opt.encoded"
            :disabled="opt.disabled"
            class="relative flex cursor-pointer select-none items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-sm text-[var(--atelier-ink)] outline-none transition-colors data-[disabled]:pointer-events-none data-[disabled]:opacity-45 data-[highlighted]:bg-[var(--atelier-hover)] data-[state=checked]:font-medium"
          >
            <SelectItemText class="truncate">
              {{ opt.label }}
            </SelectItemText>
            <SelectItemIndicator class="ml-2 shrink-0 text-[var(--atelier-ink)]">
              <svg class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2">
                <path d="m3.5 8.5 3 3 6-6.5" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import {
  SelectContent,
  SelectIcon,
  SelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectPortal,
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectViewport
} from 'reka-ui'
import { overlayHostKey } from '@/composables/overlayHost'

export interface UiSelectOption {
  value: string | number | boolean | null
  label: string
  disabled?: boolean
}

const EMPTY_SENTINEL = '__atelier_empty__'
const NULL_SENTINEL = '__atelier_null__'

const props = withDefaults(
  defineProps<{
    modelValue: string | number | boolean | null | undefined
    options: UiSelectOption[]
    placeholder?: string
    disabled?: boolean
    size?: 'sm' | 'md'
    block?: boolean
    id?: string
    ariaLabel?: string
  }>(),
  {
    placeholder: '',
    disabled: false,
    size: 'md',
    block: true,
    id: undefined,
    ariaLabel: undefined
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number | boolean | null]
  change: [value: string | number | boolean | null]
}>()

const overlayHost = inject(overlayHostKey, null)
const portalTarget = computed(() => overlayHost?.value || 'body')

function encodeValue(val: string | number | boolean | null | undefined): string {
  if (val === null || val === undefined) return NULL_SENTINEL
  if (val === '') return EMPTY_SENTINEL
  return `${typeof val}:${String(val)}`
}

const normalizedOptions = computed(() =>
  props.options.map((opt, idx) => ({
    ...opt,
    encoded: encodeValue(opt.value),
    key: `${idx}_${encodeValue(opt.value)}`
  }))
)

const internalValue = computed(() => encodeValue(props.modelValue))

const sizeClass = computed(() =>
  props.size === 'sm' ? 'h-8 px-2.5 text-xs' : 'h-[38px] px-3 text-sm'
)

function handleSelect(encoded: string) {
  const match = normalizedOptions.value.find((o) => o.encoded === encoded)
  const next = match ? match.value : ''
  emit('update:modelValue', next)
  emit('change', next)
}
</script>
