<template>
  <div v-if="loading || status?.enabled" class="overflow-hidden rounded-2xl border border-amber-200 bg-gradient-to-br from-white via-amber-50/50 to-white text-slate-900 shadow-lg shadow-amber-200/30 dark:border-amber-500/25 dark:bg-slate-950 dark:bg-none dark:text-slate-100 dark:shadow-amber-900/10">
    <div class="border-b border-amber-200 p-5 dark:border-amber-500/20">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="flex items-center gap-2">
            <span class="rounded-lg bg-amber-100 px-2 py-1 text-amber-700 dark:bg-amber-400/10 dark:text-amber-300">🎁</span>
            <h2 class="text-lg font-semibold">Token 激励计划</h2>
            <span v-if="claimableBalance > 0" class="rounded-full bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">可领取</span>
          </div>
          <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">
            本周期累计消耗 Token 达到档位后，可领取对应奖励余额。奖励需在本周期内领取，过期不可补领。
          </p>
        </div>
        <button type="button" class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-700 hover:bg-white dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800" @click="$emit('refresh')">
          刷新
        </button>
      </div>
    </div>

    <div v-if="loading" class="p-5 text-sm text-slate-500 dark:text-slate-400">加载激励进度中...</div>

    <div v-else-if="status" class="space-y-5 p-5">
      <div class="grid gap-4 md:grid-cols-3">
        <div>
          <p class="text-xs text-slate-500 dark:text-slate-400">本周期已消耗</p>
          <p class="mt-1 text-2xl font-bold">{{ formatTokens(status.total_tokens) }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-500 dark:text-slate-400">当前目标</p>
          <p class="mt-1 text-2xl font-bold">{{ formatTokens(status.next_threshold_tokens) }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-500 dark:text-slate-400">可领余额</p>
          <p class="mt-1 text-2xl font-bold text-amber-600 dark:text-amber-300">¥{{ claimableBalance.toFixed(2) }}</p>
        </div>
      </div>

      <div>
        <div class="mb-2 flex items-center justify-between text-sm text-slate-600 dark:text-slate-300">
          <span>本期进度</span>
          <span v-if="status.remaining_tokens > 0">还差 {{ formatTokens(status.remaining_tokens) }}</span>
          <span v-else>已达成全部目标</span>
        </div>
        <div class="h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
          <div class="h-full rounded-full bg-gradient-to-r from-amber-500 to-emerald-400" :style="{ width: progressPercent + '%' }"></div>
        </div>
        <p class="mt-2 text-xs text-slate-500 dark:text-slate-500">周期结束：{{ formatDateTime(status.period_end) }}</p>
      </div>

      <div class="grid gap-3 md:grid-cols-3 xl:grid-cols-7">
        <div v-for="tier in status.tiers" :key="tier.threshold_tokens" class="rounded-xl border p-3" :class="tierClass(tier.status)">
          <div class="flex items-center justify-between gap-2">
            <span class="text-sm text-slate-600 dark:text-slate-300">{{ formatTokens(tier.threshold_tokens) }}</span>
            <span class="text-xs" :class="statusClass(tier.status)">{{ statusText(tier.status) }}</span>
          </div>
          <p class="mt-3 text-xl font-semibold">¥{{ tier.reward_balance.toFixed(2) }}</p>
          <button
            v-if="tier.status === 'claimable'"
            type="button"
            class="mt-3 w-full rounded-lg bg-amber-400 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-amber-300 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="claimingThreshold === tier.threshold_tokens"
            @click="$emit('claim', tier.threshold_tokens)"
          >
            {{ claimingThreshold === tier.threshold_tokens ? '领取中...' : '领取' }}
          </button>
          <p v-else-if="tier.status === 'locked'" class="mt-3 text-xs text-slate-500 dark:text-slate-500">还差 {{ formatTokens(tier.remaining_tokens || 0) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TokenIncentiveStatus, TokenIncentiveTierStatus } from '@/api/usage'

const props = defineProps<{
  status: TokenIncentiveStatus | null
  loading: boolean
  claimingThreshold: number | null
}>()

defineEmits<{
  refresh: []
  claim: [thresholdTokens: number]
}>()

const claimableBalance = computed(() => props.status?.claimable_balance ?? 0)
const maxThreshold = computed(() => props.status?.tiers.at(-1)?.threshold_tokens || 1_000_000_000)
const progressPercent = computed(() => {
  const total = props.status?.total_tokens ?? 0
  return Math.min(100, Math.max(0, (total / maxThreshold.value) * 100)).toFixed(2)
})

const formatTokens = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(2) + 'B'
  if (value >= 1e6) return (value / 1e6).toFixed(2) + 'M'
  if (value >= 1e3) return (value / 1e3).toFixed(2) + 'K'
  return value.toLocaleString()
}

const formatDateTime = (value: string) => new Date(value).toLocaleString()

const statusText = (status: TokenIncentiveTierStatus) => ({
  locked: '未达成',
  claimable: '可领取',
  claimed: '已领取',
  expired: '已过期',
}[status])

const statusClass = (status: TokenIncentiveTierStatus) => ({
  locked: 'text-slate-500',
  claimable: 'text-amber-600 dark:text-amber-300',
  claimed: 'text-emerald-600 dark:text-emerald-300',
  expired: 'text-rose-600 dark:text-rose-300',
}[status])

const tierClass = (status: TokenIncentiveTierStatus) => ({
  locked: 'border-slate-200 bg-white/70 dark:border-slate-700 dark:bg-slate-900/60',
  claimable: 'border-amber-300 bg-amber-100/70 dark:border-amber-400/60 dark:bg-amber-500/10',
  claimed: 'border-emerald-200 bg-emerald-100/60 dark:border-emerald-400/40 dark:bg-emerald-500/10',
  expired: 'border-rose-200 bg-rose-100/60 dark:border-rose-400/40 dark:bg-rose-500/10',
}[status])
</script>
