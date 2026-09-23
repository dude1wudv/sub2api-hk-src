import { mount } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useGlacierInteraction } from '../useGlacierInteraction'

function makeMediaQuery(initial: boolean) {
  const listeners = new Set<(event: MediaQueryListEvent) => void>()
  const media = {
    matches: initial,
    media: '',
    onchange: null,
    addEventListener: vi.fn((_: string, callback: (event: MediaQueryListEvent) => void) => listeners.add(callback)),
    removeEventListener: vi.fn((_: string, callback: (event: MediaQueryListEvent) => void) => listeners.delete(callback)),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
    setMatches(value: boolean) {
      this.matches = value
      listeners.forEach(callback => callback({ matches: value } as MediaQueryListEvent))
    },
  }
  return media as unknown as MediaQueryList & { setMatches(value: boolean): void }
}

function mountInteraction(enabled = true) {
  const active = ref(enabled)
  const component = defineComponent({
    setup() {
      useGlacierInteraction(active)
      return () => h('span')
    },
  })
  return { wrapper: mount(component), active }
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('useGlacierInteraction', () => {
  it('attaches only for fine pointers without reduced motion and removes hover state when disabled', async () => {
    const fine = makeMediaQuery(true)
    const reduced = makeMediaQuery(false)
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => { callback(0); return 1 })
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
    vi.stubGlobal('matchMedia', vi.fn((query: string) => query.includes('hover') ? fine : reduced))
    const add = vi.spyOn(document, 'addEventListener')
    const remove = vi.spyOn(document, 'removeEventListener')
    const { wrapper, active } = mountInteraction()
    const target = document.createElement('button')
    target.className = 'btn'
    document.body.appendChild(target)

    expect(add).toHaveBeenCalledWith('pointermove', expect.any(Function), { passive: true })
    const move = new MouseEvent('pointermove', { bubbles: true, clientX: 20, clientY: 30 })
    Object.defineProperty(move, 'pointerType', { value: 'mouse' })
    target.dispatchEvent(move)
    expect(target.hasAttribute('data-glass-hover')).toBe(true)
    expect(target.style.getPropertyValue('--glass-x')).toBeTruthy()

    active.value = false
    await wrapper.vm.$nextTick()
    expect(target.hasAttribute('data-glass-hover')).toBe(false)
    expect(remove).toHaveBeenCalledWith('pointermove', expect.any(Function))
    wrapper.unmount()
    target.remove()
  })

  it('respects reduced motion and touch, then cleans up media and global listeners on unmount', async () => {
    const fine = makeMediaQuery(true)
    const reduced = makeMediaQuery(true)
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => { callback(0); return 1 })
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
    vi.stubGlobal('matchMedia', vi.fn((query: string) => query.includes('hover') ? fine : reduced))
    const removeDocument = vi.spyOn(document, 'removeEventListener')
    const removeWindow = vi.spyOn(window, 'removeEventListener')
    const { wrapper } = mountInteraction()
    const target = document.createElement('button')
    target.className = 'btn'
    document.body.appendChild(target)
    const touchMove = new MouseEvent('pointermove', { bubbles: true })
    Object.defineProperty(touchMove, 'pointerType', { value: 'touch' })
    target.dispatchEvent(touchMove)
    expect(target.hasAttribute('data-glass-hover')).toBe(false)

    reduced.setMatches(false)
    const mouseMove = new MouseEvent('pointermove', { bubbles: true, clientX: 2, clientY: 3 })
    Object.defineProperty(mouseMove, 'pointerType', { value: 'mouse' })
    target.dispatchEvent(mouseMove)
    expect(target.hasAttribute('data-glass-hover')).toBe(true)
    wrapper.unmount()

    expect(fine.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function))
    expect(reduced.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function))
    expect(removeDocument).toHaveBeenCalledWith('pointermove', expect.any(Function))
    expect(removeDocument).toHaveBeenCalledWith('pointerleave', expect.any(Function))
    expect(removeWindow).toHaveBeenCalledWith('blur', expect.any(Function))
    expect(target.hasAttribute('data-glass-hover')).toBe(false)
    target.remove()
  })
})
