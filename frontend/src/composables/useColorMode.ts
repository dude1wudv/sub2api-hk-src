import { computed, readonly, ref } from 'vue'

export type ColorMode = 'light' | 'dark'

const mode = ref<ColorMode>('light')
let preference: ColorMode | null = null
let systemQuery: MediaQueryList | undefined
let disposeListeners: (() => void) | undefined

function validMode(value: unknown): value is ColorMode {
  return value === 'light' || value === 'dark'
}

function applyMode(value: ColorMode) {
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', value === 'dark')
    document.documentElement.style.colorScheme = value
    document.documentElement.removeAttribute('data-style')
  }
  // Apply the class before notifying chart consumers that read computed tokens.
  mode.value = value
}

function applyPreference() {
  applyMode(preference ?? (systemQuery?.matches ? 'dark' : 'light'))
}

/** Start once before mounting so every route and body-teleported surface agrees. */
export function initColorMode() {
  disposeListeners?.()
  let saved: string | null = null
  try {
    saved = localStorage.getItem('theme')
    localStorage.removeItem('appearance-style')
    localStorage.removeItem('appearance-style-migration')
    if (saved !== null && !validMode(saved)) localStorage.removeItem('theme')
  } catch { /* Storage can be unavailable; follow the system for this session. */ }
  preference = validMode(saved) ? saved : null
  systemQuery = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-color-scheme: dark)') : undefined
  applyPreference()

  const onSystemChange = () => { if (preference === null) applyPreference() }
  const onStorageChange = (event: StorageEvent) => {
    if (event.key !== 'theme' && event.key !== null) return
    preference = validMode(event.newValue) ? event.newValue : null
    applyPreference()
  }
  systemQuery?.addEventListener('change', onSystemChange)
  if (typeof window !== 'undefined') window.addEventListener('storage', onStorageChange)
  const observedQuery = systemQuery
  disposeListeners = () => {
    observedQuery?.removeEventListener('change', onSystemChange)
    if (typeof window !== 'undefined') window.removeEventListener('storage', onStorageChange)
  }
  return disposeListeners
}

export function useColorMode() {
  function setMode(value: ColorMode) {
    if (!validMode(value)) return
    preference = value
    applyMode(value)
    try { localStorage.setItem('theme', value) } catch { /* Keep the session choice. */ }
  }
  function toggleMode() { setMode(mode.value === 'dark' ? 'light' : 'dark') }
  return { mode: readonly(mode), isDark: computed(() => mode.value === 'dark'), setMode, toggleMode }
}

if (import.meta.hot) import.meta.hot.dispose(() => disposeListeners?.())
