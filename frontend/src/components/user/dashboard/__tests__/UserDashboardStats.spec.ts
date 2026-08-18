import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UserDashboardStats from '../UserDashboardStats.vue'
import type { UserDashboardStats as UserDashboardStatsType } from '@/api/usage'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const stats: UserDashboardStatsType = {
  total_api_keys: 2,
  active_api_keys: 1,
  total_requests: 20,
  total_input_tokens: 40,
  total_output_tokens: 50,
  total_cache_creation_tokens: 30,
  total_cache_read_tokens: 70,
  total_tokens: 190,
  total_cost: 0.2,
  total_actual_cost: 0.1,
  today_requests: 2,
  today_input_tokens: 4,
  today_output_tokens: 5,
  today_cache_creation_tokens: 30,
  today_cache_read_tokens: 70,
  today_tokens: 19,
  today_cost: 0.02,
  today_actual_cost: 0.01,
  average_duration_ms: 100,
  rpm: 1,
  tpm: 2
}

describe('UserDashboardStats', () => {
  it('shows the combined cache creation and read tokens for today and total', () => {
    const wrapper = mount(UserDashboardStats, {
      props: {
        stats,
        balance: 1,
        isSimple: true
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text().match(/dashboard\.cache: 100/g)).toHaveLength(2)
  })
})
