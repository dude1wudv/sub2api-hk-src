import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import TablePageLayout from '../TablePageLayout.vue'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../TablePageLayout.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('TablePageLayout responsive table scrolling', () => {
  it('keeps the tablet table layout at 768px and switches below it without losing slots', async () => {
    const width = window.innerWidth
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 768 })
    const wrapper = mount(TablePageLayout, { slots: { filters: '<button>Filter</button>', table: '<table><tbody><tr><td>Resource</td></tr></tbody></table>' } })
    await nextTick()
    expect(wrapper.classes()).not.toContain('mobile-mode')
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 767 })
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(wrapper.classes()).toContain('mobile-mode')
    expect(wrapper.get('button').text()).toBe('Filter')
    expect(wrapper.get('td').text()).toBe('Resource')
    wrapper.unmount()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
  })
  it('does not disable the table horizontal scroll container in mobile mode', () => {
    const tableWrapperBlocks = Array.from(
      componentSource.matchAll(/([^{}]*:deep\(\.table-wrapper\)[^{}]*)\{([^{}]*)\}/g)
    )

    expect(tableWrapperBlocks.length).toBeGreaterThan(0)

    const baseBlock = tableWrapperBlocks.find(([selector]) => !selector.includes('.mobile-mode'))
    const mobileBlocks = tableWrapperBlocks.filter(([selector]) => selector.includes('.mobile-mode'))

    expect(baseBlock?.[2]).toContain('overflow-x-auto')
    expect(mobileBlocks.every(([, , declarations]) => !declarations.includes('overflow-visible'))).toBe(
      true
    )
  })
})
