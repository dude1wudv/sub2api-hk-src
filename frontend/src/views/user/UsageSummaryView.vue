<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('usageSummary.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('usageSummary.description') }}
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <router-link to="/usage" class="btn btn-secondary">
            <Icon name="document" size="sm" class="mr-2" />
            {{ t('usageSummary.viewRecords') }}
          </router-link>
          <button class="btn btn-primary" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="sm" class="mr-2" />
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>

      <div
        v-if="errorInfo"
        class="card border-red-200 bg-red-50 p-5 dark:border-red-800/50 dark:bg-red-900/20"
      >
        <div class="flex items-start gap-3">
          <Icon name="exclamationCircle" class="mt-0.5 text-red-600 dark:text-red-400" />
          <div>
            <h2 class="text-sm font-semibold text-red-800 dark:text-red-300">
              {{ errorInfo.title }}
            </h2>
            <p class="mt-1 text-sm text-red-700 dark:text-red-400">
              {{ errorInfo.description }}
            </p>
            <p class="mt-2 text-xs font-medium text-red-700 dark:text-red-300">
              {{ errorInfo.action }}
            </p>
          </div>
        </div>
      </div>

      <section class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="card p-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.balance') }}
              </p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                ${{ formatMoney(user?.balance ?? 0) }}
              </p>
            </div>
            <div class="rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30">
              <Icon name="dollar" class="text-emerald-600 dark:text-emerald-400" />
            </div>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">
            {{ t('usageSummary.walletHint') }}
          </p>
        </div>

        <div class="card p-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.todayCost') }}
              </p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                ${{ formatMoney(stats?.today_actual_cost ?? 0) }}
              </p>
            </div>
            <div class="rounded-lg bg-sky-100 p-2 dark:bg-sky-900/30">
              <Icon name="chart" class="text-sky-600 dark:text-sky-400" />
            </div>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">
            {{ formatNumber(stats?.today_requests ?? 0) }} {{ t('usageSummary.requests') }}
          </p>
        </div>

        <div class="card p-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.todayTokens') }}
              </p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                {{ formatCompact(stats?.today_tokens ?? 0) }}
              </p>
            </div>
            <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
              <Icon name="cube" class="text-amber-600 dark:text-amber-400" />
            </div>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">
            {{ t('usageSummary.inputOutput', {
              input: formatCompact(stats?.today_input_tokens ?? 0),
              output: formatCompact(stats?.today_output_tokens ?? 0)
            }) }}
          </p>
        </div>

        <div class="card p-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.realtime') }}
              </p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                {{ formatNumber(stats?.rpm ?? 0) }} / {{ formatCompact(stats?.tpm ?? 0) }}
              </p>
            </div>
            <div class="rounded-lg bg-violet-100 p-2 dark:bg-violet-900/30">
              <Icon name="bolt" class="text-violet-600 dark:text-violet-400" />
            </div>
          </div>
          <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">
            RPM / TPM
          </p>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div class="card p-6 xl:col-span-2">
          <div class="flex items-center justify-between gap-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('usageSummary.trend') }}
              </h2>
              <p class="text-sm text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.lastSevenDays') }}
              </p>
            </div>
          </div>
          <div class="mt-6 space-y-4">
            <div v-for="point in trendBars" :key="point.date" class="grid grid-cols-[88px_1fr_92px] items-center gap-3">
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ point.date }}</span>
              <div class="h-3 overflow-hidden rounded bg-gray-100 dark:bg-dark-800">
                <div
                  class="h-full rounded bg-primary-500"
                  :style="{ width: point.width + '%' }"
                ></div>
              </div>
              <span class="text-right text-xs font-medium text-gray-700 dark:text-dark-200">
                ${{ formatMoney(point.actual_cost) }}
              </span>
            </div>
            <div v-if="!loading && trendBars.length === 0" class="empty-state py-8">
              <Icon name="chart" size="xl" class="mb-3 text-gray-400" />
              <p class="text-sm text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.noUsageYet') }}
              </p>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('usageSummary.entitlements') }}
          </h2>
          <div class="mt-5 space-y-4">
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.apiKeys') }}
              </p>
              <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
                {{ stats?.active_api_keys ?? 0 }} / {{ stats?.total_api_keys ?? 0 }}
              </p>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.concurrency') }}
              </p>
              <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
                {{ user?.concurrency ?? 0 }}
              </p>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.activeSubscriptions') }}
              </p>
              <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
                {{ subscriptionSummary?.active_count ?? 0 }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('usageSummary.modelSpend') }}
            </h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="model in topModels" :key="model.model" class="flex items-center justify-between gap-4 px-6 py-4">
              <div class="min-w-0">
                <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ model.model }}</p>
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ formatNumber(model.requests) }} {{ t('usageSummary.requests') }} · {{ formatCompact(model.total_tokens) }} tokens
                </p>
              </div>
              <span class="text-sm font-semibold text-gray-900 dark:text-white">
                ${{ formatMoney(model.actual_cost) }}
              </span>
            </div>
            <div v-if="!loading && topModels.length === 0" class="empty-state py-8">
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('usageSummary.noUsageYet') }}</p>
            </div>
          </div>
        </div>

        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('usageSummary.activeSubscriptionList') }}
            </h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="sub in subscriptionSummary?.subscriptions ?? []"
              :key="sub.id"
              class="px-6 py-4"
            >
              <div class="flex items-center justify-between gap-4">
                <p class="text-sm font-medium text-gray-900 dark:text-white">{{ sub.group_name }}</p>
                <span class="rounded bg-emerald-100 px-2 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                  {{ sub.days_remaining ?? '-' }} {{ t('usageSummary.daysLeft') }}
                </span>
              </div>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('usageSummary.expiresAt') }} {{ formatDate(sub.expires_at) }}
              </p>
            </div>
            <div v-if="!loading && !subscriptionSummary?.subscriptions?.length" class="empty-state py-8">
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('usageSummary.noActiveSubscriptions') }}</p>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI, type ModelStatsResponse, type TrendResponse, type UserDashboardStats } from '@/api/usage'
import subscriptionAPI, { type SubscriptionSummary } from '@/api/subscriptions'
import { useAuthStore } from '@/stores/auth'
import { explainHumanApiError, type HumanApiError } from '@/utils/apiError'

const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(false)
const errorInfo = ref<HumanApiError | null>(null)
const stats = ref<UserDashboardStats | null>(null)
const trend = ref<TrendResponse | null>(null)
const models = ref<ModelStatsResponse | null>(null)
const subscriptionSummary = ref<SubscriptionSummary | null>(null)

const user = computed(() => authStore.user)

const formatMoney = (value: number): string => {
  return Number.isFinite(value) ? value.toFixed(4).replace(/0+$/, '').replace(/\.$/, '') : '0'
}

const formatNumber = (value: number): string => {
  return Math.round(value).toLocaleString()
}

const formatCompact = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return formatNumber(value)
}

const formatDate = (value?: string | null): string => {
  if (!value) return '-'
  return new Date(value).toLocaleString(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const todayRange = () => {
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - 6)
  const fmt = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  return { start_date: fmt(start), end_date: fmt(end) }
}

const trendBars = computed(() => {
  const rows = trend.value?.trend ?? []
  const max = Math.max(...rows.map((row) => row.actual_cost), 0)
  return rows.map((row) => ({
    ...row,
    width: max > 0 ? Math.max(4, Math.round((row.actual_cost / max) * 100)) : 0,
  }))
})

const topModels = computed(() => {
  return [...(models.value?.models ?? [])]
    .sort((a, b) => b.actual_cost - a.actual_cost)
    .slice(0, 6)
})

const loadAll = async () => {
  loading.value = true
  errorInfo.value = null
  try {
    const range = todayRange()
    const [statsData, trendData, modelData, subData] = await Promise.all([
      usageAPI.getDashboardStats(),
      usageAPI.getDashboardTrend({ ...range, granularity: 'day' }),
      usageAPI.getDashboardModels(range),
      subscriptionAPI.getSubscriptionSummary(),
    ])
    stats.value = statsData
    trend.value = trendData
    models.value = modelData
    subscriptionSummary.value = subData
  } catch (error) {
    errorInfo.value = explainHumanApiError(error, t('usageSummary.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadAll()
})
</script>
