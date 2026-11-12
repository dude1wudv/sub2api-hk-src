<script setup lang="ts">
import PortalPageHeader from '@/components/portal/PortalPageHeader.vue'
import { useI18n } from 'vue-i18n'
import { PATRICK_CARD_ORDER_URL } from '@/utils/portalPurchase'
import PortalRedeemHistory from '@/components/portal/PortalRedeemHistory.vue'

const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
</script>

<template>
  <div class="portal-page redemption-page">
    <PortalPageHeader :title="copy('兑换记录', 'Redemption history')" />
    <nav class="redemption-tabs" :aria-label="copy('充值与兑换记录', 'Top up and redemption history')">
      <RouterLink to="/purchase">{{ copy('充值与兑换', 'Top up & redeem') }}</RouterLink>
      <RouterLink to="/orders" class="active" aria-current="page">{{ copy('兑换记录', 'Redemption history') }}</RouterLink>
    </nav>
    <aside class="redemption-orders-note">
      <div>
        <h2>{{ copy('本站兑换记录', 'Redemptions on patrickapi') }}</h2>
        <p>{{ copy('这里显示在本站完成的兑换及额度调整。发卡网的购买订单、付款和发货记录，请前往发卡网查询。', 'This page shows redemptions and allowance adjustments on patrickapi. Purchase, payment, and delivery records are available in the card shop.') }}</p>
      </div>
      <a :href="PATRICK_CARD_ORDER_URL" target="_blank" rel="noopener noreferrer" class="portal-button secondary">{{ copy('查询发卡网订单', 'Card-shop orders') }} <span aria-hidden="true">↗</span></a>
    </aside>
    <PortalRedeemHistory />
  </div>
</template>

<style scoped>
.redemption-tabs { display: flex; gap: 28px; border-bottom: 1px solid rgb(var(--line)); margin-bottom: 26px; }
.redemption-tabs a { padding-bottom: 13px; color: rgb(var(--ink-3)); font: 500 14.5px/1.5 var(--font-sans); text-decoration: none; transition: color var(--motion-fast) ease; }
.redemption-tabs a:hover { color: rgb(var(--ink)); }
.redemption-tabs .active { color: rgb(var(--ink)); font-weight: 600; border-bottom: 2px solid rgb(var(--brand-moss)); margin-bottom: -1px; }
.redemption-orders-note { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 22px 24px; margin-bottom: 30px; border: 1px solid rgb(var(--brand-moss) / .24); border-radius: var(--radius-card); background: linear-gradient(135deg, rgb(var(--brand-moss) / .08) 0%, rgb(var(--surface)) 70%); box-shadow: var(--shadow-panel); }
.redemption-orders-note h2 { margin: 0 0 6px; font: 650 17px/1.4 var(--font-sans); letter-spacing: -0.01em; }
.redemption-orders-note p { max-width: 720px; margin: 0; color: rgb(var(--ink-2)); font-size: 13.5px; line-height: 1.7; }
.redemption-orders-note > a { flex-shrink: 0; }
@media (max-width: 899px) { .redemption-orders-note { align-items: flex-start; flex-direction: column; gap: 16px; } }
@media (max-width: 767px) { .redemption-orders-note { padding: 18px; margin-bottom: 24px; } .redemption-tabs { gap: 22px; margin-bottom: 20px; } }
</style>
