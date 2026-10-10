<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAPI } from '@/api/payment'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { SubscriptionPlan } from '@/types/payment'
import PortalDialog from './PortalDialog.vue'

const props = defineProps<{ plan: SubscriptionPlan | null; unavailable?: string }>()
const emit = defineEmits<{ close: []; purchased: [warning: string] }>()
const { t, locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const auth = useAuthStore()
const subscriptions = useSubscriptionStore()
const submitting = ref(false)
const error = ref('')
const attempts = new Map<number, string>()
const usd = (value: number) => new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD' }).format(value)
const balance = computed(() => auth.user?.balance ?? 0)
const validation = computed(() => {
  if (!props.plan) return ''
  if (props.unavailable) return props.unavailable
  if (!['balance', 'both'].includes(props.plan.purchase_mode || '')) return text('此计划未开放余额购买。', 'Balance purchase is not available for this plan.')
  if (balance.value < props.plan.price) return text('账户余额不足，请先购买兑换码并充值。', 'Insufficient balance. Top up with a redemption code first.')
  return ''
})
function close() { if (!submitting.value) { error.value = ''; emit('close') } }
async function purchase() {
  if (!props.plan || submitting.value || validation.value) return
  const plan = props.plan
  submitting.value = true
  error.value = ''
  if (!attempts.has(plan.id)) attempts.set(plan.id, `balance-subscription-${plan.id}-${crypto.randomUUID()}`)
  try {
    await paymentAPI.purchaseSubscriptionWithBalance(plan.id, attempts.get(plan.id)!)
    attempts.delete(plan.id)
    const refresh = await Promise.allSettled([auth.refreshUser(), subscriptions.fetchActiveSubscriptions(true)])
    const warning = refresh.some(result => result.status === 'rejected')
      ? text('订阅已购买，账户信息暂未刷新，请重新加载。', 'Subscription purchased. Refresh the page to update account information.') : ''
    emit('purchased', warning)
  } catch (cause) {
    error.value = extractI18nErrorMessage(cause, t, 'payment.errors', text('订阅未完成，请重试。', 'Could not complete the subscription. Please retry.'))
  } finally { submitting.value = false }
}
</script>

<template>
  <PortalDialog :open="!!plan" :title="text('确认订阅', 'Confirm subscription')" @close="close">
    <template v-if="plan">
      <h3>{{ plan.name }}</h3>
      <dl class="subscription-confirm-summary">
        <div><dt>{{ text('账户余额', 'Account balance') }}</dt><dd>{{ usd(balance) }}</dd></div>
        <div><dt>{{ text('本次扣除', 'Amount to deduct') }}</dt><dd><strong>{{ usd(plan.price) }}</strong></dd></div>
        <div><dt>{{ text('支付方式', 'Payment method') }}</dt><dd>{{ text('账户余额', 'Account balance') }}</dd></div>
      </dl>
      <p class="portal-muted">{{ text('确认后从余额扣款并开通或续期此订阅。', 'Confirm to deduct your balance and activate or extend this subscription.') }}</p>
      <p v-if="validation" class="portal-error" role="alert">{{ validation }}</p>
      <p v-if="error" class="portal-error" role="alert">{{ error }}</p>
      <footer class="portal-dialog-actions">
        <button class="portal-button secondary" :disabled="submitting" @click="close">{{ text('取消', 'Cancel') }}</button>
        <RouterLink v-if="balance < plan.price" class="portal-button" to="/purchase" @click="close">{{ text('前往充值', 'Top up') }}</RouterLink>
        <button v-else class="portal-button" :disabled="submitting || !!validation" @click="purchase">{{ submitting ? text('处理中…', 'Processing…') : text('确认使用余额订阅', 'Confirm balance purchase') }}</button>
      </footer>
    </template>
  </PortalDialog>
</template>

<style scoped>
.subscription-confirm-summary { display: grid; gap: 14px; margin: 24px 0; padding: 24px 0; border-block: 1px solid #efede3; }
.subscription-confirm-summary > div { display: flex; justify-content: space-between; gap: 20px; }
.subscription-confirm-summary dt { color: #777; }
.subscription-confirm-summary dd { font-variant-numeric: tabular-nums; }
.portal-error { margin-top: 16px; }
</style>
