import { onMounted, onUnmounted, watch, type Ref } from 'vue'

/** One delegated, frame-limited pointer listener; never moves text or table rows. */
export function useGlacierInteraction(enabled: Ref<boolean>) {
  let frame = 0
  let active: HTMLElement | null = null
  let pending: PointerEvent | null = null
  let fine: MediaQueryList | undefined
  let reduced: MediaQueryList | undefined
  let attached = false
  let stopWatch: (() => void) | undefined
  const selector = '.btn, .select-trigger, .sidebar-link, .appearance-controls, .glacier-row-action, .glacier-action-trigger, .glacier-disclosure > summary, .card[data-metric], .hero-instrument'

  function clear() {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
    pending = null
    active?.removeAttribute('data-glass-hover')
    active?.style.removeProperty('--glass-x')
    active?.style.removeProperty('--glass-y')
    active = null
  }

  function render() {
    frame = 0
    const event = pending
    pending = null
    if (!event || !(event.target instanceof Element)) return
    const target = event.target.closest<HTMLElement>(selector)
    if (target !== active) {
      active?.removeAttribute('data-glass-hover')
      active?.style.removeProperty('--glass-x')
      active?.style.removeProperty('--glass-y')
      active = target
    }
    if (!target || target.matches(':disabled, [aria-disabled="true"]')) return
    const rect = target.getBoundingClientRect()
    target.style.setProperty('--glass-x', `${event.clientX - rect.left}px`)
    target.style.setProperty('--glass-y', `${event.clientY - rect.top}px`)
    target.setAttribute('data-glass-hover', '')
  }

  function move(event: PointerEvent) {
    if (event.pointerType === 'touch') return
    pending = event
    if (!frame) frame = requestAnimationFrame(render)
  }

  function sync() {
    const shouldAttach = enabled.value && !!fine?.matches && !reduced?.matches
    if (shouldAttach === attached) return
    attached = shouldAttach
    if (attached) {
      document.addEventListener('pointermove', move, { passive: true })
      document.addEventListener('pointerleave', clear)
      window.addEventListener('blur', clear)
    } else {
      document.removeEventListener('pointermove', move)
      document.removeEventListener('pointerleave', clear)
      window.removeEventListener('blur', clear)
      clear()
    }
  }

  onMounted(() => {
    fine = window.matchMedia('(hover: hover) and (pointer: fine)')
    reduced = window.matchMedia('(prefers-reduced-motion: reduce)')
    fine.addEventListener('change', sync)
    reduced.addEventListener('change', sync)
    stopWatch = watch(enabled, sync, { immediate: true })
  })
  onUnmounted(() => {
    stopWatch?.()
    fine?.removeEventListener('change', sync)
    reduced?.removeEventListener('change', sync)
    document.removeEventListener('pointermove', move)
    document.removeEventListener('pointerleave', clear)
    window.removeEventListener('blur', clear)
    clear()
  })
}
