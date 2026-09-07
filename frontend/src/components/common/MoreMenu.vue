<template>
  <div ref="rootRef" class="relative inline-flex">
    <slot name="trigger" :open="open" :toggle="toggle" :close="close">
      <button
        ref="triggerRef"
        type="button"
        class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
        :aria-label="ariaLabel"
        aria-haspopup="menu"
        :aria-expanded="open"
        @click="toggle"
        @keydown.down.prevent="openMenu"
      >
        <Icon name="more" size="sm" aria-hidden="true" />
      </button>
    </slot>
    <div
      v-if="open"
      ref="menuRef"
      class="absolute z-40 mt-1 min-w-40 overflow-hidden rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-700 dark:bg-dark-900"
      :class="align === 'left' ? 'left-0' : 'right-0'"
      role="menu"
      @keydown="handleKeydown"
      @click.capture="handleMenuClick"
    >
      <slot :close="close" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  ariaLabel: string
  align?: 'left' | 'right'
}

const props = withDefaults(defineProps<Props>(), { align: 'right' })
const emit = defineEmits<{ 'update:open': [value: boolean] }>()
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const open = ref(false)

const close = (restoreFocus = false) => {
  if (!open.value) return
  open.value = false
  emit('update:open', false)
  if (restoreFocus) triggerRef.value?.focus()
}

const menuItems = () => Array.from(menuRef.value?.querySelectorAll<HTMLElement>('[role="menuitem"], button:not(:disabled), a[href]') ?? [])

const openMenu = async () => {
  if (open.value) return
  open.value = true
  emit('update:open', true)
  await nextTick()
  menuItems()[0]?.focus()
}

const toggle = () => {
  if (open.value) close()
  else void openMenu()
}

const handleDocumentPointerDown = (event: PointerEvent) => {
  if (!rootRef.value?.contains(event.target as Node)) close()
}

const handleMenuClick = (event: MouseEvent) => {
  const target = event.target
  if (!(target instanceof Element)) return
  if (target.closest('[data-menu-keep-open]')) return
  if (target.closest('[role="menuitem"], button, a[href]')) close()
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    event.preventDefault()
    close(true)
    return
  }
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp' && event.key !== 'Home' && event.key !== 'End') return
  const items = menuItems()
  if (!items.length) return
  event.preventDefault()
  const currentIndex = items.indexOf(document.activeElement as HTMLElement)
  if (event.key === 'Home') items[0]?.focus()
  else if (event.key === 'End') items[items.length - 1]?.focus()
  else if (event.key === 'ArrowDown') items[(currentIndex + 1 + items.length) % items.length]?.focus()
  else items[(currentIndex - 1 + items.length) % items.length]?.focus()
}

onMounted(() => document.addEventListener('pointerdown', handleDocumentPointerDown))
onUnmounted(() => document.removeEventListener('pointerdown', handleDocumentPointerDown))

defineExpose({ close, openMenu, toggle })
</script>
