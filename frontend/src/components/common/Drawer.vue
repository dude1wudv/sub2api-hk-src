<template>
  <Teleport to="body">
    <Transition name="drawer-backdrop">
      <div v-if="show" class="fixed inset-0 z-[60] bg-black/45" @click.self="closeFromOutside" />
    </Transition>
    <Transition name="drawer-panel" @after-enter="focusFirstControl">
      <aside
        v-if="show"
        ref="panelRef"
        class="fixed inset-y-0 right-0 z-[61] flex w-full max-w-full flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900"
        :style="{ width: `${resolvedWidth}px` }"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        tabindex="-1"
        @keydown="handleKeydown"
      >
        <header class="flex h-14 shrink-0 items-center justify-between border-b border-gray-200 px-5 dark:border-dark-700">
          <h2 :id="titleId" class="text-sm font-semibold text-gray-900 dark:text-white">{{ title }}</h2>
          <button
            type="button"
            class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :aria-label="closeLabel"
            @click="closeFromButton"
          >
            <Icon name="x" size="sm" aria-hidden="true" />
          </button>
        </header>
        <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="shrink-0 border-t border-gray-200 px-5 py-4 dark:border-dark-700">
          <slot name="footer" />
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'

let drawerId = 0

interface Props {
  show: boolean
  title: string
  width?: number
  closeLabel: string
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  width: 480,
  closeOnEscape: true,
  closeOnClickOutside: true
})
const emit = defineEmits<{ 'update:show': [value: boolean]; close: [] }>()
const panelRef = ref<HTMLElement | null>(null)
const titleId = `drawer-title-${++drawerId}`
let previousActiveElement: HTMLElement | null = null

const resolvedWidth = computed(() => Math.min(600, Math.max(420, props.width)))

const closeFromButton = () => {
  emit('update:show', false)
  emit('close')
}

const closeFromOutside = () => {
  if (props.closeOnClickOutside) closeFromButton()
}

const focusFirstControl = async () => {
  await nextTick()
  panelRef.value?.querySelector<HTMLElement>('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')?.focus()
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.closeOnEscape) {
    event.preventDefault()
    closeFromButton()
    return
  }
  if (event.key !== 'Tab' || !panelRef.value) return
  const controls = Array.from(panelRef.value.querySelectorAll<HTMLElement>('button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'))
  if (!controls.length) return
  const first = controls[0]
  const last = controls[controls.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(
  () => props.show,
  (isOpen) => {
    if (isOpen) {
      previousActiveElement = document.activeElement as HTMLElement | null
      document.body.classList.add('modal-open')
      return
    }
    document.body.classList.remove('modal-open')
    previousActiveElement?.focus()
    previousActiveElement = null
  }
)

onUnmounted(() => document.body.classList.remove('modal-open'))
</script>

<style scoped>
.drawer-backdrop-enter-active,
.drawer-backdrop-leave-active { transition: opacity 160ms ease-out; }
.drawer-backdrop-enter-from,
.drawer-backdrop-leave-to { opacity: 0; }
.drawer-panel-enter-active,
.drawer-panel-leave-active { transition: transform 200ms ease-out; }
.drawer-panel-enter-from,
.drawer-panel-leave-to { transform: translateX(100%); }
@media (prefers-reduced-motion: reduce) {
  .drawer-backdrop-enter-active,
  .drawer-backdrop-leave-active,
  .drawer-panel-enter-active,
  .drawer-panel-leave-active { transition-duration: 1ms; }
}
</style>
