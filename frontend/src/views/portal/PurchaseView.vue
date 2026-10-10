<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { formatPortalBalance, PATRICK_CARD_ORDER_URL, PATRICK_CARD_SHOP_URL } from '@/utils/portalPurchase'
import PortalRedeemForm from '@/components/portal/PortalRedeemForm.vue'
import PortalRedeemHistory from '@/components/portal/PortalRedeemHistory.vue'

const { locale } = useI18n()
const auth = useAuthStore()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const balance = computed(() => auth.user ? formatPortalBalance(auth.user.balance, locale.value) : '—')
const historyVersion = ref(0)
</script>

<template>
  <div class="portal-page purchase-page">
    <nav class="purchase-tabs" :aria-label="copy('充值与兑换记录', 'Top up and redemption history')">
      <RouterLink to="/purchase" class="active" aria-current="page">{{ copy('充值', 'Top up') }}</RouterLink>
      <RouterLink to="/orders">{{ copy('兑换记录', 'Redemption history') }}</RouterLink>
    </nav>

    <section class="portal-panel purchase-balance" :aria-label="copy('账户余额', 'Account balance')">
      <div><p>{{ copy('当前余额', 'Current balance') }}</p><strong>{{ balance }}</strong></div>
      <RouterLink to="/usage" class="portal-button secondary">{{ copy('查看用量', 'View usage') }}</RouterLink>
    </section>

    <div class="purchase-grid">
      <section class="portal-panel purchase-shop" aria-labelledby="purchase-shop-title">
        <div class="purchase-step-heading"><span aria-hidden="true">01</span><h1 id="purchase-shop-title">{{ copy('购买兑换码', 'Purchase a code') }}</h1></div>
        <p class="portal-muted">{{ copy('在发卡网 patrickapi 店铺购买兑换码，完成后返回本站兑换。', 'Buy a code from the patrickapi card shop, then return here to redeem it.') }}</p>
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
.purchase-tabs { display: flex; gap: 28px; border-bottom: 1px solid #efede3; margin-bottom: 28px; }
.purchase-tabs a { padding: 0 0 15px; color: #8d8c87; font: 22px/1.3 'Times New Roman', Times, serif; text-decoration: none; }
.purchase-tabs .active { color: #000; border-bottom: 2px solid #000; margin-bottom: -1px; }
.purchase-balance { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 24px 26px; margin-bottom: 24px; }
.purchase-balance p { margin: 0 0 10px; color: #8d8c87; font-size: 13px; }
.purchase-balance strong { display: block; color: #000; font-size: 30px; line-height: 1.2; font-weight: 400; font-variant-numeric: tabular-nums; }
.purchase-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); align-items: start; gap: 24px; }
.purchase-shop { padding: 26px; }
.purchase-step-heading { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.purchase-step-heading > span { display: grid; place-items: center; width: 29px; height: 29px; border: 1px solid #efede3; border-radius: 50%; font-size: 12px; }
.purchase-step-heading h1 { margin: 0; font-size: 18px; font-weight: 400; }
.purchase-shop > p { margin: 0; font-size: 13px; line-height: 1.8; }
.purchase-shop-button { margin-top: 28px; }
.purchase-shop .purchase-shop-note { margin-top: 14px; color: #8d8c87; font-size: 12px; }
.purchase-order-help { display: flex; flex-wrap: wrap; gap: 8px 12px; margin-top: 30px; padding-top: 20px; border-top: 1px solid #efede3; color: #8d8c87; font-size: 12px; line-height: 1.8; }
.purchase-order-help a { color: #000; text-decoration: underline; text-underline-offset: 3px; }
.purchase-history { margin-top: 38px; }
@media (max-width: 899px) { .purchase-grid { grid-template-columns: 1fr; gap: 20px; } }
@media (max-width: 739px) {
  .purchase-tabs { gap: 24px; margin-bottom: 22px; }
  .purchase-balance { gap: 16px; padding: 22px 20px; }
  .purchase-balance strong { font-size: 27px; }
  .purchase-balance .portal-button { padding-inline: 14px; }
  .purchase-shop { padding: 20px; }
  .purchase-history { margin-top: 30px; }
}
</style>
