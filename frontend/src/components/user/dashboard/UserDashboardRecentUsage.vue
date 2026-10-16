<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { formatDateTime, formatDateLocalInput } from '@/utils/format'
import { usageDrilldownQuery } from '@/views/portal/usageRoute'
import type { UsageLog } from '@/types'
defineProps<{ data: UsageLog[]; loading: boolean }>()
const { t, locale } = useI18n()
const tx = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const formatCost = (value: number) => value.toFixed(4)
function usageLocation(log: UsageLog) {
  const day = formatDateLocalInput(new Date(log.created_at))
  return { path: '/usage', query: usageDrilldownQuery({ start: day, end: day, model: log.model, key: log.api_key_id }) }
}
</script>

<template>
  <section class="workspace-recent">
    <header class="workspace-section-title"><div><h2>{{ t('dashboard.recentUsage') }}</h2></div><span class="portal-tag">{{ t('dashboard.last7Days') }}</span></header>
    <div v-if="loading" class="workspace-recent-loading" role="status"><LoadingSpinner /></div>
    <EmptyState v-else-if="!data.length" :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    <ol v-else class="workspace-request-list">
      <li v-for="log in data" :key="log.id"><RouterLink :to="usageLocation(log)" class="workspace-request">
        <span class="workspace-request-model-icon"><ModelIcon :model="log.model" size="22px" /></span>
        <span class="workspace-request-identity"><strong :title="log.model">{{ log.model }}</strong><small>{{ formatDateTime(log.created_at) }}</small></span>
        <span class="workspace-request-cost"><strong :title="t('dashboard.actual')">${{ formatCost(log.actual_cost) }}</strong><small>{{ (log.input_tokens + log.output_tokens).toLocaleString() }} tokens</small><small v-if="log.output_tokens_estimated" :title="t('usage.outputTokensEstimatedHint')">{{ t('usage.outputTokensEstimated') }}</small></span>
        <Icon name="arrowRight" size="sm" class="workspace-request-arrow" />
      </RouterLink></li>
    </ol>
    <RouterLink to="/usage" class="workspace-view-all">{{ tx('查看全部请求', 'View all requests') }}<Icon name="arrowRight" size="sm" /></RouterLink>
  </section>
</template>
