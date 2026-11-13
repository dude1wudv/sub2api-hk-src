<template>
  <BaseDialog
    :show="show"
    :title="title"
    width="wide"
    @close="$emit('close')"
  >
    <div v-if="loading" class="py-8 text-center text-sm text-gray-500">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="!detail" class="py-8 text-center text-sm text-gray-500">
      {{ t('channelStatus.detailLoadError') }}
    </div>
    <div v-else class="overflow-x-auto">
      <div v-if="mobile" class="mobile-records">
          <dl v-for="m in detail.models"
            :key="m.model" class="mobile-record">
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.model') }}</dt><dd>{{ formatMonitorModel(m.model) }}</dd></div>
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.latestStatus') }}</dt><dd><span
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px]"
                :class="statusBadgeClass(m.latest_status)"
              >
                {{ statusLabel(m.latest_status) }}
              </span></dd></div>
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.latestLatency') }}</dt><dd>{{ formatLatency(m.latest_latency_ms) }}</dd></div>
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.availability7d') }}</dt><dd>{{ formatPercent(m.availability_7d) }}</dd></div>
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.availability15d') }}</dt><dd>{{ formatPercent(m.availability_15d) }}</dd></div>
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.availability30d') }}</dt><dd>{{ formatPercent(m.availability_30d) }}</dd></div>
  <div class="mobile-record-field"><dt>{{ t('channelStatus.detailColumns.avgLatency7d') }}</dt><dd>{{ formatLatency(m.avg_latency_7d_ms) }}</dd></div>
</dl>

</div>
<table v-else class="w-full text-left text-sm">
        <thead class="border-b border-gray-200 dark:border-dark-700">
          <tr class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.model') }}</th>
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.latestStatus') }}</th>
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.latestLatency') }}</th>
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.availability7d') }}</th>
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.availability15d') }}</th>
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.availability30d') }}</th>
            <th class="py-2 pr-3">{{ t('channelStatus.detailColumns.avgLatency7d') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="m in detail.models"
            :key="m.model"
            class="border-b border-gray-100 dark:border-dark-800"
          >
            <td class="py-2 pr-3 font-medium text-gray-900 dark:text-gray-100">{{ formatMonitorModel(m.model) }}</td>
            <td class="py-2 pr-3">
              <span
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px]"
                :class="statusBadgeClass(m.latest_status)"
              >
                {{ statusLabel(m.latest_status) }}
              </span>
            </td>
            <td class="py-2 pr-3 text-gray-700 dark:text-gray-300">{{ formatLatency(m.latest_latency_ms) }}</td>
            <td class="py-2 pr-3 text-gray-700 dark:text-gray-300">{{ formatPercent(m.availability_7d) }}</td>
            <td class="py-2 pr-3 text-gray-700 dark:text-gray-300">{{ formatPercent(m.availability_15d) }}</td>
            <td class="py-2 pr-3 text-gray-700 dark:text-gray-300">{{ formatPercent(m.availability_30d) }}</td>
            <td class="py-2 pr-3 text-gray-700 dark:text-gray-300">{{ formatLatency(m.avg_latency_7d_ms) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-secondary">
          {{ t('channelStatus.closeDetail') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  status as fetchChannelMonitorDetail,
  type UserMonitorDetail,
} from '@/api/channelMonitor'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const props = defineProps<{
  show: boolean
  monitorId: number | null
  title: string
}>()

defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { statusLabel, statusBadgeClass, formatLatency, formatPercent, formatMonitorModel } = useChannelMonitorFormat()

const detail = ref<UserMonitorDetail | null>(null)
const loading = ref(false)
let requestVersion = 0

async function load(id: number) {
  const version = ++requestVersion
  detail.value = null
  loading.value = true
  try {
    const result = await fetchChannelMonitorDetail(id)
    if (version === requestVersion) detail.value = result
  } catch (err: unknown) {
    if (version !== requestVersion) return
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.detailLoadError')))
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

watch(
  () => [props.show, props.monitorId] as const,
  ([show, id], _, onCleanup) => {
    onCleanup(() => { requestVersion++ })
    if (!show) {
      detail.value = null
      return
    }
    if (id != null) void load(id)
  },
  { immediate: true },
)
</script>
