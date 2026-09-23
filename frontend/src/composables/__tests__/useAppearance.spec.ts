import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { initAppearance, useAppearance } from '../useAppearance'

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
  document.documentElement.classList.remove('dark')
  delete document.documentElement.dataset.style
})

describe('appearance defaults and migration', () => {
  it.each([null, 'aurora', 'lagoon', 'graphite', 'glacier', 'invalid'])(
    'forces the previous preference %s to glacier on the first visit', (previous) => {
      if (previous) localStorage.setItem('appearance-style', previous)
      localStorage.setItem('theme', 'dark')

      initAppearance()

      expect(useAppearance().style.value).toBe('glacier')
      expect(document.documentElement.dataset.style).toBe('glacier')
      expect(localStorage.getItem('appearance-style')).toBe('glacier')
      expect(document.documentElement.classList.contains('dark')).toBe(true)
      expect(localStorage.getItem('theme')).toBe('dark')
    }
  )

  it('preserves a later user choice after reinitialization', () => {
    initAppearance()
    useAppearance().setStyle('graphite')

    initAppearance()

    expect(useAppearance().style.value).toBe('graphite')
    expect(document.documentElement.dataset.style).toBe('graphite')
    expect(localStorage.getItem('appearance-style')).toBe('graphite')
  })

  it('falls back to glacier for a missing or invalid preference after migration', () => {
    initAppearance()
    for (const value of [null, 'unknown']) {
      if (value) localStorage.setItem('appearance-style', value)
      else localStorage.removeItem('appearance-style')
      initAppearance()
      expect(useAppearance().style.value).toBe('glacier')
    }
  })

  it('applies glacier when browser storage is unavailable', () => {
    useAppearance().setStyle('aurora')
    vi.stubGlobal('localStorage', {
      getItem: () => { throw new Error('Disabled') },
      setItem: () => { throw new Error('Disabled') }
    })

    expect(() => initAppearance()).not.toThrow()
    expect(document.documentElement.dataset.style).toBe('glacier')
  })

  it('retries migration if saving the forced preference fails', () => {
    localStorage.setItem('appearance-style', 'lagoon')
    vi.stubGlobal('localStorage', {
      getItem: localStorage.getItem.bind(localStorage),
      setItem: () => { throw new Error('Full') }
    })
    initAppearance()
    vi.unstubAllGlobals()

    initAppearance()

    expect(useAppearance().style.value).toBe('glacier')
    expect(localStorage.getItem('appearance-style')).toBe('glacier')
  })
})
