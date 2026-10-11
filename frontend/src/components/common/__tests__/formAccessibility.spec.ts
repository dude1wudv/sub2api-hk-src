import { afterEach, describe, expect, it } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import Input from '../Input.vue'
import TextArea from '../TextArea.vue'

enableAutoUnmount(afterEach)

describe.each([{ component: Input, tag: 'input' }, { component: TextArea, tag: 'textarea' }])('$tag accessible field', ({ component, tag }) => {
  it('generates distinct stable IDs and associates labels and hints', async () => {
    const wrapper = mount(defineComponent({
      components: { Field: component },
      template: '<div><Field model-value="" label="First" hint="Help"/><Field model-value="" label="Second"/></div>',
    }))
    const fields = wrapper.findAll(tag)
    const firstId = fields[0].attributes('id')
    expect(firstId).toBeTruthy()
    expect(firstId).not.toBe(fields[1].attributes('id'))
    expect(wrapper.get('label').attributes('for')).toBe(firstId)
    expect(fields[0].attributes('aria-describedby')).toBe(wrapper.get('.input-hint').attributes('id'))
    await fields[0].setValue('new value')
    expect(fields[0].attributes('id')).toBe(firstId)
  })

  it('keeps a supplied ID and external description while announcing validation', async () => {
    const wrapper = mount(component, { props: { id: 'named-field', modelValue: '', label: 'Name', hint: 'Hint', 'aria-describedby': 'external-help' } })
    expect(wrapper.get(tag).attributes('aria-describedby')).toBe('external-help named-field-message')
    await wrapper.setProps({ error: 'Required' })
    expect(wrapper.get(tag).attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('[role="alert"]').attributes('id')).toBe('named-field-message')
    expect(wrapper.find('.input-hint').exists()).toBe(false)
    await wrapper.setProps({ error: '', hint: '' })
    expect(wrapper.get(tag).attributes('aria-invalid')).toBeUndefined()
    expect(wrapper.get(tag).attributes('aria-describedby')).toBe('external-help')
  })
})
