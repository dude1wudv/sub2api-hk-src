import { ref } from 'vue'

export type ThemeStyle = 'aurora' | 'lagoon' | 'graphite'
const styles: ThemeStyle[] = ['aurora', 'lagoon', 'graphite']
const style = ref<ThemeStyle>('aurora')
const dark = ref(false)

function persist(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* Storage may be disabled by the browser. */ }
}

export function initAppearance() {
  let savedStyle: string | null = null
  let savedMode: string | null = null
  try {
    savedStyle = localStorage.getItem('appearance-style')
    savedMode = localStorage.getItem('theme')
  } catch { /* Use system appearance without persistent storage. */ }
  style.value = styles.includes(savedStyle as ThemeStyle) ? savedStyle as ThemeStyle : 'aurora'
  dark.value = savedMode ? savedMode === 'dark' : window.matchMedia('(prefers-color-scheme: dark)').matches
  document.documentElement.dataset.style = style.value
  document.documentElement.classList.toggle('dark', dark.value)
}

export function useAppearance() {
  function setStyle(value: ThemeStyle) {
    if (!styles.includes(value)) return
    style.value = value
    document.documentElement.dataset.style = value
    persist('appearance-style', value)
  }
  function toggleTheme() {
    dark.value = !document.documentElement.classList.contains('dark')
    document.documentElement.classList.toggle('dark', dark.value)
    persist('theme', dark.value ? 'dark' : 'light')
  }
  return { style, isDark: dark, setStyle, toggleTheme }
}
