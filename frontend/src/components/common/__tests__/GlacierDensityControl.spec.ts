import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GlacierDensityControl from '../GlacierDensityControl.vue'
import { useAppearance } from '@/composables/useAppearance'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' } }) }))

describe('GlacierDensityControl', () => {
  beforeEach(() => {
    localStorage.clear()
    useAppearance().setStyle('aurora')
  })

  it('shares a persisted density choice across all themes', async () => {
    const wrapper = mount(GlacierDensityControl)
    expect(wrapper.find('[role="group"]').exists()).toBe(true)

    useAppearance().setStyle('glacier')
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="group"]').attributes('aria-label')).toBe('表格密度')
    expect(wrapper.get('button[aria-pressed="true"]').text()).toBe('舒适')

    await wrapper.get('button:nth-child(2)').trigger('click')
    expect(localStorage.getItem('glacier-table-density')).toBe('compact')
    expect(wrapper.get('button[aria-pressed="true"]').text()).toBe('紧凑')

    for (const theme of ['aurora', 'lagoon', 'graphite', 'glacier'] as const) {
      useAppearance().setStyle(theme)
      await wrapper.vm.$nextTick()
      expect(wrapper.get('button[aria-pressed="true"]').text()).toBe('紧凑')
    }
    wrapper.unmount()
    useAppearance().setStyle('aurora')
  })
})
