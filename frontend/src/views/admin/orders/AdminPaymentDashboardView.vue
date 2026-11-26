<template>
  <div class="admin-page admin-page--analysis" data-admin-page="payment-dashboard">
    <div class="space-y-6">
      <!-- Header with Day Switcher -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <span class="text-xs font-medium uppercase tracking-wider text-[var(--atelier-ink-muted)]">{{ t('admin.dashboard.timeRange') }}</span>
        <div class="flex items-center gap-2">
          <UiSegmented
            v-model="days"
            :options="daysSegmentedOptions"
            size="sm"
            :aria-label="t('admin.dashboard.timeRange')"
          />
          <UiButton variant="secondary" size="sm" :disabled="loading" :title="t('common.refresh')" @click="loadDashboard">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </UiButton>
        </div>
      </div>

      <!-- Dashboard Content -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>
      <template v-else-if="stats">
        <UiMetricRibbon
          :primary-label="t('payment.admin.totalRevenue')"
          :primary-value="formatCurrencySummary(stats.total_amount)"
          :primary-meta="`${stats.total_count} ${t('payment.admin.orders')}`"
          :metrics="ribbonMetrics"
        />

        <DailyRevenueChart :data="stats.daily_series || []" :loading="loading" />

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div class="atelier-surface p-5 lg:col-span-5">
            <h3 class="mb-4 text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">{{ t('payment.admin.paymentDistribution') }}</h3>
            <div v-if="!stats.payment_methods?.length" class="flex h-32 items-center justify-center text-sm text-[var(--atelier-ink-muted)]">{{ t('payment.admin.noData') }}</div>
            <div v-else class="divide-y divide-[var(--atelier-border-subtle)]">
              <div v-for="method in stats.payment_methods" :key="method.type" class="flex items-center justify-between py-3 first:pt-0 last:pb-0">
                <div class="flex items-center gap-2.5">
                  <UiStatusDot :tone="methodTone(method.type)" />
                  <span class="text-sm text-[var(--atelier-ink)]">{{ t('payment.methods.' + method.type, method.type) }}</span>
                </div>
                <div class="text-right">
                  <span v-for="[currency, amount] in sortedAmounts(method.amount)" :key="currency" class="block font-mono text-sm font-medium tabular-nums text-[var(--atelier-ink)]">{{ formatMoney(currency, amount) }}</span>
                  <span class="font-mono text-xs tabular-nums text-[var(--atelier-ink-muted)]">({{ method.count }})</span>
                </div>
              </div>
            </div>
          </div>

          <div class="atelier-surface p-5 lg:col-span-7">
            <h3 class="mb-4 text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">{{ t('payment.admin.topUsers') }}</h3>
            <div v-if="!hasTopUsers(stats.top_users)" class="flex h-32 items-center justify-center text-sm text-[var(--atelier-ink-muted)]">{{ t('payment.admin.noData') }}</div>
            <div v-else class="space-y-4">
              <div v-for="[currency, users] in sortedTopUsers(stats.top_users)" :key="currency" class="space-y-1.5">
                <p class="font-mono text-xs font-medium uppercase tracking-wider text-[var(--atelier-ink-muted)]">{{ currency }}</p>
                <div class="divide-y divide-[var(--atelier-border-subtle)]">
                  <div v-for="(user, idx) in users" :key="user.user_id" class="flex items-center justify-between rounded-lg px-2.5 py-2 transition-colors hover:bg-[var(--atelier-hover)]">
                    <div class="flex min-w-0 items-center gap-3">
                      <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-md font-mono text-xs font-semibold tabular-nums" :class="rankClass(idx)">{{ idx + 1 }}</span>
                      <span class="truncate text-sm text-[var(--atelier-ink)]">{{ user.email }}</span>
                    </div>
                    <span class="ml-3 shrink-0 font-mono text-sm font-medium tabular-nums text-[var(--atelier-ink)]">{{ formatMoney(currency, user.amount) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { CurrencyAmounts, DashboardStats, TopUserPaymentStats } from '@/types/payment'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import DailyRevenueChart from '@/components/admin/payment/DailyRevenueChart.vue'
import { UiButton, UiMetricRibbon, UiSegmented, UiStatusDot, type RibbonMetric, type SegmentedOption } from '@/ui'

const { t } = useI18n()
const appStore = useAppStore()

const DAYS_OPTIONS = [7, 30, 90] as const
const days = ref<number>(30)
const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

const daysSegmentedOptions = computed<SegmentedOption<number>[]>(() =>
  DAYS_OPTIONS.map((d) => ({ value: d, label: `${d}${t('payment.admin.daySuffix')}` }))
)

const ribbonMetrics = computed<RibbonMetric[]>(() => {
  if (!stats.value) return []
  return [
    {
      label: t('payment.admin.todayRevenue'),
      value: formatCurrencySummary(stats.value.today_amount),
      sub: `${stats.value.today_count} ${t('payment.admin.orders')}`,
      dot: 'ok',
    },
    {
      label: t('payment.admin.todayOrders'),
      value: String(stats.value.today_count),
      dot: 'info',
    },
    {
      label: t('payment.admin.avgAmount'),
      value: formatCurrencySummary(stats.value.avg_amount),
    },
  ]
})

function methodTone(type: string): 'ok' | 'info' | 'warn' | 'neutral' {
  if (type === 'wxpay' || type === 'wxpay_direct') return 'ok'
  if (type === 'alipay' || type === 'alipay_direct' || type === 'alipay_manual') return 'info'
  if (type === 'stripe' || type === 'airwallex') return 'warn'
  return 'neutral'
}

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-[var(--atelier-ink)] text-[var(--atelier-ink-inverse)]'
  return 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-secondary)]'
}

function sortedAmounts(amounts: CurrencyAmounts): [string, number][] {
  return Object.entries(amounts).sort(([left], [right]) => left.localeCompare(right))
}

function formatCurrencySummary(amounts: CurrencyAmounts): string {
  const entries = sortedAmounts(amounts)
  if (!entries.length) return formatMoney('USD', 0)
  return entries.map(([currency, amount]) => formatMoney(currency, amount)).join(' · ')
}

function sortedTopUsers(usersByCurrency: Record<string, TopUserPaymentStats[]>): [string, TopUserPaymentStats[]][] {
  return Object.entries(usersByCurrency).sort(([left], [right]) => left.localeCompare(right))
}

function hasTopUsers(usersByCurrency: Record<string, TopUserPaymentStats[]>): boolean {
  return Object.values(usersByCurrency).some(users => users.length > 0)
}

function formatMoney(currency: string, amount: number): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(amount)
}

async function loadDashboard() {
  loading.value = true
  try {
    const res = await adminPaymentAPI.getDashboard(days.value)
    stats.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

watch(days, () => loadDashboard())
onMounted(() => loadDashboard())
</script>
