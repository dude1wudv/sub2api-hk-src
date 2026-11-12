<template>
  <div
    ref="containerRef"
    class="filter-menu relative"
    :class="compact ? 'min-w-[6.5rem] sm:min-w-[7.25rem]' : 'min-w-[150px] sm:min-w-[160px]'"
  >
    <button
      ref="triggerRef"
      type="button"
      class="select-trigger flex cursor-pointer list-none items-center justify-between gap-1.5 text-left"
      :class="[
        isOpen ? 'select-trigger-open' : '',
        modelValue.length > 0 ? 'select-trigger-filtered' : '',
        compact ? 'h-8 rounded-lg !px-2 !py-1 text-xs' : 'h-[42px]',
      ]"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      :aria-label="label"
      @click="toggleOpen"
    >
      <span
        class="select-value min-w-0 truncate tabular-nums"
        :class="compact ? 'max-w-[5.5rem] sm:max-w-[6.5rem]' : 'max-w-[11rem]'"
      >
        {{ t('channelMonitorV2.filters.labelValue', { label, value: selectionLabel }) }}
      </span>
      <span class="select-icon shrink-0 transition-transform" :class="isOpen ? 'rotate-180' : ''">
        <Icon name="chevronDown" size="sm" />
      </span>
    </button>

    <Teleport to="body">
      <Transition name="select-dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="select-dropdown-portal dropdown ui-popover filter-dropdown"
          :class="[instanceId]"
          :style="dropdownStyle"
          role="listbox"
          aria-multiselectable="true"
          @click.stop
          @mousedown.stop
        >
          <button
            type="button"
            class="dropdown-item select-option select-option-group flex w-full items-center justify-between px-3.5 py-2 text-left text-xs font-semibold"
            @click="clear"
          >
            <span>{{ allLabel }}</span>
            <Icon v-if="modelValue.length === 0" name="check" size="sm" class="text-primary-500" />
          </button>

          <button
            v-for="option in options"
            :key="option.value"
            type="button"
            role="option"
            class="dropdown-item select-option flex w-full items-center justify-between gap-3 px-3.5 py-2 text-left text-[13px]"
            :class="modelValue.includes(option.value) ? 'select-option-selected' : ''"
            :aria-selected="modelValue.includes(option.value)"
            @click="toggle(option.value)"
          >
            <span class="flex min-w-0 flex-1 items-center gap-2">
              <span
                class="checkbox flex h-4 w-4 items-center justify-center rounded border text-primary-500"
                :class="modelValue.includes(option.value) ? 'checkbox-checked' : 'checkbox-unchecked'"
              >
                <Icon v-if="modelValue.includes(option.value)" name="check" size="sm" class="text-primary-500" />
              </span>
              <span class="min-w-0 flex-1 truncate">{{ option.label }}</span>
            </span>
            <small v-if="option.count != null" class="option-count font-mono text-xs tabular-nums">{{ formatCount(option.count) }}</small>
          </button>
          <p v-if="options.length === 0" class="filter-empty px-4 py-3 text-center text-xs">{{ t('channelMonitorV2.filters.empty') }}</p>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import type { CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { monitorIntlLocale } from '@/features/channel-monitor-v2/monitorFormat'

interface FilterOption {
  value: string
  label: string
  count?: number
}

const props = withDefaults(
  defineProps<{
    label: string
    allLabel: string
    modelValue: string[]
    /** Options for this picker (parent may cascade by platform). */
    options: FilterOption[]
    /** Compact trigger for single-row toolbars. */
    compact?: boolean
  }>(),
  { compact: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const { t, locale } = useI18n()

const containerRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const instanceId = `filter-select-${Math.random().toString(36).slice(2, 9)}`

const selectionLabel = computed(() => {
  if (props.modelValue.length === 0) return props.allLabel
  if (props.modelValue.length === 1) {
    return props.options.find((item) => item.value === props.modelValue[0])?.label || props.modelValue[0]
  }
  return t('channelMonitorV2.filters.selectedCount', { count: props.modelValue.length })
})

const dropdownStyle = computed<CSSProperties>(() => {
  const trigger = triggerRef.value
  if (!trigger) return {}
  const rect = trigger.getBoundingClientRect()
  const padding = 8
  const viewportRight = Math.max(padding, window.innerWidth - padding)
  const left = Math.min(Math.max(padding, rect.left), viewportRight)
  const availableWidth = Math.max(0, viewportRight - left)
  const preferredMinWidth = Math.max(200, rect.width)
  const minWidth = Math.min(preferredMinWidth, availableWidth)
  return {
    position: 'fixed' as const,
    left: `${left}px`,
    top: `${rect.bottom + 4}px`,
    minWidth: `${minWidth}px`,
    maxWidth: `${availableWidth}px`,
    zIndex: '100000020',
  }
})

function clear() {
  emit('update:modelValue', [])
  // Stay open so users can re-select without reopening.
}

function toggle(value: string) {
  const selected = new Set(props.modelValue)
  if (selected.has(value)) selected.delete(value)
  else selected.add(value)
  emit('update:modelValue', [...selected])
  // Stay open on toggle (multi-select).
}

function toggleOpen() {
  isOpen.value ? close() : open()
}

function open() {
  isOpen.value = true
  void nextTick(() => positionDropdown())
}

function close() {
  isOpen.value = false
}

function positionDropdown() {
  const trigger = triggerRef.value
  const dropdown = dropdownRef.value
  if (!trigger || !dropdown) return
  const rect = trigger.getBoundingClientRect()
  const padding = 8
  const viewportRight = Math.max(padding, window.innerWidth - padding)
  const left = Math.min(Math.max(padding, rect.left), viewportRight)
  const availableWidth = Math.max(0, viewportRight - left)
  const preferredMinWidth = Math.max(200, rect.width)
  const minWidth = Math.min(preferredMinWidth, availableWidth)
  dropdown.style.left = `${left}px`
  dropdown.style.top = `${rect.bottom + 4}px`
  dropdown.style.minWidth = `${minWidth}px`
  dropdown.style.maxWidth = `${availableWidth}px`
}

function formatCount(value: number) {
  return Intl.NumberFormat(locale.value || monitorIntlLocale(), {
    notation: value >= 10000 ? 'compact' : 'standard',
  }).format(value)
}

function onDocumentMouseDown(event: MouseEvent) {
  if (!isOpen.value) return
  const target = event.target as Node | null
  if (!target) return
  if (containerRef.value?.contains(target)) return
  if (dropdownRef.value?.contains(target)) return
  close()
}

function onDocumentKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

function onWindowChange() {
  if (isOpen.value) positionDropdown()
}

watch(isOpen, async (open) => {
  if (open) {
    await nextTick()
    positionDropdown()
    document.addEventListener('mousedown', onDocumentMouseDown)
    document.addEventListener('keydown', onDocumentKeyDown)
    window.addEventListener('resize', onWindowChange)
    window.addEventListener('scroll', onWindowChange, true)
  } else {
    document.removeEventListener('mousedown', onDocumentMouseDown)
    document.removeEventListener('keydown', onDocumentKeyDown)
    window.removeEventListener('resize', onWindowChange)
    window.removeEventListener('scroll', onWindowChange, true)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentMouseDown)
  document.removeEventListener('keydown', onDocumentKeyDown)
  window.removeEventListener('resize', onWindowChange)
  window.removeEventListener('scroll', onWindowChange, true)
})
</script>

<style scoped>
.select-trigger {
  @apply flex w-full items-center justify-between gap-2;
  @apply rounded-xl px-4 py-2.5 text-sm;
  background: rgb(var(--surface));
  border: 1px solid rgb(var(--line));
  color: rgb(var(--ink));
  transition: border-color var(--motion-fast) var(--motion-ease), background-color var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
  @apply focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20;
  @apply cursor-pointer;
}

.select-trigger:hover {
  border-color: rgb(var(--line-strong));
  background: rgb(var(--surface-2) / 0.55);
}

.select-trigger-open {
  @apply border-primary-500 ring-2 ring-primary-500/20;
}

.select-trigger-filtered {
  border-color: rgb(var(--brand-moss) / 0.45);
  background: rgb(var(--brand-moss) / 0.06);
}

.select-icon {
  color: rgb(var(--ink-3));
}

.filter-menu summary::-webkit-details-marker {
  display: none;
}

.filter-dropdown {
  @apply w-max min-w-[200px] max-h-[min(50vh,360px)] overflow-y-auto rounded-xl;
  background: rgb(var(--surface-elevated));
  border: 1px solid rgb(var(--line));
  color: rgb(var(--ink));
  box-shadow: var(--shadow-overlay);
}

.dropdown-item {
  @apply cursor-pointer;
  color: rgb(var(--ink-2));
}

.dropdown-item:hover {
  background: rgb(var(--surface-2));
  color: rgb(var(--ink));
}

.select-option-group {
  color: rgb(var(--ink));
  border-bottom: 1px solid rgb(var(--line));
}

.select-option-selected {
  background: rgb(var(--brand-moss) / 0.08);
  color: rgb(var(--ink));
  font-weight: 500;
}

.checkbox {
  flex: none;
}

.checkbox-unchecked {
  border-color: rgb(var(--line-strong));
  background: rgb(var(--surface));
}

.checkbox-checked {
  border-color: rgb(var(--brand-moss));
  background: rgb(var(--brand-moss) / 0.12);
}

.option-count,
.filter-empty {
  color: rgb(var(--ink-3));
}

@media (max-width: 640px) {
  .filter-menu {
    min-width: 100%;
  }
}
</style>
