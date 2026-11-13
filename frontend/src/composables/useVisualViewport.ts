import { onScopeDispose, readonly, shallowRef, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue'

export interface VisualViewportRect {
  left: number
  top: number
  width: number
  height: number
  right: number
  bottom: number
}

/** Visible space, including an on-screen keyboard, without changing content or focus. */
export function useVisualViewport(active: MaybeRefOrGetter<boolean>): Readonly<Ref<VisualViewportRect>> {
  const rect = shallowRef<VisualViewportRect>({ left: 0, top: 0, width: 0, height: 0, right: 0, bottom: 0 })
  const size = (...values: (number | undefined)[]) => values.find(value => Number.isFinite(value) && value! > 0) ?? 0
  const stop = watch(() => toValue(active), (enabled, _, onCleanup) => {
    if (!enabled || typeof window === 'undefined' || typeof document === 'undefined') return
    const viewport = window.visualViewport
    let frame: number | undefined
    const measure = () => {
      const left = Number.isFinite(viewport?.offsetLeft) ? Math.max(0, viewport!.offsetLeft) : 0
      const top = Number.isFinite(viewport?.offsetTop) ? Math.max(0, viewport!.offsetTop) : 0
      const width = size(viewport?.width, document.documentElement.clientWidth, window.innerWidth)
      const height = size(viewport?.height, document.documentElement.clientHeight, window.innerHeight)
      const next = { left, top, width, height, right: left + width, bottom: top + height }
      if (Object.keys(next).some(key => next[key as keyof VisualViewportRect] !== rect.value[key as keyof VisualViewportRect])) rect.value = next
    }
    const schedule = () => {
      if (frame !== undefined) return
      frame = window.requestAnimationFrame(() => { frame = undefined; measure() })
    }
    measure()
    window.addEventListener('resize', schedule)
    window.addEventListener('scroll', schedule, true)
    viewport?.addEventListener('resize', schedule)
    viewport?.addEventListener('scroll', schedule)
    onCleanup(() => {
      if (frame !== undefined) window.cancelAnimationFrame(frame)
      window.removeEventListener('resize', schedule)
      window.removeEventListener('scroll', schedule, true)
      viewport?.removeEventListener('resize', schedule)
      viewport?.removeEventListener('scroll', schedule)
    })
  }, { immediate: true, flush: 'sync' })
  onScopeDispose(stop)
  return readonly(rect)
}
