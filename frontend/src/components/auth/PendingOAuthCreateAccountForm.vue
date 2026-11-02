<template>
  <form class="space-y-3" @submit.prevent="handleSubmit">
    <div v-if="settingsError" class="text-sm text-red-600" role="alert">{{ settingsError }} <button type="button" :disabled="settingsLoading || retryAfter > 0" @click="loadPendingSettings">{{ isChinese ? '重试设置' : 'Retry settings' }}</button></div>
    <p v-else-if="!settingsLoaded" class="text-sm text-gray-500" role="status">{{ isChinese ? '正在读取注册设置…' : 'Loading registration settings…' }}</p>
    <input
      v-model="email"
      :data-testid="`${testIdPrefix}-create-account-email`"
      type="email"
      class="input w-full"
      :placeholder="t('auth.emailPlaceholder')"
      :disabled="isSubmitting || isSendingCode"
    />
    <input
      v-model="password"
      :data-testid="`${testIdPrefix}-create-account-password`"
      type="password"
      autocomplete="new-password"
      :minlength="passwordPolicy.min_length"
      aria-describedby="pending-password-hint"
      class="input w-full"
      :placeholder="t('auth.passwordPlaceholder')"
      :disabled="isSubmitting"
    />
    <p v-if="retryAfter > 0" class="text-sm text-gray-500" role="status">{{ isChinese ? `请在 ${retryAfter} 秒后重试。` : `Try again in ${retryAfter} seconds.` }}</p>
    <p id="pending-password-hint" class="text-xs text-gray-500">{{ passwordHint }}</p>
    <div v-if="captchaEnabled" class="space-y-2">
      <TurnstileWidget
        ref="turnstileRef"
        :site-key="turnstileSiteKey"
        :turnstile-enabled="turnstileEnabled"
        :turnstile-site-key="turnstileSiteKey"
        :tencent-enabled="tencentCaptchaEnabled"
        :tencent-app-id="tencentCaptchaAppId"
        :tencent-region="tencentCaptchaRegion"
        :aliyun-enabled="aliyunCaptchaEnabled"
        :aliyun-scene-id="aliyunCaptchaSceneId"
        :aliyun-prefix="aliyunCaptchaPrefix"
        :aliyun-region="aliyunCaptchaRegion"
        @verify="onTurnstileVerify"
        @expire="onTurnstileExpire"
        @error="onTurnstileError"
      />
    </div>
    <div v-if="emailVerifyEnabled" class="flex gap-3">
      <input
        v-model="verifyCode"
        :data-testid="`${testIdPrefix}-create-account-verify-code`"
        type="text"
        inputmode="numeric"
        maxlength="6"
        class="input min-w-0 flex-1"
        placeholder="123456"
        :disabled="isSubmitting"
      />
      <button
        :data-testid="`${testIdPrefix}-create-account-send-code`"
        type="button"
        class="btn btn-secondary shrink-0"
        :disabled="isSubmitting || !settingsLoaded || retryAfter > 0 || isSendingCode || countdown > 0 || !email.trim() || (turnstileEnabled && !turnstileToken)"
        @click="handleSendCode"
      >
        {{
          isSendingCode
            ? t('auth.sendingCode')
            : countdown > 0
              ? t('auth.resendCountdown', { countdown })
              : t('auth.sendCode')
        }}
      </button>
    </div>
    <p v-if="emailVerifyEnabled && sendCodeSuccess" class="text-sm text-green-600 dark:text-green-400">
      {{ t('auth.codeSentSuccess') }}
    </p>
    <p v-else-if="emailVerifyEnabled" class="text-xs text-gray-500 dark:text-dark-400">
      {{ t('auth.verificationCodeHint') }}
    </p>
    <input
      v-if="invitationCodeEnabled"
      v-model="invitationCode"
      :data-testid="`${testIdPrefix}-create-account-invitation-code`"
      type="text"
      class="input w-full"
      :placeholder="t('auth.invitationCodePlaceholder')"
      :disabled="isSubmitting"
    />
    <button
      :data-testid="`${testIdPrefix}-create-account-submit`"
      type="button"
      class="btn btn-primary w-full"
      :disabled="isSubmitting || !settingsLoaded || retryAfter > 0 || !email.trim() || validateNewPassword(password, passwordPolicy) !== null || (invitationCodeEnabled && !invitationCode.trim()) || (turnstileEnabled && !turnstileToken)"
      @click="handleSubmit"
    >
      {{ isSubmitting ? t('common.processing') : t('auth.createAccount') }}
    </button>
    <button
      type="button"
      class="btn btn-secondary w-full"
      :disabled="isSubmitting"
      @click="emitSwitchToBind"
    >
      {{ t('auth.alreadyHaveAccount') }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { resolvePasswordPolicy, validateNewPassword, newPasswordError, passwordPolicyHint } from '@/utils/passwordPolicy'
import TurnstileWidget from '@/components/CaptchaChallenge.vue'
import { getPublicSettings, sendPendingOAuthVerifyCode } from '@/api/auth'
import { useAppStore } from '@/stores'
import { userFacingError } from '@/utils/userFacingError'
import { useRetryCooldown } from '@/composables/useRetryCooldown'

export type PendingOAuthCreateAccountPayload = {
  email: string
  password: string
  verifyCode: string
  turnstileToken?: string
  tencentCaptchaTicket?: string
  tencentCaptchaRandstr?: string
  invitationCode?: string
}

const props = defineProps<{
  initialEmail: string
  testIdPrefix: string
  isSubmitting: boolean
  errorMessage?: string
}>()

const emit = defineEmits<{
  submit: [payload: PendingOAuthCreateAccountPayload]
  switchToBind: [email: string]
}>()

const { t, locale } = useI18n()
const { remaining: retryAfter, start: startRetry } = useRetryCooldown()
const isChinese = computed(() => (locale?.value || 'en').startsWith('zh'))
const appStore = useAppStore()
const passwordPolicy = computed(() => resolvePasswordPolicy(appStore.cachedPublicSettings))
const passwordHint = computed(() => passwordPolicyHint(locale?.value || 'en', passwordPolicy.value))

const settingsLoaded = ref(false)
const settingsLoading = ref(false)
const settingsError = ref('')
const email = ref('')
const password = ref('')
const verifyCode = ref('')
const invitationCode = ref('')
const isSendingCode = ref(false)
const sendCodeError = ref('')
const sendCodeSuccess = ref(false)
const countdown = ref(0)
const invitationCodeEnabled = ref(false)
const emailVerifyEnabled = ref(true)
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const tencentCaptchaEnabled = ref(false)
const tencentCaptchaAppId = ref('')
const tencentCaptchaRegion = ref('cn')
const aliyunCaptchaEnabled = ref(false)
const aliyunCaptchaSceneId = ref('')
const aliyunCaptchaPrefix = ref('')
const aliyunCaptchaRegion = ref('cn')
const turnstileToken = ref('')
const tencentCaptchaRandstr = ref('')
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const aliyunCaptchaReady = computed(
  () =>
    aliyunCaptchaEnabled.value &&
    Boolean(aliyunCaptchaSceneId.value) &&
    Boolean(aliyunCaptchaPrefix.value)
)
// 动作触发式验证码（腾讯/阿里云）：发送验证码、提交时弹窗验证
const actionCaptchaEnabled = computed(
  () =>
    (tencentCaptchaEnabled.value && Boolean(tencentCaptchaAppId.value)) ||
    aliyunCaptchaReady.value
)
const captchaEnabled = computed(
  () =>
    (turnstileEnabled.value && Boolean(turnstileSiteKey.value)) || actionCaptchaEnabled.value
)

let countdownTimer: ReturnType<typeof setInterval> | null = null
let disposed = false

watch(
  () => props.initialEmail,
  value => {
    email.value = value || ''
  },
  { immediate: true }
)

watch(sendCodeError, value => {
  if (value) {
    appStore.showError(value)
  }
})

watch(
  () => props.errorMessage,
  value => {
    if (value) {
      appStore.showError(value)
      if (captchaEnabled.value) {
        resetTurnstile()
      }
    }
  }
)

function clearCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function startCountdown(seconds: number) {
  clearCountdown()
  countdown.value = Math.max(0, seconds)

  if (countdown.value <= 0) {
    return
  }

  countdownTimer = setInterval(() => {
    if (countdown.value <= 1) {
      countdown.value = 0
      clearCountdown()
      return
    }

    countdown.value -= 1
  }, 1000)
}

function getRequestErrorMessage(error: unknown, fallback: string): string {
  startRetry(error)
  return userFacingError(error, locale?.value || 'en', fallback)
}

function resetTurnstile() {
  turnstileToken.value = ''
  tencentCaptchaRandstr.value = ''
  turnstileRef.value?.reset()
}

function onTurnstileVerify(token: string, randstr = '') {
  turnstileToken.value = token
  tencentCaptchaRandstr.value = randstr
  sendCodeError.value = ''
}

function onTurnstileExpire() {
  turnstileToken.value = ''
  tencentCaptchaRandstr.value = ''
  sendCodeError.value = t('auth.turnstileExpired')
}

function onTurnstileError() {
  turnstileToken.value = ''
  tencentCaptchaRandstr.value = ''
  sendCodeError.value = t('auth.turnstileFailed')
}

async function acquireActionProof(): Promise<boolean> {
  if (!actionCaptchaEnabled.value) return true

  const proof = await turnstileRef.value?.verifyAction()
  if (!proof) return false

  turnstileToken.value = proof.token
  tencentCaptchaRandstr.value = proof.randstr
  return true
}

async function handleSendCode() {
  if (!settingsLoaded.value || isSendingCode.value || props.isSubmitting || countdown.value > 0 || retryAfter.value > 0) return
  const trimmedEmail = email.value.trim()
  if (!trimmedEmail) {
    return
  }

  if (turnstileEnabled.value && !turnstileToken.value) {
    sendCodeError.value = t('auth.completeVerification')
    return
  }

  if (!(await acquireActionProof())) {
    return
  }

  isSendingCode.value = true
  sendCodeError.value = ''
  sendCodeSuccess.value = false

  try {
    const response = await sendPendingOAuthVerifyCode({
      email: trimmedEmail,
      turnstile_token:
        turnstileEnabled.value || aliyunCaptchaEnabled.value ? turnstileToken.value : undefined,
      tencent_captcha_ticket: tencentCaptchaEnabled.value ? turnstileToken.value : undefined,
      tencent_captcha_randstr: tencentCaptchaEnabled.value ? tencentCaptchaRandstr.value : undefined
    })
    if (disposed) return
    sendCodeSuccess.value = true
    startCountdown(response.countdown)
  } catch (error: unknown) {
    sendCodeError.value = getRequestErrorMessage(error, t('auth.sendCodeFailed'))
  } finally {
    if (captchaEnabled.value) {
      resetTurnstile()
    }
    isSendingCode.value = false
  }
}

async function handleSubmit() {
  if (!settingsLoaded.value || props.isSubmitting || retryAfter.value > 0) return
  const trimmedEmail = email.value.trim()
  if (!trimmedEmail || validateNewPassword(password.value, passwordPolicy.value) !== null) {
    sendCodeError.value = !trimmedEmail ? t('auth.emailRequired') : newPasswordError(password.value, locale?.value || 'en', passwordPolicy.value)
    return
  }

  // Turnstile 票据一次性：发送验证码已消耗上一枚，reset 后要等新票据回调。
  // 缺票时不能提交——create-account 端点会校验验证码，空 token 直接被判失败。
  // 表单的隐式提交（输入框回车）绕得过按钮的 disabled，所以这里必须再挡一次。
  if (turnstileEnabled.value && !turnstileToken.value) {
    sendCodeError.value = t('auth.completeVerification')
    return
  }

  if (!(await acquireActionProof())) {
    return
  }

  emit('submit', {
    email: trimmedEmail,
    password: password.value,
    verifyCode: emailVerifyEnabled.value ? verifyCode.value.trim() : '',
    ...((turnstileEnabled.value || aliyunCaptchaEnabled.value) && turnstileToken.value
      ? { turnstileToken: turnstileToken.value }
      : {}),
    ...(tencentCaptchaEnabled.value && turnstileToken.value
      ? {
          tencentCaptchaTicket: turnstileToken.value,
          tencentCaptchaRandstr: tencentCaptchaRandstr.value
        }
      : {}),
    invitationCode: invitationCode.value.trim() || undefined
  })

  if (actionCaptchaEnabled.value) {
    resetTurnstile()
  }
}

function emitSwitchToBind() {
  emit('switchToBind', email.value.trim())
}

async function loadPendingSettings() {
  if (settingsLoading.value || retryAfter.value > 0) return
  settingsLoading.value = true
  settingsError.value = ''
  try {
    const settings = await getPublicSettings()
    if (disposed) return
    invitationCodeEnabled.value = settings.invitation_code_enabled === true
    emailVerifyEnabled.value = settings.email_verify_enabled !== false
    turnstileEnabled.value = settings.turnstile_enabled === true
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    tencentCaptchaEnabled.value = settings.tencent_captcha_enabled === true
    tencentCaptchaAppId.value = settings.tencent_captcha_app_id || ''
    tencentCaptchaRegion.value = settings.tencent_captcha_region || 'cn'
    aliyunCaptchaEnabled.value = settings.aliyun_captcha_enabled === true
    aliyunCaptchaSceneId.value = settings.aliyun_captcha_scene_id || ''
    aliyunCaptchaPrefix.value = settings.aliyun_captcha_prefix || ''
    aliyunCaptchaRegion.value = settings.aliyun_captcha_region || 'cn'
    settingsLoaded.value = true
  } catch (error) {
    startRetry(error)
    settingsError.value = userFacingError(error, locale?.value || 'en', isChinese.value ? '注册设置读取失败，请重试。' : 'Could not load registration settings. Please retry.')
  } finally { settingsLoading.value = false }
}
onMounted(() => { void loadPendingSettings() })

onUnmounted(() => {
  disposed = true
  clearCountdown()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
