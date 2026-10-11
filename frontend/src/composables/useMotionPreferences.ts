import { computed, readonly, ref } from 'vue'

export type MotionPreference = 'auto' | 'reduced'
const storageKey = 'motion-preference'
const preference = ref<MotionPreference>('auto')
const systemReduced = ref(false)
const finePointer = ref(false)
const wideViewport = ref(false)
const pageVisible = ref(true)
const reducedMotion = computed(() => preference.value === 'reduced' || systemReduced.value)
const pointerEffectsEnabled = computed(() => !reducedMotion.value && finePointer.value && wideViewport.value && pageVisible.value)
let disposeListeners: (() => void) | undefined

function validPreference(value: unknown): value is MotionPreference {
  return value === 'auto' || value === 'reduced'
}

function applyPreference() {
  if (typeof document !== 'undefined') document.documentElement.dataset.motion = reducedMotion.value ? 'reduced' : 'full'
}

/** One shared set of listeners, initialized before mounting or on first use. */
export function initMotionPreferences() {
  disposeListeners?.()
  preference.value = 'auto'
  try {
    const saved = localStorage.getItem(storageKey)
    if (validPreference(saved)) preference.value = saved
    else if (saved !== null) localStorage.removeItem(storageKey)
  } catch { /* System preference remains available when storage is blocked. */ }

  const listen = (query: string, update: (matches: boolean) => void) => {
    const media = typeof window !== 'undefined' && typeof window.matchMedia === 'function' ? window.matchMedia(query) : undefined
    const changed = () => { update(media?.matches ?? false); applyPreference() }
    changed()
    media?.addEventListener('change', changed)
    return () => media?.removeEventListener('change', changed)
  }
  const stops = [
    listen('(prefers-reduced-motion: reduce)', value => { systemReduced.value = value }),
    listen('(hover: hover) and (pointer: fine)', value => { finePointer.value = value }),
    listen('(min-width: 768px)', value => { wideViewport.value = value }),
  ]
  const onVisibility = () => { pageVisible.value = typeof document === 'undefined' || document.visibilityState !== 'hidden' }
  const onStorage = (event: StorageEvent) => {
    if (event.key !== storageKey && event.key !== null) return
    preference.value = validPreference(event.newValue) ? event.newValue : 'auto'
    applyPreference()
  }
  onVisibility()
  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibility)
  if (typeof window !== 'undefined') window.addEventListener('storage', onStorage)
  const cleanup = () => {
    stops.forEach(stop => stop())
    if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibility)
    if (typeof window !== 'undefined') window.removeEventListener('storage', onStorage)
    if (disposeListeners === cleanup) disposeListeners = undefined
  }
  disposeListeners = cleanup
  applyPreference()
  return cleanup
}

export function useMotionPreferences() {
  if (!disposeListeners) initMotionPreferences()
  function setPreference(value: MotionPreference) {
    if (!validPreference(value)) return
    preference.value = value
    applyPreference()
    try { localStorage.setItem(storageKey, value) } catch { /* Keep the session choice. */ }
  }
  return { preference: readonly(preference), reducedMotion, pointerEffectsEnabled, setPreference }
}

if (import.meta.hot) import.meta.hot.dispose(() => disposeListeners?.())
