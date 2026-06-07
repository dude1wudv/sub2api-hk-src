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
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'

const rechargeContact = 'QQ:320985943'
const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()

const copyContact = () => {
  void copyToClipboard(rechargeContact, t('common.copiedToClipboard'))
}
</script>
