<template>
  <button ref="trigger" type="button" class="glacier-action-trigger" :aria-label="label" :title="label" aria-haspopup="menu" :aria-expanded="open" @click.stop="toggle" @keydown.down.prevent="show">
    <Icon name="more" size="sm" />
  </button>
  <Teleport to="body">
    <div v-if="open" ref="panel" role="menu" :aria-label="label" class="glacier-action-menu" :style="position" @keydown="onKeydown">
      <slot :close="close" />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, nextTick, onUnmounted, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppearance } from '@/composables/useAppearance'
defineProps<{ label: string }>()
const { style } = useAppearance()
const open = ref(false)
const trigger = ref<HTMLButtonElement>()
const panel = ref<HTMLDivElement>()
const position = ref({ left: '0px', top: '0px' })
const buttons = () => Array.from(panel.value?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') ?? [])

function close(restoreFocus = true) {
  open.value = false
  document.removeEventListener('pointerdown', outside)
  window.removeEventListener('resize', dismiss)
  window.removeEventListener('scroll', scroll, true)
  if (restoreFocus) trigger.value?.focus()
}
function dismiss() { close(false) }
function scroll(event: Event) {
  if (event.target instanceof Node && panel.value?.contains(event.target)) return
  dismiss()
}
function outside(event: PointerEvent) {
  if (event.target instanceof Node && !panel.value?.contains(event.target) && !trigger.value?.contains(event.target)) dismiss()
}
async function show() {
  if (open.value || !trigger.value) return
  open.value = true
  await nextTick()
  if (!open.value || !trigger.value || !panel.value) return
  const rect = trigger.value.getBoundingClientRect()
  const height = panel.value.offsetHeight
  const width = panel.value.offsetWidth
  position.value = {
    left: `${Math.max(12, Math.min(rect.right - width, window.innerWidth - width - 12))}px`,
    top: `${Math.max(12, rect.bottom + 8 + height <= window.innerHeight - 12 ? rect.bottom + 8 : rect.top - height - 8)}px`
  }
  buttons().forEach(button => { button.setAttribute('role', 'menuitem'); button.tabIndex = -1 })
  buttons()[0]?.focus()
  document.addEventListener('pointerdown', outside)
  window.addEventListener('resize', dismiss)
  window.addEventListener('scroll', scroll, true)
}
function toggle() { if (open.value) close(); else void show() }
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close(); return }
  if (event.key === 'Tab') { close(); return }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const items = buttons()
  if (!items.length) return
  const index = items.indexOf(document.activeElement as HTMLButtonElement)
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (index + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length
  items[next]?.focus()
}
watch(style, dismiss)
onUnmounted(dismiss)
</script>
