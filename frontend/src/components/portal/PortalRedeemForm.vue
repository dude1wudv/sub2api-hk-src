<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { redeemAPI } from '@/api/redeem'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatPortalBalance, maskPortalRedeemCode } from '@/utils/portalPurchase'

const emit = defineEmits<{ redeemed: [] }>()
const { locale } = useI18n()
const auth = useAuthStore()
const subscriptions = useSubscriptionStore()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const code = ref('')
const submitting = ref(false)
const refreshing = ref(false)
const error = ref('')
const completed = ref<{ type: string; value: number } | null>(null)
const refreshProblems = ref<('profile' | 'subscription')[]>([])
let disposed = false

const successDescription = computed(() => {
  if (completed.value?.type === 'balance') return copy(`已兑换 ${formatPortalBalance(completed.value.value, locale.value)} 账户余额。`, `${formatPortalBalance(completed.value.value, locale.value)} has been credited to your account.`)
  if (completed.value?.type === 'subscription') return copy('订阅兑换已完成。', 'Your subscription code has been redeemed.')
  if (completed.value?.type === 'concurrency') return copy('并发额度兑换已完成。', 'Your concurrency code has been redeemed.')
  return copy('兑换已完成。', 'Your code has been redeemed.')
})
const refreshWarning = computed(() => {
  const profile = refreshProblems.value.includes('profile')
  const subscription = refreshProblems.value.includes('subscription')
  if (profile && subscription) return copy('余额和订阅信息暂未刷新。兑换已经成功，无需再次提交兑换码。', 'Balance and subscription details could not be refreshed. Redemption succeeded; do not submit the code again.')
  if (subscription) return copy('订阅信息暂未刷新。兑换已经成功，无需再次提交兑换码。', 'Subscription details could not be refreshed. Redemption succeeded; do not submit the code again.')
  return copy('余额信息暂未刷新。兑换已经成功，无需再次提交兑换码。', 'Balance details could not be refreshed. Redemption succeeded; do not submit the code again.')
})

async function refreshAccount() {
  if (refreshing.value || !completed.value) return
  refreshing.value = true
  refreshProblems.value = []
  const updateSubscription = completed.value.type === 'subscription'
  const results = await Promise.allSettled([
    Promise.resolve().then(() => auth.refreshUser()),
    ...(updateSubscription ? [Promise.resolve().then(() => {
      subscriptions.invalidateCache()
      return subscriptions.fetchActiveSubscriptions(true)
    })] : [])
  ])
  if (!disposed) {
    if (results[0]?.status === 'rejected') refreshProblems.value.push('profile')
    if (updateSubscription && results[1]?.status === 'rejected') refreshProblems.value.push('subscription')
    refreshing.value = false
  }
}

async function submit() {
  if (submitting.value || refreshing.value) return
  const submitted = code.value.trim()
  if (!submitted) {
    error.value = copy('请输入兑换码。', 'Enter a redemption code.')
    return
  }
  submitting.value = true
  error.value = ''
  completed.value = null
  refreshProblems.value = []
  try {
    const result = await redeemAPI.redeem(submitted)
    // Keep only the fields needed for the acknowledgement; the API may include the raw code.
    completed.value = { type: result.type, value: result.value }
    code.value = ''
    if (!disposed) emit('redeemed')
    await refreshAccount()
  } catch (cause) {
    if (!disposed) {
      const message = extractApiErrorMessage(cause, copy('兑换失败，请检查兑换码后重试。', 'Redemption failed. Check the code and try again.'))
      error.value = message.split(submitted).join(maskPortalRedeemCode(submitted))
    }
  } finally {
    if (!disposed) submitting.value = false
  }
}

onBeforeUnmount(() => { disposed = true; code.value = '' })
</script>

<template>
  <section class="portal-panel redeem-entry" aria-labelledby="portal-redeem-title">
    <div class="redeem-entry-heading"><span aria-hidden="true">02</span><h2 id="portal-redeem-title">{{ copy('使用兑换码', 'Redeem your code') }}</h2></div>
    <p class="portal-muted">{{ copy('购买后，将发卡网提供的兑换码粘贴到这里。', 'After purchase, paste the code provided by the card shop here.') }}</p>
    <form class="redeem-form" @submit.prevent="submit">
      <label class="portal-field" for="portal-redeem-code">
        {{ copy('兑换码', 'Redemption code') }}
        <input id="portal-redeem-code" v-model="code" type="password" autocomplete="off" autocapitalize="off" :spellcheck="false" :disabled="submitting || refreshing" :placeholder="copy('请输入或粘贴兑换码', 'Enter or paste your code')" :aria-invalid="Boolean(error)" :aria-describedby="error ? 'portal-redeem-error' : undefined" />
      </label>
      <p v-if="error" id="portal-redeem-error" class="portal-error" role="alert">{{ error }}</p>
      <button type="submit" class="portal-button" :disabled="submitting || refreshing || !code.trim()">
        {{ submitting ? copy('正在兑换…', 'Redeeming…') : copy('确认兑换', 'Redeem code') }}
      </button>
    </form>
    <div v-if="completed" class="redeem-success" role="status" aria-live="polite">
      <strong>{{ copy('兑换成功', 'Redemption successful') }}</strong>
      <p>{{ successDescription }}</p>
      <p v-if="refreshing" class="portal-muted">{{ copy('正在更新账户信息…', 'Updating account details…') }}</p>
      <div v-else-if="refreshProblems.length" class="redeem-refresh-warning">
        <p>{{ refreshWarning }}</p>
        <button type="button" class="portal-button secondary" :disabled="refreshing" @click="refreshAccount">{{ copy('刷新账户信息', 'Refresh account details') }}</button>
      </div>
      <RouterLink to="/orders" class="redeem-history-link">{{ copy('查看兑换记录', 'View redemption history') }} <span aria-hidden="true">→</span></RouterLink>
    </div>
  </section>
</template>

<style scoped>
.redeem-entry { padding: 26px; }
.redeem-entry-heading { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.redeem-entry-heading > span { display: grid; place-items: center; width: 29px; height: 29px; border: 1px solid #efede3; border-radius: 50%; font-size: 12px; }
.redeem-entry-heading h2 { margin: 0; font-size: 18px; font-weight: 400; }
.redeem-entry > p { margin: 0; font-size: 13px; line-height: 1.8; }
.redeem-form { display: grid; gap: 18px; margin-top: 24px; }
.redeem-form .portal-button { justify-self: start; }
.redeem-form input { background: #fffdf7; }
.redeem-form input:disabled { opacity: .55; }
.redeem-form .portal-error { margin: 0; font-size: 13px; }
.redeem-success { margin-top: 24px; padding-top: 20px; border-top: 1px solid #efede3; }
.redeem-success > strong { display: block; color: #25744f; font-weight: 500; }
.redeem-success p { margin: 6px 0 0; font-size: 13px; line-height: 1.8; }
.redeem-history-link { display: inline-flex; align-items: center; gap: 6px; margin-top: 12px; color: #000; font-size: 13px; text-decoration: underline; text-underline-offset: 3px; }
.redeem-refresh-warning { margin-top: 12px; color: #796230; }
.redeem-refresh-warning .portal-button { margin-top: 12px; font-size: 12px; }
@media (max-width: 739px) { .redeem-entry { padding: 20px; } }
</style>
