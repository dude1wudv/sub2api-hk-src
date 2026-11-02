<template>
  <AuthLayout>
    <div class="space-y-6">
      <!-- Title -->
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.resetPasswordTitle') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('auth.resetPasswordHint') }}
        </p>
      </div>

      <!-- Invalid Link State -->
      <div v-if="isInvalidLink" class="space-y-6">
        <div class="rounded-xl border border-amber-200 bg-amber-50 p-6 dark:border-amber-800/50 dark:bg-amber-900/20">
          <div class="flex flex-col items-center gap-4 text-center">
            <div class="flex h-12 w-12 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-800/50">
              <Icon name="exclamationCircle" size="lg" class="text-amber-600 dark:text-amber-400" />
            </div>
            <div>
              <h3 class="text-lg font-semibold text-amber-800 dark:text-amber-200">
                {{ t('auth.invalidResetLink') }}
              </h3>
              <p class="mt-2 text-sm text-amber-700 dark:text-amber-300">
                {{ t('auth.invalidResetLinkHint') }}
              </p>
            </div>
          </div>
        </div>

        <div class="text-center">
          <router-link
            to="/forgot-password"
            class="inline-flex items-center gap-2 font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
          >
            {{ t('auth.requestNewResetLink') }}
          </router-link>
        </div>
      </div>

      <div v-else-if="settingsError" role="alert" class="rounded-xl border border-red-200 p-4 text-sm">
        <p>{{ settingsError }}</p>
        <button type="button" class="btn btn-secondary mt-3" :disabled="retryAfter > 0" @click="loadResetSettings(true)">{{ retryAfter > 0 ? `${retryAfter}s` : copy('重试', 'Retry') }}</button>
      </div>
      <p v-else-if="!settingsLoaded" role="status" class="text-sm text-gray-500">{{ copy('正在读取密码重置设置…', 'Loading password reset settings…') }}</p>
      <div v-else-if="!resetEnabled" role="status" class="rounded-xl border border-gray-200 p-4 text-sm">{{ copy('本站暂未开放密码重置，请联系帮助与客服。', 'Password reset is not available on this site. Please contact support.') }}</div>
      <!-- Success State -->
      <div v-else-if="isSuccess" class="space-y-6">
        <div class="rounded-xl border border-green-200 bg-green-50 p-6 dark:border-green-800/50 dark:bg-green-900/20">
          <div class="flex flex-col items-center gap-4 text-center">
            <div class="flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-800/50">
              <Icon name="checkCircle" size="lg" class="text-green-600 dark:text-green-400" />
            </div>
            <div>
              <h3 class="text-lg font-semibold text-green-800 dark:text-green-200">
                {{ t('auth.passwordResetSuccess') }}
              </h3>
              <p class="mt-2 text-sm text-green-700 dark:text-green-300">
                {{ t('auth.passwordResetSuccessHint') }}
              </p>
            </div>
          </div>
        </div>

        <div class="text-center">
          <router-link
            to="/login"
            class="btn btn-primary inline-flex items-center gap-2"
          >
            <Icon name="login" size="md" />
            {{ t('auth.signIn') }}
          </router-link>
        </div>
      </div>

      <!-- Form State -->
      <form v-else @submit.prevent="handleSubmit" class="space-y-5">
        <!-- Email (readonly) -->
        <div>
          <label for="email" class="input-label">
            {{ t('auth.emailLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="mail" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="email"
              :value="email"
              type="email"
              readonly
              disabled
              class="input pl-11 bg-gray-50 dark:bg-dark-700"
            />
          </div>
        </div>

        <!-- New Password Input -->
        <div>
          <label for="password" class="input-label">
            {{ t('auth.newPassword') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="lock" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="password"
              v-model="formData.password"
              :type="showPassword ? 'text' : 'password'"
              required
              autocomplete="new-password"
              :disabled="isLoading"
              class="input pl-11 pr-11"
              :class="{ 'input-error': errors.password }"
              :placeholder="t('auth.newPasswordPlaceholder')"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300"
            >
              <Icon v-if="showPassword" name="eyeOff" size="md" />
              <Icon v-else name="eye" size="md" />
            </button>
          </div>
        </div>

        <p class="text-sm text-gray-500">{{ passwordHint }}</p>
        <!-- Confirm Password Input -->
        <div>
          <label for="confirmPassword" class="input-label">
            {{ t('auth.confirmPassword') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="lock" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="confirmPassword"
              v-model="formData.confirmPassword"
              :type="showConfirmPassword ? 'text' : 'password'"
              required
              autocomplete="new-password"
              :disabled="isLoading"
              class="input pl-11 pr-11"
              :class="{ 'input-error': errors.confirmPassword }"
              :placeholder="(isChinese ? '请再次输入密码' : 'Enter the password again')"
            />
            <button
              type="button"
              @click="showConfirmPassword = !showConfirmPassword"
              class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300"
            >
              <Icon v-if="showConfirmPassword" name="eyeOff" size="md" />
              <Icon v-else name="eye" size="md" />
            </button>
          </div>
        </div>

        <p v-if="retryAfter > 0" role="status" class="text-sm text-gray-500">{{ copy(`请在 ${retryAfter} 秒后重试。`, `Try again in ${retryAfter} seconds.`) }}</p>
        <!-- Submit Button -->
        <button
          type="submit"
          :disabled="isLoading || retryAfter > 0"
          class="btn btn-primary w-full"
        >
          <svg
            v-if="isLoading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          <Icon v-else name="checkCircle" size="md" class="mr-2" />
          {{ isLoading ? t('auth.resettingPassword') : t('auth.resetPassword') }}
        </button>
      </form>
    </div>

    <!-- Footer -->
    <template #footer>
      <p class="text-gray-500 dark:text-dark-400">
        {{ t('auth.rememberedPassword') }}
        <router-link
          to="/login"
          class="font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
        >
          {{ t('auth.signIn') }}
        </router-link>
      </p>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { resolvePasswordPolicy, validateNewPassword, newPasswordError, passwordPolicyHint } from '@/utils/passwordPolicy'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { resetPassword, getPublicSettings } from '@/api/auth'
import { userFacingError } from '@/utils/userFacingError'
import { extractApiErrorCode } from '@/utils/apiError'
import { useRetryCooldown } from '@/composables/useRetryCooldown'

const { t, locale } = useI18n()
const isChinese = computed(() => (locale?.value || 'en').startsWith('zh'))
const copy = (zh: string, en: string) => isChinese.value ? zh : en
const { remaining: retryAfter, start: startRetry } = useRetryCooldown()
const settingsLoaded = ref(false)
const settingsLoading = ref(false)
const settingsError = ref('')
const resetEnabled = ref(false)

// ==================== Router & Stores ====================

const route = useRoute()
const appStore = useAppStore()
const passwordPolicy = computed(() => resolvePasswordPolicy(appStore.cachedPublicSettings))
const passwordHint = computed(() => passwordPolicyHint(locale?.value || 'en', passwordPolicy.value))

// ==================== State ====================

const isLoading = ref<boolean>(false)
const isSuccess = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)
const showConfirmPassword = ref<boolean>(false)

// URL parameters
const email = ref<string>('')
const token = ref<string>('')

const formData = reactive({
  password: '',
  confirmPassword: ''
})

const errors = reactive({
  password: '',
  confirmPassword: ''
})

const validationToastMessage = computed(
  () => errors.password || errors.confirmPassword || ''
)

watch(validationToastMessage, (value, previousValue) => {
  if (value && value !== previousValue) {
    appStore.showError(value)
  }
})

// Check if the reset link is valid (has email and token)
const isInvalidLink = computed(() => !email.value || !token.value)

// ==================== Lifecycle ====================

onMounted(() => {
  // Get email and token from URL query parameters
  email.value = (route.query.email as string) || ''
  token.value = (route.query.token as string) || ''

  if (!email.value || !token.value) {
    appStore.showError(t('auth.invalidResetLink'))
  } else {
    void loadResetSettings()
  }
})

async function loadResetSettings(force = false) {
  if (settingsLoading.value || retryAfter.value > 0) return
  settingsLoading.value = true
  settingsError.value = ''
  try {
    const settings = (!force && (appStore.cachedPublicSettings || window.__APP_CONFIG__)) || await getPublicSettings()
    resetEnabled.value = settings.password_reset_enabled === true && settings.backend_mode_enabled !== true
    settingsLoaded.value = true
  } catch (cause) {
    startRetry(cause)
    settingsError.value = userFacingError(cause, locale?.value || 'en', copy('密码重置设置读取失败，请重试。', 'Could not load password reset settings. Please retry.'))
  } finally { settingsLoading.value = false }
}

// ==================== Validation ====================

function validateForm(): boolean {
  errors.password = ''
  errors.confirmPassword = ''

  let isValid = true

  // Password validation
  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    isValid = false
  } else if (validateNewPassword(formData.password, passwordPolicy.value) !== null) {
    errors.password = newPasswordError(formData.password, locale?.value || 'en', passwordPolicy.value)
    isValid = false
  }

  // Confirm password validation
  if (!formData.confirmPassword) {
    errors.confirmPassword = t('auth.confirmPasswordRequired')
    isValid = false
  } else if (formData.password !== formData.confirmPassword) {
    errors.confirmPassword = t('auth.passwordsDoNotMatch')
    isValid = false
  }

  return isValid
}

// ==================== Form Handlers ====================

async function handleSubmit(): Promise<void> {
  if (isLoading.value || isInvalidLink.value || !settingsLoaded.value || !resetEnabled.value || retryAfter.value > 0) return
  errorMessage.value = ''

  if (!validateForm()) {
    return
  }

  isLoading.value = true

  try {
    await resetPassword({
      email: email.value,
      token: token.value,
      new_password: formData.password
    })

    isSuccess.value = true
    appStore.showSuccess(t('auth.passwordResetSuccess'))
  } catch (error: unknown) {
    startRetry(error)
    errorMessage.value = extractApiErrorCode(error) === 'INVALID_RESET_TOKEN'
      ? t('auth.invalidOrExpiredToken')
      : userFacingError(error, locale?.value || 'en', t('auth.resetPasswordFailed'))
    appStore.showError(errorMessage.value)
  } finally {
    isLoading.value = false
  }
}
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
