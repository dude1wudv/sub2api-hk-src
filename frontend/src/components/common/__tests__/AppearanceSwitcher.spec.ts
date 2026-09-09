import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'

import AppearanceSwitcher from '../AppearanceSwitcher.vue'
import { initAppearance } from '@/composables/useAppearance'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'en' },
    t: (key: string) => key
  })
}))

const styleValues = ['aurora', 'lagoon', 'graphite', 'glacier'] as const

function mountSwitcher() {
  return mount(AppearanceSwitcher, {
    global: { stubs: { Icon: true } }
  })
}

function selectedStyle(wrapper: VueWrapper) {
  return (wrapper.get('select').element as HTMLSelectElement).value
}

beforeEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
  localStorage.setItem('theme', 'light')
  initAppearance()
})

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
  document.documentElement.classList.remove('dark')
  document.documentElement.removeAttribute('data-style')
})

describe('AppearanceSwitcher appearance consumer behavior', () => {
  it('switches among all four supported styles and persists each selection', async () => {
    const wrapper = mountSwitcher()
    const select = wrapper.get('select')
    const options = select.findAll('option').map((option) => option.element.value)

    expect(options).toEqual(styleValues)

    for (const value of styleValues) {
      await select.setValue(value)
      expect(selectedStyle(wrapper)).toBe(value)
      expect(document.documentElement.dataset.style).toBe(value)
      expect(localStorage.getItem('appearance-style')).toBe(value)
    }

    wrapper.unmount()
  })

  it('restores glacier from localStorage when appearance initialization runs again', async () => {
    const wrapper = mountSwitcher()
    await wrapper.get('select').setValue('glacier')
    expect(localStorage.getItem('appearance-style')).toBe('glacier')
    wrapper.unmount()

    document.documentElement.dataset.style = 'aurora'
    initAppearance()

    const restoredWrapper = mountSwitcher()
    expect(document.documentElement.dataset.style).toBe('glacier')
    expect(selectedStyle(restoredWrapper)).toBe('glacier')
    restoredWrapper.unmount()
  })

  it.each(['light', 'dark'] as const)('keeps %s mode unchanged while switching styles', async (mode) => {
    localStorage.setItem('theme', mode)
    initAppearance()
    const initialIsDark = document.documentElement.classList.contains('dark')
    const wrapper = mountSwitcher()

    for (const value of styleValues) {
      await wrapper.get('select').setValue(value)
      expect(document.documentElement.classList.contains('dark')).toBe(initialIsDark)
      expect(localStorage.getItem('theme')).toBe(mode)
    }

    const modeButton = wrapper.get('button.appearance-mode')
    await modeButton.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(!initialIsDark)
    expect(document.documentElement.dataset.style).toBe('glacier')
    expect(selectedStyle(wrapper)).toBe('glacier')
    expect(localStorage.getItem('theme')).toBe(initialIsDark ? 'light' : 'dark')

    await modeButton.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(initialIsDark)
    expect(document.documentElement.dataset.style).toBe('glacier')
    expect(selectedStyle(wrapper)).toBe('glacier')
    expect(localStorage.getItem('theme')).toBe(mode)

    wrapper.unmount()
  })

  it('keeps the legacy preferences instead of migrating them', () => {
    for (const savedStyle of ['aurora', 'lagoon', 'graphite'] as const) {
      localStorage.setItem('appearance-style', savedStyle)
      initAppearance()
      const wrapper = mountSwitcher()

      expect(document.documentElement.dataset.style).toBe(savedStyle)
      expect(selectedStyle(wrapper)).toBe(savedStyle)
      expect(localStorage.getItem('appearance-style')).toBe(savedStyle)

      wrapper.unmount()
    }
  })

  it('falls back to aurora for an invalid saved style', () => {
    localStorage.setItem('appearance-style', 'midnight')
    initAppearance()
    const wrapper = mountSwitcher()

    expect(document.documentElement.dataset.style).toBe('aurora')
    expect(selectedStyle(wrapper)).toBe('aurora')

    wrapper.unmount()
  })

  it('still switches styles when persistent storage is unavailable', async () => {
    const storage = window.localStorage
    vi.spyOn(storage, 'getItem').mockImplementation(() => {
      throw new Error('storage unavailable')
    })
    vi.spyOn(storage, 'setItem').mockImplementation(() => {
      throw new Error('storage unavailable')
    })

    initAppearance()
    const wrapper = mountSwitcher()
    await wrapper.get('select').setValue('glacier')

    expect(document.documentElement.dataset.style).toBe('glacier')
    expect(selectedStyle(wrapper)).toBe('glacier')

    wrapper.unmount()
  })
})
