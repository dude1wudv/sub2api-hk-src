import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAppearance } from '../useAppearance'
import { useGlacierTableSurface } from '../useGlacierTableSurface'

let callbacks: Map<number, FrameRequestCallback>
let nextFrame: number
const position = { left: 100, top: 200 }
const Host = defineComponent({
  props: { label: { type: String, default: 'Name' } },
  setup(props) {
    const wrapper = ref<HTMLElement | null>(null)
    useGlacierTableSurface(wrapper)
    return () => h('div', { ref: wrapper }, h('table', [
      h('thead', h('tr', h('th', { class: 'sticky-header-cell' }, props.label))),
      h('tbody', h('tr', h('td', { class: 'sticky-col' }, 'Value')))
    ]))
  }
})

function paint() {
  const pending = [...callbacks.values()]
  callbacks.clear()
  pending.forEach(callback => callback(0))
}

beforeEach(() => {
  callbacks = new Map()
  nextFrame = 0
  position.left = 100
  position.top = 200
  vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
    callbacks.set(++nextFrame, callback)
    return nextFrame
  }))
  vi.stubGlobal('cancelAnimationFrame', vi.fn((id: number) => callbacks.delete(id)))
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => ({
    ...position, x: position.left, y: position.top, width: 80, height: 40,
    bottom: position.top + 40, right: position.left + 80, toJSON() {}
  }))
  useAppearance().setStyle('glacier')
})
afterEach(() => {
  useAppearance().setStyle('aurora')
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('useGlacierTableSurface', () => {
  it('aligns header and pinned cells after scroll and rendered content changes, batching events', async () => {
    const wrapper = mount(Host)
    await nextTick()
    paint()
    const cell = wrapper.get('td').element
    expect(cell.style.getPropertyValue('--glacier-cell-top')).toBe('200px')
    expect(wrapper.get('th').element.style.getPropertyValue('--glacier-cell-left')).toBe('100px')
    position.top = 90
    document.dispatchEvent(new Event('scroll'))
    document.dispatchEvent(new Event('scroll'))
    expect(callbacks.size).toBe(1)
    paint()
    expect(cell.style.getPropertyValue('--glacier-cell-top')).toBe('90px')
    position.left = 60
    await wrapper.setProps({ label: 'New column' })
    paint()
    expect(cell.style.getPropertyValue('--glacier-cell-left')).toBe('60px')
    wrapper.unmount()
  })

  it('keeps scene alignment across themes and cancels pending work on unmount', async () => {
    const wrapper = mount(Host)
    await nextTick()
    paint()
    for (const theme of ['aurora', 'lagoon', 'graphite', 'glacier'] as const) {
      useAppearance().setStyle(theme)
      await nextTick()
      position.left += 10
      document.dispatchEvent(new Event('scroll'))
      expect(callbacks.size).toBe(1)
      paint()
      expect(wrapper.get('td').element.style.getPropertyValue('--glacier-cell-left')).toBe(`${position.left}px`)
    }
    document.dispatchEvent(new Event('scroll'))
    expect(callbacks.size).toBe(1)
    wrapper.unmount()
    expect(callbacks.size).toBe(0)
    document.dispatchEvent(new Event('scroll'))
    expect(callbacks.size).toBe(0)
  })
})
