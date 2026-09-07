<template>
  <Teleport to="body">
    <dialog ref="dialogRef" class="command-dialog" aria-labelledby="command-title" @cancel.prevent="close" @click="handleBackdrop" @keydown="handleKeydown" @close="handleClosed">
      <div class="command-panel">
        <h2 id="command-title" class="sr-only">{{ t('console.searchPages') }}</h2>
        <div class="flex items-center gap-3 border-b border-gray-200 px-4 dark:border-dark-700">
          <Icon name="search" size="sm" class="text-gray-500" />
          <input ref="inputRef" v-model="query" class="command-input" role="combobox" aria-autocomplete="list" aria-controls="command-results" :aria-expanded="true" :aria-activedescendant="results.length ? `command-result-${selected}` : undefined" :aria-label="t('console.searchPages')" :placeholder="t('console.searchPages')" />
          <button type="button" class="btn-ghost px-2 py-1 text-xs" :aria-label="t('common.close')" @click="close">Esc</button>
        </div>
        <div id="command-results" class="command-results" role="listbox" :aria-label="t('console.searchPages')">
          <div v-for="(item, index) in results" :id="`command-result-${index}`" :key="item.path" role="option" :aria-selected="selected === index" class="command-result" :class="{ 'command-result-active': selected === index }" @mousemove="selected = index" @click="activate(item)">
            <span class="min-w-0"><span class="block truncate text-[13px] font-medium">{{ item.label }}</span><span class="text-xs text-gray-500 dark:text-dark-400">{{ item.group }}</span></span>
            <span class="ml-auto hidden truncate font-mono text-xs text-gray-500 sm:block">{{ item.path }}</span>
            <Icon v-if="selected === index" name="arrowRight" size="sm" />
          </div>
          <p v-if="!results.length" class="px-4 py-10 text-center text-sm text-gray-500" role="status">{{ t('console.noPages') }}</p>
        </div>
        <div class="border-t border-gray-200 px-4 py-2 text-xs text-gray-500 dark:border-dark-700">{{ t('console.commandHint') }}</div>
      </div>
    </dialog>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { NavigationCommand } from './navigation'
const props = defineProps<{ open: boolean; commands: NavigationCommand[] }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()
const { t } = useI18n()
const dialogRef = ref<HTMLDialogElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const query = ref('')
const selected = ref(0)
let returnFocus: HTMLElement | null = null
const results = computed(() => {
  const words = query.value.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean)
  return props.commands.filter(item => words.every(word => `${item.label} ${item.group} ${item.path}`.toLocaleLowerCase().includes(word)))
})
watch(results, () => { selected.value = 0 })
watch(() => props.open, async open => {
  if (open) {
    returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    query.value = ''
    selected.value = 0
    await nextTick()
    if (!props.open) return
    dialogRef.value?.showModal()
    inputRef.value?.focus()
  } else dialogRef.value?.close()
})
function close() { emit('update:open', false) }
function handleClosed() {
  close()
  if (returnFocus?.isConnected) returnFocus.focus()
  returnFocus = null
}
function handleBackdrop(event: MouseEvent) {
  const dialog = dialogRef.value
  if (!dialog || event.target !== dialog) return
  const bounds = dialog.getBoundingClientRect()
  if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) close()
}
function activate(item: NavigationCommand) {
  close()
  item.run()
}
function handleKeydown(event: KeyboardEvent) {
  if (event.isComposing) return
  if (event.key === 'Escape') { event.preventDefault(); close(); return }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    if (!results.value.length) return
    selected.value = (selected.value + (event.key === 'ArrowDown' ? 1 : -1) + results.value.length) % results.value.length
    void nextTick(() => document.getElementById(`command-result-${selected.value}`)?.scrollIntoView({ block: 'nearest' }))
  } else if (event.key === 'Enter' && event.target === inputRef.value) {
    event.preventDefault()
    const item = results.value[selected.value]
    if (item) activate(item)
  } else if (event.key === 'Tab') {
    const controls = Array.from(dialogRef.value?.querySelectorAll<HTMLElement>('input, button:not(:disabled)') ?? [])
    const first = controls[0]
    const last = controls[controls.length - 1]
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus() }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
  }
}
onBeforeUnmount(() => { dialogRef.value?.close(); if (returnFocus?.isConnected) returnFocus.focus() })
</script>

<style scoped>
.command-dialog { width: min(600px, calc(100vw - 32px)); max-height: calc(100dvh - 64px); margin: min(16vh, 120px) auto auto; padding: 0; color: rgb(var(--ink)); background: rgb(var(--surface)); border: 1px solid rgb(var(--line)); border-radius: 12px; box-shadow: 0 24px 80px rgb(0 0 0 / .3); }
.command-dialog::backdrop { background: rgb(0 0 0 / .55); }
.command-panel { display: flex; flex-direction: column; max-height: calc(100dvh - 96px); }
.command-input { min-width: 0; flex: 1; height: 56px; border: 0; outline: none; box-shadow: none; background: transparent; font-size: 14px; }
.command-results { overflow-y: auto; padding: 6px; }
.command-result { display: flex; align-items: center; gap: 12px; min-height: 52px; padding: 8px 12px; border-radius: 6px; cursor: pointer; }
.command-result-active { background: rgb(var(--selection-bg)); color: rgb(var(--selection-ink)); }
</style>
