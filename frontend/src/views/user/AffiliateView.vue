<template>
  <div class="mx-auto max-w-[1240px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ t('affiliate.title') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ t('affiliate.description') }}
      </p>
    </header>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-[var(--atelier-ink)] border-t-transparent" />
    </div>

    <div
      v-if="loadError"
      class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]"
      role="alert"
    >
      <span>{{ loadError }}</span>
      <UiButton variant="secondary" size="sm" :disabled="loading || retryAfter > 0" @click="loadAffiliateDetail()">
        {{ retryAfter > 0 ? `${retryAfter}s` : copy('重试', 'Retry') }}
      </UiButton>
    </div>

    <template v-if="detail">
      <!-- Borderless Metric Ribbon -->
      <UiMetricRibbon
        :primary-label="t('affiliate.stats.availableQuota')"
        :primary-value="formatCurrency(detail.aff_quota)"
        :primary-meta="detail.aff_frozen_quota > 0 ? `${t('affiliate.stats.frozenQuota')}: ${formatCurrency(detail.aff_frozen_quota)}` : t('affiliate.stats.rebateRateHint')"
        :metrics="ribbonMetrics"
        :show-dots="true"
      >
        <template #primaryAction>
          <UiButton
            variant="primary"
            size="md"
            :disabled="transferring || refreshRequired || retryAfter > 0 || detail.aff_quota <= 0"
            :loading="transferring"
            @click="openTransferConfirmation"
          >
            <Icon v-if="!transferring" name="dollar" size="sm" />
            <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
          </UiButton>
        </template>
      </UiMetricRibbon>

      <!-- Transfer Status Messages -->
      <div v-if="transferNotice || transferError || refreshRequired || retryAfter > 0 || detail.aff_quota <= 0" class="space-y-2">
        <p v-if="transferNotice" class="rounded-lg border border-[var(--atelier-ok)]/25 bg-[var(--atelier-ok-bg)] px-4 py-2.5 text-xs text-[var(--atelier-ok)]" role="status">
          {{ transferNotice }}
        </p>
        <p v-if="transferError" class="rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-2.5 text-xs text-[var(--atelier-err)]" role="alert">
          {{ transferError }}
        </p>
        <div
          v-if="refreshRequired"
          class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-2.5 text-xs text-[var(--atelier-err)]"
          role="alert"
        >
          <span>{{ copy('账户信息尚未刷新，请刷新并核对后再操作。', 'Account information could not be refreshed. Refresh and check before another transfer.') }}</span>
          <UiButton variant="secondary" size="sm" :disabled="transferring || retryAfter > 0" @click="refreshAfterTransfer">
            {{ retryAfter > 0 ? `${retryAfter}s` : copy('刷新返利与余额', 'Refresh rebate and balance') }}
          </UiButton>
        </div>
        <p v-if="retryAfter > 0" class="text-xs text-[var(--atelier-ink-muted)]" role="status">
          {{ copy(`请在 ${retryAfter} 秒后重试。`, `Try again in ${retryAfter} seconds.`) }}
        </p>
        <p v-if="detail.aff_quota <= 0" class="text-xs text-[var(--atelier-ink-muted)]">
          {{ t('affiliate.transfer.empty') }}
        </p>
      </div>

      <!-- Invite Code & Link + Rules (Asymmetric 7/5) -->
      <div class="grid grid-cols-1 gap-5 lg:grid-cols-12">
        <section class="atelier-surface space-y-5 p-6 lg:col-span-7">
          <div>
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ t('affiliate.inviteLink') }}
            </h2>
            <p class="mt-1 text-xs text-[var(--atelier-ink-secondary)]">
              {{ t('affiliate.transfer.description') }}
            </p>
          </div>

          <div class="space-y-4">
            <div class="space-y-1.5">
              <p class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-3 py-2 sm:flex-row sm:items-center">
                <code class="min-w-0 flex-1 break-all font-mono text-sm font-semibold text-[var(--atelier-ink)] sm:truncate">
                  {{ detail.aff_code }}
                </code>
                <UiButton variant="secondary" size="sm" class="sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </UiButton>
              </div>
            </div>

            <div class="space-y-1.5">
              <p class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-3 py-2 sm:flex-row sm:items-center">
                <code class="min-w-0 flex-1 break-all font-mono text-xs text-[var(--atelier-ink-secondary)] sm:truncate">
                  {{ inviteLink }}
                </code>
                <UiButton variant="secondary" size="sm" class="sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </UiButton>
              </div>
            </div>
          </div>
        </section>

        <aside class="atelier-surface-subtle p-6 lg:col-span-5">
          <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ t('affiliate.tips.title') }}
          </h2>
          <ul class="mt-3 space-y-2 text-xs leading-relaxed text-[var(--atelier-ink-secondary)]">
            <li>1. {{ t('affiliate.tips.line1') }}</li>
            <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
            <li>3. {{ t('affiliate.tips.line3') }}</li>
            <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
          </ul>
        </aside>
      </div>

      <!-- Invitees Table -->
      <section class="atelier-surface overflow-hidden">
        <header class="border-b border-[var(--atelier-border)] px-6 py-4">
          <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ t('affiliate.invitees.title') }}
          </h2>
        </header>

        <div class="p-6">
          <div
            v-if="detail.invitees.length === 0"
            class="rounded-xl border border-dashed border-[var(--atelier-border-strong)] py-10 text-center text-sm text-[var(--atelier-ink-muted)]"
          >
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <div v-if="mobile" class="space-y-3">
              <dl
                v-for="item in detail.invitees"
                :key="item.user_id"
                class="space-y-2 rounded-xl border border-[var(--atelier-border)] p-3.5 text-xs"
              >
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('affiliate.invitees.columns.email') }}</dt>
                  <dd class="font-medium text-[var(--atelier-ink)]">{{ item.email || '-' }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('affiliate.invitees.columns.username') }}</dt>
                  <dd class="text-[var(--atelier-ink-secondary)]">{{ item.username || '-' }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('affiliate.invitees.columns.rebate') }}</dt>
                  <dd class="font-mono font-semibold tabular-nums text-[var(--atelier-ok)]">{{ formatCurrency(item.total_rebate) }}</dd>
                </div>
                <div class="flex justify-between gap-2">
                  <dt class="text-[var(--atelier-ink-muted)]">{{ t('affiliate.invitees.columns.joinedAt') }}</dt>
                  <dd class="font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ formatDateTime(item.created_at) || '-' }}</dd>
                </div>
              </dl>
            </div>

            <table v-else class="atelier-table min-w-[540px] text-xs">
              <thead>
                <tr>
                  <th>{{ t('affiliate.invitees.columns.email') }}</th>
                  <th>{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="!text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th>{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in detail.invitees" :key="item.user_id">
                  <td class="font-medium text-[var(--atelier-ink)]">{{ item.email || '-' }}</td>
                  <td class="text-[var(--atelier-ink-secondary)]">{{ item.username || '-' }}</td>
                  <td class="text-right font-mono font-semibold tabular-nums text-[var(--atelier-ok)]">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="font-mono tabular-nums text-[var(--atelier-ink-secondary)]">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </template>

    <BaseDialog :show="confirmTransfer" :title="copy('确认转入余额', 'Confirm balance transfer')" :busy="transferring" width="narrow" @close="cancelTransfer">
      <p class="text-sm text-[var(--atelier-ink)]">
        {{ copy(`将当前可用返利 ${formatCurrency(detail?.aff_quota ?? 0)} 全部转入 API 账户余额，用于本站消费。`, `Transfer all currently available rebate (${formatCurrency(detail?.aff_quota ?? 0)}) to your API account balance for use on this site.`) }}
      </p>
      <p class="mt-2 text-xs text-[var(--atelier-ink-muted)]">
        {{ copy('到账金额以实际处理结果为准。', 'The credited amount is determined when the transfer is processed.') }}
      </p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UiButton variant="secondary" size="md" type="button" :disabled="transferring" @click="cancelTransfer">
            {{ copy('取消', 'Cancel') }}
          </UiButton>
          <UiButton
            variant="primary"
            size="md"
            type="button"
            data-testid="confirm-transfer"
            :disabled="transferring || refreshRequired || retryAfter > 0"
            :loading="transferring"
            @click="transferQuota"
          >
            {{ transferring ? copy('正在转入…', 'Transferring…') : copy('确认转入', 'Confirm transfer') }}
          </UiButton>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
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
import { UiButton, UiMetricRibbon, type RibbonMetric } from '@/ui'

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

const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

const ribbonMetrics = computed<RibbonMetric[]>(() => {
  if (!detail.value) return []
  return [
    { label: t('affiliate.stats.rebateRate'), value: `${formattedRebateRate.value}%` },
    { label: t('affiliate.stats.invitedUsers'), value: formatCount(detail.value.aff_count) },
    { label: t('affiliate.stats.totalQuota'), value: formatCurrency(detail.value.aff_history_quota) },
  ]
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (retryAfter.value > 0) return
  loadError.value = ''
  if (!silent) loading.value = true
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    loadError.value = userFacingError(error, currentLocale.value, t('affiliate.loadFailed'))
    startRetry(error)
  } finally {
    if (!silent) loading.value = false
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
