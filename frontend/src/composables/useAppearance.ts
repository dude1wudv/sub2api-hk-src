import { ref } from 'vue'

export type ThemeStyle = 'aurora' | 'lagoon' | 'graphite' | 'glacier'
const styles: ThemeStyle[] = ['aurora', 'lagoon', 'graphite', 'glacier']
const defaultStyle: ThemeStyle = 'glacier'
const styleMigrationKey = 'appearance-style-migration'
const styleMigrationVersion = 'glacier-default-v1'
const style = ref<ThemeStyle>(defaultStyle)
const dark = ref(false)

function persist(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* Storage may be disabled by the browser. */ }
}

export function initAppearance() {
  let savedStyle: string | null = null
  let savedMode: string | null = null
  let savedMigration: string | null = null
  try {
    savedStyle = localStorage.getItem('appearance-style')
    savedMode = localStorage.getItem('theme')
    savedMigration = localStorage.getItem(styleMigrationKey)
  } catch { /* Use system appearance without persistent storage. */ }
  // Reset existing browser preferences once; later explicit choices still persist.
  if (savedMigration !== styleMigrationVersion) {
    savedStyle = defaultStyle
    try {
      localStorage.setItem('appearance-style', defaultStyle)
      localStorage.setItem(styleMigrationKey, styleMigrationVersion)
    } catch { /* Apply the default even when persistent storage is unavailable. */ }
  }
  style.value = styles.includes(savedStyle as ThemeStyle) ? savedStyle as ThemeStyle : defaultStyle
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
