<template>
  <div class="card p-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('payment.redeemCodeOnlyTitle') }}
        </p>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('payment.redeemCodeOnlyDescription') }}
        </p>
      </div>
      <a
        :href="shopUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="btn btn-primary shrink-0"
      >
        {{ t('payment.buyRedeemCode') }}
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const shopUrl = 'https://pay.ldxp.cn/shop/EFKQE86C'
const { t } = useI18n()
</script>
