import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PlatformUsageBreakdown from '../PlatformUsageBreakdown.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)

describe('platform usage details', () => {
  it('opens by tap with complete totals and closes on a second tap', async () => {
    const wrapper = mount(PlatformUsageBreakdown, {
      props: { today: 5, total: 20, byPlatform: [{ platform: 'openai', today_actual_cost: 2, total_actual_cost: 12 }] },
    })
    const trigger = wrapper.get('button')
    const panel = document.body.querySelector<HTMLElement>('[role="dialog"]')!
    expect(panel.textContent).not.toContain('OpenAI')
    await trigger.trigger('click'); await flushPromises()
    expect(panel.textContent).toContain('OpenAI')
    expect(panel.textContent).toContain('$2.0000')
    expect(panel.textContent).toContain('$12.0000')
    expect(panel.textContent).toContain('admin.users.platformOther')
    expect(panel.textContent).toContain('$3.0000')
    expect(panel.textContent).toContain('$8.0000')
    await trigger.trigger('click'); await flushPromises()
    expect(panel.style.display).toBe('none')
  })
})
