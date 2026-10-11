import { computed, getCurrentInstance, onActivated, onDeactivated, onScopeDispose, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useMotionPreferences } from './useMotionPreferences'

interface PointerSurfaceOptions {
  enabled?: MaybeRefOrGetter<boolean>
  magnetic?: MaybeRefOrGetter<boolean>
  lift?: MaybeRefOrGetter<boolean>
}

const activeSurfaces = new Set<() => void>()
let surfaceOwners = 0
function listenForInvalidation() {
  const resetActive = () => activeSurfaces.forEach(reset => reset())
  window.addEventListener('blur', resetActive)
  window.addEventListener('scroll', resetActive, true)
  window.addEventListener('resize', resetActive)
  return () => {
    window.removeEventListener('blur', resetActive)
    window.removeEventListener('scroll', resetActive, true)
    window.removeEventListener('resize', resetActive)
  }
}
let stopInvalidation: (() => void) | undefined
function retainInvalidation() {
  if (++surfaceOwners === 1) stopInvalidation = listenForInvalidation()
  return () => {
    if (--surfaceOwners === 0) { stopInvalidation?.(); stopInvalidation = undefined }
  }
}

/** Pointer coordinates stay outside Vue's render loop; button hit areas never move. */
export function usePointerSurface(target: MaybeRefOrGetter<HTMLElement | null | undefined>, options: PointerSurfaceOptions = {}) {
  const { pointerEffectsEnabled } = useMotionPreferences()
  const suspended = ref(false)
  if (getCurrentInstance()) {
    onActivated(() => { suspended.value = false })
    onDeactivated(() => { suspended.value = true })
  }
  const enabled = computed(() => !suspended.value && pointerEffectsEnabled.value && toValue(options.enabled ?? true))
  let cleanup = () => {}
  watch([() => toValue(target), enabled], ([element, active]) => {
    cleanup()
    cleanup = () => {}
    if (!element || !active) return
    let frame = 0
    let bounds: DOMRect | null = null
    let clientX = 0, clientY = 0
    let inside = false
    const releaseInvalidation = retainInvalidation()
    const blocked = () => element.matches(':disabled, [aria-disabled="true"]') || !!element.closest('[inert]')
    const reset = () => {
      if (frame) cancelAnimationFrame(frame)
      frame = 0
      bounds = null
      inside = false
      activeSurfaces.delete(reset)
      element.dataset.pointerActive = 'false'
      for (const name of ['--ink-pointer-x', '--ink-pointer-y', '--ink-magnet-x', '--ink-magnet-y', '--ink-lift']) element.style.removeProperty(name)
    }
    const draw = () => {
      frame = 0
      if (!inside || blocked()) { reset(); return }
      bounds ??= element.getBoundingClientRect()
      if (!bounds.width || !bounds.height) return
      const x = Math.min(bounds.width, Math.max(0, clientX - bounds.left))
      const y = Math.min(bounds.height, Math.max(0, clientY - bounds.top))
      element.style.setProperty('--ink-pointer-x', `${x.toFixed(1)}px`)
      element.style.setProperty('--ink-pointer-y', `${y.toFixed(1)}px`)
      const magnetic = toValue(options.magnetic ?? false)
      element.style.setProperty('--ink-magnet-x', `${magnetic ? ((x / bounds.width - .5) * 6).toFixed(2) : 0}px`)
      element.style.setProperty('--ink-magnet-y', `${magnetic ? ((y / bounds.height - .5) * 6).toFixed(2) : 0}px`)
      element.style.setProperty('--ink-lift', toValue(options.lift ?? false) ? '-2px' : '0px')
      element.dataset.pointerActive = 'true'
    }
    const move = (event: PointerEvent) => {
      if (event.pointerType !== 'mouse' || blocked()) { reset(); return }
      const nearest = event.composedPath().find(node => node instanceof HTMLElement && node.hasAttribute('data-ink-surface'))
      if (nearest && nearest !== element) { reset(); return }
      inside = true
      activeSurfaces.add(reset)
      clientX = event.clientX
      clientY = event.clientY
      if (!frame) frame = requestAnimationFrame(draw)
    }
    element.addEventListener('pointerenter', move)
    element.addEventListener('pointermove', move)
    element.addEventListener('pointerleave', reset)
    element.addEventListener('pointercancel', reset)
    cleanup = () => {
      reset()
      releaseInvalidation()
      element.removeEventListener('pointerenter', move)
      element.removeEventListener('pointermove', move)
      element.removeEventListener('pointerleave', reset)
      element.removeEventListener('pointercancel', reset)
    }
  }, { immediate: true, flush: 'post' })
  onScopeDispose(() => cleanup())
}
