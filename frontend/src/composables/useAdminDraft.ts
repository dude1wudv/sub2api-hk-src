import { computed, nextTick, onBeforeUnmount, watch, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'

/** In-memory comparison only: credentials and draft data are never persisted. */
export function useAdminDraft(open: () => boolean, snapshot: () => unknown, busy: () => boolean = () => false, ready: () => boolean = () => true) {
  const { locale } = useI18n()
  const baseline = ref<string | null>(null)
  const dirty = computed(() => open() && baseline.value !== null && JSON.stringify(snapshot()) !== baseline.value)
  const resetDraft = () => { baseline.value = JSON.stringify(snapshot()) }
  watch([open, ready], async ([visible, hydrated]) => {
    if (!visible) baseline.value = null
    else if (hydrated && baseline.value === null) { await nextTick(); if (open() && ready() && baseline.value === null) resetDraft() }
  }, { immediate: true, flush: 'post' })
  const message = () => locale?.value?.startsWith('zh') ? '更改尚未保存，确定放弃修改并离开？' : 'You have unsaved changes. Discard them and leave?'
  onBeforeRouteLeave(() => busy() ? false : !dirty.value || window.confirm(message()))
  function beforeUnload(event: BeforeUnloadEvent) {
    if (dirty.value || (open() && busy())) { event.preventDefault(); event.returnValue = '' }
  }
  window.addEventListener('beforeunload', beforeUnload)
  onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
  return { dirty, resetDraft }
}
