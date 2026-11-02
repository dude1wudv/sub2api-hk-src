<template>
  <div class="portal-page user-service-page">
    <PortalPageHeader :title="t('nav.availableChannels')" />
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-80">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="input pl-10"
              />
            </div>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              @click="loadChannels"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh', 'Refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <PortalDataState v-if="loadError" :error="loadError" :retry-label="copy('重试', 'Retry')" @retry="loadChannels" />
        <PortalDataState v-if="ratesError" :error="ratesError" :retry-label="copy('刷新价格', 'Refresh pricing')" @retry="loadChannels" />
        <button v-if="!loading && !loadError && searchQuery.trim() && !filteredChannels.length" type="button" class="portal-inline-button" @click="searchQuery = ''">{{ copy('清除搜索', 'Clear search') }}</button>
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
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import PortalDataState from '@/components/portal/PortalDataState.vue'
import { userFacingError } from '@/utils/userFacingError'

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

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
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
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
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
