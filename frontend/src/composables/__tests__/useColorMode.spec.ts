import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { initColorMode, useColorMode } from '../useColorMode'

let matches = false
const listeners = new Set<() => void>()
let dispose: (() => void) | undefined

function systemChanges(dark: boolean) {
  matches = dark
  listeners.forEach(listener => listener())
}

beforeEach(() => {
  localStorage.clear()
  matches = false
  listeners.clear()
  document.documentElement.classList.remove('dark')
  vi.stubGlobal('matchMedia', vi.fn(() => ({
    get matches() { return matches },
    addEventListener: (_: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_: string, listener: () => void) => listeners.delete(listener),
  })))
})
afterEach(() => { dispose?.(); vi.restoreAllMocks(); vi.unstubAllGlobals() })

describe('color mode', () => {
  it.each(['light', 'dark'] as const)('preserves a valid %s preference and only removes obsolete appearance keys', saved => {
    localStorage.setItem('theme', saved)
    localStorage.setItem('appearance-style', 'lagoon')
    localStorage.setItem('appearance-style-migration', 'glacier-default-v1')
    localStorage.setItem('unrelated-setting', 'keep')
    document.documentElement.dataset.style = 'lagoon'
    matches = saved === 'light'
    dispose = initColorMode()
    expect(useColorMode().mode.value).toBe(saved)
    expect(document.documentElement.classList.contains('dark')).toBe(saved === 'dark')
    expect(document.documentElement.style.colorScheme).toBe(saved)
    expect(document.documentElement.hasAttribute('data-style')).toBe(false)
    expect(localStorage.getItem('theme')).toBe(saved)
    expect(localStorage.getItem('unrelated-setting')).toBe('keep')
    expect(localStorage.getItem('appearance-style')).toBeNull()
    expect(localStorage.getItem('appearance-style-migration')).toBeNull()
  })

  it('follows system changes until the user chooses a mode', () => {
    matches = true
    dispose = initColorMode()
    const first = useColorMode(), second = useColorMode()
    expect(first.mode.value).toBe('dark')
    expect(localStorage.getItem('theme')).toBeNull()
    systemChanges(false)
    expect(second.isDark.value).toBe(false)
    first.toggleMode()
    expect(second.isDark.value).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    systemChanges(false)
    expect(first.mode.value).toBe('dark')
  })

  it('ignores invalid saved modes and uses the system default', () => {
    localStorage.setItem('theme', 'sepia')
    matches = true
    dispose = initColorMode()
    expect(useColorMode().mode.value).toBe('dark')
    expect(localStorage.getItem('theme')).toBeNull()
  })

  it('accepts mode changes from another tab and resumes system mode when its choice is cleared', () => {
    dispose = initColorMode()
    window.dispatchEvent(new StorageEvent('storage', { key: 'theme', newValue: 'dark' }))
    expect(useColorMode().isDark.value).toBe(true)
    window.dispatchEvent(new StorageEvent('storage', { key: 'other', newValue: 'light' }))
    expect(useColorMode().isDark.value).toBe(true)
    window.dispatchEvent(new StorageEvent('storage', { key: 'theme', newValue: null }))
    expect(useColorMode().isDark.value).toBe(false)
  })

  it('does not duplicate listeners when initialized again', () => {
    initColorMode()
    dispose = initColorMode()
    expect(listeners.size).toBe(1)
    dispose()
    expect(listeners.size).toBe(0)
  })

  it('keeps a usable session mode when browser storage is blocked', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('blocked') })
    matches = true
    dispose = initColorMode()
    expect(useColorMode().isDark.value).toBe(true)
    useColorMode().setMode('light')
    expect(useColorMode().isDark.value).toBe(false)
  })
})
