import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope, ref, type EffectScope } from 'vue'
import { useVisualViewport } from '../useVisualViewport'

const scopes: EffectScope[] = []
afterEach(() => {
  scopes.splice(0).forEach(scope => scope.stop())
  vi.restoreAllMocks(); vi.unstubAllGlobals()
  Reflect.deleteProperty(document.documentElement, 'clientWidth')
  Reflect.deleteProperty(document.documentElement, 'clientHeight')
})

describe('useVisualViewport', () => {
  it('coalesces keyboard resize and offsets without replacing unchanged measurements', () => {
    const viewport = Object.assign(new EventTarget(), { width: 390, height: 844, offsetLeft: 0, offsetTop: 0 })
    vi.stubGlobal('visualViewport', viewport)
    let pending: FrameRequestCallback | undefined
    const raf = vi.spyOn(window, 'requestAnimationFrame').mockImplementation(callback => { pending = callback; return 1 })
    const scope = effectScope(); scopes.push(scope)
    const rect = scope.run(() => useVisualViewport(true))!
    expect(rect.value).toEqual({ left: 0, top: 0, width: 390, height: 844, right: 390, bottom: 844 })
    viewport.height = 360; viewport.offsetTop = 40
    viewport.dispatchEvent(new Event('resize')); viewport.dispatchEvent(new Event('scroll'))
    expect(raf).toHaveBeenCalledTimes(1)
    pending!(0)
    expect(rect.value.bottom).toBe(400)
    const previous = rect.value
    const body = document.createElement('div'); document.body.append(body); body.scrollTop = 123
    body.dispatchEvent(new Event('scroll', { bubbles: true })); pending!(0)
    expect(rect.value).toBe(previous)
    expect(body.scrollTop).toBe(123)
    body.remove()
  })

  it('falls back to document dimensions and then window dimensions for invalid sizes', () => {
    vi.stubGlobal('visualViewport', undefined)
    Object.defineProperty(document.documentElement, 'clientWidth', { configurable: true, value: 320 })
    Object.defineProperty(document.documentElement, 'clientHeight', { configurable: true, value: 568 })
    const scope = effectScope(); scopes.push(scope)
    const active = ref(true)
    const rect = scope.run(() => useVisualViewport(active))!
    expect(rect.value.width).toBe(320); expect(rect.value.height).toBe(568)
    active.value = false
    Object.defineProperty(document.documentElement, 'clientWidth', { configurable: true, value: 0 })
    Object.defineProperty(document.documentElement, 'clientHeight', { configurable: true, value: 0 })
    vi.stubGlobal('visualViewport', { width: NaN, height: -1, offsetTop: NaN, offsetLeft: -1, addEventListener: vi.fn(), removeEventListener: vi.fn() })
    active.value = true
    expect(rect.value.width).toBe(window.innerWidth); expect(rect.value.height).toBe(window.innerHeight)
    expect(rect.value.top).toBe(0); expect(rect.value.left).toBe(0)
  })

  it('removes listeners and cancels a queued frame on closing and disposal', () => {
    const viewport = Object.assign(new EventTarget(), { width: 390, height: 844, offsetLeft: 0, offsetTop: 0 })
    vi.stubGlobal('visualViewport', viewport)
    const remove = vi.spyOn(viewport, 'removeEventListener')
    const raf = vi.spyOn(window, 'requestAnimationFrame').mockReturnValue(42)
    const cancel = vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
    const scope = effectScope(); scopes.push(scope)
    const active = ref(false)
    const rect = scope.run(() => useVisualViewport(active))!
    expect(rect.value.height).toBe(0)
    active.value = true; viewport.dispatchEvent(new Event('resize')); active.value = false
    expect(cancel).toHaveBeenCalledWith(42)
    expect(remove).toHaveBeenCalledTimes(2)
    raf.mockClear()
    window.dispatchEvent(new Event('resize')); window.dispatchEvent(new Event('scroll'))
    expect(raf).not.toHaveBeenCalled()
    active.value = true; scope.stop()
    expect(remove).toHaveBeenCalledTimes(4)
  })

  it('returns zeros without reading the DOM outside a browser', () => {
    vi.stubGlobal('window', undefined)
    const scope = effectScope(); scopes.push(scope)
    expect(scope.run(() => useVisualViewport(true))!.value).toEqual({ left: 0, top: 0, width: 0, height: 0, right: 0, bottom: 0 })
  })
})
