import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
import SlowTTFTSettings from '../SlowTTFTSettings.vue'
import { defaultSlowTTFT } from '../slowTTFT'

const Host = defineComponent({
  components: { SlowTTFTSettings },
  setup() { return { settings: ref(defaultSlowTTFT()) } },
  template: '<SlowTTFTSettings v-model="settings" />'
})

describe('SlowTTFTSettings', () => {
  it('toggles protection and exposes editable numeric thresholds', async () => {
    const wrapper = mount(Host)
    expect(wrapper.findAll('input[type="number"]')).toHaveLength(0)
    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.findAll('input[type="number"]')).toHaveLength(5)
    expect(wrapper.get('[data-testid="threshold_seconds"]').element).toMatchObject({ value: '15', max: '3600' })
    expect(wrapper.get('[data-testid="consecutive_count"]').element).toMatchObject({ value: '2', max: '1000' })
    expect(wrapper.get('[data-testid="window_seconds"]').element).toMatchObject({ value: '300', max: '86400' })
    expect(wrapper.get('[data-testid="window_count"]').element).toMatchObject({ value: '3', max: '1000' })
    expect(wrapper.get('[data-testid="pause_seconds"]').element).toMatchObject({ value: '1800', max: '604800' })
    await wrapper.get('[data-testid="window_count"]').setValue('6')
    expect((wrapper.get('[data-testid="window_count"]').element as HTMLInputElement).value).toBe('6')
  })
})
