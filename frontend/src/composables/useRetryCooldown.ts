import { onBeforeUnmount, ref } from 'vue'
import { retryAfterSeconds } from '@/utils/userFacingError'

export function useRetryCooldown() {
  const remaining = ref(0)
  let timer: ReturnType<typeof setInterval> | undefined
  function clear() {
    if (timer) clearInterval(timer)
    timer = undefined
    remaining.value = 0
  }
  function start(error: unknown) {
    clear()
    const seconds = retryAfterSeconds(error)
    if (!seconds) return
    const until = Date.now() + seconds * 1000
    remaining.value = seconds
    timer = setInterval(() => {
      remaining.value = Math.max(0, Math.ceil((until - Date.now()) / 1000))
      if (!remaining.value) clear()
    }, 250)
  }
  onBeforeUnmount(clear)
  return { remaining, start, clear }
}
