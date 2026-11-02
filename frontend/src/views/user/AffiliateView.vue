<template>
  <div class="portal-page user-service-page">
    <PortalPageHeader :title="t('affiliate.title')" />
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <div v-if="loadError" class="portal-error" role="alert">{{ loadError }} <button type="button" :disabled="loading || retryAfter > 0" @click="loadAffiliateDetail()">{{ retryAfter > 0 ? `${retryAfter}s` : copy('重试', 'Retry') }}</button></div>
      <template v-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || refreshRequired || retryAfter > 0 || detail.aff_quota <= 0"
              @click="openTransferConfirmation"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="transferNotice" class="portal-muted mt-3" role="status">{{ transferNotice }}</p>
          <p v-if="transferError" class="portal-error mt-3" role="alert">{{ transferError }}</p>
          <div v-if="refreshRequired" class="portal-error mt-3" role="alert">
            {{ copy('账户信息尚未刷新，请刷新并核对后再操作。', 'Account information could not be refreshed. Refresh and check before another transfer.') }}
            <button type="button" :disabled="transferring || retryAfter > 0" @click="refreshAfterTransfer">{{ retryAfter > 0 ? `${retryAfter}s` : copy('刷新返利与余额', 'Refresh rebate and balance') }}</button>
          </div>
          <p v-if="retryAfter > 0" class="portal-muted mt-3" role="status">{{ copy(`请在 ${retryAfter} 秒后重试。`, `Try again in ${retryAfter} seconds.`) }}</p>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
    <BaseDialog :show="confirmTransfer" :title="copy('确认转入余额', 'Confirm balance transfer')" :busy="transferring" width="narrow" @close="cancelTransfer">
      <p>{{ copy(`将当前可用返利 ${formatCurrency(detail?.aff_quota ?? 0)} 全部转入 API 账户余额，用于本站消费。`, `Transfer all currently available rebate (${formatCurrency(detail?.aff_quota ?? 0)}) to your API account balance for use on this site.`) }}</p>
      <p class="portal-muted mt-2">{{ copy('到账金额以实际处理结果为准。', 'The credited amount is determined when the transfer is processed.') }}</p>
      <template #footer><div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="transferring" @click="cancelTransfer">{{ copy('取消', 'Cancel') }}</button>
        <button type="button" class="btn btn-primary" data-testid="confirm-transfer" :disabled="transferring || refreshRequired || retryAfter > 0" @click="transferQuota">{{ transferring ? copy('正在转入…', 'Transferring…') : copy('确认转入', 'Confirm transfer') }}</button>
      </div></template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { userFacingError } from '@/utils/userFacingError'
import { useRetryCooldown } from '@/composables/useRetryCooldown'

const { t, locale } = useI18n()
const currentLocale = computed(() => locale?.value || 'en')
const copy = (zh: string, en: string) => currentLocale.value.startsWith('zh') ? zh : en
const { remaining: retryAfter, start: startRetry } = useRetryCooldown()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const confirmTransfer = ref(false)
const refreshRequired = ref(false)
const loadError = ref('')
const transferError = ref('')
const transferNotice = ref('')

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (retryAfter.value > 0) return
  loadError.value = ''
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    loadError.value = userFacingError(error, currentLocale.value, t('affiliate.loadFailed'))
    startRetry(error)
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

function openTransferConfirmation() {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value || refreshRequired.value || retryAfter.value > 0) return
  transferError.value = ''
  transferNotice.value = ''
  confirmTransfer.value = true
}
function cancelTransfer() {
  if (!transferring.value) confirmTransfer.value = false
}
async function refreshAfterTransfer() {
  refreshRequired.value = true
  const results = await Promise.allSettled([userAPI.getAffiliateDetail(), authStore.refreshUser()])
  if (results[0].status === 'fulfilled') detail.value = results[0].value
  refreshRequired.value = results.some(result => result.status === 'rejected')
  for (const result of results) if (result.status === 'rejected') startRetry(result.reason)
}
async function transferQuota(): Promise<void> {
  if (!confirmTransfer.value || !detail.value || detail.value.aff_quota <= 0 || transferring.value || refreshRequired.value || retryAfter.value > 0) return
  transferring.value = true
  transferError.value = ''
  transferNotice.value = ''
  try {
    const resp = await userAPI.transferAffiliateQuota()
    confirmTransfer.value = false
    const message = t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) })
    transferNotice.value = message
    appStore.showSuccess(message)
    await refreshAfterTransfer()
  } catch (error) {
    confirmTransfer.value = false
    const status = (error as { status?: number; response?: { status?: number } } | null)?.status ?? (error as { response?: { status?: number } } | null)?.response?.status
    if (!status || status >= 500) {
      transferNotice.value = copy('请求结果尚未确认，正在查询返利与余额；请核对后再操作，不要重复提交。', 'The transfer result is uncertain. Checking your rebate and balance; verify them before trying again.')
      await refreshAfterTransfer()
    } else {
      transferError.value = userFacingError(error, currentLocale.value, t('affiliate.transferFailed'))
    }
    startRetry(error)
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
