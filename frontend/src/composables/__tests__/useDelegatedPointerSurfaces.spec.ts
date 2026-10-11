import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref, type EffectScope } from 'vue'
import { useDelegatedPointerSurfaces } from '../useDelegatedPointerSurfaces'
import { usePointerSurface } from '../usePointerSurface'
import { initMotionPreferences, useMotionPreferences } from '../useMotionPreferences'
import { dispatchPointer, mockAnimationFrames, mockMotionMedia, motionQueries } from './motionTestUtils'

const scopes: EffectScope[] = []
let cleanup: () => void
let media: ReturnType<typeof mockMotionMedia>
let frames: ReturnType<typeof mockAnimationFrames>

function card(className = 'card', tag = 'section') {
  const element = document.createElement(tag)
  element.className = className
  element.innerHTML = '<header>Heading</header><button type="button">Action</button>'
  document.body.append(element)
  const bounds = vi.spyOn(element, 'getBoundingClientRect').mockReturnValue(new DOMRect(10, 20, 200, 120))
  return { element, bounds }
}
function install() {
  const route = ref('/dashboard')
  const scope = effectScope()
  scopes.push(scope)
  scope.run(() => useDelegatedPointerSurfaces(route))
  return { scope, route }
}
const selected = () => document.querySelector<HTMLElement>('[data-ink-hover-frame]')

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

describe('delegated page surfaces', () => {
  it('coalesces pointer writes, caches geometry and leaves card children, cursor and hit area unchanged', () => {
    install()
    const { element, bounds } = card('portal-api-access-strip')
    const children = [...element.children]
    element.style.transform = 'scale(1)'
    const clicked = vi.fn()
    const button = element.querySelector('button')!
    button.addEventListener('click', clicked)
    dispatchPointer(element, 'pointerover', 20, 30)
    dispatchPointer(button, 'pointermove', 160, 110)
    expect(frames.frames.size).toBe(1)
    expect(selected()?.dataset.inkHoverFrame).toBe('pending')
    frames.flush()
    expect(bounds).toHaveBeenCalledTimes(1)
    expect(selected()?.style.getPropertyValue('--ink-hover-x')).toBe('150.0px')
    expect(selected()?.style.getPropertyValue('--ink-hover-y')).toBe('90.0px')
    expect(selected()?.dataset.inkHoverFrame).toBe('idle')
    expect(document.querySelector('[data-ink-hover-layer]')).toBeNull()
    expect(element.dataset.inkHoverActive).toBe('true')
    expect([...element.children]).toEqual(children)
    expect(element.firstElementChild).toBe(children[0])
    expect(element.lastElementChild).toBe(button)
    expect(element.style.transform).toBe('scale(1)')
    expect(element.style.cursor).toBe('')
    button.click()
    expect(clicked).toHaveBeenCalledOnce()
    dispatchPointer(element, 'pointermove', 100, 100)
    frames.flush()
    expect(bounds).toHaveBeenCalledTimes(1)
  })

  it('preserves existing background images and restores owned variables on exit', () => {
    install()
    const { element } = card()
    element.style.backgroundImage = 'linear-gradient(red, blue)'
    element.style.setProperty('--ink-hover-x', '12px', 'important')
    const originalStyle = element.getAttribute('style')
    dispatchPointer(element)
    frames.flush()
    expect(element.style.getPropertyValue('--ink-hover-base-image')).toBe('linear-gradient(red, blue)')
    expect(element.style.backgroundImage).toBe('linear-gradient(red, blue)')
    dispatchPointer(element, 'pointerout')
    expect(element.style.getPropertyValue('--ink-hover-base-image')).toBe('')
    expect(element.style.getPropertyValue('--ink-hover-x')).toBe('12px')
    expect(element.getAttribute('style')).toBe(originalStyle)
    expect(element.style.backgroundImage).toBe('linear-gradient(red, blue)')
  })

  it('respects opt-out descendants before selecting an outer card or a nested InkSurface', () => {
    install()
    const outer = card().element
    const optOut = document.createElement('div')
    optOut.dataset.inkHover = 'off'
    const inner = card('portal-panel').element
    outer.append(optOut)
    optOut.append(inner)
    dispatchPointer(outer)
    expect(frames.frames.size).toBe(1)
    dispatchPointer(optOut)
    expect(frames.frames.size).toBe(0)
    expect(selected()).toBeNull()
    dispatchPointer(inner)
    expect(frames.frames.size).toBe(0)
    outer.setAttribute('data-ink-surface', '')
    const scope = effectScope()
    scopes.push(scope)
    scope.run(() => usePointerSurface(outer))
    dispatchPointer(optOut)
    expect(frames.frames.size).toBe(0)
    expect(outer.dataset.pointerActive).toBe('false')
  })

  it('uses the nearest surface and coordinates nested InkSurface without duplicate decoration', () => {
    install()
    const outer = card('workspace-launchpad').element
    const inner = card('workspace-action-card', 'button').element
    outer.append(inner)
    inner.setAttribute('data-ink-surface', '')
    const inkScope = effectScope()
    scopes.push(inkScope)
    inkScope.run(() => usePointerSurface(inner))
    dispatchPointer(outer)
    frames.flush()
    expect(outer.dataset.inkHoverActive).toBe('true')
    dispatchPointer(inner)
    expect(selected()).toBeNull()
    expect(frames.frames.size).toBe(1)
    frames.flush()
    expect(inner.dataset.pointerActive).toBe('true')
    expect(outer.hasAttribute('data-ink-hover-active')).toBe(false)

    const nested = card('portal-panel').element
    inner.append(nested)
    dispatchPointer(nested)
    expect(frames.frames.size).toBe(1)
    frames.flush()
    expect(nested.dataset.inkHoverActive).toBe('true')
    expect(inner.dataset.pointerActive).toBe('false')
    expect(document.querySelectorAll('[data-ink-hover-active]')).toHaveLength(1)
  })

  it('cancels on departure, scroll, route change, node removal and owner disposal', async () => {
    const { scope, route } = install()
    const { element, bounds } = card()
    for (const cancel of [
      () => dispatchPointer(element, 'pointerout'),
      () => window.dispatchEvent(new Event('scroll')),
      () => { route.value = '/keys' },
    ]) {
      dispatchPointer(element)
      expect(frames.frames.size).toBe(1)
      cancel()
      expect(frames.frames.size).toBe(0)
      expect(selected()).toBeNull()
    }
    expect(bounds).toHaveBeenCalledTimes(3)
    dispatchPointer(element)
    element.remove()
    await nextTick()
    expect(frames.frames.size).toBe(0)
    expect(selected()).toBeNull()
    document.body.append(element)
    dispatchPointer(element)
    scope.stop()
    expect(frames.frames.size).toBe(0)
    dispatchPointer(element)
    expect(selected()).toBeNull()
  })

  it('disables pending effects for system/user reduced motion, narrow/coarse pointers and hidden pages', async () => {
    install()
    const { element } = card()
    for (const disable of [
      () => useMotionPreferences().setPreference('reduced'),
      () => media.set(motionQueries.reduced, true),
      () => media.set(motionQueries.pointer, false),
      () => media.set(motionQueries.desktop, false),
      () => media.setVisible(false),
    ]) {
      dispatchPointer(element)
      expect(frames.frames.size).toBe(1)
      disable()
      expect(frames.frames.size).toBe(0)
      expect(selected()).toBeNull()
      dispatchPointer(element)
      expect(frames.frames.size).toBe(0)
      useMotionPreferences().setPreference('auto')
      media.set(motionQueries.reduced, false)
      media.set(motionQueries.pointer, true)
      media.set(motionQueries.desktop, true)
      media.setVisible(true)
      await nextTick()
    }
  })

  it.each(['dialog', 'modal', 'menu'])('pauses globally while a %s overlay opens', async kind => {
    install()
    const { element } = card()
    dispatchPointer(element)
    const overlay = document.createElement(kind === 'dialog' ? 'dialog' : 'div')
    if (kind === 'dialog') overlay.setAttribute('open', '')
    else if (kind === 'modal') overlay.setAttribute('aria-modal', 'true')
    else overlay.setAttribute('role', 'menu')
    document.body.append(overlay)
    await nextTick()
    expect(frames.frames.size).toBe(0)
    expect(selected()).toBeNull()
    dispatchPointer(element)
    expect(selected()).toBeNull()
    overlay.remove()
    dispatchPointer(element)
    expect(frames.frames.size).toBe(1)
  })

  it('ignores touch, disabled/inert targets, generic divs and table rows', () => {
    install()
    const { element } = card('workspace-action-card', 'button')
    dispatchPointer(element, 'pointermove', 20, 30, 'touch')
    expect(selected()).toBeNull()
    element.setAttribute('disabled', '')
    dispatchPointer(element)
    expect(selected()).toBeNull()
    element.removeAttribute('disabled')
    element.setAttribute('inert', '')
    dispatchPointer(element)
    expect(selected()).toBeNull()
    for (const tag of ['div', 'tr']) {
      const plain = card('', tag).element
      dispatchPointer(plain)
      expect(selected()).toBeNull()
    }
  })
})
