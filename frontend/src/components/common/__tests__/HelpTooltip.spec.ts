import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key === 'common.close' ? 'Close' : 'Help' }) }))

function getTooltipElement(): HTMLDivElement {
  const tooltip = document.body.querySelector('[role="tooltip"], [role="dialog"]')
  if (!(tooltip instanceof HTMLDivElement)) {
    throw new Error('tooltip element not found')
  }
  return tooltip
}

describe('HelpTooltip', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('keeps the existing hover interaction by default', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'hover details',
      },
    })

    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('mouseenter')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    await trigger.trigger('mouseleave')
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('keeps a hover tooltip open while the pointer moves between the trigger and the tooltip', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'copyable details',
      },
    })

    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    await trigger.trigger('mouseenter')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    await trigger.trigger('mouseleave', { relatedTarget: tooltip })
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    tooltip.dispatchEvent(new MouseEvent('mouseleave', { relatedTarget: trigger.element }))
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    tooltip.dispatchEvent(new MouseEvent('mouseleave', { relatedTarget: null }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('supports click-to-toggle details and closes on outside click', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'click details',
        trigger: 'click',
      },
    })

    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    expect(tooltip.textContent).toContain('click details')

    const closeButton = tooltip.querySelector('button[aria-label="Close"]')
    if (!(closeButton instanceof HTMLButtonElement)) {
      throw new Error('close button not found')
    }
    closeButton.click()
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('opens hover tooltips on focus, exposes the linked ARIA target, and closes on focusout or Escape', async () => {
    const tooltipId = 'help-tooltip-focus'
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'focus details',
        id: tooltipId,
      },
      slots: {
        trigger: '<button type="button" aria-describedby="help-tooltip-focus">Focus help</button>',
      },
    })

    const trigger = wrapper.get('.group')
    const button = wrapper.get('button')
    const tooltip = getTooltipElement()
    expect(button.attributes('aria-describedby')).toBe(tooltipId)
    expect(tooltip.id).toBe(tooltipId)
    expect(tooltip.getAttribute('role')).toBe('tooltip')
    expect(document.getElementById(button.attributes('aria-describedby')!)).toBe(tooltip)
    expect(tooltip.style.display).toBe('none')

    await button.trigger('focusin')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    expect(tooltip.textContent).toContain('focus details')

    await button.trigger('focusout', { relatedTarget: document.body })
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    await trigger.trigger('focusin')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('closes click-triggered tooltips on Escape', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'escape details',
        trigger: 'click',
      },
    })
    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()

    await trigger.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    wrapper.unmount()
  })

  it('pins a both-mode tooltip on click after hover or focus and closes through click, outside click, or Escape', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: { content: 'hybrid details', trigger: 'both', label: 'Hybrid help' },
    })
    const root = wrapper.get('.group')
    const button = wrapper.get('button')
    const tooltip = getTooltipElement()

    expect(button.attributes('aria-haspopup')).toBe('dialog')
    expect(button.attributes('aria-expanded')).toBe('false')

    // Touch-oriented click opens and pins; a later touch/click closes it.
    await button.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    expect(button.attributes('aria-expanded')).toBe('true')
    await root.trigger('mouseleave')
    expect(tooltip.style.display).not.toBe('none')
    await button.trigger('click')
    await nextTick()
    expect(tooltip.style.display).toBe('none')
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur()

    // Hover and focus open it; the first click pins the focused popover.
    await root.trigger('mouseenter')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    await root.trigger('mouseleave')
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    await button.trigger('focusin')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    await button.trigger('click')
    await root.trigger('mouseleave')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    expect(button.attributes('aria-expanded')).toBe('true')

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')

    await button.trigger('click')
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')
    expect(button.attributes('aria-expanded')).toBe('false')

    wrapper.unmount()
  })

  it('clamps fixed coordinates to the viewport and flips below a trigger near its top edge', async () => {
    vi.stubGlobal('innerWidth', 390)
    vi.stubGlobal('innerHeight', 844)
    vi.stubGlobal('scrollX', 150)
    vi.stubGlobal('scrollY', 800)
    const wrapper = mount(HelpTooltip, { attachTo: document.body, props: { content: 'Details', widthClass: 'w-80' } })
    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()
    const bounds = vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue(rect(355, 300, 24, 24))
    vi.spyOn(tooltip, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 320, 180))

    await trigger.trigger('mouseenter')
    await flushPromises()
    expect(tooltip.style.left).toBe('58px')
    expect(tooltip.style.top).toBe('112px')
    expect(tooltip.style.maxWidth).toBe('366px')
    expect(tooltip.dataset.placement).toBe('top')

    bounds.mockReturnValue(rect(0, 15, 24, 24))
    window.dispatchEvent(new Event('scroll'))
    await flushPromises()
    expect(tooltip.style.left).toBe('12px')
    expect(tooltip.style.top).toBe('47px')
    expect(tooltip.dataset.placement).toBe('bottom')
    wrapper.unmount()
  })

  it('positions against visualViewport offsets on a short screen and repositions after viewport changes', async () => {
    const viewport = new EventTarget() as VisualViewport
    Object.defineProperties(viewport, {
      width: { configurable: true, value: 320, writable: true },
      height: { configurable: true, value: 240, writable: true },
      offsetLeft: { configurable: true, value: 30, writable: true },
      offsetTop: { configurable: true, value: 100, writable: true },
    })
    vi.stubGlobal('visualViewport', viewport)

    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: { content: 'short viewport details', trigger: 'click', widthClass: 'w-96' },
    })
    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()
    const triggerBounds = vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue(rect(300, 130, 24, 24))
    vi.spyOn(tooltip, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 384, 500))

    await trigger.trigger('click')
    await flushPromises()
    expect(tooltip.style.maxWidth).toBe('296px')
    expect(tooltip.style.maxHeight).toBe('166px')
    expect(tooltip.style.left).toBe('42px')
    expect(tooltip.style.top).toBe('162px')
    expect(tooltip.dataset.placement).toBe('bottom')

    Object.defineProperties(viewport, {
      offsetLeft: { configurable: true, value: 50, writable: true },
      offsetTop: { configurable: true, value: 180, writable: true },
    })
    triggerBounds.mockReturnValue(rect(280, 230, 24, 24))
    viewport.dispatchEvent(new Event('scroll'))
    await flushPromises()
    expect(tooltip.style.left).toBe('62px')
    expect(tooltip.style.top).toBe('262px')
    expect(tooltip.dataset.placement).toBe('bottom')

    wrapper.unmount()
  })

  it('observes trigger and tooltip size changes while open and disconnects on unmount', async () => {
    let onResize: ResizeObserverCallback | undefined
    const observe = vi.fn()
    const disconnect = vi.fn()
    class TestResizeObserver {
      constructor(callback: ResizeObserverCallback) {
        onResize = callback
      }
      observe = observe
      disconnect = disconnect
      unobserve = vi.fn()
    }
    vi.stubGlobal('ResizeObserver', TestResizeObserver)

    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: { content: 'resized details' },
    })
    const trigger = wrapper.get('.group')
    const tooltip = getTooltipElement()
    const bounds = vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue(rect(60, 160, 24, 24))
    vi.spyOn(tooltip, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 256, 100))

    expect(observe).toHaveBeenCalledTimes(2)
    await trigger.trigger('mouseenter')
    await flushPromises()
    expect(tooltip.style.left).toBe('12px')

    bounds.mockReturnValue(rect(250, 160, 24, 24))
    onResize?.([], {} as ResizeObserver)
    await flushPromises()
    expect(tooltip.style.left).toBe('134px')

    wrapper.unmount()
    expect(disconnect).toHaveBeenCalledOnce()
  })

  it('keeps long interactive details scrollable and returns keyboard focus on Escape', async () => {
    vi.stubGlobal('innerWidth', 390)
    vi.stubGlobal('innerHeight', 390)
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: { trigger: 'click', label: 'Quota policy', widthClass: 'w-80' },
      slots: { default: '<p>Long quota explanation</p><a href="https://docs.example.test/quotas">Read full policy</a>' },
    })
    const button = wrapper.get('button')
    const tooltip = getTooltipElement()
    vi.spyOn(wrapper.element, 'getBoundingClientRect').mockReturnValue(rect(180, 180, 24, 24))
    vi.spyOn(tooltip, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 320, 700))

    expect(button.attributes('type')).toBe('button')
    expect(button.attributes('aria-controls')).toBe(tooltip.id)
    expect(button.attributes('aria-haspopup')).toBe('dialog')
    await button.trigger('keydown', { key: 'ArrowDown' })
    await flushPromises()
    expect(tooltip.getAttribute('role')).toBe('dialog')
    expect(button.attributes('aria-expanded')).toBe('true')
    expect(document.activeElement).toBe(tooltip)
    expect(tooltip.style.maxHeight).toBe('166px')
    const scrollable = tooltip.querySelector<HTMLElement>('.overflow-y-auto')!
    expect(scrollable.style.maxHeight).toBe('166px')
    const measurements = vi.mocked(tooltip.getBoundingClientRect).mock.calls.length
    scrollable.scrollTop = 100
    scrollable.dispatchEvent(new Event('scroll', { bubbles: true }))
    await flushPromises()
    expect(scrollable.scrollTop).toBe(100)
    expect(tooltip.getBoundingClientRect).toHaveBeenCalledTimes(measurements)
    const link = tooltip.querySelector<HTMLAnchorElement>('a')!
    expect(link.href).toBe('https://docs.example.test/quotas')
    link.focus()
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(tooltip.style.display).toBe('none')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(button.element)
    wrapper.unmount()
  })
})

function rect(left: number, top: number, width: number, height: number): DOMRect {
  return { left, top, width, height, right: left + width, bottom: top + height, x: left, y: top, toJSON: () => ({}) }
}
