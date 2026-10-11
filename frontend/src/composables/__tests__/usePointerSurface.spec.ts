import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, type EffectScope } from 'vue'
import { usePointerSurface } from '../usePointerSurface'
import { initMotionPreferences, useMotionPreferences } from '../useMotionPreferences'
import { dispatchPointer, mockAnimationFrames, mockMotionMedia, motionQueries } from './motionTestUtils'

const scopes: EffectScope[] = []
let cleanup: () => void
let media: ReturnType<typeof mockMotionMedia>
let frames: ReturnType<typeof mockAnimationFrames>
function surface(tag = 'button') {
  const element = document.createElement(tag)
  element.setAttribute('data-ink-surface', '')
  document.body.append(element)
  const bounds = vi.spyOn(element, 'getBoundingClientRect').mockReturnValue(new DOMRect(10, 20, 100, 50))
  const scope = effectScope()
  scopes.push(scope)
  scope.run(() => usePointerSurface(element, { magnetic: true, lift: tag !== 'button' }))
  return { element, bounds, scope }
}
beforeEach(() => {
  localStorage.clear()
  media = mockMotionMedia()
  cleanup = initMotionPreferences()
  frames = mockAnimationFrames()
})
afterEach(() => {
  scopes.splice(0).forEach(scope => scope.stop())
  cleanup()
  document.body.innerHTML = ''
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('pointer surfaces', () => {
  it('coalesces moves into one frame, caches bounds, and leaves the button hit box untouched', () => {
    const { element, bounds } = surface()
    element.style.transform = 'scale(1)'
    dispatchPointer(element, 'pointerenter', 20, 25)
    dispatchPointer(element, 'pointermove', 130, 100)
    expect(frames.frames.size).toBe(1)
    frames.flush()
    expect(bounds).toHaveBeenCalledTimes(1)
    expect(element.style.getPropertyValue('--ink-pointer-x')).toBe('100.0px')
    expect(element.style.getPropertyValue('--ink-magnet-x')).toBe('3.00px')
    expect(element.style.getPropertyValue('--ink-magnet-y')).toBe('3.00px')
    expect(element.style.getPropertyValue('--ink-lift')).toBe('0px')
    expect(element.style.transform).toBe('scale(1)')
    dispatchPointer(element, 'pointermove', 60, 30)
    frames.flush()
    expect(bounds).toHaveBeenCalledTimes(1)
    expect(element.style.getPropertyValue('--ink-magnet-x')).toBe('0.00px')
  })

  it('limits card lift, invalidates cached layout on scroll, and cancels on leave and disposal', () => {
    const { element, bounds, scope } = surface('article')
    dispatchPointer(element)
    frames.flush()
    expect(element.style.getPropertyValue('--ink-lift')).toBe('-2px')
    window.dispatchEvent(new Event('scroll'))
    expect(element.dataset.pointerActive).toBe('false')
    dispatchPointer(element)
    frames.flush()
    expect(bounds).toHaveBeenCalledTimes(2)
    dispatchPointer(element)
    dispatchPointer(element, 'pointerleave')
    expect(frames.frames.size).toBe(0)
    expect(element.style.getPropertyValue('--ink-pointer-x')).toBe('')
    dispatchPointer(element)
    scope.stop()
    expect(frames.frames.size).toBe(0)
    dispatchPointer(element)
    expect(frames.frames.size).toBe(0)
  })

  it('stops pending effects immediately after reduced, hidden, narrow or coarse settings are applied', async () => {
    const { element } = surface()
    for (const disable of [
      () => useMotionPreferences().setPreference('reduced'),
      () => media.setVisible(false),
      () => media.set(motionQueries.pointer, false),
      () => media.set(motionQueries.desktop, false),
    ]) {
      dispatchPointer(element)
      expect(frames.frames.size).toBe(1)
      disable()
      await nextTick()
      expect(frames.frames.size).toBe(0)
      expect(element.dataset.pointerActive).toBe('false')
      useMotionPreferences().setPreference('auto')
      media.setVisible(true)
      media.set(motionQueries.pointer, true)
      media.set(motionQueries.desktop, true)
      await nextTick()
    }
  })

  it('ignores touch, disabled and inert controls and lets only the innermost surface respond', () => {
    const outer = surface('article').element
    const inner = surface().element
    outer.append(inner)
    dispatchPointer(inner, 'pointermove', 30, 40, 'touch')
    expect(frames.frames.size).toBe(0)
    inner.setAttribute('disabled', '')
    dispatchPointer(inner)
    expect(frames.frames.size).toBe(0)
    inner.removeAttribute('disabled')
    outer.setAttribute('inert', '')
    dispatchPointer(inner)
    expect(frames.frames.size).toBe(0)
    outer.removeAttribute('inert')
    dispatchPointer(inner)
    expect(frames.frames.size).toBe(1)
    frames.flush()
    expect(inner.dataset.pointerActive).toBe('true')
    expect(outer.dataset.pointerActive).toBe('false')
  })

  it('shares window invalidation listeners across mounted surfaces', () => {
    const add = vi.spyOn(window, 'addEventListener')
    const remove = vi.spyOn(window, 'removeEventListener')
    const first = surface()
    const second = surface()
    expect(add.mock.calls.filter(([event]) => event === 'scroll')).toHaveLength(1)
    first.scope.stop()
    expect(remove.mock.calls.filter(([event]) => event === 'scroll')).toHaveLength(0)
    second.scope.stop()
    expect(remove.mock.calls.filter(([event]) => event === 'scroll')).toHaveLength(1)
  })
})
