<template>
  <Teleport to="body">
    <dialog ref="dialog" class="portal-dialog" :class="{ wide }" :aria-labelledby="titleId" @cancel.prevent="close" @click="onBackdrop">
      <header class="portal-dialog-header"><h2 :id="titleId">{{ title }}</h2><button type="button" class="portal-inline-button" :aria-label="zh ? '关闭' : 'Close'" @click="close">✕</button></header>
      <slot />
    </dialog>
  </Teleport>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ open: boolean; title: string; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const dialog = ref<HTMLDialogElement | null>(null)
const titleId = useId()
function close() { emit('close') }
function onBackdrop(event: MouseEvent) {
  if (event.target !== dialog.value || !dialog.value) return
  const rect = dialog.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) close()
}
watch(() => props.open, async (open) => {
  await nextTick()
  if (open && dialog.value && !dialog.value.open) dialog.value.showModal()
  else if (!open) dialog.value?.close()
}, { immediate: true })
onBeforeUnmount(() => dialog.value?.close())
</script>
<style scoped>
dialog.portal-dialog { margin:auto; }
dialog.portal-dialog::backdrop { background:#00000040; }
</style>
