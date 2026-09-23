import { onBeforeUnmount, onUpdated, watch, type Ref } from 'vue'
import { useGlacierPreferences } from './useGlacierPreferences'

/** Align opaque scene crops with the glass behind a sticky table cell.
 * A translucent td or backdrop-filter cannot reliably mask sibling cells in Chromium.
 * Explicit viewport coordinates also avoid background-attachment:fixed becoming local
 * inside a filtered ancestor. Only visible, mounted cells are measured, once per frame.
 */
export function useGlacierTableSurface(wrapper: Ref<HTMLElement | null>) {
  const { isGlassLayout: enabled } = useGlacierPreferences()
  let frame = 0
  let detach: (() => void) | undefined

  function schedule() {
    if (!enabled.value || !wrapper.value || frame) return
    frame = requestAnimationFrame(() => {
      frame = 0
      if (!enabled.value || !wrapper.value) return
      // Read the whole batch before writing styles to avoid repeated layout flushes.
      const positions = [...wrapper.value.querySelectorAll<HTMLElement>('.sticky-header-cell, tbody .sticky-col')]
        .map(cell => ({ cell, rect: cell.getBoundingClientRect() }))
        .filter(({ rect }) => rect.bottom >= 0 && rect.top <= window.innerHeight)
      for (const { cell, rect } of positions) {
        cell.style.setProperty('--glacier-cell-left', `${rect.left}px`)
        cell.style.setProperty('--glacier-cell-top', `${rect.top}px`)
      }
    })
  }

  watch([enabled, wrapper], ([active, element]) => {
    detach?.()
    detach = undefined
    if (!active || !element) return
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(schedule)
    observer?.observe(element)
    const table = element.querySelector('table')
    if (table) observer?.observe(table)
    // Capture includes table scrolling and scrolling of its page/outer containers.
    document.addEventListener('scroll', schedule, { capture: true, passive: true })
    window.addEventListener('resize', schedule, { passive: true })
    detach = () => {
      observer?.disconnect()
      document.removeEventListener('scroll', schedule, true)
      window.removeEventListener('resize', schedule)
      cancelAnimationFrame(frame)
      frame = 0
    }
    schedule()
  }, { flush: 'post', immediate: true })

  // Sorting, virtual windows, column settings and density can move existing cells.
  onUpdated(schedule)
  onBeforeUnmount(() => detach?.())
}
