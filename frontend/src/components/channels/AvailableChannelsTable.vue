<template>
  <!-- .table-wrapper 是 TablePageLayout 滚动链的挂载点：外层 .table-scroll-container
       负责卡片外观并 overflow-hidden，本层接收 overflow-y-auto 才能在内容超高时滚动。 -->
  <div class="table-wrapper">
    <div data-testid="available-channels" class="channel-list">
      <div v-if="loading" data-testid="mobile-loading" class="py-10 text-center">
        <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
      </div>
      <div v-else-if="rows.length === 0" data-testid="mobile-empty" class="py-12 text-center">
        <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
      </div>
      <section
        v-else
        v-for="(channel, chIdx) in rows"
        :key="`mobile-${channel.name}-${chIdx}`"
        class="channel-card"
      >
        <header class="mb-3 min-w-0">
          <h3 class="break-words text-sm font-semibold text-gray-900 dark:text-white">
            {{ channel.name }}
          </h3>
          <p class="mt-1 break-words text-xs leading-5 text-gray-500 dark:text-gray-400">
            {{ channel.description || '-' }}
          </p>
        </header>

        <div class="divide-y divide-gray-100 dark:divide-dark-700/60">
          <div
            v-for="section in channel.platforms"
            :key="`mobile-${channel.name}-${section.platform}`"
            class="min-w-0 py-3 first:pt-0 last:pb-0"
          >
            <span
              :class="[
                'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                platformBadgeClass(section.platform),
              ]"
            >
              <PlatformIcon :platform="section.platform as GroupPlatform" size="xs" />
              {{ section.platform }}
            </span>

            <dl class="channel-section">
              <div class="min-w-0">
                <dt class="mb-1.5 text-[11px] font-medium text-gray-500 dark:text-gray-400">
                  {{ columns.groups }}
                </dt>
                <dd class="flex min-w-0 flex-col gap-2">
                  <div
                    v-if="exclusiveGroups(section).length > 0"
                    class="flex min-w-0 flex-wrap items-center gap-1.5"
                  >
                    <span
                      class="inline-flex items-center gap-0.5 text-[10px] font-medium uppercase text-purple-600 dark:text-purple-400"
                      :title="t('availableChannels.exclusiveTooltip')"
                    >
                      <Icon name="shield" size="xs" class="h-3 w-3" />
                      {{ t('availableChannels.exclusive') }}
                    </span>
                    <div
                      v-for="g in exclusiveGroups(section)"
                      :key="`mobile-ex-${g.id}`"
                      class="inline-flex max-w-full min-w-0 flex-wrap items-center gap-1"
                    >
                      <GroupBadge
                        class="max-w-full"
                        :name="g.name"
                        :platform="g.platform as GroupPlatform"
                        :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                        :rate-multiplier="g.rate_multiplier"
                        :user-rate-multiplier="userGroupRates[g.id] ?? null"
                        always-show-rate
                      />
                      <span
                        v-if="hasPeakRate(g)"
                        class="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                        :title="peakRateTitle(g)"
                      >
                        <Icon name="clock" size="xs" class="h-3 w-3" />
                        {{ peakRateLabel(g) }}
                      </span>
                    </div>
                  </div>
                  <div
                    v-if="publicGroups(section).length > 0"
                    class="flex min-w-0 flex-wrap items-center gap-1.5"
                  >
                    <span
                      class="inline-flex items-center gap-0.5 text-[10px] font-medium uppercase text-gray-500 dark:text-gray-400"
                      :title="t('availableChannels.publicTooltip')"
                    >
                      <Icon name="globe" size="xs" class="h-3 w-3" />
                      {{ t('availableChannels.public') }}
                    </span>
                    <div
                      v-for="g in publicGroups(section)"
                      :key="`mobile-pub-${g.id}`"
                      class="inline-flex max-w-full min-w-0 flex-wrap items-center gap-1"
                    >
                      <GroupBadge
                        class="max-w-full"
                        :name="g.name"
                        :platform="g.platform as GroupPlatform"
                        :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                        :rate-multiplier="g.rate_multiplier"
                        :user-rate-multiplier="userGroupRates[g.id] ?? null"
                        always-show-rate
                      />
                      <span
                        v-if="hasPeakRate(g)"
                        class="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                        :title="peakRateTitle(g)"
                      >
                        <Icon name="clock" size="xs" class="h-3 w-3" />
                        {{ peakRateLabel(g) }}
                      </span>
                    </div>
                  </div>
                  <span v-if="section.groups.length === 0" class="text-xs text-gray-400">-</span>
                </dd>
              </div>

              <div class="min-w-0">
                <dt class="mb-1.5 text-[11px] font-medium text-gray-500 dark:text-gray-400">
                  {{ columns.supportedModels }} <span class="ml-1 tabular-nums">{{ section.supported_models.length }}</span>
                </dt>
                <dd class="flex min-w-0 flex-wrap gap-1">
                  <SupportedModelChip
                    v-for="m in visibleModels(channel, section)"
                    :key="`mobile-${section.platform}-${m.name}`"
                    class="max-w-full [&>span]:max-w-full [&>span]:truncate"
                    :model="m"
                    :pricing-key-prefix="pricingKeyPrefix"
                    :no-pricing-label="noPricingLabel"
                    :show-platform="false"
                    :platform-hint="section.platform"
                  />
                  <button v-if="section.supported_models.length > 6 && !searchQuery.trim()" type="button" class="row-action" :aria-expanded="expandedSections.has(sectionKey(channel, section))" @click="toggleModels(channel, section)">
                    {{ expandedSections.has(sectionKey(channel, section)) ? (zh ? '收起' : 'Show less') : (zh ? '查看全部 ' : 'Show all ') + section.supported_models.length }}
                    <Icon :name="expandedSections.has(sectionKey(channel, section)) ? 'chevronDown' : 'chevronRight'" size="xs" />
                  </button>
                  <span v-if="section.supported_models.length === 0" class="text-xs text-gray-400">
                    {{ noModelsLabel }}
                  </span>
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from './SupportedModelChip.vue'
import type { UserAvailableChannel, UserAvailableGroup, UserChannelPlatformSection } from '@/api/channels'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformBadgeClass } from '@/utils/platformColors'
import { useAppStore } from '@/stores/app'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'

const props = defineProps<{
  columns: {
    name: string
    description: string
    platform: string
    groups: string
    supportedModels: string
  }
  searchQuery?: string
  rows: UserAvailableChannel[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  /** 用户专属倍率（group_id → multiplier）；无专属时由 GroupBadge 仅显示默认倍率。 */
  userGroupRates: Record<number, number>
}>()

// Suppress unused warning — props is accessed via template automatically but
// the explicit reference here keeps the linter from flagging userGroupRates.
void props.userGroupRates

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const searchQuery = computed(() => props.searchQuery ?? '')
const expandedSections = ref(new Set<string>())
function sectionKey(channel: UserAvailableChannel, section: UserChannelPlatformSection) { return JSON.stringify([channel.name, section.platform]) }
function toggleModels(channel: UserAvailableChannel, section: UserChannelPlatformSection) {
  const key = sectionKey(channel, section)
  const next = new Set(expandedSections.value)
  if (next.has(key)) next.delete(key); else next.add(key)
  expandedSections.value = next
}
function visibleModels(channel: UserAvailableChannel, section: UserChannelPlatformSection) {
  const query = searchQuery.value.trim().toLowerCase()
  if (query) {
    const matches = section.supported_models.filter(model => model.name.toLowerCase().includes(query))
    return matches.length ? matches : section.supported_models
  }
  return expandedSections.value.has(sectionKey(channel, section)) ? section.supported_models : section.supported_models.slice(0, 6)
}

function exclusiveGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => g.is_exclusive)
}

function publicGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => !g.is_exclusive)
}

const appStore = useAppStore()

function hasPeakRate(group: UserAvailableGroup): boolean {
  return groupHasPeakRate(group)
}

function peakRateLabel(group: UserAvailableGroup): string {
  return formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

function peakRateTitle(group: UserAvailableGroup): string {
  return t('common.peakRateTooltip', { window: peakRateLabel(group) }) + t('common.peakRateImageNote')
}
</script>
