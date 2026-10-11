import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import InkSurface from '../InkSurface.vue'
import { initMotionPreferences } from '@/composables/useMotionPreferences'
import { dispatchPointer, mockAnimationFrames, mockMotionMedia } from '@/composables/__tests__/motionTestUtils'

let cleanup: () => void
let frames: ReturnType<typeof mockAnimationFrames>
const wrappers: VueWrapper[] = []
beforeEach(() => {
  localStorage.clear()
  mockMotionMedia()
  cleanup = initMotionPreferences()
  frames = mockAnimationFrames()
})
afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  cleanup()
  document.body.innerHTML = ''
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('InkSurface', () => {
  it('keeps native button focus, attributes, events and hit area on the stable outer element', async () => {
    const onClick = vi.fn()
    const wrapper = mount(InkSurface, {
      attachTo: document.body,
      props: { as: 'button', magnetic: true },
      attrs: { onClick, 'aria-label': 'Save account', class: 'save-control' },
      slots: { default: 'Save' },
    })
    wrappers.push(wrapper)
    await nextTick()
    const button = wrapper.element as HTMLButtonElement
    expect(button.tagName).toBe('BUTTON')
    expect(button.type).toBe('button')
    expect(button.getAttribute('aria-label')).toBe('Save account')
    expect(button.classList.contains('save-control')).toBe(true)
    expect(wrapper.find('.ink-surface__wash').attributes('aria-hidden')).toBe('true')
    button.focus()
    expect(document.activeElement).toBe(button)
    vi.spyOn(button, 'getBoundingClientRect').mockReturnValue(new DOMRect(0, 0, 100, 44))
    dispatchPointer(button, 'pointermove', 100, 44)
    frames.flush()
    expect(button.style.transform).toBe('')
    expect(button.style.getPropertyValue('--ink-lift')).toBe('0px')
    await wrapper.trigger('click')
    expect(onClick).toHaveBeenCalledOnce()
    expect(wrapper.find('.ink-surface__magnet').text()).toBe('Save')
  })

  it('passes explicit submit and disabled semantics through to the native control', async () => {
    const onClick = vi.fn()
    const wrapper = mount(InkSurface, { props: { as: 'button', magnetic: true }, attrs: { type: 'submit', disabled: true, onClick } })
    wrappers.push(wrapper)
    expect((wrapper.element as HTMLButtonElement).type).toBe('submit')
    expect((wrapper.element as HTMLButtonElement).disabled).toBe(true)
    await wrapper.trigger('click')
    expect(onClick).not.toHaveBeenCalled()
  })

  it('cancels effects while cached by KeepAlive and resumes on activation', async () => {
    const visible = ref(true)
    const wrapper = mount({
      components: { InkSurface },
      setup: () => ({ visible }),
      template: '<KeepAlive><InkSurface v-if="visible" as="article" interactive>Card</InkSurface></KeepAlive>',
    }, { attachTo: document.body })
    wrappers.push(wrapper)
    await nextTick()
    const element = wrapper.find('article').element as HTMLElement
    dispatchPointer(element)
    expect(frames.frames.size).toBe(1)
    visible.value = false
    await nextTick()
    expect(frames.frames.size).toBe(0)
    dispatchPointer(element)
    expect(frames.frames.size).toBe(0)
    visible.value = true
    await nextTick()
    dispatchPointer(element)
    expect(frames.frames.size).toBe(1)
  })
})
