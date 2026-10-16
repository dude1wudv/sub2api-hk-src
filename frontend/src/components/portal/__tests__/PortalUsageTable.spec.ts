import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import PortalUsageTable from '../PortalUsageTable.vue'

const mobile = ref(false)
vi.mock('@vueuse/core', () => ({ useMediaQuery: () => mobile }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

function row(first_token_ms: number | null, duration_ms: number | null) {
  return {
    id: `${first_token_ms}-${duration_ms}`,
    created_at: '2026-03-08T00:00:00Z',
    request_type: 'stream',
    billing_mode: 'token',
    input_tokens: 10,
    output_tokens: 20,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    actual_cost: 0,
    total_cost: 0,
    first_token_ms,
    duration_ms,
    image_count: 0,
    image_input_tokens: 0,
    image_output_tokens: 0,
  } as any
}

describe('PortalUsageTable latency states', () => {
  beforeEach(() => { mobile.value = false })

  it.each([false, true])('labels estimated output in desktop/mobile usage (mobile=%s)', (isMobile) => {
    mobile.value = isMobile
    const wrapper = mount(PortalUsageTable, {
      props: {
        rows: [{ ...row(1_000, 3_000), output_tokens_estimated: true }, { ...row(2_000, 4_000), output_tokens_estimated: false }],
        columns: ['tokens'], sortBy: 'created_at', sortOrder: 'desc',
      },
    })
    const badges = wrapper.findAll('[data-testid="output-tokens-estimated"]')
    expect(badges).toHaveLength(1)
    expect(badges[0].text()).toBe('usage.outputTokensEstimated')
    expect(badges[0].attributes('title')).toBe('usage.outputTokensEstimatedHint')
    expect(wrapper.findAll('.usage-estimate-note')).toHaveLength(1)
    wrapper.unmount()
  })

  it('shows provider and semantic token icons without losing cache, pricing, or request details', () => {
    const wrapper = mount(PortalUsageTable, {
      props: {
        rows: [{ ...row(1_000, 3_000), model: 'claude-sonnet-4', stream: true, cache_read_tokens: 90, cache_creation_tokens: 5, actual_cost: 0.12, total_cost: 0.24 }],
        columns: ['model', 'type', 'tokens', 'cost'], sortBy: 'created_at', sortOrder: 'desc',
      },
    })
    expect(wrapper.get('.model-icon path').attributes('fill')).toBe('#D97706')
    expect(wrapper.get('.usage-input-icon').attributes('aria-hidden')).toBe('true')
    expect(wrapper.find('.usage-output-icon').exists()).toBe(true)
    expect(wrapper.get('.usage-cache-read').text()).toBe('90')
    expect(wrapper.get('.usage-cache-write').text()).toBe('5')
    expect(wrapper.get('.usage-token-rates').text()).toContain('85.7%')
    expect(wrapper.get('.usage-token-rates').text()).toContain('10.0 tok/s')
    expect(wrapper.get('.usage-actual-cost').text()).toContain('0.12')
    expect(wrapper.get('.usage-standard-cost').text()).toContain('0.24')
    expect(wrapper.find('.usage-request-stream svg').exists()).toBe(true)
    wrapper.unmount()
  })

  it('preserves the four latency health thresholds and a neutral state for missing values', () => {
    const wrapper = mount(PortalUsageTable, {
      props: {
        rows: [row(5_000, 30_000), row(10_000, 60_000), row(30_000, 180_000), row(60_000, 300_000), row(null, null)],
        columns: ['latency'] as any,
        sortBy: 'created_at',
        sortOrder: 'desc',
      },
    })
    const firstColors = wrapper.findAll('.usage-latency-values > b:nth-of-type(1)').map(node => node.element.style.color)
    const durationColors = wrapper.findAll('.usage-latency-values > b:nth-of-type(2)').map(node => node.element.style.color)

    expect(firstColors).toEqual(['rgb(var(--text-ok))', 'rgb(var(--text-warn))', 'rgb(var(--text-warn))', 'rgb(var(--text-err))', 'rgb(var(--ink-3))'])
    expect(durationColors).toEqual(['rgb(var(--text-ok))', 'rgb(var(--text-warn))', 'rgb(var(--text-warn))', 'rgb(var(--text-err))', 'rgb(var(--ink-3))'])
    wrapper.unmount()
  })

  it('renders a mobile card with the full model name in one element', async () => {
    mobile.value = true
    const wrapper = mount(PortalUsageTable, {
      props: {
        rows: [{ ...row(1_000, 3_000), model: 'gpt-6.1-sol', stream: true, api_key: { name: 'pro' } }],
        columns: ['created_at', 'model', 'tokens', 'api_key'],
        sortBy: 'created_at',
        sortOrder: 'desc',
      },
    })
    expect(wrapper.find('table').exists()).toBe(false)
    const model = wrapper.get('.usage-card .usage-model-text')
    expect(model.text()).toBe('gpt-6.1-sol')
    expect(model.element.childElementCount).toBe(0)
    expect(wrapper.get('.usage-card-extra').text()).toContain('pro')
    await wrapper.get('.usage-card .usage-detail-link').trigger('click')
    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ model: 'gpt-6.1-sol' })
    wrapper.unmount()
  })
})
