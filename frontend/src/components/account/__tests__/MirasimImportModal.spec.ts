import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MirasimImportModal from '../MirasimImportModal.vue'

const BaseDialogStub = defineComponent({
  props: { show: Boolean, title: String },
  emits: ['close'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const ProxySelectorStub = defineComponent({
  props: { modelValue: Number, disabled: Boolean },
  emits: ['update:modelValue'],
  template: '<button type="button" data-testid="choose-proxy" @click="$emit(\'update:modelValue\', 17)">proxy</button>'
})

const GroupSelectorStub = defineComponent({
  props: { modelValue: Array, platform: String },
  emits: ['update:modelValue'],
  template: '<button type="button" data-testid="choose-group" @click="$emit(\'update:modelValue\', [23])">group</button>'
})

function mountModal(submitting = false) {
  return mount(MirasimImportModal, {
    props: {
      show: true,
      submitting,
      proxies: [],
      groups: [
        { id: 23, name: 'Mirasim pricing', platform: 'mirasim', status: 'active' },
        { id: 24, name: 'Inactive', platform: 'mirasim', status: 'inactive' },
        { id: 25, name: 'Other platform', platform: 'openai', status: 'active' }
      ] as any
    },
    global: { stubs: { BaseDialog: BaseDialogStub, ProxySelector: ProxySelectorStub, GroupSelector: GroupSelectorStub } }
  })
}

describe('MirasimImportModal', () => {
  it('submits the chosen import options and defaults concurrency to five', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-testid="choose-proxy"]').trigger('click')
    await wrapper.get('[data-testid="choose-group"]').trigger('click')
    await wrapper.get('input[max="100"]').setValue('9')
    await wrapper.get('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toEqual([[{
      name: '', proxy_id: 17, group_ids: [23], concurrency: 9
    }]])
    wrapper.unmount()
  })

  it('prevents duplicate submits and cancel while submitting', async () => {
    const wrapper = mountModal(true)
    await wrapper.get('form').trigger('submit.prevent')
    await wrapper.get('button.btn-secondary').trigger('click')
    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.emitted('close')).toBeUndefined()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
  })

  it('cancels before import', async () => {
    const wrapper = mountModal()
    await wrapper.get('button.btn-secondary').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })
})
