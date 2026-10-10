<template>
  <Teleport to="body">
    <dialog ref="element" class="portal-account-modal" aria-labelledby="portal-account-dialog-title" @cancel.prevent="close" @click="onBackdropClick">
      <header class="portal-account-modal__header">
        <h2 id="portal-account-dialog-title">{{ title }}</h2>
        <button type="button" class="portal-account-close" :aria-label="closeLabel" :disabled="busy" @click="close">×</button>
      </header>
      <slot />
    </dialog>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const props = withDefaults(defineProps<{ title: string; busy?: boolean; closeLabel?: string }>(), {
  busy: false,
  closeLabel: '关闭'
})
const emit = defineEmits<{ close: [] }>()
const element = ref<HTMLDialogElement | null>(null)
let previousFocus: HTMLElement | null = null

function close() {
  if (!props.busy) emit('close')
}

function onBackdropClick(event: MouseEvent) {
  const dialog = element.value
  if (!dialog || event.target !== dialog) return
  const bounds = dialog.getBoundingClientRect()
  if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) close()
}

onMounted(async () => {
  previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  await nextTick()
  element.value?.showModal()
  element.value?.querySelector<HTMLElement>('[autofocus], input, select, textarea')?.focus()
})

onBeforeUnmount(() => {
  element.value?.close()
  if (previousFocus?.isConnected) previousFocus.focus()
})
</script>
