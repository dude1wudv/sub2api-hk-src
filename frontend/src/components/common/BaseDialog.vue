<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="modal-overlay"
        :class="{ 'admin-detail-overlay': adminSurfaceActive && presentation === 'drawer' }"
        :style="zIndexStyle"
        :aria-labelledby="dialogId"
        role="dialog"
        aria-modal="true"
        @click.self="handleClose"
      >
        <!-- Modal panel -->
        <div ref="dialogRef" :class="['modal-content', widthClasses, { 'admin-edit-modal': adminSurfaceActive && presentation === 'editor' }]" tabindex="-1" @click.stop>
          <!-- Header -->
          <div class="modal-header">
            <h3 :id="dialogId" class="modal-title">
              {{ title }}
            </h3>
            <button
              v-if="showCloseButton"
              type="button"
              @click="requestClose"
              :disabled="busy"
              class="modal-close -mr-2 rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 focus-visible:ring-offset-2 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-white dark:focus-visible:ring-offset-dark-900"
              :aria-label="zh ? '关闭弹窗' : 'Close modal'"
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
          <div v-if="confirmDiscard" ref="discardRef" class="admin-discard-confirm" role="alertdialog" :aria-label="zh ? '未保存的修改' : 'Unsaved changes'" aria-modal="true">
            <p>{{ zh ? '更改尚未保存。关闭后将放弃本次修改。' : 'Your changes have not been saved. Closing will discard them.' }}</p>
            <div><button type="button" class="btn btn-secondary" @click="keepEditing">{{ zh ? '继续编辑' : 'Keep editing' }}</button><button type="button" class="btn btn-danger" @click="discard">{{ zh ? '放弃修改' : 'Discard changes' }}</button></div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts">
let dialogIdCounter = 0
const openDialogs = new Set<string>()
</script>

<script setup lang="ts">
import { computed, watch, onMounted, onUnmounted, ref, nextTick, getCurrentInstance } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { adminSurfaceActive } from '@/composables/adminSurface'
import { useScrollLock } from '@/composables/useScrollLock'

// 生成唯一ID以避免多个对话框时ID冲突
const dialogId = `modal-title-${++dialogIdCounter}`

// 焦点管理
const dialogRef = ref<HTMLElement | null>(null)
const modalBodyRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null

type DialogWidth = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'

interface Props {
  show: boolean
  title: string
  width?: DialogWidth
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  showCloseButton?: boolean
  zIndex?: number
  presentation?: 'modal' | 'drawer' | 'editor'
  dirty?: boolean
  busy?: boolean
}

interface Emits {
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  width: 'normal',
  closeOnEscape: true,
  closeOnClickOutside: false,
  showCloseButton: true,
  zIndex: 50,
  presentation: 'modal',
  dirty: false,
  busy: false
})

const emit = defineEmits<Emits>()
useScrollLock(() => props.show)
const instance = getCurrentInstance()
const zh = computed(() => String(instance?.appContext.config.globalProperties.$i18n?.locale ?? document.documentElement.lang).startsWith('zh'))
const confirmDiscard = ref(false)
const discardRef = ref<HTMLElement | null>(null)
let editingFocus: HTMLElement | null = null
function requestClose() {
  if (props.busy) return
  if (props.dirty) {
    editingFocus = document.activeElement as HTMLElement
    confirmDiscard.value = true
    void nextTick(() => discardRef.value?.querySelector('button')?.focus())
  } else emit('close')
}
function keepEditing() { confirmDiscard.value = false; void nextTick(() => editingFocus?.focus()) }
function discard() { confirmDiscard.value = false; emit('close') }
defineExpose({ requestClose })

// Custom z-index style (overrides the default z-50 from CSS)
const zIndexStyle = computed(() => {
  return props.zIndex !== 50 ? { zIndex: props.zIndex } : undefined
})

const widthClasses = computed(() => {
  // Width guidance: narrow=confirm/short prompts, normal=standard forms,
  // wide=multi-section forms or rich content, extra-wide=analytics/tables,
  // full=full-screen or very dense layouts.
  const widths: Record<DialogWidth, string> = {
    narrow: 'max-w-md',
    normal: 'max-w-lg',
    wide: 'w-full sm:max-w-2xl md:max-w-3xl lg:max-w-4xl',
    'extra-wide': 'w-full sm:max-w-3xl md:max-w-4xl lg:max-w-5xl xl:max-w-6xl',
    full: 'w-full sm:max-w-4xl md:max-w-5xl lg:max-w-6xl xl:max-w-7xl'
  }
  return widths[props.width]
})

const handleClose = () => {
  if (props.closeOnClickOutside) {
    requestClose()
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (!props.show || [...openDialogs].pop() !== dialogId) return
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    if (props.closeOnEscape) {
      if (confirmDiscard.value) keepEditing()
      else requestClose()
    }
  }
  if (event.key === 'Tab') {
    const panel = confirmDiscard.value ? discardRef.value : dialogRef.value
    const focusable = [...(panel?.querySelectorAll<HTMLElement>('button:not([disabled]),a[href],input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])') ?? [])]
      .filter(el => !el.closest('[hidden],[inert]') && getComputedStyle(el).visibility !== 'hidden' && !hasHiddenParent(el, panel))
    const first = focusable[0], last = focusable[focusable.length - 1]
    if (!first) { event.preventDefault(); panel?.focus(); return }
    if (event.shiftKey && (document.activeElement === first || !panel?.contains(document.activeElement))) { event.preventDefault(); last.focus() }
    else if (!event.shiftKey && (document.activeElement === last || !panel?.contains(document.activeElement))) { event.preventDefault(); first.focus() }
  }
}
function hasHiddenParent(el: HTMLElement, panel: HTMLElement | null) {
  for (let node: HTMLElement | null = el; node && node !== panel; node = node.parentElement) if (getComputedStyle(node).display === 'none') return true
  return false
}

const updateDialogStack = (isOpen: boolean) => {
  if (isOpen) openDialogs.add(dialogId)
  else openDialogs.delete(dialogId)
}

function restoreFocus() {
  if (previousActiveElement?.isConnected && !previousActiveElement.closest('[inert]')) previousActiveElement.focus({ preventScroll: true })
  previousActiveElement = null
}

// Prevent body scroll when modal is open and manage focus
watch(
  () => props.show,
  async (isOpen) => {
    confirmDiscard.value = false
    if (isOpen) {
      // 保存当前焦点元素
      previousActiveElement = document.activeElement as HTMLElement
      updateDialogStack(true)

      // 等待DOM更新后设置焦点到对话框
      await nextTick()
      if (!props.show) return
      if (modalBodyRef.value) {
        modalBodyRef.value.scrollTop = 0
      }
      if (dialogRef.value) {
        const firstFocusable = dialogRef.value.querySelector<HTMLElement>(
          '[autofocus]:not([disabled]), button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
        )
        ;(firstFocusable || dialogRef.value).focus({ preventScroll: true })
      }
    } else {
      updateDialogStack(false)
      // 恢复之前的焦点
      restoreFocus()
    }
  },
  { immediate: true }
)

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape)
  // 确保组件卸载时移除滚动锁定
  updateDialogStack(false)
  restoreFocus()
})
</script>

<style scoped>
.modal-overlay { padding: max(24px, env(safe-area-inset-top, 0px)) max(24px, env(safe-area-inset-right, 0px)) max(24px, env(safe-area-inset-bottom, 0px)) max(24px, env(safe-area-inset-left, 0px)); }
.modal-content { max-height: calc(100dvh - 48px); color: rgb(var(--ink)); background: rgb(var(--surface)); font-family: var(--font-sans); }
.modal-body { overscroll-behavior: contain; }
.modal-close { display: inline-grid; place-items: center; flex-shrink: 0; transition-duration: var(--motion-fast, 160ms); }
.modal-title { font-family: var(--font-sans); }
.modal-enter-active, .modal-leave-active { transition-duration: var(--motion-overlay, 260ms); }
.modal-enter-active .modal-content, .modal-leave-active .modal-content { transition-duration: var(--motion-overlay, 260ms); transition-timing-function: var(--motion-ease, cubic-bezier(.2,.8,.2,1)); }
@media (max-width: 767px) {
  .modal-overlay { padding: max(12px, env(safe-area-inset-top, 0px)) max(12px, env(safe-area-inset-right, 0px)) max(12px, env(safe-area-inset-bottom, 0px)) max(12px, env(safe-area-inset-left, 0px)); }
  .modal-content { max-height: calc(100dvh - 24px); }
  .modal-close { min-width: 44px; min-height: 44px; }
  .modal-header { padding-top: max(16px, env(safe-area-inset-top, 0px)); }
  .modal-footer { padding-bottom: max(16px, env(safe-area-inset-bottom, 0px)); }
  .modal-body :deep(input:not([type='checkbox']):not([type='radio'])), .modal-body :deep(select), .modal-body :deep(textarea) { font-size: 16px; }
}
@media (prefers-reduced-motion: reduce) {
  .modal-enter-active, .modal-leave-active, .modal-enter-active .modal-content, .modal-leave-active .modal-content { transition-duration: .01ms; }
}
</style>
