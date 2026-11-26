<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { formatPortalBalance, PATRICK_CARD_ORDER_URL, PATRICK_CARD_SHOP_URL } from '@/utils/portalPurchase'
import PortalRedeemForm from '@/components/portal/PortalRedeemForm.vue'
import PortalRedeemHistory from '@/components/portal/PortalRedeemHistory.vue'
import { UiMetricRibbon, UiTabs } from '@/ui'

const { locale } = useI18n()
const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const denominations = computed(() => app.cachedPublicSettings?.card_shop_denominations_usd ?? [1, 10, 50])
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const balance = computed(() => auth.user ? formatPortalBalance(auth.user.balance, locale.value) : '—')
const subscriptionEnabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription))
const historyVersion = ref(0)

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
  if (val === 'orders') void router.push('/orders')
  else if (val === 'subscriptions') void router.push('/subscriptions')
}
</script>

<template>
  <div class="mx-auto max-w-[1240px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ copy('充值与兑换', 'Top up & redeem') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ copy('通过兑换码快速为账户充值余额或激活权益。', 'Top up your account balance or activate allowances using redemption codes.') }}
      </p>
    </header>

    <UiTabs
      v-if="billingTabs.length > 1"
      model-value="purchase"
      :tabs="billingTabs"
      :aria-label="copy('计费与兑换导航', 'Billing and redemption navigation')"
      @update:model-value="handleTabChange"
    />

    <!-- Balance Ribbon -->
    <UiMetricRibbon
      :primary-label="copy('当前余额', 'Current balance')"
      :primary-value="balance"
      :primary-meta="copy('可用于按量计费的模型调用', 'Available for pay-as-you-go model requests')"
      :show-dots="true"
    >
      <template #primaryAction>
        <RouterLink to="/usage" class="atelier-btn atelier-btn-secondary h-9 px-3.5 text-xs">
          {{ copy('查看用量', 'View usage') }}
        </RouterLink>
      </template>
    </UiMetricRibbon>

    <!-- Two-Column Purchase & Redeem Grid -->
    <div class="grid grid-cols-1 items-start gap-5 lg:grid-cols-2">
      <section class="atelier-surface p-6" aria-labelledby="purchase-shop-title">
        <div class="flex items-center gap-3">
          <span
            class="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] font-mono text-xs font-semibold text-[var(--atelier-ink)]"
            aria-hidden="true"
          >
            01
          </span>
          <h2 id="purchase-shop-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ copy('购买兑换码', 'Purchase a code') }}
          </h2>
        </div>

        <p class="mt-3 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
          {{ copy('在发卡网 patrickapi 店铺购买兑换码，完成后返回本站兑换。', 'Buy a code from the patrickapi card shop, then return here to redeem it.') }}
        </p>

        <div class="mt-5 flex flex-wrap items-center gap-2 text-xs text-[var(--atelier-ink-secondary)]">
          <span class="mr-1">{{ copy('到账面额', 'Balance denominations') }}</span>
          <strong
            v-for="amount in denominations"
            :key="amount"
            class="rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2.5 py-1 font-mono text-xs font-semibold tabular-nums text-[var(--atelier-ink)]"
          >
            {{ formatPortalBalance(amount, locale) }}
          </strong>
        </div>

        <div class="mt-6">
          <a
            :href="PATRICK_CARD_SHOP_URL"
            target="_blank"
            rel="noopener noreferrer"
            class="atelier-btn atelier-btn-primary h-10 px-4 text-sm"
          >
            {{ copy('前往发卡网购买', 'Visit card shop') }}
            <span aria-hidden="true">↗</span>
          </a>
        </div>

        <p class="mt-3 text-xs text-[var(--atelier-ink-muted)]">
          {{ copy('面额、售价和库存以发卡网页面为准。', 'Available denominations, prices, and stock are shown in the card shop.') }}
        </p>

        <div class="mt-6 flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-[var(--atelier-border)] pt-4 text-xs text-[var(--atelier-ink-muted)]">
          <span>{{ copy('已购买但找不到兑换码？', 'Already purchased but cannot find your code?') }}</span>
          <a
            :href="PATRICK_CARD_ORDER_URL"
            target="_blank"
            rel="noopener noreferrer"
            class="font-medium text-[var(--atelier-ink)] underline underline-offset-4 hover:opacity-80"
          >
            {{ copy('查询发卡网订单', 'Find your card-shop order') }} <span aria-hidden="true">↗</span>
          </a>
        </div>
      </section>

      <PortalRedeemForm @redeemed="historyVersion++" />
    </div>

    <PortalRedeemHistory compact :refresh-key="historyVersion" />
  </div>
</template>
