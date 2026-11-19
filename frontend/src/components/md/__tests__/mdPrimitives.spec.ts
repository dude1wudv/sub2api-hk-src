import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import {
  MdButton,
  MdCard,
  MdCheckbox,
  MdFilterChip,
  MdPageHeader,
  MdSelect,
  MdSlider,
  MdSwitch,
  MdTabs,
  MdTextField
} from '@/components/md'

describe('Material 3 Vue primitive wrappers', () => {
  it('renders MdButton variants and blocks clicks while loading or disabled', async () => {
    const wrapper = mount(MdButton, {
      props: { variant: 'tonal', loading: false },
      slots: { default: 'Save' }
    })
    expect(wrapper.element.tagName.toLowerCase()).toBe('md-filled-tonal-button')
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)

    await wrapper.setProps({ loading: true })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
  })

  it('adapts MdTextField input/change to v-model', async () => {
    const wrapper = mount(MdTextField, {
      props: { modelValue: 'hello', label: 'API Key' }
    })
    const el = wrapper.element as HTMLElement & { value?: string }
    el.value = 'sk-test-123'
    await wrapper.trigger('input')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['sk-test-123'])
  })

  it('adapts MdSwitch selected property to boolean v-model', async () => {
    const wrapper = mount(MdSwitch, {
      props: { modelValue: false, label: 'Stream' }
    })
    const switchEl = wrapper.find('md-switch').element as HTMLElement & { selected?: boolean }
    switchEl.selected = true
    await wrapper.find('md-switch').trigger('change')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })

  it('adapts MdCheckbox checked property to boolean v-model', async () => {
    const wrapper = mount(MdCheckbox, {
      props: { modelValue: false, label: 'Agree' }
    })
    const cbEl = wrapper.find('md-checkbox').element as HTMLElement & { checked?: boolean }
    cbEl.checked = true
    await wrapper.find('md-checkbox').trigger('change')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })

  it('adapts MdSelect change event to string v-model', async () => {
    const wrapper = mount(MdSelect, {
      props: {
        modelValue: 'openai',
        options: [
          { value: 'openai', label: 'OpenAI' },
          { value: 'anthropic', label: 'Anthropic' }
        ]
      }
    })
    const selectEl = wrapper.element as HTMLElement & { value?: string }
    Object.defineProperty(selectEl, 'value', { configurable: true, get: () => 'anthropic' })
    await wrapper.trigger('change')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['anthropic'])
  })

  it('adapts MdSlider numeric value to v-model', async () => {
    const wrapper = mount(MdSlider, {
      props: { modelValue: 0.7, min: 0, max: 2, step: 0.1, label: 'Temperature' }
    })
    const sliderEl = wrapper.find('md-slider').element as HTMLElement & { value?: number }
    sliderEl.value = 1.2
    await wrapper.find('md-slider').trigger('input')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([1.2])
  })

  it('switches MdTabs by item id', async () => {
    const wrapper = mount(MdTabs, {
      props: {
        modelValue: 'usage',
        items: [
          { id: 'usage', label: 'Usage' },
          { id: 'errors', label: 'Errors', badge: 3 },
          { id: 'traffic', label: 'Traffic' }
        ]
      }
    })
    const tabs = wrapper.findAll('md-primary-tab')
    expect(tabs).toHaveLength(3)
    await tabs[2].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['traffic'])
  })

  it('toggles MdFilterChip selection and renders MdCard / MdPageHeader', async () => {
    const chip = mount(MdFilterChip, {
      props: { modelValue: false, label: 'Today' }
    })
    await chip.trigger('click')
    expect(chip.emitted('update:modelValue')?.[0]).toEqual([true])

    const card = mount(MdCard, {
      props: { title: 'Overview', variant: 'tonal' },
      slots: { default: 'Body' }
    })
    expect(card.text()).toContain('Overview')
    expect(card.classes()).toContain('md-card--tonal')

    const header = mount(MdPageHeader, {
      props: { title: 'Playground', eyebrow: 'API Tools', description: 'Test models live' }
    })
    expect(header.find('h1').text()).toBe('Playground')
  })
})
