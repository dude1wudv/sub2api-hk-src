<template>
  <div class="card p-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <p class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('payment.redeemCodeOnlyTitle') }}
        </p>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('payment.redeemCodeOnlyDescription') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-primary shrink-0 sm:mt-1"
        @click="copyContact"
      >
        {{ copied ? t('common.copied') : t('payment.copyRechargeContact') }}
      </button>
    </div>

    <div class="mt-5 overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700">
      <div class="bg-gray-50 px-4 py-3 dark:bg-dark-800/60">
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('payment.rechargePricingTitle') }}</p>
      </div>
      <div class="divide-y divide-gray-100 dark:divide-dark-700">
        <div
          v-for="plan in rechargePlans"
          :key="plan.name"
          class="flex items-center justify-between gap-4 px-4 py-3"
        >
          <div class="min-w-0">
            <span class="mb-1 inline-flex rounded-md bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
              {{ t('payment.rechargeCardTag') }}
            </span>
            <p class="truncate text-sm font-medium text-gray-700 dark:text-gray-200">{{ plan.name }}</p>
          </div>
          <div class="shrink-0 text-sm font-semibold text-gray-900 dark:text-white">
            {{ plan.price }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'

const rechargeContact = 'QQ:320985943'
const rechargePlans = [
  { name: '2.1元100刀余额不限时长', price: '2.1' },
  { name: 'kiro Claude', price: '9.9' },
  { name: 'claude站内300刀额度', price: '29.9' },
  { name: '1.9兑换100刀', price: '1.9' }
]
const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()

const copyContact = () => {
  void copyToClipboard(rechargeContact, t('common.copiedToClipboard'))
}
</script>
