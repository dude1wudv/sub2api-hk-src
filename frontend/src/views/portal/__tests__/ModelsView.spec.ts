import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ModelsView from '../ModelsView.vue'

const { api, appStore, route } = vi.hoisted(() => ({
  api: { get: vi.fn() },
  appStore: {
    publicSettingsLoaded: true,
    cachedPublicSettings: { model_plaza_enabled: true },
    fetchPublicSettings: vi.fn(),
  },
  route: { query: { model: 'vision' } },
}))

vi.mock('@/api/modelPlaza', () => ({ getModelPlaza: api.get }))
vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => ({ isAuthenticated: false }),
}))
vi.mock('vue-router', async () => ({
  ...await vi.importActual<typeof import('vue-router')>('vue-router'),
  useRoute: () => route,
}))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ locale: { value: 'en-US' } }),
}))

describe('ModelsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = { model: 'vision' }
    api.get.mockResolvedValue({
      description: '',
      groups: [
        {
          id: 1,
          name: 'General',
          description: '',
          platform: 'openai',
          subscription_type: 'standard',
          rate_multiplier: 1,
          peak_rate_enabled: false,
          peak_start: '',
          peak_end: '',
          peak_rate_multiplier: 1,
          is_exclusive: false,
          image_rate_independent: false,
          image_rate_multiplier: 1,
          video_rate_independent: false,
          video_rate_multiplier: 1,
          long_context_pricing_enabled: true,
          models: [{ name: 'chat-main', platform: 'openai', pricing: null, official_pricing: null }],
        },
        {
          id: 2,
          name: 'Vision models',
          description: '',
          platform: 'openai',
          subscription_type: 'standard',
          rate_multiplier: 1,
          peak_rate_enabled: false,
          peak_start: '',
          peak_end: '',
          peak_rate_multiplier: 1,
          is_exclusive: false,
          image_rate_independent: false,
          image_rate_multiplier: 1,
          video_rate_independent: false,
          video_rate_multiplier: 1,
          long_context_pricing_enabled: true,
          models: [{ name: 'vision-pro', platform: 'openai', pricing: null, official_pricing: null }],
        },
      ],
    })
  })

  it('selects the group matching the bookmarked model after async catalog load', async () => {
    const wrapper = mount(ModelsView, { global: { stubs: {
      RouterLink: RouterLinkStub,
      Icon: true,
      PortalDialog: true,
    } } })

    await flushPromises()

    const selectedGroup = wrapper.find('button[aria-pressed="true"]')
    expect(selectedGroup.exists()).toBe(true)
    expect(selectedGroup.text()).toContain('Vision models')
    wrapper.unmount()
  })
})
