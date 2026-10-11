import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import ActiveIndicator from '../ActiveIndicator.vue'
import { initMotionPreferences, useMotionPreferences } from '@/composables/useMotionPreferences'
import { mockAnimationFrames, mockMotionMedia } from '@/composables/__tests__/motionTestUtils'

let cleanup: () => void
let frames: ReturnType<typeof mockAnimationFrames>
let wrapper: VueWrapper | undefined
let container: HTMLElement
let first: HTMLButtonElement
let second: HTMLButtonElement
const resizeObservers: Array<{ callback: () => void; disconnect: ReturnType<typeof vi.fn> }> = []
const mutationObservers: Array<{ callback: () => void; disconnect: ReturnType<typeof vi.fn> }> = []

async function paint() { frames.flush(); await nextTick() }
beforeEach(() => {
  localStorage.clear()
  mockMotionMedia()
  cleanup = initMotionPreferences()
  frames = mockAnimationFrames()
  vi.stubGlobal('ResizeObserver', class {
    disconnect = vi.fn()
    observe = vi.fn()
    unobserve = vi.fn()
    constructor(public callback: () => void) { resizeObservers.push(this) }
  })
  vi.stubGlobal('MutationObserver', class {
    disconnect = vi.fn()
    observe = vi.fn()
    constructor(public callback: () => void) { mutationObservers.push(this) }
  })
  container = document.createElement('nav')
  container.innerHTML = '<button data-active-indicator-item="first" aria-current="page">First</button><button data-active-indicator-item="second">Second</button>'
  document.body.append(container)
  ;[first, second] = [...container.querySelectorAll('button')]
  vi.spyOn(container, 'getBoundingClientRect').mockReturnValue(new DOMRect(100, 40, 300, 44))
  vi.spyOn(first, 'getBoundingClientRect').mockReturnValue(new DOMRect(130, 48, 80, 32))
  vi.spyOn(second, 'getBoundingClientRect').mockReturnValue(new DOMRect(230, 48, 110, 32))
  Object.defineProperty(container, 'clientLeft', { value: 2 })
  Object.defineProperty(container, 'clientTop', { value: 2 })
  container.scrollLeft = 50
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  cleanup()
  document.body.innerHTML = ''
  resizeObservers.length = 0
  mutationObservers.length = 0
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('ActiveIndicator', () => {
  it('positions one decorative layer inside a scrolled container without moving or focusing its controls', async () => {
    first.focus()
    wrapper = mount(ActiveIndicator, { props: { container, activeKey: 'first' } })
    await paint()
    const indicator = container.querySelector<HTMLElement>('[data-active-indicator]')!
    expect(indicator.getAttribute('aria-hidden')).toBe('true')
    expect(indicator.style.transform).toBe('translate3d(78px, 6px, 0)')
    expect(indicator.style.width).toBe('80px')
    expect(indicator.style.height).toBe('32px')
    expect(indicator.classList.contains('active-indicator--settled')).toBe(false)
    expect(first.style.transform).toBe('')
    expect(document.activeElement).toBe(first)

    await wrapper.setProps({ activeKey: 'second' })
    await paint()
    expect(indicator.style.transform).toBe('translate3d(178px, 6px, 0)')
    expect(indicator.classList.contains('active-indicator--settled')).toBe(true)
    expect(indicator.querySelector('.active-indicator__ink--arriving')).not.toBeNull()
    expect(document.activeElement).toBe(first)
  })

  it('remeasures changed labels without replaying feedback for the same active item', async () => {
    wrapper = mount(ActiveIndicator, { props: { container, activeKey: 'first' } })
    await paint()
    await wrapper.setProps({ activeKey: 'second' })
    await paint()
    const ink = container.querySelector('.active-indicator__ink')
    vi.mocked(second.getBoundingClientRect).mockReturnValue(new DOMRect(230, 48, 130, 32))
    mutationObservers[0].callback()
    resizeObservers[0].callback()
    expect(frames.frames.size).toBe(1)
    await paint()
    expect(container.querySelector<HTMLElement>('[data-active-indicator]')?.style.width).toBe('130px')
    expect(container.querySelector('.active-indicator__ink')).toBe(ink)
  })

  it('falls back to selected ARIA state and updates immediately when motion is reduced', async () => {
    useMotionPreferences().setPreference('reduced')
    wrapper = mount(ActiveIndicator, { props: { container, activeKey: null } })
    await paint()
    first.removeAttribute('aria-current')
    second.setAttribute('aria-selected', 'true')
    mutationObservers[0].callback()
    await paint()
    const indicator = container.querySelector<HTMLElement>('[data-active-indicator]')!
    expect(indicator.style.width).toBe('110px')
    expect(indicator.classList.contains('active-indicator--settled')).toBe(false)
    expect(indicator.querySelector('.active-indicator__ink--arriving')).toBeNull()
    second.hidden = true
    mutationObservers[0].callback()
    await paint()
    expect(indicator.style.display).toBe('none')
  })

  it('cancels pending measurements and disconnects observers when unmounted', () => {
    wrapper = mount(ActiveIndicator, { props: { container, activeKey: 'first' } })
    expect(frames.frames.size).toBe(1)
    wrapper.unmount()
    wrapper = undefined
    expect(frames.frames.size).toBe(0)
    expect(resizeObservers[0].disconnect).toHaveBeenCalledOnce()
    expect(mutationObservers[0].disconnect).toHaveBeenCalledOnce()
    expect(container.querySelector('[data-active-indicator]')).toBeNull()
  })
})
