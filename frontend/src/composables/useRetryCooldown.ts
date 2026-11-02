import { onBeforeUnmount, ref } from 'vue'
import { retryAfterSeconds } from '@/utils/userFacingError'

export function useRetryCooldown() {
  const remaining = ref(0)
  let timer: ReturnType<typeof setInterval> | undefined
  let until = 0
  function clear() {
    if (timer) clearInterval(timer)
    timer = undefined
    until = 0
    remaining.value = 0
  }
  function start(error: unknown) {
    const seconds = retryAfterSeconds(error)
    if (!seconds) return
    until = Math.max(until, Date.now() + seconds * 1000)
    remaining.value = Math.max(0, Math.ceil((until - Date.now()) / 1000))
    if (timer) return
    timer = setInterval(() => {
      remaining.value = Math.max(0, Math.ceil((until - Date.now()) / 1000))
      if (!remaining.value) clear()
    }, 250)
  }
  onBeforeUnmount(clear)
  return { remaining, start, clear }
}
