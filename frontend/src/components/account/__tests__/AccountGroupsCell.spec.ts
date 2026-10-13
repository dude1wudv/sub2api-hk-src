import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import AccountGroupsCell from '../AccountGroupsCell.vue'
import type { Group } from '@/types'

vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })

describe('account groups details', () => {
  it.each([320, 360, 390, 1280])('fits long groups at both edges of a %spx viewport', async (width) => {
    vi.stubGlobal('innerWidth', width)
    vi.stubGlobal('innerHeight', 400)
    let triggerLeft = 0
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      return (this.getAttribute('role') === 'dialog'
        ? { left: 0, right: 384, top: 0, bottom: 900, width: 384, height: 900 }
        : { left: triggerLeft, right: triggerLeft + 20, top: 180, bottom: 200, width: 20, height: 20 }) as DOMRect
    })
    const groups = Array.from({ length: 30 }, (_, id) => ({ id, name: `LongGroupName${id}` })) as Group[]
    const wrapper = mount(AccountGroupsCell, { props: { groups }, global: { stubs: { GroupBadge: { props: ['name'], template: '<span>{{ name }}</span>' } } } })
    const trigger = wrapper.get('button')
    for (const left of [0, width - 20]) {
      triggerLeft = left
      await trigger.trigger('click'); await flushPromises()
      const panel = document.body.querySelector<HTMLElement>('[role="dialog"]')!
      expect(panel.textContent).toContain('LongGroupName29')
      expect(Number.parseFloat(panel.style.left)).toBeGreaterThanOrEqual(12)
      expect(Number.parseFloat(panel.style.left) + Math.min(384, Number.parseFloat(panel.style.maxWidth))).toBeLessThanOrEqual(width - 12)
      expect(Number.parseFloat(panel.style.maxHeight)).toBeLessThanOrEqual(180)
      expect(panel.querySelector('.overflow-y-auto')).not.toBeNull()
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' })); await flushPromises()
      expect(panel.style.display).toBe('none')
    }
  })
})
