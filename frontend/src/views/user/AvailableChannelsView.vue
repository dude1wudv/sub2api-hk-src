<template>
  <div class="mx-auto max-w-[1440px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ t('nav.availableChannels') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ copy('浏览当前账户可用的模型渠道、分组与实时计费倍率。', 'Browse available model channels, routing groups, and active rate multipliers for your account.') }}
      </p>
    </header>

    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
          <div class="relative w-full sm:max-w-sm">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('availableChannels.searchPlaceholder')"
              class="atelier-input !pl-9"
            />
          </div>

          <div class="flex items-center justify-end gap-2">
            <UiButton
              v-if="!loading && !loadError && searchQuery.trim() && !filteredChannels.length"
              variant="ghost"
              size="sm"
              type="button"
              @click="searchQuery = ''"
            >
              {{ copy('清除搜索', 'Clear search') }}
            </UiButton>
            <UiButton
              variant="secondary"
              size="md"
              :disabled="loading"
              :title="t('common.refresh', 'Refresh')"
              @click="loadChannels"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh', 'Refresh') }}</span>
            </UiButton>
          </div>
        </div>
      </template>

      <template #table>
        <PortalDataState v-if="loadError" :error="loadError" :retry-label="copy('重试', 'Retry')" @retry="loadChannels" />
        <PortalDataState v-if="ratesError" :error="ratesError" :retry-label="copy('刷新价格', 'Refresh pricing')" @retry="loadChannels" />
        <AvailableChannelsTable
          v-if="!loadError || channels.length"
          :columns="columnLabels"
          :rows="filteredChannels"
          :search-query="searchQuery"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="searchQuery.trim() ? copy('没有符合搜索条件的渠道，请调整或清除搜索。', 'No channels match this search. Change or clear the search.') : t('availableChannels.empty')"
        />
      </template>
    </TablePageLayout>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import PortalDataState from '@/components/portal/PortalDataState.vue'
import { userFacingError } from '@/utils/userFacingError'
import { UiButton } from '@/ui'

const { t, locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const loadError = ref('')
const ratesError = ref('')

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')

const columnLabels = computed(() => ({
  name: t('availableChannels.columns.name'),
  description: t('availableChannels.columns.description'),
  platform: t('availableChannels.columns.platform'),
  groups: t('availableChannels.columns.groups'),
  supportedModels: t('availableChannels.columns.supportedModels'),
}))

const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return channels.value
  return channels.value
    .map((ch) => {
      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      if (nameHit || descHit) return ch
      const matchingSections = ch.platforms.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

async function loadChannels() {
  loading.value = true
  loadError.value = ''
  ratesError.value = ''
  try {
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        ratesError.value = userFacingError(err, locale.value, copy('个人价格暂未加载，当前价格仅供参考。', 'Your account pricing is unavailable; displayed prices are a preview.'))
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    loadError.value = userFacingError(err, locale.value, copy('接入信息加载失败，请重试。', 'Access information could not be loaded. Please retry.'))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>
