<template>
  <div class="mx-auto max-w-[1160px] space-y-6 pb-10" data-testid="portal-account">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ copy('账户设置', 'Account settings') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ copy('管理个人资料、第三方登录绑定、安全凭据与通知偏好。', 'Manage your profile, connected identities, security credentials, and preferences.') }}
      </p>
    </header>

    <p v-if="notice" class="rounded-xl border border-[var(--atelier-ok)]/25 bg-[var(--atelier-ok-bg)] px-4 py-3 text-sm text-[var(--atelier-ok)]" role="status">
      {{ notice }}
    </p>
    <p v-if="loadError" class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]" role="alert">
      <span>{{ copy('账户资料读取失败：', 'Could not load your profile: ') }}{{ loadError }}</span>
      <UiButton variant="secondary" size="sm" :disabled="loading || profileRetryAfter > 0" @click="loadProfile">
        {{ profileRetryAfter > 0 ? `${profileRetryAfter}s` : copy('重试资料', 'Retry profile') }}
      </UiButton>
    </p>
    <p v-if="settingsError" class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]" role="alert">
      <span>{{ copy('账户功能设置读取失败：', 'Could not load account settings: ') }}{{ settingsError }}</span>
      <UiButton variant="secondary" size="sm" :disabled="settingsLoading || settingsRetryAfter > 0" @click="loadSettings(true)">
        {{ settingsRetryAfter > 0 ? `${settingsRetryAfter}s` : copy('重试设置', 'Retry settings') }}
      </UiButton>
    </p>
    <p v-if="loading && !user" class="atelier-surface py-12 text-center text-sm text-[var(--atelier-ink-muted)]" role="status">
      {{ copy('正在读取账户…', 'Loading account…') }}
    </p>

    <div v-if="user" class="grid grid-cols-1 items-start gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
      <!-- Left Identity & Anchor Nav -->
      <aside class="atelier-surface p-5 lg:sticky lg:top-16">
        <div class="flex items-center gap-3">
          <img
            v-if="safeAvatarUrl"
            :src="safeAvatarUrl"
            :alt="user.username || 'Avatar'"
            class="h-11 w-11 shrink-0 rounded-full border border-[var(--atelier-border)] object-cover"
          />
          <span
            v-else
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] font-mono text-sm font-semibold text-[var(--atelier-ink)]"
            aria-hidden="true"
          >
            {{ initial }}
          </span>
          <div class="min-w-0">
            <strong class="block truncate text-sm font-semibold text-[var(--atelier-ink)]">
              {{ user.username?.trim() || copy('未设置昵称', 'Display name not set') }}
            </strong>
            <span class="block truncate text-xs text-[var(--atelier-ink-muted)]">
              {{ displayEmail || copy('未绑定邮箱', 'No email bound') }}
            </span>
          </div>
        </div>

        <nav class="mt-5 flex flex-col gap-1 border-t border-[var(--atelier-border)] pt-4" :aria-label="copy('账户设置分区', 'Account settings sections')">
          <a
            href="#account-identity"
            class="flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          >
            <span>{{ copy('资料与登录', 'Profile & sign-in') }}</span>
            <span aria-hidden="true" class="text-[var(--atelier-ink-muted)]">↗</span>
          </a>
          <a
            href="#account-security"
            class="flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          >
            <span>{{ copy('账户安全', 'Account security') }}</span>
            <span aria-hidden="true" class="text-[var(--atelier-ink-muted)]">↗</span>
          </a>
          <a
            v-if="settings?.balance_low_notify_enabled"
            href="#account-notifications"
            class="flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          >
            <span>{{ copy('余额通知', 'Balance notifications') }}</span>
            <span aria-hidden="true" class="text-[var(--atelier-ink-muted)]">↗</span>
          </a>
          <a
            href="#account-appearance"
            class="flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-medium text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          >
            <span>{{ copy('动效偏好', 'Motion preferences') }}</span>
            <span aria-hidden="true" class="text-[var(--atelier-ink-muted)]">↗</span>
          </a>
        </nav>
      </aside>

      <!-- Right Sections -->
      <div class="space-y-5">
        <!-- Profile & Sign-in -->
        <section id="account-identity" class="atelier-surface scroll-mt-16 overflow-hidden" :aria-label="copy('账户信息', 'Account information')">
          <header class="border-b border-[var(--atelier-border)] px-6 py-4">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ copy('资料与登录', 'Profile & sign-in') }}
            </h2>
          </header>

          <div class="divide-y divide-[var(--atelier-border)] px-6">
            <p v-if="bindingError" class="py-3 text-xs text-[var(--atelier-err)]" role="alert">{{ bindingError }}</p>

            <div class="flex flex-wrap items-center justify-between gap-4 py-4">
              <div class="flex items-center gap-3.5">
                <img
                  v-if="safeAvatarUrl"
                  :src="safeAvatarUrl"
                  :alt="user.username || 'Avatar'"
                  class="h-10 w-10 shrink-0 rounded-full border border-[var(--atelier-border)] object-cover"
                />
                <span
                  v-else
                  class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[var(--atelier-border-strong)] bg-[var(--atelier-subtle)] font-mono text-sm font-semibold text-[var(--atelier-ink)]"
                  aria-hidden="true"
                >
                  {{ initial }}
                </span>
                <div>
                  <strong class="block text-sm font-medium text-[var(--atelier-ink)]">
                    {{ user.username?.trim() || copy('未设置昵称', 'Display name not set') }}
                  </strong>
                  <small class="text-xs text-[var(--atelier-ink-muted)]">{{ copy('账户昵称与头像', 'Display name & avatar') }}</small>
                </div>
              </div>
              <div class="flex flex-wrap gap-2">
                <UiButton variant="secondary" size="sm" @click="openEditor('name')">{{ copy('更改昵称', 'Edit name') }}</UiButton>
                <UiButton variant="secondary" size="sm" @click="openEditor('avatar')">{{ copy('设置头像', 'Avatar') }}</UiButton>
              </div>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-4 py-4">
              <div class="flex items-center gap-3.5">
                <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] text-[var(--atelier-ink-secondary)]" aria-hidden="true">
                  <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
                    <rect x="3" y="5" width="18" height="14" rx="2" />
                    <path d="m4 7 8 6 8-6" />
                  </svg>
                </span>
                <div>
                  <strong class="block text-sm font-medium text-[var(--atelier-ink)]">{{ copy('邮箱', 'Email') }}</strong>
                  <small class="text-xs text-[var(--atelier-ink-muted)]">{{ displayEmail || copy('尚未绑定', 'Not connected') }}</small>
                </div>
              </div>
              <UiButton variant="secondary" size="sm" @click="openEditor('email')">
                {{ emailBound ? copy('更改', 'Change') : copy('绑定', 'Connect') }}
              </UiButton>
            </div>

            <div v-for="item in providers" :key="item.provider" class="flex flex-wrap items-center justify-between gap-4 py-4">
              <div class="flex items-center gap-3.5">
                <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] font-mono text-xs font-semibold text-[var(--atelier-ink)]" aria-hidden="true">
                  {{ item.monogram }}
                </span>
                <div>
                  <strong class="block text-sm font-medium text-[var(--atelier-ink)]">{{ item.label }}</strong>
                  <small class="block text-xs text-[var(--atelier-ink-muted)]">
                    {{ item.details?.display_name || item.details?.subject_hint || (item.bound ? copy('已绑定', 'Connected') : copy('尚未绑定', 'Not connected')) }}
                  </small>
                  <small v-if="item.bound && item.details?.bound_count && item.details.bound_count > 1" class="block text-xs text-[var(--atelier-ink-muted)]">
                    {{ copy(`${item.details.bound_count} 个关联身份`, `${item.details.bound_count} connected identities`) }}
                  </small>
                  <small v-if="item.bound && !item.canUnbind" class="block text-xs text-[var(--atelier-ink-muted)]">
                    {{ copy('保留至少一种可用的登录方式', 'Keep at least one available sign-in method') }}
                  </small>
                  <small v-else-if="!item.bound && item.provider === 'wechat' && !item.canBind" class="block text-xs text-[var(--atelier-ink-muted)]">
                    {{ wechatHint }}
                  </small>
                </div>
              </div>
              <UiButton v-if="item.canUnbind" variant="secondary" size="sm" :disabled="busy" @click="openUnbind(item.provider)">
                {{ copy('解绑', 'Disconnect') }}
              </UiButton>
              <UiButton v-else-if="item.canBind" variant="secondary" size="sm" :disabled="busy || actionRetryAfter > 0" @click="bindProvider(item.provider)">
                {{ copy('绑定', 'Connect') }}
              </UiButton>
              <span v-else class="inline-flex items-center gap-1.5 rounded-full bg-[var(--atelier-subtle)] px-2.5 py-0.5 text-xs text-[var(--atelier-ink-muted)]">
                <UiStatusDot :tone="item.bound ? 'ok' : 'neutral'" />
                {{ item.bound ? copy('已关联', 'Connected') : copy('暂不可用', 'Unavailable') }}
              </span>
            </div>
          </div>
        </section>

        <!-- Account Security -->
        <section id="account-security" class="atelier-surface scroll-mt-16 overflow-hidden" :aria-label="copy('账户安全', 'Account security')">
          <header class="border-b border-[var(--atelier-border)] px-6 py-4">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ copy('账户安全', 'Account security') }}
            </h2>
          </header>

          <div class="divide-y divide-[var(--atelier-border)] px-6">
            <div class="flex flex-wrap items-center justify-between gap-4 py-4">
              <div>
                <strong class="block text-sm font-medium text-[var(--atelier-ink)]">{{ copy('登录密码', 'Password') }}</strong>
                <small class="text-xs text-[var(--atelier-ink-muted)]">{{ copy('定期更新密码以保护账户安全', 'Update your password regularly to keep your account secure') }}</small>
              </div>
              <UiButton variant="secondary" size="sm" @click="openEditor('password')">
                {{ copy('更改', 'Change') }}
              </UiButton>
            </div>

            <div class="py-2">
              <PortalAccountSecurity :passkey-enabled="settings?.passkey_enabled === true" />
            </div>

            <div class="flex flex-wrap items-center justify-between gap-4 py-4">
              <div class="max-w-lg">
                <strong class="block text-sm font-medium text-[var(--atelier-ink)]">{{ copy('登录设备与会话', 'Active device sessions') }}</strong>
                <small class="text-xs text-[var(--atelier-ink-muted)]">
                  {{ copy('立即注销所有浏览器与设备的登录态，当前设备也需要重新登录。', 'Sign out every active session across all devices immediately.') }}
                </small>
              </div>
              <UiButton
                variant="secondary"
                size="sm"
                data-testid="revoke-all-sessions"
                :disabled="busy"
                @click="openEditor('revokeAll')"
              >
                {{ copy('退出全部设备', 'Sign out all devices') }}
              </UiButton>
            </div>
          </div>
        </section>

        <!-- Notifications -->
        <section
          v-if="settings?.balance_low_notify_enabled"
          id="account-notifications"
          class="atelier-surface scroll-mt-16 p-6"
          :aria-label="copy('通知', 'Notifications')"
        >
          <PortalAccountNotifications :default-threshold="settings.balance_low_notify_threshold ?? 0" />
        </section>

        <!-- Motion Preferences -->
        <section id="account-appearance" class="atelier-surface scroll-mt-16 overflow-hidden">
          <header class="border-b border-[var(--atelier-border)] px-6 py-4">
            <h2 class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
              {{ copy('动效偏好', 'Motion preferences') }}
            </h2>
          </header>
          <div class="flex flex-wrap items-center justify-between gap-4 px-6 py-4">
            <div>
              <strong class="block text-sm font-medium text-[var(--atelier-ink)]">{{ copy('减少装饰动效', 'Reduce decorative motion') }}</strong>
              <small class="text-xs text-[var(--atelier-ink-muted)]">
                {{ copy('停止流动、跟随和位移，保留操作反馈。', 'Stop flow, pointer and movement effects while keeping action feedback.') }}
              </small>
            </div>
            <MotionPreferenceToggle show-label />
          </div>
        </section>
      </div>
    </div>

    <PortalAccountDialog v-if="editor" :title="editorTitle" :busy="busy" :close-label="copy('关闭', 'Close')" @close="closeEditor">
      <p v-if="error" class="mb-3 rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]" role="alert">{{ error }}</p>
      <p v-if="actionRetryAfter > 0" class="mb-3 text-xs text-[var(--atelier-ink-muted)]" role="status">
        {{ copy(`请在 ${actionRetryAfter} 秒后重试。`, `Try again in ${actionRetryAfter} seconds.`) }}
      </p>
      <form class="space-y-4" @submit.prevent="submit">
        <label v-if="editor === 'name'" class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
          <span>{{ copy('账户昵称', 'Display name') }}</span>
          <input v-model="name" class="atelier-input" autocomplete="nickname" required autofocus />
        </label>

        <label v-else-if="editor === 'avatar'" class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
          <span>{{ copy('头像图片链接（HTTPS URL，留空恢复首字母头像）', 'Avatar image URL (HTTPS, leave blank to reset)') }}</span>
          <input v-model="avatarInput" class="atelier-input" type="url" placeholder="https://example.com/avatar.png" autofocus />
        </label>

        <template v-else-if="editor === 'email'">
          <p class="text-xs leading-relaxed text-[var(--atelier-ink-secondary)]">
            {{ emailBound ? copy('输入新邮箱，并使用当前密码确认更改。', 'Enter the new email address and confirm with your current password.') : copy('绑定邮箱并设置密码，之后可使用邮箱登录。', 'Connect your email and set a password to enable email sign-in.') }}
          </p>
          <label class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
            <span>{{ emailBound ? copy('新邮箱', 'New email address') : copy('邮箱地址', 'Email address') }}</span>
            <input v-model="email" class="atelier-input" type="email" autocomplete="email" required autofocus />
          </label>
          <label class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
            <span>{{ copy('邮箱验证码', 'Email verification code') }}</span>
            <div class="flex gap-2">
              <input v-model="emailCode" class="atelier-input flex-1 font-mono" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" required />
              <UiButton variant="secondary" size="md" type="button" :disabled="busy || cooldown > 0 || actionRetryAfter > 0 || !validEmail" @click="sendEmailCode">
                {{ cooldown > 0 ? `${cooldown}s` : copy('发送验证码', 'Send code') }}
              </UiButton>
            </div>
          </label>
          <label class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
            <span>{{ emailBound ? copy('当前密码', 'Current password') : copy('设置密码', 'Set password') }}</span>
            <input v-model="password" class="atelier-input" type="password" :autocomplete="emailBound ? 'current-password' : 'new-password'" :minlength="emailBound ? undefined : passwordPolicy.min_length" required />
            <span v-if="!emailBound" class="block text-[11px] font-normal text-[var(--atelier-ink-muted)]">{{ passwordHint }}</span>
          </label>
        </template>

        <template v-else-if="editor === 'password'">
          <label class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
            <span>{{ copy('当前密码', 'Current password') }}</span>
            <input v-model="password" class="atelier-input" type="password" autocomplete="current-password" required autofocus />
          </label>
          <label class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
            <span>{{ copy('新密码', 'New password') }}</span>
            <input v-model="newPassword" class="atelier-input" type="password" autocomplete="new-password" :minlength="passwordPolicy.min_length" required />
            <span class="block text-[11px] font-normal text-[var(--atelier-ink-muted)]">{{ passwordHint }}</span>
          </label>
          <label class="block space-y-1.5 text-xs font-medium text-[var(--atelier-ink-secondary)]">
            <span>{{ copy('确认新密码', 'Confirm new password') }}</span>
            <input v-model="confirmPassword" class="atelier-input" type="password" autocomplete="new-password" :minlength="passwordPolicy.min_length" required />
          </label>
        </template>

        <p v-else-if="editor === 'revokeAll'" class="text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
          {{ copy('确认立即退出全部设备的登录会话？所有浏览器与客户端需重新验证登录。', 'Sign out all active sessions across every device? You will need to sign in again.') }}
        </p>

        <p v-else-if="editor === 'unbind'" class="text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
          {{ copy(`确认解除与 ${selectedProvider?.label} 的关联？解除后无法再使用这一身份登录。`, `Disconnect ${selectedProvider?.label}? You will no longer be able to use this identity to sign in.`) }}
        </p>

        <div class="flex justify-end gap-2 pt-2">
          <UiButton variant="secondary" size="md" type="button" :disabled="busy" @click="closeEditor">
            {{ copy('取消', 'Cancel') }}
          </UiButton>
          <UiButton
            :variant="editor === 'revokeAll' || editor === 'unbind' ? 'danger' : 'primary'"
            size="md"
            type="submit"
            :disabled="busy || actionRetryAfter > 0"
            :loading="busy"
          >
            {{ busy ? copy('处理中…', 'Processing…') : editor === 'unbind' ? copy('确认解绑', 'Disconnect') : editor === 'revokeAll' ? copy('确认退出全部设备', 'Sign out all') : editor === 'email' ? copy('确认', 'Confirm') : copy('保存', 'Save') }}
          </UiButton>
        </div>
      </form>
    </PortalAccountDialog>
  </div>
</template>

<script setup lang="ts">
import MotionPreferenceToggle from '@/components/common/MotionPreferenceToggle.vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { userAPI } from '@/api'
import { hasExplicitWeChatOAuthCapabilities, resolveWeChatOAuthStartStrict, revokeAllSessions, type WeChatOAuthPublicSettings } from '@/api/auth'
import { startOAuthBinding, type BindableOAuthProvider } from '@/api/user'
import { sanitizeUrl } from '@/utils/url'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { userFacingError } from '@/utils/userFacingError'
import { useRetryCooldown } from '@/composables/useRetryCooldown'
import { resolvePasswordPolicy, newPasswordError, passwordPolicyHint } from '@/utils/passwordPolicy'
import type { User, UserAuthBindingStatus, UserAuthProvider } from '@/types'
import PortalAccountDialog from '@/components/portal/PortalAccountDialog.vue'
import PortalAccountSecurity from '@/components/portal/PortalAccountSecurity.vue'
import PortalAccountNotifications from '@/components/portal/PortalAccountNotifications.vue'
import '@/styles/PortalAccount.css'
import { UiButton, UiStatusDot } from '@/ui'

type SupportedProvider = 'linuxdo' | 'oidc' | 'wechat' | 'dingtalk'
const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const route = useRoute()
const app = useAppStore()
const auth = useAuthStore()
const user = computed(() => auth.user)
const settings = computed(() => app.cachedPublicSettings)
const passwordPolicy = computed(() => resolvePasswordPolicy(settings.value))
const passwordHint = computed(() => passwordPolicyHint(locale.value, passwordPolicy.value))
const { remaining: profileRetryAfter, start: startProfileRetry } = useRetryCooldown()
const { remaining: settingsRetryAfter, start: startSettingsRetry } = useRetryCooldown()
const { remaining: actionRetryAfter, start: startActionRetry } = useRetryCooldown()
const initial = computed(() => (user.value?.username || user.value?.email || '?').slice(0, 1).toUpperCase())
const safeAvatarUrl = computed(() => sanitizeUrl((user.value as { avatar_url?: string } | null)?.avatar_url || ''))
const loading = ref(false)
const loadError = ref('')
const settingsLoading = ref(false)
const settingsError = ref('')
const bindingError = ref('')
const editor = ref<'name' | 'avatar' | 'email' | 'password' | 'unbind' | 'revokeAll' | null>(null)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const name = ref('')
const avatarInput = ref('')
const email = ref('')
const emailCode = ref('')
const password = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const targetProvider = ref<SupportedProvider | null>(null)
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined
let disposed = false
const validEmail = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim()))

function bindingDetails(provider: UserAuthProvider): UserAuthBindingStatus | null {
  const value = user.value?.auth_bindings?.[provider] ?? user.value?.identity_bindings?.[provider]
  return value && typeof value === 'object' ? value : null
}
function isBound(provider: UserAuthProvider) {
  const direct = user.value?.[`${provider}_bound` as keyof User]
  if (typeof direct === 'boolean') return direct
  const value = user.value?.auth_bindings?.[provider] ?? user.value?.identity_bindings?.[provider]
  if (typeof value === 'boolean') return value
  if (typeof value?.bound === 'boolean') return value.bound
  return Boolean(value?.provider_subject || value?.issuer || value?.provider_key)
}
const emailBound = computed(() => isBound('email'))
const displayEmail = computed(() => {
  const value = user.value?.email?.trim() || ''
  return value.endsWith('.invalid') && !emailBound.value ? '' : value
})
const wechatSettings = computed<WeChatOAuthPublicSettings | null>(() => {
  if (!settings.value) return null
  if (hasExplicitWeChatOAuthCapabilities(settings.value)) return settings.value
  return { ...settings.value, wechat_oauth_open_enabled: settings.value.wechat_oauth_enabled, wechat_oauth_mp_enabled: settings.value.wechat_oauth_enabled }
})
const wechatStart = computed(() => resolveWeChatOAuthStartStrict(wechatSettings.value))
const wechatHint = computed(() => wechatStart.value.unavailableReason === 'wechat_browser_required' ? copy('请在微信内打开后绑定', 'Open this page in WeChat to connect') : copy('请在系统浏览器中打开后绑定', 'Open this page in your browser to connect'))
const providers = computed(() => {
  const entries: { provider: SupportedProvider; label: string; monogram: string; enabled: boolean }[] = [
    { provider: 'linuxdo', label: 'Linux DO', monogram: 'L', enabled: settings.value?.linuxdo_oauth_enabled === true },
    { provider: 'oidc', label: settings.value?.oidc_oauth_provider_name || 'OIDC', monogram: 'O', enabled: settings.value?.oidc_oauth_enabled === true },
    { provider: 'wechat', label: copy('微信', 'WeChat'), monogram: 'W', enabled: wechatStart.value.openEnabled || wechatStart.value.mpEnabled },
    { provider: 'dingtalk', label: copy('钉钉', 'DingTalk'), monogram: 'D', enabled: settings.value?.dingtalk_oauth_enabled === true }
  ]
  return entries.map(item => {
    const bound = isBound(item.provider)
    const details = bindingDetails(item.provider)
    return { ...item, bound, details, canBind: !bound && item.enabled && details?.can_bind !== false && (item.provider !== 'wechat' || wechatStart.value.mode !== null), canUnbind: bound && details?.can_unbind === true }
  }).filter(item => item.enabled || item.bound)
})
const selectedProvider = computed(() => providers.value.find(item => item.provider === targetProvider.value))
const editorTitle = computed(() =>
  editor.value === 'name'
    ? copy('更改昵称', 'Edit display name')
    : editor.value === 'avatar'
      ? copy('设置头像', 'Update avatar')
      : editor.value === 'email'
        ? emailBound.value
          ? copy('更改邮箱', 'Change email')
          : copy('绑定邮箱', 'Connect email')
        : editor.value === 'password'
          ? copy('更改密码', 'Change password')
          : editor.value === 'revokeAll'
            ? copy('退出全部设备', 'Sign out all devices')
            : copy('解除绑定', 'Disconnect account')
)

function resetSecrets() {
  emailCode.value = ''
  password.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
}
function openEditor(value: 'name' | 'avatar' | 'email' | 'password' | 'revokeAll') {
  name.value = user.value?.username || ''
  avatarInput.value = (user.value as { avatar_url?: string } | null)?.avatar_url || ''
  email.value = emailBound.value ? '' : displayEmail.value
  resetSecrets()
  error.value = ''
  notice.value = ''
  editor.value = value
}
function closeEditor() {
  if (busy.value) return
  editor.value = null
  error.value = ''
  resetSecrets()
}
function openUnbind(provider: SupportedProvider) {
  targetProvider.value = provider
  error.value = ''
  notice.value = ''
  editor.value = 'unbind'
}
async function loadProfile() {
  if (loading.value || profileRetryAfter.value > 0) return
  loading.value = true
  loadError.value = ''
  try { await auth.refreshUser() }
  catch (cause) {
    if (!disposed) loadError.value = userFacingError(cause, locale.value, copy('暂时无法读取，请重试。', 'Temporarily unavailable. Please retry.'))
    startProfileRetry(cause)
  } finally { loading.value = false }
}
async function loadSettings(force = false) {
  if (settingsLoading.value || settingsRetryAfter.value > 0) return
  settingsLoading.value = true
  settingsError.value = ''
  try {
    const result = await app.fetchPublicSettings(force)
    if (!result && !disposed) {
      settingsError.value = userFacingError(app.publicSettingsError, locale.value, copy('暂时无法读取，请重试。', 'Temporarily unavailable. Please retry.'))
      startSettingsRetry(app.publicSettingsError)
    }
  } catch (cause) {
    if (!disposed) settingsError.value = userFacingError(cause, locale.value, copy('暂时无法读取，请重试。', 'Temporarily unavailable. Please retry.'))
    startSettingsRetry(cause)
  } finally { settingsLoading.value = false }
}
async function loadAccount() {
  await Promise.all([loadProfile(), loadSettings()])
}
async function bindProvider(provider: BindableOAuthProvider) {
  const item = providers.value.find(value => value.provider === provider)
  if (!item?.canBind || busy.value || actionRetryAfter.value > 0) return
  busy.value = true
  bindingError.value = ''
  try { await startOAuthBinding(provider, { redirectTo: route.fullPath || '/profile', wechatOAuthSettings: provider === 'wechat' ? wechatSettings.value : undefined }) }
  catch (cause) { startActionRetry(cause); bindingError.value = userFacingError(cause, locale.value, copy('无法开始绑定，请重试。', 'Could not start the connection. Please retry.')) }
  finally { busy.value = false }
}
async function sendEmailCode() {
  if (!validEmail.value || busy.value || cooldown.value > 0 || actionRetryAfter.value > 0) return
  busy.value = true
  error.value = ''
  try {
    await userAPI.sendEmailBindingCode(email.value.trim())
    if (disposed) return
    cooldown.value = 60
    if (timer) clearInterval(timer)
    timer = setInterval(() => {
      cooldown.value = Math.max(0, cooldown.value - 1)
      if (cooldown.value === 0) { clearInterval(timer); timer = undefined }
    }, 1000)
  } catch (cause) { startActionRetry(cause); error.value = userFacingError(cause, locale.value, copy('验证码发送失败，请重试。', 'Could not send the verification code.')) }
  finally { busy.value = false }
}
async function submit() {
  if (busy.value || actionRetryAfter.value > 0) return
  error.value = ''
  if (editor.value === 'name' && !name.value.trim()) { error.value = copy('请输入昵称。', 'Enter a display name.'); return }
  if (editor.value === 'avatar') {
    const trimmedUrl = avatarInput.value.trim()
    if (trimmedUrl && !sanitizeUrl(trimmedUrl)) {
      error.value = copy('请输入有效的 HTTP/HTTPS 头像链接。', 'Enter a valid HTTP/HTTPS image URL.')
      return
    }
  }
  if (editor.value === 'email' && (!validEmail.value || !/^\d{6}$/.test(emailCode.value) || !password.value)) { error.value = copy('请填写有效邮箱、6 位验证码和密码。', 'Enter a valid email, 6-digit verification code, and password.'); return }
  if (editor.value === 'email' && !emailBound.value) { error.value = newPasswordError(password.value, locale.value, passwordPolicy.value); if (error.value) return }
  if (editor.value === 'password') {
    if (!password.value) { error.value = copy('请输入当前密码。', 'Enter your current password.'); return }
    error.value = newPasswordError(newPassword.value, locale.value, passwordPolicy.value)
    if (error.value) return
    if (newPassword.value !== confirmPassword.value) { error.value = copy('两次输入的密码必须一致。', 'The passwords must match.'); return }
  }
  if (editor.value === 'unbind' && !selectedProvider.value?.canUnbind) { error.value = copy('当前登录方式无法解除，请先绑定另一种登录方式。', 'Connect another sign-in method before disconnecting this one.'); return }
  busy.value = true
  try {
    if (editor.value === 'name') auth.user = await userAPI.updateProfile({ username: name.value.trim() })
    else if (editor.value === 'avatar') auth.user = await userAPI.updateProfile({ avatar_url: avatarInput.value.trim() } as Record<string, string>)
    else if (editor.value === 'email') auth.user = await userAPI.bindEmailIdentity({ email: email.value.trim(), verify_code: emailCode.value, password: password.value })
    else if (editor.value === 'password') await userAPI.changePassword(password.value, newPassword.value)
    else if (editor.value === 'unbind' && targetProvider.value) auth.user = await userAPI.unbindAuthIdentity(targetProvider.value)
    else if (editor.value === 'revokeAll') {
      await revokeAllSessions()
      try { await auth.logout?.() } catch { /* local session cleanup */ }
      window.location.assign('/login')
      return
    }
    notice.value = copy('账户设置已更新。', 'Account settings updated.')
    editor.value = null
    resetSecrets()
  } catch (cause) { startActionRetry(cause); error.value = userFacingError(cause, locale.value, copy('更新失败，请重试。', 'Could not update the account. Please retry.')) }
  finally { busy.value = false }
}

onMounted(() => { void loadAccount() })
onBeforeUnmount(() => {
  disposed = true
  if (timer) clearInterval(timer)
  resetSecrets()
})
</script>
