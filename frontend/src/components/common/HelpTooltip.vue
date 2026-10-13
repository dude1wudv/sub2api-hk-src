<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, useTemplateRef, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  content?: string
  id?: string
  label?: string
  trigger?: 'hover' | 'click' | 'both'
  widthClass?: string
}>(), {
  trigger: 'hover',
  widthClass: 'w-64',
})

const { t } = useI18n()
const generatedId = useId()
const tooltipId = computed(() => props.id || `help-tooltip-${generatedId}`)
const show = ref(false)
const pinned = ref(false)
const placement = ref<'top' | 'bottom'>('top')
const triggerRef = useTemplateRef<HTMLElement>('trigger')
const tooltipRef = useTemplateRef<HTMLElement>('tooltip')
const contentRef = useTemplateRef<HTMLElement>('content')
const tooltipStyle = ref<CSSProperties>({ top: '12px', left: '12px', maxWidth: 'calc(100vw - 24px)' })
const triggerAttrs = computed(() => ({
  'aria-describedby': props.trigger === 'hover' ? tooltipId.value : undefined,
  'aria-controls': props.trigger !== 'hover' ? tooltipId.value : undefined,
  'aria-expanded': props.trigger !== 'hover' ? show.value : undefined,
  'aria-haspopup': props.trigger !== 'hover' ? 'dialog' as const : undefined,
}))
let restoringFocus = false
let resizeObserver: ResizeObserver | undefined

async function openTooltip(focusContent = false) {
  show.value = true
  await nextTick()
  await updatePosition()
  if (show.value && focusContent) tooltipRef.value?.focus({ preventScroll: true })
}

function closeTooltip(restoreFocus = false) {
  show.value = false
  pinned.value = false
  if (!restoreFocus) return
  const control = triggerRef.value?.querySelector<HTMLElement>('button, a[href], [tabindex]')
  restoringFocus = true
  control?.focus({ preventScroll: true })
  restoringFocus = false
}

function onEnter() {
  if (props.trigger !== 'click') void openTooltip()
}

function isInside(container: HTMLElement | null, target: EventTarget | null): boolean {
  return target instanceof Node && !!container?.contains(target)
}

function onLeave(event: MouseEvent) {
  if (props.trigger === 'click' || pinned.value) return
  if (isInside(tooltipRef.value, event.relatedTarget)) return
  if (isInside(triggerRef.value, document.activeElement) || isInside(tooltipRef.value, document.activeElement)) return
  closeTooltip()
}

function onTooltipLeave(event: MouseEvent) {
  if (props.trigger === 'click' || pinned.value) return
  if (isInside(triggerRef.value, event.relatedTarget)) return
  if (isInside(triggerRef.value, document.activeElement) || isInside(tooltipRef.value, document.activeElement)) return
  closeTooltip()
}

function onFocusIn() {
  if (props.trigger !== 'click' && !restoringFocus) void openTooltip()
}

function onFocusOut(event: FocusEvent) {
  if (isInside(triggerRef.value, event.relatedTarget) || isInside(tooltipRef.value, event.relatedTarget)) return
  closeTooltip()
}

function onClick(event: MouseEvent) {
  if (props.trigger === 'hover') return
  event.stopPropagation()
  if (show.value && (props.trigger === 'click' || pinned.value)) closeTooltip()
  else {
    pinned.value = true
    void openTooltip(event.detail === 0)
  }
}

function onTriggerKeydown(event: KeyboardEvent) {
  if (event.key !== 'ArrowDown') return
  event.preventDefault()
  event.stopPropagation()
  pinned.value = true
  void openTooltip(true)
}

function onDocumentClick(event: MouseEvent) {
  if (props.trigger === 'hover' || !show.value) return
  if (isInside(triggerRef.value, event.target) || isInside(tooltipRef.value, event.target)) return
  closeTooltip()
}

function onDocumentKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || !show.value) return
  event.preventDefault()
  event.stopPropagation()
  closeTooltip(isInside(tooltipRef.value, document.activeElement))
}

function onViewportChange(event: Event) {
  // Scrolling long details must not reset their available height or scroll position.
  if (isInside(tooltipRef.value, event.target)) return
  if (show.value) void updatePosition()
}

async function updatePosition() {
  const el = triggerRef.value
  const tooltip = tooltipRef.value
  if (!show.value || !el || !tooltip) return
  const viewport = window.visualViewport
  const width = viewport?.width || document.documentElement.clientWidth || window.innerWidth
  const height = viewport?.height || document.documentElement.clientHeight || window.innerHeight
  const leftEdge = (viewport?.offsetLeft || 0) + 12
  const topEdge = (viewport?.offsetTop || 0) + 12
  const maxWidth = Math.max(0, width - 24)
  const maxHeight = Math.max(0, height - 24)
  tooltipStyle.value = { ...tooltipStyle.value, maxWidth: `${maxWidth}px` }
  await nextTick()
  if (!show.value) return

  // Fixed positioning uses viewport coordinates, including when the document scrolls.
  const rect = el.getBoundingClientRect()
  const tooltipRect = tooltip.getBoundingClientRect()
  const tooltipWidth = Math.min(tooltipRect.width, maxWidth)
  const naturalHeight = Math.max(tooltipRect.height, (contentRef.value?.scrollHeight || 0) + tooltip.offsetHeight - tooltip.clientHeight)
  const tooltipHeight = Math.min(naturalHeight, maxHeight)
  const above = Math.max(0, rect.top - topEdge - 8)
  const below = Math.max(0, topEdge + maxHeight - rect.bottom - 8)
  placement.value = tooltipHeight <= above || above >= below ? 'top' : 'bottom'
  const availableHeight = Math.min(maxHeight, placement.value === 'top' ? above : below)
  const renderedHeight = Math.min(tooltipHeight, availableHeight)
  const desiredTop = placement.value === 'top' ? rect.top - 8 - renderedHeight : rect.bottom + 8
  tooltipStyle.value = {
    left: `${Math.min(Math.max(rect.left + rect.width / 2 - tooltipWidth / 2, leftEdge), leftEdge + maxWidth - tooltipWidth)}px`,
    top: `${Math.min(Math.max(desiredTop, topEdge), topEdge + maxHeight - renderedHeight)}px`,
    maxWidth: `${maxWidth}px`,
    maxHeight: `${availableHeight}px`,
  }
}

onMounted(() => {
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      if (show.value) void updatePosition()
    })
    if (triggerRef.value) resizeObserver.observe(triggerRef.value)
    if (tooltipRef.value) resizeObserver.observe(tooltipRef.value)
  }
  document.addEventListener('click', onDocumentClick, true)
  document.addEventListener('keydown', onDocumentKeydown)
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', onViewportChange, true)
  window.visualViewport?.addEventListener('resize', onViewportChange)
  window.visualViewport?.addEventListener('scroll', onViewportChange)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  document.removeEventListener('click', onDocumentClick, true)
  document.removeEventListener('keydown', onDocumentKeydown)
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', onViewportChange, true)
  window.visualViewport?.removeEventListener('resize', onViewportChange)
  window.visualViewport?.removeEventListener('scroll', onViewportChange)
})
</script>

<template>
  <div
    ref="trigger"
    class="group relative ml-1 inline-flex items-center align-middle"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
    @click="onClick"
    @keydown="onTriggerKeydown"
    @focusin="onFocusIn"
    @focusout="onFocusOut"
  >
    <slot name="trigger" :trigger-attrs="triggerAttrs" :open="show" :id="tooltipId">
      <button
        v-bind="triggerAttrs"
        type="button"
        class="inline-flex min-h-6 min-w-6 items-center justify-center rounded text-gray-400 transition-colors hover:text-primary-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 max-md:min-h-11 max-md:min-w-11 dark:text-gray-500 dark:hover:text-primary-400"
        :aria-label="props.label || t('common.info')"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </button>
    </slot>

    <Teleport to="body">
      <div
        ref="tooltip"
        v-show="show"
        :id="tooltipId"
        :role="props.trigger !== 'hover' ? 'dialog' : 'tooltip'"
        :aria-label="props.trigger !== 'hover' ? props.label || t('common.info') : undefined"
        :data-placement="placement"
        tabindex="-1"
        :class="[
          'ui-popover fixed z-[99999] rounded-lg text-xs leading-relaxed selection:bg-primary-200 selection:text-gray-900 before:absolute before:inset-x-0 before:h-2',
          placement === 'top' ? 'before:top-full' : 'before:bottom-full',
          props.widthClass,
        ]"
        :style="tooltipStyle"
        @mouseleave="onTooltipLeave"
        @focusin="onFocusIn"
        @focusout="onFocusOut"
      >
        <div
          ref="content"
          class="overflow-y-auto overscroll-contain whitespace-normal p-3 [overflow-wrap:anywhere]"
          :class="{ 'pr-12': props.trigger !== 'hover' }"
          :style="{ maxHeight: tooltipStyle.maxHeight }"
        >
          <slot :open="show">{{ content }}</slot>
        </div>
        <button
          v-if="props.trigger !== 'hover'"
          type="button"
          class="absolute right-0 top-0 flex min-h-11 min-w-11 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-gray-300 dark:hover:bg-white/10 dark:hover:text-white dark:focus-visible:ring-white"
          :aria-label="t('common.close')"
          @click.stop="closeTooltip(true)"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </Teleport>
  </div>
</template>
