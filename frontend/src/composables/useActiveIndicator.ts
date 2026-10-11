import { onScopeDispose, readonly, shallowRef, toValue, watch, type MaybeRefOrGetter } from 'vue'

export interface IndicatorGeometry {
  x: number
  y: number
  width: number
  height: number
  visible: boolean
  key: string | null
}

export function useActiveIndicator(
  container: MaybeRefOrGetter<HTMLElement | null | undefined>,
  activeKey: MaybeRefOrGetter<string | number | null>,
  selector: MaybeRefOrGetter<string> = '[data-active-indicator-item]',
) {
  const geometry = shallowRef<IndicatorGeometry>({ x: 0, y: 0, width: 0, height: 0, visible: false, key: null })
  let stop = () => {}
  let schedule = () => {}
  watch(() => toValue(container), element => {
    stop()
    stop = () => {}
    schedule = () => {}
    if (!element) { geometry.value = { ...geometry.value, visible: false }; return }
    let frame = 0
    let disposed = false
    let observedItem: HTMLElement | undefined
    const measure = () => {
      frame = 0
      if (disposed) return
      const key = toValue(activeKey)
      const items = Array.from(element.querySelectorAll<HTMLElement>(toValue(selector)))
      const selected = (key === null ? undefined : items.find(item => item.dataset.activeIndicatorItem === String(key)))
        ?? items.find(item => item.getAttribute('aria-selected') === 'true' || ['page', 'true', 'step', 'location'].includes(item.getAttribute('aria-current') || ''))
      if (selected !== observedItem) {
        if (observedItem) resize?.unobserve(observedItem)
        observedItem = selected
        if (selected) resize?.observe(selected)
      }
      const rect = selected?.getBoundingClientRect()
      if (!rect?.width || !rect.height || selected?.hidden) {
        if (geometry.value.visible) geometry.value = { ...geometry.value, visible: false }
        return
      }
      const bounds = element.getBoundingClientRect()
      const next: IndicatorGeometry = {
        x: rect.left - bounds.left + element.scrollLeft - element.clientLeft,
        y: rect.top - bounds.top + element.scrollTop - element.clientTop,
        width: rect.width,
        height: rect.height,
        visible: true,
        key: selected?.dataset.activeIndicatorItem || (key === null ? selected?.id || null : String(key)),
      }
      if (Object.keys(next).some(key => next[key as keyof IndicatorGeometry] !== geometry.value[key as keyof IndicatorGeometry])) geometry.value = next
    }
    schedule = () => { if (!frame && !disposed) frame = requestAnimationFrame(measure) }
    const resize = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(schedule)
    resize?.observe(element)
    const mutations = typeof MutationObserver === 'undefined' ? undefined : new MutationObserver(schedule)
    mutations?.observe(element, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['aria-current', 'aria-selected', 'data-active-indicator-item', 'hidden'] })
    element.addEventListener('scroll', schedule, { passive: true })
    window.addEventListener('resize', schedule)
    void document.fonts?.ready.then(() => { if (!disposed) schedule() })
    schedule()
    const scheduleForElement = schedule
    stop = () => {
      disposed = true
      if (frame) cancelAnimationFrame(frame)
      resize?.disconnect()
      mutations?.disconnect()
      element.removeEventListener('scroll', scheduleForElement)
      window.removeEventListener('resize', scheduleForElement)
    }
  }, { immediate: true, flush: 'post' })
  watch([() => toValue(activeKey), () => toValue(selector)], () => schedule(), { flush: 'post' })
  onScopeDispose(() => stop())
  return { geometry: readonly(geometry), refresh: () => schedule() }
}
