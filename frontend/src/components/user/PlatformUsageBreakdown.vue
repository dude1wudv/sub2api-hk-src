<template>
  <div class="text-sm tabular-nums">
    <div class="flex flex-wrap items-center gap-1.5">
      <span class="text-gray-500 dark:text-dark-400">{{ t('admin.users.today') }}:</span>
      <span class="font-medium text-gray-900 dark:text-white">${{ today.toFixed(4) }}</span>
      <HelpTooltip
        v-if="hasBreakdown"
        trigger="both"
        width-class="w-80"
        class="!ml-0"
        :label="t('admin.users.platformBreakdown')"
      >
        <template #default="{ open }">
          <div v-if="open">
            <div class="mb-1.5 flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 pb-1 text-[11px] opacity-80 dark:border-gray-700">
              <span>{{ t('admin.users.platformBreakdown') }}</span>
              <span class="font-mono tabular-nums">{{ t('admin.users.today') }} / {{ t('admin.users.total') }}</span>
            </div>
            <div
              v-for="item in sortedBreakdown"
              :key="item.platform"
              class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 py-0.5"
              :class="{ 'opacity-70 italic': item.isOther }"
            >
              <span class="capitalize">
                {{ item.isOther ? t('admin.users.platformOther') : platformLabel(item.platform) }}
              </span>
              <span class="min-w-0 font-mono tabular-nums [overflow-wrap:anywhere]">
                ${{ item.today_actual_cost.toFixed(4) }}
                <span class="opacity-50">/</span>
                ${{ item.total_actual_cost.toFixed(4) }}
              </span>
            </div>
          </div>
        </template>
      </HelpTooltip>
    </div>
    <div class="mt-0.5 flex flex-wrap items-center gap-1.5">
      <span class="text-gray-500 dark:text-dark-400">{{ t('admin.users.total') }}:</span>
      <span class="font-medium text-gray-900 dark:text-white">${{ total.toFixed(4) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import type { PlatformUsage } from '@/api/admin/dashboard'

const props = defineProps<{
  today: number
  total: number
  byPlatform?: PlatformUsage[]
}>()

const { t } = useI18n()

// 与 UserDashboardStats 保持一致：把"总值 - 各平台之和"的差作为"其他"行展示，
// 避免 tooltip 内各平台费用加总与列首总值对不上。
const OTHER_THRESHOLD = 0.0001

interface BreakdownRow {
  platform: string
  today_actual_cost: number
  total_actual_cost: number
  isOther?: boolean
}

const sortedBreakdown = computed<BreakdownRow[]>(() => {
  const list = props.byPlatform ?? []
  const rows: BreakdownRow[] = [...list]
    .sort((a, b) => b.total_actual_cost - a.total_actual_cost)
    .map((p) => ({ ...p }))

  const sumTotal = rows.reduce((s, r) => s + r.total_actual_cost, 0)
  const sumToday = rows.reduce((s, r) => s + r.today_actual_cost, 0)
  const diffTotal = Math.max(0, props.total - sumTotal)
  const diffToday = Math.max(0, props.today - sumToday)
  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    rows.push({
      platform: '__other__',
      today_actual_cost: diffToday,
      total_actual_cost: diffTotal,
      isOther: true
    })
  }
  return rows
})

const hasBreakdown = computed(() => sortedBreakdown.value.length > 0)

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
}

function platformLabel(platform: string): string {
  return PLATFORM_LABELS[platform] ?? platform
}
</script>
