<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { PATRICK_CARD_ORDER_URL } from '@/utils/portalPurchase'
import PortalRedeemHistory from '@/components/portal/PortalRedeemHistory.vue'
import { UiTabs } from '@/ui'

const { locale } = useI18n()
const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const subscriptionEnabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription))

const billingTabs = computed(() => {
  const tabs: Array<{ value: string; label: string }> = []
  if (!auth.isSimpleMode) {
    tabs.push({ value: 'purchase', label: copy('充值与兑换', 'Top up & redeem') })
    tabs.push({ value: 'orders', label: copy('兑换记录', 'Redemption history') })
  }
  if (subscriptionEnabled.value) {
    tabs.push({ value: 'subscriptions', label: copy('订阅计划', 'Subscriptions') })
  }
  return tabs
})

function handleTabChange(val: string) {
  if (val === 'purchase') void router.push('/purchase')
  else if (val === 'subscriptions') void router.push('/subscriptions')
}
</script>

<template>
  <div class="mx-auto max-w-[1240px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ copy('兑换记录', 'Redemption history') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ copy('查看在本站完成的兑换码核销及额度调整明细。', 'Review code redemptions and allowance adjustments completed on this site.') }}
      </p>
    </header>

    <UiTabs
      v-if="billingTabs.length > 1"
      model-value="orders"
      :tabs="billingTabs"
      :aria-label="copy('计费与兑换导航', 'Billing and redemption navigation')"
      @update:model-value="handleTabChange"
    />

    <aside class="atelier-surface-subtle flex flex-col items-start justify-between gap-4 p-5 sm:flex-row sm:items-center">
      <div class="max-w-2xl">
        <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
          {{ copy('本站兑换记录', 'Redemptions on patrickapi') }}
        </h2>
        <p class="mt-1 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
          {{ copy('这里显示在本站完成的兑换及额度调整。发卡网的购买订单、付款和发货记录，请前往发卡网查询。', 'This page shows redemptions and allowance adjustments on patrickapi. Purchase, payment, and delivery records are available in the card shop.') }}
        </p>
      </div>
      <a
        :href="PATRICK_CARD_ORDER_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="atelier-btn atelier-btn-secondary h-9 shrink-0 px-3.5 text-xs"
      >
        {{ copy('查询发卡网订单', 'Card-shop orders') }}
        <span aria-hidden="true">↗</span>
      </a>
    </aside>

    <PortalRedeemHistory />
  </div>
</template>
