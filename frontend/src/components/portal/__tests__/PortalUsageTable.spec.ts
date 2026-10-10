import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PortalUsageTable from '../PortalUsageTable.vue'

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

    expect(firstColors).toEqual(['rgb(22, 128, 74)', 'rgb(166, 108, 0)', 'rgb(194, 81, 23)', 'rgb(200, 49, 49)', 'rgb(146, 144, 136)'])
    expect(durationColors).toEqual(['rgb(22, 128, 74)', 'rgb(166, 108, 0)', 'rgb(194, 81, 23)', 'rgb(200, 49, 49)', 'rgb(146, 144, 136)'])
    wrapper.unmount()
  })
})
