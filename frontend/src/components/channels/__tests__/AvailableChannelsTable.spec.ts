import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'
import type { UserAvailableChannel } from '@/api/channels'
import { useColorMode } from '@/composables/useColorMode'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh-CN' } }),
  }
})

const componentPath = resolve(process.cwd(), 'src/components/channels/AvailableChannelsTable.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AvailableChannelsTable card layout integration', () => {
  it('mounts the channel list and its cards on the .table-wrapper scroll hook', () => {
    expect(componentSource).toMatch(/<div class="table-wrapper">\s*<div data-testid="available-channels"[^>]*class="channel-list">/)
    expect(componentSource).toMatch(/<section\s+v-else\s+v-for="\(channel, chIdx\) in rows"[\s\S]*?class="channel-card"/)
  })

  it('does not clip content with its own overflow-hidden card wrapper', () => {
    expect(componentSource).not.toMatch(/<div class="card overflow-hidden">/)
  })
})

const rows: UserAvailableChannel[] = [
  {
    name: 'Primary channel',
    description: 'Fast and reliable access',
    platforms: [
      {
        platform: 'anthropic',
        groups: [
          {
            id: 1,
            name: 'Exclusive Pro',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 1.2,
            peak_rate_enabled: true,
            peak_start: '08:00',
            peak_end: '10:00',
            peak_rate_multiplier: 1.5,
            is_exclusive: true,
          },
          {
            id: 2,
            name: 'Public',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 1,
            peak_rate_enabled: false,
            peak_start: '',
            peak_end: '',
            peak_rate_multiplier: 1,
            is_exclusive: false,
          },
        ],
        supported_models: [{ name: 'claude-test', platform: 'anthropic', pricing: null }],
      },
    ],
  },
]

const baseProps = {
  columns: {
    name: 'Channel',
    description: 'Description',
    platform: 'Platform',
    groups: 'Groups and rates',
    supportedModels: 'Models and pricing',
  },
  rows,
  loading: false,
  pricingKeyPrefix: 'availableChannels.pricing',
  noPricingLabel: 'No pricing',
  noModelsLabel: 'No models',
  emptyLabel: 'No channels',
  userGroupRates: { 1: 0.8 },
}

function mountTable(props = {}) {
  return mount(AvailableChannelsTable, {
    props: { ...baseProps, ...props },
    global: {
      plugins: [createPinia()],
      stubs: {
        Icon: { props: ['name'], template: '<i :data-icon="name" />' },
        PlatformIcon: { template: '<i data-platform-icon />' },
        GroupBadge: {
          props: ['name', 'rateMultiplier', 'userRateMultiplier'],
          template:
            '<span data-group-badge>{{ name }}:{{ rateMultiplier }}:{{ userRateMultiplier }}</span>',
        },
        SupportedModelChip: {
          props: ['model', 'noPricingLabel'],
          template: '<span data-model-chip>{{ model.name }}:{{ model.pricing?.marker || noPricingLabel }}</span>',
        },
      },
    },
  })
}

describe('AvailableChannelsTable responsive surfaces', () => {
  it.each(['light', 'dark'] as const)('shares complete channel cards in %s', theme => {
    useColorMode().setMode(theme)
    const wrapper = mountTable()
    const cards = wrapper.get('[data-testid="available-channels"]')
    expect(cards.findAll('.channel-card')).toHaveLength(1)
    expect(cards.text()).toContain('Primary channel')
    expect(cards.text()).toContain('Fast and reliable access')
    expect(cards.findAll('[data-group-badge]')).toHaveLength(2)
    expect(cards.get('[data-model-chip]').text()).toContain('claude-test:No pricing')
    wrapper.unmount()
  })

  it('renders a readable surface with groups, rates, peaks, and model pricing chips', () => {
    const wrapper = mountTable()
    const mobile = wrapper.get('[data-testid="available-channels"]')

    expect(mobile.classes()).toContain('channel-list')
    expect(mobile.text()).toContain('Primary channel')
    expect(mobile.text()).toContain('Fast and reliable access')
    expect(mobile.text()).toContain('Groups and rates')
    expect(mobile.text()).toContain('Models and pricing')
    expect(mobile.text()).toContain('availableChannels.exclusive')
    expect(mobile.text()).toContain('availableChannels.public')
    expect(mobile.get('[data-group-badge]').text()).toBe('Exclusive Pro:1.2:0.8')
    expect(mobile.findAll('[data-group-badge]')).toHaveLength(2)
    expect(mobile.get('[data-icon="clock"]')).toBeTruthy()
    expect(mobile.text()).toContain('08:00')
    expect(mobile.text()).toContain('10:00')
    expect(mobile.text()).toContain('×1.5')
    expect(mobile.get('[data-model-chip]').text()).toBe('claude-test:No pricing')
    expect(mobile.findAll('.max-w-full')).not.toHaveLength(0)
  })

  it('keeps the mobile placeholders when a platform has no groups or models', () => {
    const wrapper = mountTable({
      rows: [
        {
          name: 'Fallback channel',
          description: '',
          platforms: [{ platform: 'openai', groups: [], supported_models: [] }],
        },
      ],
    })
    const mobile = wrapper.get('[data-testid="available-channels"]')

    expect(mobile.text()).toContain('Fallback channel')
    expect(mobile.text()).toContain('openai')
    expect(mobile.text()).toContain('No models')
    expect(mobile.findAll('dd')[0].text()).toBe('-')
  })

  it('provides loading and empty states for the channel list', async () => {
    const wrapper = mountTable({ loading: true, rows: [] })

    expect(wrapper.get('[data-testid="mobile-loading"] [data-icon="refresh"]')).toBeTruthy()

    await wrapper.setProps({ loading: false })

    expect(wrapper.get('[data-testid="mobile-empty"]').text()).toContain('No channels')
  })
})

describe('AvailableChannelsTable model discovery', () => {
  const sevenModels = Array.from({ length: 7 }, (_, index) => ({
    name: `model-${index + 1}`,
    platform: 'anthropic',
    pricing: index === 6 ? { marker: 'priced' } : null,
  }))

  beforeEach(() => useColorMode().setMode('dark'))
  afterEach(() => useColorMode().setMode('light'))

  it('shows six models by default and reveals the rest with their pricing when expanded', async () => {
    const wrapper = mountTable({
      rows: [{ ...rows[0], platforms: [{ ...rows[0].platforms[0], supported_models: sevenModels }] }],
    })
    const list = wrapper.get('[data-testid="available-channels"]')
    expect(list.findAll('[data-model-chip]')).toHaveLength(6)
    expect(list.text()).toContain('7')

    await list.get('.row-action').trigger('click')
    expect(list.findAll('[data-model-chip]')).toHaveLength(7)
    expect(list.text()).toContain('model-7:priced')
    expect(list.get('.row-action').attributes('aria-expanded')).toBe('true')
    wrapper.unmount()
  })

  it('searches the full model list and keeps group multipliers and prices visible', () => {
    const wrapper = mountTable({
      searchQuery: 'model-7',
      rows: [{ ...rows[0], platforms: [{ ...rows[0].platforms[0], supported_models: sevenModels }] }],
    })
    const list = wrapper.get('[data-testid="available-channels"]')
    expect(list.findAll('[data-model-chip]')).toHaveLength(1)
    expect(list.get('[data-model-chip]').text()).toBe('model-7:priced')
    expect(list.get('[data-group-badge]').text()).toBe('Exclusive Pro:1.2:0.8')
    wrapper.unmount()
  })
})
