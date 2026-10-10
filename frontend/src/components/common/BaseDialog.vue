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
              @click="requestClose"
              :disabled="busy"
              class="-mr-2 rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 focus-visible:ring-offset-2 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-white dark:focus-visible:ring-offset-dark-900"
              aria-label="Close modal"
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
  if (props.closeOnEscape && event.key === 'Escape') {
    if (confirmDiscard.value) keepEditing()
    else requestClose()
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

const updateScrollLock = (isOpen: boolean) => {
  if (isOpen) openDialogs.add(dialogId)
  else openDialogs.delete(dialogId)
  document.body.classList.toggle('modal-open', openDialogs.size > 0)
}

// Prevent body scroll when modal is open and manage focus
watch(
  () => props.show,
  async (isOpen) => {
    confirmDiscard.value = false
    if (isOpen) {
      // 保存当前焦点元素
      previousActiveElement = document.activeElement as HTMLElement
      // 使用CSS类而不是直接操作style,更易于管理多个对话框
      updateScrollLock(true)

      // 等待DOM更新后设置焦点到对话框
      await nextTick()
      if (modalBodyRef.value) {
        modalBodyRef.value.scrollTop = 0
      }
      if (dialogRef.value) {
        const firstFocusable = dialogRef.value.querySelector<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
        firstFocusable?.focus()
      }
    } else {
      updateScrollLock(false)
      // 恢复之前的焦点
      if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
        previousActiveElement.focus()
      }
      previousActiveElement = null
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
  updateScrollLock(false)
})
</script>
