import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import MotionPreferenceToggle from '../MotionPreferenceToggle.vue'
import { initMotionPreferences } from '@/composables/useMotionPreferences'
import { mockMotionMedia, motionQueries } from '@/composables/__tests__/motionTestUtils'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('zh-CN') }) }))
let cleanup: () => void
let media: ReturnType<typeof mockMotionMedia>
let wrapper: VueWrapper | undefined
beforeEach(() => {
  localStorage.clear()
  media = mockMotionMedia()
  cleanup = initMotionPreferences()
})
afterEach(() => { wrapper?.unmount(); cleanup(); vi.restoreAllMocks() })

describe('MotionPreferenceToggle', () => {
  it('exposes and persists an explicit reduction while keeping auto as the default', async () => {
    wrapper = mount(MotionPreferenceToggle, { props: { showLabel: true } })
    expect(wrapper.attributes('aria-pressed')).toBe('false')
    expect(wrapper.text()).toContain('动效跟随系统')
    await wrapper.trigger('click')
    expect(wrapper.attributes('aria-pressed')).toBe('true')
    expect(wrapper.attributes('aria-label')).toContain('切换为跟随系统')
    expect(localStorage.getItem('motion-preference')).toBe('reduced')
    expect(document.documentElement.dataset.motion).toBe('reduced')
    await wrapper.trigger('click')
    expect(wrapper.attributes('aria-pressed')).toBe('false')
    expect(localStorage.getItem('motion-preference')).toBe('auto')
  })

  it('announces effective system reduction without claiming an explicit preference', async () => {
    wrapper = mount(MotionPreferenceToggle)
    media.set(motionQueries.reduced, true)
    await nextTick()
    expect(wrapper.attributes('aria-pressed')).toBe('false')
    expect(wrapper.attributes('aria-label')).toContain('已跟随系统减少动效')
    expect(document.documentElement.dataset.motion).toBe('reduced')
  })
})
