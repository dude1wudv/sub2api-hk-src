import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  BUILTIN_PLATFORM_CATALOG,
  resetPlatformCatalog,
  setPlatformCatalog
} from '@/constants/platformCatalog'
import ChannelsView from '@/views/admin/ChannelsView.vue'

const { listChannels, getGroups, syncPricingModels, createChannel, updateChannel, showSuccess, showError } = vi.hoisted(() => ({
  listChannels: vi.fn(),
  getGroups: vi.fn(),
  syncPricingModels: vi.fn(),
  createChannel: vi.fn(),
  updateChannel: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channels: { list: listChannels, syncPricingModels, create: createChannel, update: updateChannel },
    groups: { getAll: getGroups },
    settings: { getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false }) }
  }
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key })
}))

function mountView() {
  return mount(ChannelsView, {
    global: {
      stubs: {
        AppLayout: defineComponent({ template: '<main><slot /></main>' }),
        TablePageLayout: defineComponent({
          template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>'
        }),
        BaseDialog: defineComponent({
          props: ['show'],
          template: '<section v-if="show"><slot /><slot name="footer" /></section>'
        }),
        PricingEntryCard: defineComponent({
          name: 'PricingEntryCard',
          props: ['entry'],
          emits: ['update'],
          template: '<div class="pricing-entry">{{ entry.models.join(", ") }}</div>'
        }),
        DataTable: defineComponent({
          props: { data: { type: Array, default: () => [] } },
          template: '<div><slot v-if="data.length" name="cell-actions" :row="data[0]" /></div>',
        }),
        Pagination: true,
        ConfirmDialog: true,
        EmptyState: true,
        Select: true,
        Icon: true,
        PlatformIcon: true,
        Toggle: true
      }
    }
  })
}

async function openDialog(wrapper: ReturnType<typeof mountView>) {
  await flushPromises()
  await wrapper.findAll('button').find(button => button.text() === 'admin.channels.createChannel')!.trigger('click')
  await flushPromises()
}

async function enablePlatform(wrapper: ReturnType<typeof mountView>, platform: string) {
  const label = wrapper.findAll('label').find(label => label.text() === `admin.groups.platforms.${platform}`)
  expect(label).toBeDefined()
  await label!.get('input[type="checkbox"]').setValue(true)
  await wrapper.findAll('button.channel-tab').find(button => button.text() === `admin.groups.platforms.${platform}`)!.trigger('click')
}

function visibleSyncButtons(wrapper: ReturnType<typeof mountView>) {
  return wrapper.findAll('button').filter(button => button.isVisible() && [
    'admin.channels.form.syncLatestModels',
    'admin.channels.form.syncingModels'
  ].includes(button.text()))
}

describe('ChannelsView pricing model sync', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    listChannels.mockResolvedValue({ items: [], total: 0 })
    getGroups.mockResolvedValue([])
    syncPricingModels.mockReset()
    createChannel.mockResolvedValue({})
    updateChannel.mockResolvedValue({})
  })

  afterEach(() => {
    resetPlatformCatalog()
  })

  it.each([
    ['openai', true],
    ['opencode_go', true],
    ['typesafe', true],
    ['command_code', false],
    ['cline', false]
  ] as const)('offers pricing model sync for %s only when its pricing catalog exists', async (platform, expected) => {
    const wrapper = mountView()
    await openDialog(wrapper)
    await enablePlatform(wrapper, platform)

    expect(visibleSyncButtons(wrapper)).toHaveLength(expected ? 1 : 0)
    expect(syncPricingModels).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('reacts to a new platform gaining a pricing catalog', async () => {
    const platform = { id: 'acme_router', display_name: 'Acme Router', gateway: 'openai' as const, cn_provider: false }
    setPlatformCatalog({
      ...BUILTIN_PLATFORM_CATALOG,
      platforms: [...BUILTIN_PLATFORM_CATALOG.platforms, platform]
    })
    const wrapper = mountView()
    await openDialog(wrapper)
    await enablePlatform(wrapper, platform.id)
    expect(visibleSyncButtons(wrapper)).toHaveLength(0)

    setPlatformCatalog({
      ...BUILTIN_PLATFORM_CATALOG,
      platforms: [...BUILTIN_PLATFORM_CATALOG.platforms, { ...platform, litellm_provider: 'acme' }]
    })
    await flushPromises()
    expect(visibleSyncButtons(wrapper)).toHaveLength(1)
    wrapper.unmount()
  })

  it('blocks additional syncs while one request is in flight and adds its models once', async () => {
    let resolveSync!: (value: { models: string[] }) => void
    syncPricingModels.mockReturnValue(new Promise(resolve => { resolveSync = resolve }))
    const wrapper = mountView()
    await openDialog(wrapper)
    await enablePlatform(wrapper, 'openai')
    await visibleSyncButtons(wrapper)[0].trigger('click')
    await visibleSyncButtons(wrapper)[0].trigger('click')
    await enablePlatform(wrapper, 'opencode_go')

    expect(visibleSyncButtons(wrapper)[0].attributes('disabled')).toBeDefined()
    await visibleSyncButtons(wrapper)[0].trigger('click')
    expect(syncPricingModels).toHaveBeenCalledOnce()
    expect(syncPricingModels).toHaveBeenCalledWith('openai')

    resolveSync({ models: ['gpt-latest'] })
    await flushPromises()
    expect(wrapper.findAll('.pricing-entry')).toHaveLength(1)
    expect(wrapper.get('.pricing-entry').text()).toBe('gpt-latest')
    expect(visibleSyncButtons(wrapper)[0].attributes('disabled')).toBeUndefined()
    expect(showError).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  const pricedChannel = {
    id: 7, name: 'OpenAI prices', description: '', status: 'active',
    group_ids: [42], model_mapping: {}, account_stats_pricing_rules: [],
    model_pricing: [{ platform: 'openai', models: ['gpt-5.5'], billing_mode: 'token',
      input_price: 3e-6, output_price: 15e-6, cache_write_price: null, cache_read_price: null,
      fast_multiplier: 3, ultrafast_multiplier: 9, flex_multiplier: 0.5,
      image_input_price: null, image_output_price: null, per_request_price: null,
      intervals: [], time_pricing: null }],
  }

  it('creates a channel with independent Fast and Ultra Fast overrides', async () => {
    getGroups.mockResolvedValue([{ id: 42, name: 'OpenAI test group', platform: 'openai', rate_multiplier: 1 }])
    syncPricingModels.mockResolvedValue({ models: ['gpt-5.5'] })
    const wrapper = mountView()
    await openDialog(wrapper)
    await wrapper.get('#channel-form input[type="text"]').setValue('New speed pricing')
    await enablePlatform(wrapper, 'openai')
    await wrapper.findAll('label').find(label => label.text().includes('OpenAI test group'))!.get('input[type="checkbox"]').setValue(true)
    await visibleSyncButtons(wrapper)[0].trigger('click')
    await flushPromises()
    const card = wrapper.getComponent({ name: 'PricingEntryCard' })
    expect(card.props('entry')).toMatchObject({ fast_multiplier: null, ultrafast_multiplier: null, flex_multiplier: null })
    card.vm.$emit('update', { ...card.props('entry'), fast_multiplier: '2.75', ultrafast_multiplier: '8.5' })
    await flushPromises()
    await wrapper.get('#channel-form').trigger('submit')
    await flushPromises()
    expect(createChannel).toHaveBeenCalledWith(expect.objectContaining({
      group_ids: [42], model_pricing: [expect.objectContaining({ fast_multiplier: 2.75, ultrafast_multiplier: 8.5, flex_multiplier: null })],
    }))
    expect(showError).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  async function editPricedChannel() {
    getGroups.mockResolvedValue([{ id: 42, name: 'OpenAI test group', platform: 'openai', rate_multiplier: 1 }])
    listChannels.mockResolvedValue({ items: [pricedChannel], total: 1 })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'common.edit')!.trigger('click')
    await flushPromises()
    return wrapper
  }

  it('loads, edits, and saves tier multipliers without converting them as token prices', async () => {
    const wrapper = await editPricedChannel()
    const card = wrapper.getComponent({ name: 'PricingEntryCard' })
    expect(card.props('entry')).toMatchObject({ input_price: 3, fast_multiplier: 3, ultrafast_multiplier: 9, flex_multiplier: 0.5 })
    card.vm.$emit('update', { ...card.props('entry'), ultrafast_multiplier: '11.25' })
    await flushPromises()
    await wrapper.get('#channel-form').trigger('submit')
    await flushPromises()
    expect(updateChannel).toHaveBeenCalledWith(7, expect.objectContaining({
      model_pricing: [expect.objectContaining({ input_price: 3e-6, fast_multiplier: 3, ultrafast_multiplier: 11.25, flex_multiplier: 0.5 })],
    }))
    expect(showError).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it.each(['0', '-1', 'Infinity', 'not-a-number'])('rejects invalid Ultra Fast multiplier %s before saving', async multiplier => {
    const wrapper = await editPricedChannel()
    const card = wrapper.getComponent({ name: 'PricingEntryCard' })
    card.vm.$emit('update', { ...card.props('entry'), ultrafast_multiplier: multiplier })
    await flushPromises()
    await wrapper.get('#channel-form').trigger('submit')
    await flushPromises()
    expect(updateChannel).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith(expect.stringContaining('admin.channels.form.multiplierPositive'))
    wrapper.unmount()
  })

  it('clears only Ultra Fast to restore its default while retaining the other multipliers', async () => {
    const wrapper = await editPricedChannel()
    const card = wrapper.getComponent({ name: 'PricingEntryCard' })
    card.vm.$emit('update', { ...card.props('entry'), ultrafast_multiplier: '' })
    await flushPromises()
    await wrapper.get('#channel-form').trigger('submit')
    await flushPromises()
    expect(updateChannel).toHaveBeenCalledWith(7, expect.objectContaining({
      model_pricing: [expect.objectContaining({ fast_multiplier: 3, ultrafast_multiplier: null, flex_multiplier: 0.5 })],
    }))
    wrapper.unmount()
  })
})
