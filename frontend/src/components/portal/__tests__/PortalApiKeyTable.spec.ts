import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import PortalApiKeyTable from '../PortalApiKeyTable.vue'
import type { ApiKey } from '@/types'

const mobile = ref(false)
vi.mock('@vueuse/core', () => ({ useMediaQuery: () => mobile }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('zh-CN'), t: (key: string) => key }) }
})

function key(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: 3,
    user_id: 1,
    key: 'sk-test-key-value-1234',
    name: 'pro',
    group_id: 7,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    last_used_ip: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '2026-10-10T11:01:38Z',
    updated_at: '2026-10-10T11:01:38Z',
    current_concurrency: 1,
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    routing_group_ids: [7, 8],
    group: { id: 7, name: 'ChatGPT-Pro', platform: 'openai', rate_multiplier: 1 } as ApiKey['group'],
    ...overrides,
  }
}

function render(apiKey = key(), props: Record<string, unknown> = {}) {
  return mount(PortalApiKeyTable, {
    props: {
      keys: [apiKey],
      groups: [],
      groupRates: {},
      usageStats: {},
      busyIds: new Set<number>(),
      selectedIds: [],
      ...props,
    },
    global: {
      stubs: {
        Icon: true,
        GroupBadge: { props: ['name'], template: '<span class="group-badge">{{ name }}</span>' },
      },
    },
  })
}

describe('PortalApiKeyTable groups and mobile cards', () => {
  beforeEach(() => { mobile.value = false })

  it('opens the editor when the group is clicked on desktop', async () => {
    const apiKey = key()
    const wrapper = render(apiKey)
    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-key-card"]').exists()).toBe(false)
    expect(wrapper.get('.portal-api-routing').text()).toBe('智能路由 · 2')
    await wrapper.get('[data-test="edit-group"]').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([apiKey])
    wrapper.unmount()
  })

  it('renders a mobile card with every action and a clickable group', async () => {
    mobile.value = true
    const apiKey = key()
    const wrapper = render(apiKey)
    expect(wrapper.find('table').exists()).toBe(false)
    const card = wrapper.get('[data-test="api-key-card"]')
    expect(card.get('.portal-api-card-name').text()).toBe('pro')
    expect(card.get('.group-badge').text()).toBe('ChatGPT-Pro')
    for (const label of ['使用密钥', '导入到 CCS', '禁用', '编辑', '删除']) {
      expect(card.text()).toContain(label)
    }
    await card.get('[data-test="edit-group"]').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([apiKey])
    wrapper.unmount()
  })

  it('disables the group button while the key is busy and hides CCS import when requested', () => {
    mobile.value = true
    const wrapper = render(key(), { hideCcs: true, busyIds: new Set([3]) })
    expect(wrapper.find('[data-test="import-ccs"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="edit-group"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
