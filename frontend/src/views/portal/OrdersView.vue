<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { PATRICK_CARD_ORDER_URL } from '@/utils/portalPurchase'
import PortalRedeemHistory from '@/components/portal/PortalRedeemHistory.vue'

const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
</script>

<template>
  <div class="portal-page redemption-page">
    <nav class="redemption-tabs" :aria-label="copy('充值与兑换记录', 'Top up and redemption history')">
      <RouterLink to="/purchase">{{ copy('充值', 'Top up') }}</RouterLink>
      <RouterLink to="/orders" class="active" aria-current="page">{{ copy('兑换记录', 'Redemption history') }}</RouterLink>
    </nav>
    <aside class="redemption-orders-note">
      <div>
        <h1>{{ copy('本站兑换记录', 'Redemptions on patrickapi') }}</h1>
        <p>{{ copy('这里显示在本站完成的兑换及额度调整。发卡网的购买订单、付款和发货记录，请前往发卡网查询。', 'This page shows redemptions and allowance adjustments on patrickapi. Purchase, payment, and delivery records are available in the card shop.') }}</p>
      </div>
      <a :href="PATRICK_CARD_ORDER_URL" target="_blank" rel="noopener noreferrer" class="portal-button secondary">{{ copy('查询发卡网订单', 'Card-shop orders') }} <span aria-hidden="true">↗</span></a>
    </aside>
    <PortalRedeemHistory />
  </div>
</template>

<style scoped>
.redemption-tabs { display: flex; gap: 28px; border-bottom: 1px solid #efede3; margin-bottom: 28px; }
.redemption-tabs a { padding-bottom: 15px; color: #8d8c87; font: 22px/1.3 'Times New Roman', Times, serif; text-decoration: none; }
.redemption-tabs .active { color: #000; border-bottom: 2px solid #000; margin-bottom: -1px; }
.redemption-orders-note { display: flex; align-items: center; justify-content: space-between; gap: 26px; padding: 24px; margin-bottom: 34px; border: 1px solid #efede3; border-radius: 12px; background: #fffdf7; }
.redemption-orders-note h1 { margin: 0 0 10px; font: 400 22px/1.4 'Times New Roman', Times, serif; }
.redemption-orders-note p { max-width: 720px; margin: 0; color: #8d8c87; font-size: 13px; line-height: 1.8; }
.redemption-orders-note > a { flex-shrink: 0; }
@media (max-width: 899px) { .redemption-orders-note { align-items: flex-start; flex-direction: column; gap: 18px; } }
@media (max-width: 739px) { .redemption-orders-note { padding: 20px; margin-bottom: 28px; } .redemption-tabs { gap: 24px; margin-bottom: 22px; } }
</style>
