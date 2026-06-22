import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CyberBackground from '../CyberBackground.vue'

describe('CyberBackground', () => {
  it('renders the component', () => {
    const wrapper = mount(CyberBackground)
    expect(wrapper.exists()).toBe(true)
  })

  it('contains the main background container', () => {
    const wrapper = mount(CyberBackground)
    const container = wrapper.find('.cyber-background')
    expect(container.exists()).toBe(true)
  })

  it('contains animated grid canvas', () => {
    const wrapper = mount(CyberBackground)
    const canvas = wrapper.find('.grid-canvas')
    expect(canvas.exists()).toBe(true)
  })

  it('contains SVG map lines layer', () => {
    const wrapper = mount(CyberBackground)
    const svg = wrapper.find('.map-lines')
    expect(svg.exists()).toBe(true)
    expect(svg.element.tagName).toBe('svg')
  })

  it('contains data streams layer', () => {
    const wrapper = mount(CyberBackground)
    const streams = wrapper.find('.data-streams')
    expect(streams.exists()).toBe(true)
  })

  it('contains corner glows', () => {
    const wrapper = mount(CyberBackground)
    const glows = wrapper.findAll('.corner-glow')
    expect(glows.length).toBeGreaterThan(0)
  })

  it('has proper CSS classes for animation', () => {
    const wrapper = mount(CyberBackground)
    const container = wrapper.find('.cyber-background')
    expect(container.classes()).toContain('cyber-background')
  })
})
