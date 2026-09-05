<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="modal-overlay"
        :style="zIndexStyle"
        :aria-labelledby="dialogId"
        role="dialog"
        aria-modal="true"
        @click.self="handleOverlayClick"
      >
        <!-- Modal panel -->
        <div ref="dialogRef" :class="['modal-content', widthClasses]" tabindex="-1" @click.stop>
          <!-- Header -->
          <div class="modal-header">
            <h3 :id="dialogId" class="modal-title">
              {{ title }}
            </h3>
            <button
              v-if="showCloseButton"
              type="button"
              @click="handleClose"
              class="-mr-2 rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 focus-visible:ring-offset-2 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-white dark:focus-visible:ring-offset-dark-900"
              :aria-label="t('common.close')"
            >
              <Icon name="x" size="md" />
            </button>
          </div>

          <!-- Body -->
          <div ref="modalBodyRef" class="modal-body">
            <slot></slot>
          </div>

          <!-- Footer -->
          <div v-if="$slots.footer" class="modal-footer">
            <slot name="footer"></slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts">
let dialogIdCounter = 0

type DialogWidth = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'

interface DialogStackEntry {
  focus: () => void
  contains: (element: HTMLElement) => boolean
  replaceRestoreTarget: (closedDialog: DialogStackEntry, replacement: HTMLElement | null) => void
}

const openDialogs: DialogStackEntry[] = []

const getTopDialog = () => openDialogs[openDialogs.length - 1]

const syncScrollLock = () => {
  document.body.classList.toggle('modal-open', openDialogs.length > 0)
}

const registerDialog = (dialog: DialogStackEntry) => {
  openDialogs.push(dialog)
  syncScrollLock()
}

const unregisterDialog = (dialog: DialogStackEntry) => {
  const index = openDialogs.indexOf(dialog)
  if (index === -1) return false

  const wasTop = index === openDialogs.length - 1
  openDialogs.splice(index, 1)
  syncScrollLock()
  return wasTop
}

const isTopDialog = (dialog: DialogStackEntry) => getTopDialog() === dialog
</script>

<script setup lang="ts">
import { computed, watch, onMounted, onBeforeUnmount, ref, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const dialogId = `modal-title-${++dialogIdCounter}`

const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[contenteditable="true"]',
  '[tabindex]:not([tabindex="-1"])'
].join(', ')

const dialogRef = ref<HTMLElement | null>(null)
const modalBodyRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null
let isRegistered = false

const dialogEntry: DialogStackEntry = {
  focus: () => focusInitialElement(),
  contains: (element) => dialogRef.value?.contains(element) ?? false,
  replaceRestoreTarget: (closedDialog, replacement) => {
    if (previousActiveElement && closedDialog.contains(previousActiveElement)) {
      previousActiveElement = replacement
    }
  }
}

interface Props {
  show: boolean
  title: string
  width?: DialogWidth
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  showCloseButton?: boolean
  zIndex?: number
}

interface Emits {
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  width: 'normal',
  closeOnEscape: true,
  closeOnClickOutside: false,
  showCloseButton: true,
  zIndex: 50
})

const emit = defineEmits<Emits>()

const zIndexStyle = computed(() => {
  return props.zIndex !== 50 ? { zIndex: props.zIndex } : undefined
})

const widthClasses = computed(() => {
  const widths: Record<DialogWidth, string> = {
    narrow: 'max-w-md',
    normal: 'max-w-lg',
    wide: 'w-full sm:max-w-2xl md:max-w-3xl lg:max-w-4xl',
    'extra-wide': 'w-full sm:max-w-3xl md:max-w-4xl lg:max-w-5xl xl:max-w-6xl',
    full: 'w-full sm:max-w-4xl md:max-w-5xl lg:max-w-6xl xl:max-w-7xl'
  }
  return widths[props.width]
})

const getFocusableElements = () => {
  if (!dialogRef.value) return []

  return Array.from(dialogRef.value.querySelectorAll<HTMLElement>(focusableSelector)).filter(
    (element) => {
      if (element.tabIndex < 0 || element.matches(':disabled') || element.closest('[hidden], [inert], [aria-hidden="true"]')) return false
      if (getComputedStyle(element).visibility === 'hidden' || getComputedStyle(element).visibility === 'collapse') return false
      for (let ancestor: HTMLElement | null = element; ancestor; ancestor = ancestor.parentElement) {
        if (getComputedStyle(ancestor).display === 'none') return false
      }
      return true
    }
  )
}

function focusInitialElement() {
  const [firstFocusable] = getFocusableElements()
  ;(firstFocusable ?? dialogRef.value)?.focus()
}

const handleClose = () => {
  if (isTopDialog(dialogEntry)) emit('close')
}

const handleOverlayClick = () => {
  if (props.closeOnClickOutside && isTopDialog(dialogEntry)) {
    emit('close')
  }
}

const handleTab = (event: KeyboardEvent) => {
  const focusableElements = getFocusableElements()
  const activeElement = document.activeElement as HTMLElement | null

  if (focusableElements.length === 0) {
    event.preventDefault()
    dialogRef.value?.focus()
    return
  }

  const firstFocusable = focusableElements[0]
  const lastFocusable = focusableElements[focusableElements.length - 1]
  const activeElementIsFocusable = activeElement ? focusableElements.includes(activeElement) : false

  if (
    (event.shiftKey && (!activeElementIsFocusable || activeElement === firstFocusable)) ||
    (!event.shiftKey && (!activeElementIsFocusable || activeElement === lastFocusable))
  ) {
    event.preventDefault()
    ;(event.shiftKey ? lastFocusable : firstFocusable).focus()
  }
}

const handleKeydown = (event: KeyboardEvent) => {
  if (!props.show || !isTopDialog(dialogEntry)) return

  if (event.key === 'Tab') {
    handleTab(event)
  } else if (event.key === 'Escape' && props.closeOnEscape) {
    event.preventDefault()
    emit('close')
  }
}

const canRestoreFocus = (element: HTMLElement) => {
  if (!element.isConnected) return false
  return !element.closest('.modal-content') || openDialogs.some((dialog) => dialog.contains(element))
}

const openDialog = async () => {
  if (isRegistered) return

  previousActiveElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
  isRegistered = true
  registerDialog(dialogEntry)

  await nextTick()
  if (!isRegistered || !props.show || !isTopDialog(dialogEntry)) return

  if (modalBodyRef.value) {
    modalBodyRef.value.scrollTop = 0
  }
  focusInitialElement()
}

const closeDialog = () => {
  if (!isRegistered) return

  const restoreTarget = previousActiveElement
  previousActiveElement = null
  isRegistered = false

  const wasTop = unregisterDialog(dialogEntry)
  openDialogs.forEach((dialog) => dialog.replaceRestoreTarget(dialogEntry, restoreTarget))
  if (!wasTop) return

  const nextTopDialog = getTopDialog()
  void nextTick().then(() => {
    if (getTopDialog() !== nextTopDialog) return
    if (restoreTarget && canRestoreFocus(restoreTarget) && (!nextTopDialog || nextTopDialog.contains(restoreTarget))) {
      restoreTarget.focus()
    } else {
      nextTopDialog?.focus()
    }
  })
}

watch(
  () => props.show,
  (isOpen) => {
    if (isOpen) {
      void openDialog()
    } else {
      closeDialog()
    }
  },
  { immediate: true }
)

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown)
  closeDialog()
})
</script>
