<template>
  <div class="portal-page developer-dashboard">
    <MdPageHeader
      :eyebrow="zh ? '开发者工作台' : 'Developer Workspace'"
      :title="zh ? '工作台概览' : 'Workspace Overview'"
      :description="zh ? '实时掌握账户余额续航、平台配额水位、主力模型分布与调用趋势。' : 'Monitor balance runway, platform quota watermarks, model mix, and live usage trends.'"
    >
      <template #badge>
        <MdBadge tone="neutral">{{ new Date().toLocaleDateString(locale, { month: 'short', day: 'numeric', weekday: 'short' }) }}</MdBadge>
      </template>
      <template #actions>
        <MdButton variant="outlined" size="sm" :loading="loading" @click="refreshAll">
          <template #icon><Icon name="refresh" size="xs" /></template>
          {{ t('common.refresh') }}
        </MdButton>
        <MdButton variant="tonal" size="sm" @click="router.push('/playground')">
          <template #icon><Icon name="play" size="xs" /></template>
          {{ zh ? 'API 调试台' : 'Playground' }}
        </MdButton>
        <MdButton variant="filled" size="sm" @click="router.push('/keys?create=1')">
          <template #icon><Icon name="plus" size="xs" /></template>
          {{ zh ? '创建密钥' : 'Create key' }}
        </MdButton>
      </template>
    </MdPageHeader>

    <!-- Quick Connect Bar -->
    <div v-if="openAIBase" class="dashboard-quick-connect">
      <div class="dashboard-qc-endpoint">
        <span class="dashboard-qc-label">Base URL</span>
        <code>{{ openAIBase }}</code>
      </div>
      <div class="dashboard-qc-actions">
        <MdButton variant="text" size="sm" @click="copyBaseUrl">
          <template #icon><Icon name="copy" size="xs" /></template>
          {{ zh ? '复制地址' : 'Copy URL' }}
        </MdButton>
        <MdButton variant="text" size="sm" @click="router.push('/keys?quickstart=1')">
          <template #icon><Icon name="bolt" size="xs" /></template>
          {{ zh ? '一键接入配置' : 'Quickstart' }}
        </MdButton>
      </div>
    </div>

    <div v-if="error" role="alert" class="portal-error workspace-error"><p>{{ error }}</p><button class="portal-button secondary" :disabled="loading" @click="refreshAll">{{ zh ? '重试' : 'Retry' }}</button></div>
    <div v-if="loading && !stats" class="portal-panel flex items-center justify-center py-12" role="status"><LoadingSpinner /></div>
    <template v-if="stats">
      <UserOnboardingChecklist v-if="stats.total_requests === 0" :key-count="stats.total_api_keys" />
      <template v-else>
      <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" :platform-quotas="platformQuotas" />
      <UsageInsightsCard
        :balance="user?.balance || 0"
        :stats="stats"
        :trend="trendData"
        :models="modelStats"
        :is-simple="authStore.isSimpleMode"
      />
      <section class="workspace-activity">
        <header class="workspace-section-title"><div><h2>{{ zh ? '用量统计' : 'Usage statistics' }}</h2></div><RouterLink to="/usage" class="portal-inline-button">{{ zh ? '用量明细' : 'Usage details' }} <Icon name="arrowRight" size="sm" /></RouterLink></header>
        <p v-if="chartError" class="portal-error" role="alert">{{ chartError }} <button @click="loadCharts">{{ zh ? '重试' : 'Retry' }}</button></p>
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" @refresh="loadCharts" @drilldown="drilldown" />
      </section>
      <div class="workspace-bottom-grid">
        <div class="min-w-0"><p v-if="recentError" class="portal-error" role="alert">{{ recentError }} <button @click="loadRecent">{{ zh ? '重试' : 'Retry' }}</button></p><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
        <UserDashboardQuickActions />
      </div>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { MdBadge, MdButton, MdPageHeader } from '@/components/md'
import { usageDrilldownQuery, type UsageDrilldown } from '@/views/portal/usageRoute'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserOnboardingChecklist from '@/components/user/dashboard/UserOnboardingChecklist.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UsageInsightsCard from '@/components/user/dashboard/UsageInsightsCard.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useClipboard } from '@/composables/useClipboard'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { formatDateLocalInput } from '@/utils/format'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const authStore = useAuthStore()
const appStore = useAppStore()
const router = useRouter()
const { openAIBase } = usePatrickApiBase()
const { copyToClipboard } = useClipboard()
async function copyBaseUrl() {
  if (openAIBase.value && (await copyToClipboard(openAIBase.value))) {
    appStore.showSuccess(zh.value ? '已复制 API Base URL' : 'API Base URL copied')
  }
}
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
let disposed = false
const chartError = ref('')
const recentError = ref('')
const today = formatDateLocalInput(new Date())
function drilldown(value: UsageDrilldown) { void router.push({ path: '/usage', query: usageDrilldownQuery(value) }) }

async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    await authStore.refreshUser()
    const data = await usageAPI.getDashboardStats()
    if (!disposed) stats.value = data
  } catch (cause) {
    error.value = zh.value ? '暂时无法获取账户概览，请重试。' : 'Unable to load your overview. Please try again.'
    console.error('Failed to load dashboard stats:', cause)
  } finally { loading.value = false }
}
async function loadCharts() {
  const request = ++chartRequest
  loadingCharts.value = true
  chartError.value = ''
  try {
    const [trend, models] = await Promise.all([
      usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as 'day' | 'hour' }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })
    ])
    if (disposed || request !== chartRequest) return
    trendData.value = trend.trend || []
    modelStats.value = models.models || []
  } catch (cause) {
    if (!disposed && request === chartRequest) chartError.value = zh.value ? '趋势暂时无法加载，请重试。' : 'Unable to load trends. Please retry.'
    console.error('Failed to load charts:', cause)
  } finally { if (request === chartRequest) loadingCharts.value = false }
}
async function loadRecent() {
  loadingUsage.value = true
  recentError.value = ''
  try {
    const res = await usageAPI.getByDateRange(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)), today)
    if (!disposed) recentUsage.value = res.items.slice(0, 5)
  } catch (cause) { recentError.value = zh.value ? '最近请求加载失败，请重试。' : 'Unable to load recent requests. Please retry.'; console.error('Failed to load recent usage:', cause) }
  finally { loadingUsage.value = false }
}
async function loadPlatformQuotas() {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (cause) {
    console.warn('Failed to load platform quotas:', cause)
    platformQuotas.value = null
  }
}
function refreshAll() {
  void loadStats()
  void loadCharts()
  void loadRecent()
  void loadPlatformQuotas()
}
onMounted(refreshAll)
onBeforeUnmount(() => { disposed = true; chartRequest++ })
</script>

<style scoped>
.dashboard-quick-connect {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 18px;
  padding: 10px 14px;
  border-radius: var(--md-sys-shape-corner-large);
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
}

.dashboard-qc-endpoint {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.dashboard-qc-label {
  padding: 2px 8px;
  border-radius: 9999px;
  font-size: 0.72rem;
  font-weight: 600;
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
}

.dashboard-qc-endpoint code {
  font-family: var(--font-mono);
  font-size: 0.82rem;
  color: var(--md-sys-color-on-surface);
  word-break: break-all;
}

.dashboard-qc-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
}
</style>
