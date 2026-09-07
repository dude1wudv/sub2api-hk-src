<template>
  <AppLayout>
    <div class="developer-dashboard space-y-6">
      <section class="workspace-hero">
        <div>
          <p class="hero-eyebrow">DEVELOPER WORKSPACE / API COCKPIT</p>
          <h2>{{ zh ? '你的下一次创造，从这里开始。' : 'Your next idea starts here.' }}</h2>
          <p class="hero-description">{{ zh ? '密钥、用量与模型，一目了然。把复杂留给平台，把专注留给创造。' : 'Your keys, usage and models, in one clear view. Less administration. More creation.' }}</p>
          <div class="hero-actions">
            <router-link to="/keys" class="btn btn-primary"><Icon name="key" size="sm" />{{ t('nav.apiKeys') }}</router-link>
            <router-link to="/usage" class="btn btn-secondary"><Icon name="chart" size="sm" />{{ t('dashboard.viewUsage') }}</router-link>
          </div>
        </div>
        <div class="hero-instrument">
          <span>{{ t('dashboard.todayRequests') }}</span>
          <strong>{{ stats ? stats.today_requests.toLocaleString() : '—' }}</strong>
          <div class="instrument-rule"></div>
          <span>{{ zh ? '每一次调用，都是新的可能' : 'Every call, a new possibility' }}</span>
        </div>
      </section>
      <div class="dashboard-section-heading">
        <h2>{{ zh ? '账户与用量概览' : 'Account & usage overview' }}</h2>
        <button class="btn btn-ghost btn-sm" :disabled="loading" @click="refreshAll"><Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />{{ t('common.refresh') }}</button>
      </div>
      <div v-if="error" role="alert" class="card flex flex-wrap items-center justify-between gap-3 p-5">
        <p class="text-sm text-err-600 dark:text-err-300">{{ error }}</p>
        <button class="btn btn-secondary" :disabled="loading" @click="refreshAll">{{ zh ? '重试' : 'Retry' }}</button>
      </div>
      <div v-if="loading && !stats" class="card flex items-center justify-center py-12" role="status"><LoadingSpinner /></div>
      <template v-if="stats">
        <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" :platform-quotas="platformQuotas" />
        <div class="dashboard-section-heading"><h2>{{ zh ? '了解你的 API 活动' : 'Understand your API activity' }}</h2><span>USAGE ANALYTICS</span></div>
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" @refresh="refreshAll" />
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="min-w-0 lg:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
          <div class="min-w-0 lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { formatDateLocalInput } from '@/utils/format'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const error = ref('')
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref('day')
let chartRequest = 0

async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (cause) {
    error.value = zh.value ? '暂时无法获取账户概览，请重试。' : 'Unable to load your overview. Please try again.'
    console.error('Failed to load dashboard stats:', cause)
  } finally { loading.value = false }
}
async function loadCharts() {
  const request = ++chartRequest
  loadingCharts.value = true
  try {
    const [trend, models] = await Promise.all([
      usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as 'day' | 'hour' }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })
    ])
    if (request !== chartRequest) return
    trendData.value = trend.trend || []
    modelStats.value = models.models || []
  } catch (cause) {
    console.error('Failed to load charts:', cause)
  } finally { if (request === chartRequest) loadingCharts.value = false }
}
async function loadRecent() {
  loadingUsage.value = true
  try {
    const res = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = res.items.slice(0, 5)
  } catch (cause) { console.error('Failed to load recent usage:', cause) }
  finally { loadingUsage.value = false }
}
async function loadPlatformQuotas() {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (cause) {
    console.warn('Failed to load platform quotas:', cause)
    platformQuotas.value = []
  }
}
function refreshAll() {
  void loadStats()
  void loadCharts()
  void loadRecent()
  void loadPlatformQuotas()
}
onMounted(refreshAll)
</script>
