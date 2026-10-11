import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { ref } from 'vue'
import AdminFormTabs from '../AdminFormTabs.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('zh-CN') }) }))
const tabs = [
  { id: 'basic', zh: '基础信息', en: 'General' },
  { id: 'scheduling', zh: '调度与分组', en: 'Scheduling' },
  { id: 'advanced', zh: '高级设置', en: 'Advanced' },
]
let wrapper: VueWrapper | undefined
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

describe('AdminFormTabs', () => {
  it('keeps roving focus, panel associations and arrow-key wrapping with the decorative indicator', async () => {
    wrapper = mount({
      components: { AdminFormTabs },
      setup: () => ({ selected: ref('basic'), tabs }),
      template: '<AdminFormTabs id="editor" v-model="selected" :tabs="tabs" />',
    }, { attachTo: document.body, global: { stubs: { ActiveIndicator: true } } })
    const buttons = wrapper.findAll('button')
    expect(wrapper.find('[role="tablist"]').attributes('aria-label')).toBe('编辑分类')
    expect(buttons[0].attributes('aria-controls')).toBe('editor-basic-panel')
    expect(buttons[0].attributes('tabindex')).toBe('0')
    expect(buttons[1].attributes('data-active-indicator-item')).toBe('scheduling')
    await buttons[0].trigger('keydown', { key: 'ArrowRight' })
    expect(document.activeElement).toBe(buttons[1].element)
    expect(buttons[1].attributes('aria-selected')).toBe('true')
    expect(buttons[0].attributes('tabindex')).toBe('-1')
    await buttons[1].trigger('keydown', { key: 'End' })
    expect(document.activeElement).toBe(buttons[2].element)
    await buttons[2].trigger('keydown', { key: 'ArrowRight' })
    expect(document.activeElement).toBe(buttons[0].element)
    await buttons[0].trigger('keydown', { key: 'ArrowLeft' })
    expect(document.activeElement).toBe(buttons[2].element)
    await buttons[2].trigger('keydown', { key: 'Home' })
    expect(document.activeElement).toBe(buttons[0].element)
  })

  it('preserves pointer selection and renders an empty tab set safely', async () => {
    wrapper = mount(AdminFormTabs, { props: { id: 'editor', modelValue: 'basic', tabs }, global: { stubs: { ActiveIndicator: true } } })
    await wrapper.findAll('button')[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['scheduling']])
    await wrapper.setProps({ tabs: [] })
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(0)
  })
})
