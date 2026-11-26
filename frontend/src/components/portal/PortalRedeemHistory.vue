<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { redeemAPI, type RedeemHistoryItem } from '@/api/redeem'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatPortalBalance, maskPortalRedeemCode } from '@/utils/portalPurchase'
import { UiButton, UiSelect, type UiSelectOption } from '@/ui'

const props = withDefaults(defineProps<{ compact?: boolean; refreshKey?: number }>(), { compact: false, refreshKey: 0 })
const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
type HistoryRow = Omit<RedeemHistoryItem, 'code'> & { maskedCode: string }
const rows = ref<HistoryRow[]>([])
const loading = ref(true)
const error = ref('')
const page = ref(1)
const pageSize = ref(props.compact ? 5 : 20)
const selectedPageSize = ref(pageSize.value)
const total = ref(0)
const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const pageSizeOptions: UiSelectOption[] = [
  { value: 20, label: '20' },
  { value: 50, label: '50' },
  { value: 100, label: '100' },
]
let request = 0
let failedTarget: { page: number; size: number } | null = null

function typeLabel(type: string) {
  const names: Record<string, [string, string]> = {
    balance: ['余额兑换', 'Balance redemption'],
    subscription: ['订阅兑换', 'Subscription redemption'],
    concurrency: ['并发额度兑换', 'Concurrency redemption'],
    admin_balance: ['余额调整', 'Balance adjustment'],
    admin_concurrency: ['并发额度调整', 'Concurrency adjustment'],
    invitation: ['邀请兑换', 'Invitation redemption']
  }
  const label = names[type]
  return label ? copy(...label) : copy('其他兑换', 'Other redemption')
}

function valueLabel(item: HistoryRow) {
  if (item.type === 'balance' || item.type === 'admin_balance') {
    return `${item.value > 0 ? '+' : ''}${formatPortalBalance(item.value, locale.value)}`
  }
  if (item.type === 'subscription') {
    const days = item.validity_days ?? item.value
    return Number.isFinite(days) && days > 0 ? copy(`${days} 天`, `${days} days`) : '—'
  }
  if (item.type === 'concurrency' || item.type === 'admin_concurrency') {
    return Number.isFinite(item.value) ? `${item.value > 0 ? '+' : ''}${item.value} ${copy('并发', 'concurrent requests')}` : '—'
  }
  return '—'
}

function formattedDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', { hour12: false })
}

async function load(targetPage = page.value, targetSize = pageSize.value) {
  const currentRequest = ++request
  loading.value = true
  error.value = ''
  failedTarget = null
  try {
    const result = await redeemAPI.getHistory(targetPage, targetSize)
    if (currentRequest !== request) return
    const resultTotal = Number.isFinite(result.total) ? Math.max(0, result.total) : 0
    const lastPage = Math.max(1, Math.ceil(resultTotal / targetSize))
    if (targetPage > lastPage) { await load(lastPage, targetSize); return }
    rows.value = (result.items || []).map(({ code, ...item }) => ({
      ...item,
      maskedCode: item.type.startsWith('admin_') ? '—' : maskPortalRedeemCode(code)
    }))
    total.value = resultTotal
    page.value = targetPage
    pageSize.value = targetSize
    selectedPageSize.value = targetSize
  } catch (cause) {
    if (currentRequest !== request) return
    selectedPageSize.value = pageSize.value
    failedTarget = { page: targetPage, size: targetSize }
    error.value = extractApiErrorMessage(cause, copy('兑换记录加载失败，请重试。', 'Could not load redemption history. Please retry.'))
  } finally {
    if (currentRequest === request) loading.value = false
  }
}

function handlePageSizeChange(val: string | number | boolean | null) {
  const next = Number(val) || 20
  selectedPageSize.value = next
  void load(1, next)
}

function retry() { void load(failedTarget?.page ?? page.value, failedTarget?.size ?? pageSize.value) }
watch(() => props.refreshKey, () => { void load(1, pageSize.value) })
onMounted(() => { void load(1, pageSize.value) })
onBeforeUnmount(() => { request++ })
</script>

<template>
  <section class="atelier-surface overflow-hidden" :aria-label="copy('本站兑换记录', 'Redemption history on this site')">
    <header class="flex items-center justify-between gap-4 border-b border-[var(--atelier-border)] px-5 py-4">
      <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
        {{ compact ? copy('最近兑换', 'Recent redemptions') : copy('本站兑换记录', 'Redemption history') }}
      </h2>
      <RouterLink
        v-if="compact"
        to="/orders"
        class="text-xs font-medium text-[var(--atelier-ink-secondary)] hover:text-[var(--atelier-ink)] hover:underline"
      >
        {{ copy('查看全部', 'View all') }} <span aria-hidden="true">→</span>
      </RouterLink>
      <UiButton v-else variant="secondary" size="sm" type="button" :disabled="loading" @click="load()">
        {{ copy('刷新', 'Refresh') }}
      </UiButton>
    </header>

    <div class="p-5">
      <div
        v-if="error"
        class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-xs text-[var(--atelier-err)]"
        role="alert"
      >
        <p class="m-0">{{ error }}</p>
        <UiButton variant="secondary" size="sm" type="button" :disabled="loading" @click="retry">
          {{ copy('重试', 'Retry') }}
        </UiButton>
      </div>

      <div class="overflow-x-auto" :aria-busy="loading">
        <div v-if="mobile" class="space-y-3">
          <div v-if="loading" class="py-10 text-center text-xs text-[var(--atelier-ink-muted)]" role="status">
            {{ copy('正在加载兑换记录…', 'Loading redemption history…') }}
          </div>
          <div v-else-if="!rows.length" class="py-10 text-center text-xs text-[var(--atelier-ink-muted)]">
            {{ error ? copy('记录暂时无法显示', 'History is temporarily unavailable') : copy('暂无兑换记录', 'No redemptions yet') }}
          </div>
          <dl
            v-for="item in loading ? [] : rows"
            :key="item.id"
            class="space-y-2 rounded-xl border border-[var(--atelier-border)] p-3.5 text-xs"
          >
            <div class="flex justify-between gap-2">
              <dt class="text-[var(--atelier-ink-muted)]">{{ copy('时间', 'Date') }}</dt>
              <dd class="text-right font-mono">
                <time :datetime="item.used_at || item.created_at">{{ formattedDate(item.used_at || item.created_at) }}</time>
                <span class="block text-[11px] text-[var(--atelier-ink-muted)]">#{{ item.id }}</span>
              </dd>
            </div>
            <div class="flex justify-between gap-2">
              <dt class="text-[var(--atelier-ink-muted)]">{{ copy('类型', 'Type') }}</dt>
              <dd class="text-right">
                {{ typeLabel(item.type) }}
                <span v-if="item.type === 'subscription' && item.group?.name" class="block text-[11px] text-[var(--atelier-ink-muted)]">
                  {{ item.group.name }}
                </span>
              </dd>
            </div>
            <div class="flex justify-between gap-2">
              <dt class="text-[var(--atelier-ink-muted)]">{{ copy('金额 / 额度', 'Amount / allowance') }}</dt>
              <dd class="font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">{{ valueLabel(item) }}</dd>
            </div>
            <div class="flex justify-between gap-2">
              <dt class="text-[var(--atelier-ink-muted)]">{{ copy('兑换码', 'Code') }}</dt>
              <dd class="font-mono text-[var(--atelier-ink-secondary)]">{{ item.maskedCode }}</dd>
            </div>
          </dl>
        </div>

        <table v-else class="atelier-table min-w-[540px] text-xs">
          <thead>
            <tr>
              <th scope="col">{{ copy('时间', 'Date') }}</th>
              <th scope="col">{{ copy('类型', 'Type') }}</th>
              <th scope="col">{{ copy('金额 / 额度', 'Amount / allowance') }}</th>
              <th scope="col">{{ copy('兑换码', 'Code') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="py-12 text-center text-[var(--atelier-ink-muted)]" role="status">
                {{ copy('正在加载兑换记录…', 'Loading redemption history…') }}
              </td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="4" class="py-12 text-center text-[var(--atelier-ink-muted)]">
                {{ error ? copy('记录暂时无法显示', 'History is temporarily unavailable') : copy('暂无兑换记录', 'No redemptions yet') }}
              </td>
            </tr>
            <tr v-for="item in loading ? [] : rows" :key="item.id">
              <td class="whitespace-nowrap">
                <time :datetime="item.used_at || item.created_at" class="font-mono tabular-nums text-[var(--atelier-ink)]">
                  {{ formattedDate(item.used_at || item.created_at) }}
                </time>
                <span class="block font-mono text-[11px] text-[var(--atelier-ink-muted)]">#{{ item.id }}</span>
              </td>
              <td class="whitespace-nowrap">
                <span class="text-[var(--atelier-ink)]">{{ typeLabel(item.type) }}</span>
                <span v-if="item.type === 'subscription' && item.group?.name" class="block text-[11px] text-[var(--atelier-ink-muted)]">
                  {{ item.group.name }}
                </span>
              </td>
              <td class="whitespace-nowrap font-mono font-semibold tabular-nums text-[var(--atelier-ink)]">
                {{ valueLabel(item) }}
              </td>
              <td class="whitespace-nowrap font-mono text-[var(--atelier-ink-secondary)]">
                {{ item.maskedCode }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="!compact && total > 0"
        class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--atelier-border)] pt-4 text-xs text-[var(--atelier-ink-secondary)]"
      >
        <div class="flex items-center gap-2">
          <span>{{ copy('每页', 'Per page') }}</span>
          <div class="w-20">
            <UiSelect
              :model-value="selectedPageSize"
              :options="pageSizeOptions"
              :disabled="loading"
              size="sm"
              :aria-label="copy('每页条数', 'Rows per page')"
              @update:model-value="handlePageSizeChange"
            />
          </div>
        </div>

        <div class="flex items-center gap-3">
          <span class="font-mono tabular-nums text-[var(--atelier-ink-muted)]">
            {{ copy(`共 ${total} 条`, `${total} records`) }} · {{ page }} / {{ pages }}
          </span>
          <div class="flex items-center gap-1.5">
            <UiButton variant="secondary" size="sm" type="button" :disabled="loading || page <= 1" @click="load(page - 1)">
              {{ copy('上一页', 'Previous') }}
            </UiButton>
            <UiButton variant="secondary" size="sm" type="button" :disabled="loading || page >= pages" @click="load(page + 1)">
              {{ copy('下一页', 'Next') }}
            </UiButton>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
