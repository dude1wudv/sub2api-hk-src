import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { initMotionPreferences, useMotionPreferences } from '../useMotionPreferences'
import { useReducedMotion } from '../useReducedMotion'
import { mockMotionMedia, motionQueries } from './motionTestUtils'

let cleanup: () => void
let media: ReturnType<typeof mockMotionMedia>
beforeEach(() => {
  localStorage.clear()
  media = mockMotionMedia()
  cleanup = initMotionPreferences()
})
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  localStorage.clear()
  delete document.documentElement.dataset.motion
})

describe('shared motion preferences', () => {
  it('shares one media subscription and preserves the legacy readonly ref', () => {
    const first = useMotionPreferences()
    const second = useMotionPreferences()
    expect(first.preference.value).toBe('auto')
    expect(first.reducedMotion).toBe(second.reducedMotion)
    expect(useReducedMotion()).toBe(first.reducedMotion)
    expect(media.matchMedia).toHaveBeenCalledTimes(3)
    expect([...media.media.values()].every(item => item.listeners.size === 1)).toBe(true)
    expect(document.documentElement.dataset.motion).toBe('full')
    cleanup()
    expect([...media.media.values()].every(item => item.listeners.size === 0)).toBe(true)
  })

  it('persists the explicit reduction and never overrides the system reduction', () => {
    const state = useMotionPreferences()
    state.setPreference('reduced')
    expect(state.reducedMotion.value).toBe(true)
    expect(localStorage.getItem('motion-preference')).toBe('reduced')
    expect(document.documentElement.dataset.motion).toBe('reduced')
    media.set(motionQueries.reduced, true)
    state.setPreference('auto')
    expect(state.reducedMotion.value).toBe(true)
    media.set(motionQueries.reduced, false)
    expect(state.reducedMotion.value).toBe(false)
    expect(document.documentElement.dataset.motion).toBe('full')
  })

  it('restores saved choices, discards invalid values, and handles cross-tab clearing', () => {
    localStorage.setItem('motion-preference', 'reduced')
    cleanup = initMotionPreferences()
    const state = useMotionPreferences()
    expect(state.preference.value).toBe('reduced')
    window.dispatchEvent(new StorageEvent('storage', { key: 'motion-preference', newValue: 'auto' }))
    expect(state.preference.value).toBe('auto')
    window.dispatchEvent(new StorageEvent('storage', { key: 'unrelated', newValue: 'reduced' }))
    expect(state.preference.value).toBe('auto')
    state.setPreference('reduced')
    window.dispatchEvent(new StorageEvent('storage', { key: null }))
    expect(state.preference.value).toBe('auto')
    localStorage.setItem('motion-preference', 'full')
    cleanup = initMotionPreferences()
    expect(state.preference.value).toBe('auto')
    expect(localStorage.getItem('motion-preference')).toBeNull()
  })

  it('keeps a session choice when storage is unavailable', () => {
    vi.spyOn(localStorage, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => { throw new Error('blocked') })
    cleanup = initMotionPreferences()
    expect(() => useMotionPreferences().setPreference('reduced')).not.toThrow()
    expect(useMotionPreferences().reducedMotion.value).toBe(true)
  })

  it('allows pointer effects only on a visible, wide, fine-pointer surface', () => {
    const state = useMotionPreferences()
    expect(state.pointerEffectsEnabled.value).toBe(true)
    for (const query of [motionQueries.pointer, motionQueries.desktop]) {
      media.set(query, false)
      expect(state.pointerEffectsEnabled.value).toBe(false)
      media.set(query, true)
      expect(state.pointerEffectsEnabled.value).toBe(true)
    }
    media.setVisible(false)
    expect(state.pointerEffectsEnabled.value).toBe(false)
    media.setVisible(true)
    expect(state.pointerEffectsEnabled.value).toBe(true)
    state.setPreference('reduced')
    expect(state.pointerEffectsEnabled.value).toBe(false)
  })
})
