<script setup lang="ts">
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatPortalBalance, PATRICK_CARD_ORDER_URL, PATRICK_CARD_SHOP_URL } from '@/utils/portalPurchase'
import PortalRedeemForm from '@/components/portal/PortalRedeemForm.vue'
import PortalRedeemHistory from '@/components/portal/PortalRedeemHistory.vue'

const { locale } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const denominations = computed(() => app.cachedPublicSettings?.card_shop_denominations_usd ?? [1, 10, 50])
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const balance = computed(() => auth.user ? formatPortalBalance(auth.user.balance, locale.value) : '—')
const historyVersion = ref(0)
</script>

<template>
  <div class="portal-page purchase-page">
    <PortalPageHeader :title="copy('充值与兑换', 'Top up & redeem')" />
    <nav class="purchase-tabs" :aria-label="copy('充值与兑换记录', 'Top up and redemption history')">
      <RouterLink to="/purchase" class="active" aria-current="page">{{ copy('充值与兑换', 'Top up & redeem') }}</RouterLink>
      <RouterLink to="/orders">{{ copy('兑换记录', 'Redemption history') }}</RouterLink>
    </nav>

    <section class="portal-panel purchase-balance" :aria-label="copy('账户余额', 'Account balance')">
      <div><p>{{ copy('当前余额', 'Current balance') }}</p><strong>{{ balance }}</strong><small>{{ copy('可用于按量计费的模型调用', 'Available for pay-as-you-go model requests') }}</small></div>
      <RouterLink to="/usage" class="portal-button secondary">{{ copy('查看用量', 'View usage') }}</RouterLink>
    </section>

    <div class="purchase-grid">
      <section class="portal-panel purchase-shop" aria-labelledby="purchase-shop-title">
        <div class="purchase-step-heading"><span aria-hidden="true">01</span><h2 id="purchase-shop-title">{{ copy('购买兑换码', 'Purchase a code') }}</h2></div>
        <p class="portal-muted">{{ copy('在发卡网 patrickapi 店铺购买兑换码，完成后返回本站兑换。', 'Buy a code from the patrickapi card shop, then return here to redeem it.') }}</p>
        <div class="purchase-denominations"><span>{{ copy('到账面额', 'Balance denominations') }}</span><strong v-for="amount in denominations" :key="amount">{{ formatPortalBalance(amount, locale) }}</strong></div>
        <a :href="PATRICK_CARD_SHOP_URL" target="_blank" rel="noopener noreferrer" class="portal-button purchase-shop-button">
          {{ copy('前往发卡网购买', 'Visit card shop') }} <span aria-hidden="true">↗</span>
        </a>
        <p class="purchase-shop-note">{{ copy('面额、售价和库存以发卡网页面为准。', 'Available denominations, prices, and stock are shown in the card shop.') }}</p>
        <div class="purchase-order-help">
          <span>{{ copy('已购买但找不到兑换码？', 'Already purchased but cannot find your code?') }}</span>
          <a :href="PATRICK_CARD_ORDER_URL" target="_blank" rel="noopener noreferrer">{{ copy('查询发卡网订单', 'Find your card-shop order') }} <span aria-hidden="true">↗</span></a>
        </div>
      </section>

      <PortalRedeemForm @redeemed="historyVersion++" />
    </div>

    <div class="purchase-history"><PortalRedeemHistory compact :refresh-key="historyVersion" /></div>
  </div>
</template>

<style scoped>
.purchase-tabs { display: flex; gap: 28px; border-bottom: 1px solid rgb(var(--line)); margin-bottom: 26px; }
.purchase-tabs a { padding: 0 0 13px; color: rgb(var(--ink-3)); font: 500 14.5px/1.5 var(--font-sans); text-decoration: none; transition: color var(--motion-fast) ease; }
.purchase-tabs a:hover { color: rgb(var(--ink)); }
.purchase-tabs .active { color: rgb(var(--ink)); font-weight: 600; border-bottom: 2px solid rgb(var(--brand-moss)); margin-bottom: -1px; }
.purchase-balance {
  background: linear-gradient(155deg, #0f172a 0%, #162235 62%, #0f2926 100%);
  border-color: #1e293b;
  color: #f8fafc;
  box-shadow: 0 8px 24px -6px rgb(15 23 42 / .24), inset 0 1px 0 rgb(255 255 255 / .08);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 26px;
  margin-bottom: 24px;
}
:global(:root.dark) .purchase-balance {
  background: linear-gradient(155deg, #16202e 0%, #111b27 58%, #0f2924 100%);
  border-color: rgb(var(--brand-moss) / .38);
  box-shadow: 0 8px 28px -6px rgb(0 0 0 / .45), inset 0 1px 0 rgb(255 255 255 / .06);
}
.purchase-balance p { margin: 0 0 8px; color: #cbd5e1; font-size: 13px; font-weight: 550; }
.purchase-balance strong { display: block; color: #ffffff; font-size: 32px; line-height: 1.18; font-weight: 650; letter-spacing: -0.025em; font-variant-numeric: tabular-nums; }
.purchase-balance small { display: block; margin-top: 10px; font-size: 13px; color: #94a3b8; }
.purchase-balance .portal-button.secondary { background: rgb(255 255 255 / .1); border-color: rgb(255 255 255 / .2); color: #f8fafc; }
.purchase-balance .portal-button.secondary:hover { background: rgb(255 255 255 / .18); border-color: rgb(255 255 255 / .34); }
.purchase-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); align-items: start; gap: 22px; }
.purchase-shop { padding: 26px; }
.purchase-step-heading { display: flex; align-items: center; gap: 11px; margin-bottom: 12px; }
.purchase-step-heading > span { display: grid; place-items: center; width: 28px; height: 28px; background: rgb(var(--brand-moss) / .12); color: rgb(var(--brand-moss)); border: 1px solid rgb(var(--brand-moss) / .24); border-radius: 8px; font-family: var(--font-mono); font-size: 12.5px; font-weight: 650; }
.purchase-step-heading h2 { margin: 0; font-size: 17.5px; font-weight: 650; letter-spacing: -0.015em; }
.purchase-shop > p { margin: 0; font-size: 13.5px; line-height: 1.7; }
.purchase-denominations { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-top: 20px; font-size: 13px; color: rgb(var(--ink-2)); }
.purchase-denominations strong { padding: 6px 12px; border: 1px solid rgb(var(--line)); border-radius: 9px; background: rgb(var(--surface-2)); color: rgb(var(--ink)); font-weight: 600; font-variant-numeric: tabular-nums; }
.purchase-shop-button { margin-top: 24px; }
.purchase-shop .purchase-shop-note { margin-top: 14px; color: rgb(var(--ink-3)); font-size: 13px; }
.purchase-order-help { display: flex; flex-wrap: wrap; gap: 8px 12px; margin-top: 26px; padding-top: 18px; border-top: 1px solid rgb(var(--line)); color: rgb(var(--ink-3)); font-size: 13px; line-height: 1.7; }
.purchase-order-help a { color: rgb(var(--brand-moss)); font-weight: 550; text-decoration: underline; text-underline-offset: 3px; }
.purchase-history { margin-top: 34px; }
@media (max-width: 899px) { .purchase-grid { grid-template-columns: 1fr; gap: 18px; } }
@media (max-width: 767px) {
  .purchase-tabs { gap: 22px; margin-bottom: 20px; }
  .purchase-balance { flex-wrap: wrap; gap: 16px; padding: 20px; }
  .purchase-balance strong { font-size: 27px; }
  .purchase-balance .portal-button { padding-inline: 14px; }
  .purchase-shop { padding: 20px; }
  .purchase-history { margin-top: 28px; }
}
</style>
