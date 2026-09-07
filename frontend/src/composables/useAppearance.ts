import { ref } from 'vue'

export type ThemeStyle = 'aurora' | 'lagoon' | 'graphite'
export type ThemeMode = 'dark' | 'light' | 'system'
const styles: ThemeStyle[] = ['aurora', 'lagoon', 'graphite']
const style = ref<ThemeStyle>('aurora')
const mode = ref<ThemeMode>('dark')
const dark = ref(true)
let systemTheme: MediaQueryList | undefined

function persist(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* Storage may be disabled by the browser. */ }
}

function applyMode() {
  dark.value = mode.value === 'dark' || (mode.value === 'system' && !!systemTheme?.matches)
  document.documentElement.classList.toggle('dark', dark.value)
}

export function initAppearance() {
  let savedStyle: string | null = null
  let savedMode: string | null = null
  try {
    savedStyle = localStorage.getItem('appearance-style')
    savedMode = localStorage.getItem('theme')
  } catch { /* Use dark appearance without persistent storage. */ }
  style.value = styles.includes(savedStyle as ThemeStyle) ? savedStyle as ThemeStyle : 'aurora'
  mode.value = savedMode === 'light' || savedMode === 'system' ? savedMode : 'dark'
  if (!systemTheme) {
    systemTheme = window.matchMedia('(prefers-color-scheme: dark)')
    systemTheme.addEventListener('change', applyMode)
  }
  document.documentElement.dataset.style = style.value
  applyMode()
}

export function useAppearance() {
  function setStyle(value: ThemeStyle) {
    if (!styles.includes(value)) return
    style.value = value
    document.documentElement.dataset.style = value
    persist('appearance-style', value)
  }
  function setMode(value: ThemeMode) {
    if (!['dark', 'light', 'system'].includes(value)) return
    mode.value = value
    applyMode()
    persist('theme', value)
  }
  function toggleTheme() {
    setMode(dark.value ? 'light' : 'dark')
  }
  return { style, mode, isDark: dark, setStyle, setMode, toggleTheme }
}
