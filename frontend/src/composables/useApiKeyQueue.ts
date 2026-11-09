import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { getConcurrencyQueue, type APIKeyQueueItem } from '@/api/keys'
import type { ApiKey } from '@/types'

/** Poll only the displayed keys; never keep a hidden page's queue request alive. */
export function useApiKeyQueue(keys: Ref<ApiKey[]>) {
  const counts = ref<Record<number, APIKeyQueueItem>>({})
  const unavailable = ref(false)
  let timer: ReturnType<typeof setInterval> | undefined
  let controller: AbortController | undefined
  let mounted = false
  async function refresh() {
    if (!mounted || document.hidden) return
    controller?.abort()
    const current = new AbortController()
    controller = current
    const ids = keys.value.map(key => key.id)
    if (!ids.length) { counts.value = {}; unavailable.value = false; return }
    try {
      const chunks = []
      for (let offset = 0; offset < ids.length; offset += 100) chunks.push(ids.slice(offset, offset + 100))
      const results = await Promise.all(chunks.map(chunk => getConcurrencyQueue(chunk, current.signal)))
      if (!mounted || current.signal.aborted) return
      counts.value = Object.fromEntries(results.flatMap(result => result.items).map(item => [item.id, item]))
      unavailable.value = false
    } catch {
      if (mounted && !current.signal.aborted) { counts.value = {}; unavailable.value = true }
    }
  }
  function visibility() {
    if (document.hidden) { controller?.abort(); counts.value = {} }
    else void refresh()
  }
  watch(() => keys.value.map(key => key.id).join(','), () => { counts.value = {}; void refresh() })
  onMounted(() => {
    mounted = true
    document.addEventListener('visibilitychange', visibility)
    timer = setInterval(() => { void refresh() }, 5000)
    void refresh()
  })
  onBeforeUnmount(() => {
    mounted = false
    clearInterval(timer)
    controller?.abort()
    document.removeEventListener('visibilitychange', visibility)
  })
  return { counts, unavailable, refresh }
}
