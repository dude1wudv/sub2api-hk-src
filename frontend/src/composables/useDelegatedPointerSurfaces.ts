import { onScopeDispose, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { nearestPointerSurface } from './usePointerSurface'
import { useMotionPreferences } from './useMotionPreferences'
import '@/styles/ink-hover.css'

const overlaySelector = 'dialog[open], [aria-modal="true"], [role="menu"], [role="listbox"]'
const blockedSelector = ':disabled, [aria-disabled="true"], [inert], [hidden], [aria-hidden="true"], [data-ink-hover="off"]'
const variables = ['--ink-hover-x', '--ink-hover-y', '--ink-hover-base-image', '--ink-hover-base-size', '--ink-hover-base-position', '--ink-hover-base-repeat']

/** Background-only decoration: no extra children, text scaling or foreground overlay. */
export function useDelegatedPointerSurfaces(resetOn?: MaybeRefOrGetter<unknown>) {
  const { pointerEffectsEnabled } = useMotionPreferences()
  let active: HTMLElement | undefined
  let bounds: DOMRect | undefined
  let previousVariables = new Map<string, { value: string; priority: string }>()
  let frame = 0
  let x = 0, y = 0
  let mutations: MutationObserver | undefined
  let resize: ResizeObserver | undefined

  const hasOverlay = () => [...document.querySelectorAll<HTMLElement>(overlaySelector)]
    .some(element => !element.hidden && element.getAttribute('aria-hidden') !== 'true' && getComputedStyle(element).display !== 'none')

  function reset() {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
    mutations?.disconnect()
    resize?.disconnect()
    mutations = undefined
    resize = undefined
    if (active) {
      active.removeAttribute('data-ink-hover-active')
      active.removeAttribute('data-ink-hover-frame')
      for (const name of variables) {
        const previous = previousVariables.get(name)
        if (previous?.value) active.style.setProperty(name, previous.value, previous.priority)
        else active.style.removeProperty(name)
      }
    }
    previousVariables.clear()
    active = undefined
    bounds = undefined
  }

  function activate(element: HTMLElement) {
    reset()
    if (!element.isConnected || element.closest(blockedSelector) || hasOverlay()) return
    const rect = element.getBoundingClientRect()
    if (!rect.width || !rect.height) return
    const style = getComputedStyle(element)
    // Geometry/background are read once on entry. All frame work below is writes only.
    active = element
    bounds = rect
    previousVariables = new Map(variables.map(name => [name, { value: element.style.getPropertyValue(name), priority: element.style.getPropertyPriority(name) }]))
    element.style.setProperty('--ink-hover-base-image', style.backgroundImage || 'none')
    element.style.setProperty('--ink-hover-base-size', style.backgroundSize || 'auto')
    element.style.setProperty('--ink-hover-base-position', style.backgroundPosition || '0% 0%')
    element.style.setProperty('--ink-hover-base-repeat', style.backgroundRepeat || 'repeat')
    element.dataset.inkHoverFrame = 'idle'
    mutations = new MutationObserver(() => {
      if (!active?.isConnected || active.closest(blockedSelector) || hasOverlay()) reset()
    })
    mutations.observe(document.body, { childList: true, subtree: true, attributes: true,
      attributeFilter: ['open', 'aria-modal', 'inert', 'disabled', 'aria-disabled', 'hidden', 'aria-hidden', 'data-ink-hover'] })
    if (typeof ResizeObserver !== 'undefined') {
      let previousSize: string | undefined
      resize = new ResizeObserver(entries => {
        const entry = entries[0]
        if (!entry) return
        const size = `${entry.contentRect.width}:${entry.contentRect.height}`
        if (previousSize !== undefined && previousSize !== size) reset()
        previousSize = size
      })
      resize.observe(element)
    }
  }

  function draw() {
    frame = 0
    if (!active?.isConnected || !bounds || active.closest(blockedSelector)) { reset(); return }
    active.dataset.inkHoverFrame = 'idle'
    active.style.setProperty('--ink-hover-x', `${Math.min(bounds.width, Math.max(0, x - bounds.left)).toFixed(1)}px`)
    active.style.setProperty('--ink-hover-y', `${Math.min(bounds.height, Math.max(0, y - bounds.top)).toFixed(1)}px`)
    active.dataset.inkHoverActive = 'true'
  }

  function move(event: PointerEvent) {
    if (event.pointerType !== 'mouse') { reset(); return }
    const element = nearestPointerSurface(event)
    if (!element || element.hasAttribute('data-ink-surface') || element.closest(blockedSelector)) { reset(); return }
    if (element !== active) activate(element)
    if (!active) return
    x = event.clientX
    y = event.clientY
    if (!frame) {
      active.dataset.inkHoverFrame = 'pending'
      frame = requestAnimationFrame(draw)
    }
  }
  function leave(event: PointerEvent) {
    if (!(event.relatedTarget instanceof Node) || !active?.contains(event.relatedTarget)) reset()
  }

  const stop = watch(pointerEffectsEnabled, (enabled, _previous, onCleanup) => {
    reset()
    if (!enabled) return
    document.addEventListener('pointerover', move, true)
    document.addEventListener('pointermove', move, true)
    document.addEventListener('pointerout', leave, true)
    document.addEventListener('pointercancel', reset, true)
    window.addEventListener('scroll', reset, true)
    window.addEventListener('resize', reset)
    window.addEventListener('blur', reset)
    onCleanup(() => { unlisten(); reset() })
  }, { immediate: true, flush: 'sync' })
  // Listener removal runs before a preference is re-enabled, and on App disposal.
  function unlisten() {
    document.removeEventListener('pointerover', move, true)
    document.removeEventListener('pointermove', move, true)
    document.removeEventListener('pointerout', leave, true)
    document.removeEventListener('pointercancel', reset, true)
    window.removeEventListener('scroll', reset, true)
    window.removeEventListener('resize', reset)
    window.removeEventListener('blur', reset)
  }
  if (resetOn !== undefined) watch(() => toValue(resetOn), reset, { flush: 'sync' })
  onScopeDispose(() => { stop(); unlisten(); reset() })
  return { reset }
}
