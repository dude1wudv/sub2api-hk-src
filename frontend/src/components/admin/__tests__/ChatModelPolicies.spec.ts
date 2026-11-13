import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ChatModelPoliciesEditor from '../ChatModelPoliciesEditor.vue'
import { parseChatModelPolicies } from '@/utils/chatModelPolicies'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('chat model policies', () => {
  it('uses exact public models and never enables unspecified capabilities', () => {
    const policies = parseChatModelPolicies(JSON.stringify({ 'same-public-model': { transport: 'chat_completions' } }))
    expect(policies['same-public-model']).toEqual({
      transport: 'chat_completions', image_input: false, document_input: false,
      native_deepseek_extensions: false, native_files: false, native_search: false, native_read_url: false,
    })
  })

  it.each([
    { 'model-*': { transport: 'messages' } },
    { model: { transport: 'responses', native_files: true } },
    { model: { transport: 'messages', document_input: true } },
    { model: { transport: 'chat_completions', api_key: 'never-store' } },
    { model: { transport: 'chat_completions', image_input: 'true' } },
    null, [],
  ])('rejects invalid policy %j', value => {
    expect(() => parseChatModelPolicies(JSON.stringify(value))).toThrow()
  })

  it('blocks invalid JSON without overwriting the last valid policy', async () => {
    const wrapper = mount(ChatModelPoliciesEditor, { props: { modelValue: {} } })
    const input = wrapper.get('textarea')
    await input.setValue('{')
    expect(wrapper.find('[role=alert]').exists()).toBe(true)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(wrapper.vm.validate()).toBe(false)
    await input.setValue(JSON.stringify({ 'public-model': { transport: 'messages', document_input: true, native_files: true } }))
    expect(wrapper.find('[role=alert]').exists()).toBe(false)
    const emitted = wrapper.emitted('update:modelValue')!
    expect(emitted.at(-1)?.[0]).toMatchObject({ 'public-model': { transport: 'messages', native_files: true } })
    await wrapper.setProps({ modelValue: {} })
    expect(input.element.value).toBe('{}')
  })
})
