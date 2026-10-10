<template>
  <div class="portal-page">
  <main class="portal-account" data-testid="portal-account">
    <header class="portal-account__heading">
      <h1>{{ copy('账户管理', 'Account management') }}</h1>
      <p>{{ copy('管理你的账户信息与登录方式', 'Manage your account and sign-in methods') }}</p>
    </header>
    <p v-if="notice" class="portal-account-success" role="status">{{ notice }}</p>
    <p v-if="loadError" class="portal-account-error" role="alert">
      {{ loadError }}
      <button type="button" class="portal-account-action" :disabled="loading" @click="loadAccount">{{ copy('重试', 'Retry') }}</button>
    </p>
    <p v-if="loading && !user" class="portal-account-note" role="status">{{ copy('正在读取账户…', 'Loading account…') }}</p>

    <div v-if="user" class="portal-account-panel">
      <section class="portal-account-section" :aria-label="copy('账户信息', 'Account information')">
        <div class="portal-account-row">
          <div class="portal-account-row__identity">
            <span class="portal-account-avatar" aria-hidden="true">{{ initial }}</span>
            <div class="portal-account-row__text">
              <strong>{{ user.username || 'patrickapi' }}</strong>
              <small>{{ copy('账户昵称', 'Display name') }}</small>
            </div>
          </div>
          <button type="button" class="portal-account-action" @click="openEditor('name')">{{ copy('更改', 'Edit') }}</button>
        </div>
        <div class="portal-account-row">
          <div class="portal-account-row__identity">
            <span class="portal-account-provider" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="5" width="18" height="14" rx="2" /><path d="m4 7 8 6 8-6" /></svg></span>
            <div class="portal-account-row__text">
              <strong>{{ copy('邮箱', 'Email') }}</strong>
              <small>{{ displayEmail || copy('尚未绑定', 'Not connected') }}</small>
            </div>
          </div>
          <button type="button" class="portal-account-action" @click="openEditor('email')">{{ emailBound ? copy('更改', 'Change') : copy('绑定', 'Connect') }}</button>
        </div>
        <div v-for="item in providers" :key="item.provider" class="portal-account-row">
          <div class="portal-account-row__identity">
            <span class="portal-account-provider" aria-hidden="true">{{ item.monogram }}</span>
            <div class="portal-account-row__text">
              <strong>{{ item.label }}</strong>
              <small>{{ item.details?.display_name || item.details?.subject_hint || (item.bound ? copy('已绑定', 'Connected') : copy('尚未绑定', 'Not connected')) }}</small>
              <small v-if="item.bound && item.details?.bound_count && item.details.bound_count > 1">{{ copy(`${item.details.bound_count} 个关联身份`, `${item.details.bound_count} connected identities`) }}</small>
              <small v-if="item.bound && !item.canUnbind">{{ copy('保留至少一种可用的登录方式', 'Keep at least one available sign-in method') }}</small>
              <small v-else-if="!item.bound && item.provider === 'wechat' && !item.canBind">{{ wechatHint }}</small>
            </div>
          </div>
          <button v-if="item.canUnbind" type="button" class="portal-account-action" :disabled="busy" @click="openUnbind(item.provider)">{{ copy('解绑', 'Disconnect') }}</button>
          <button v-else-if="item.canBind" type="button" class="portal-account-action" :disabled="busy" @click="bindProvider(item.provider)">{{ copy('绑定', 'Connect') }}</button>
          <span v-else class="portal-account-status">{{ item.bound ? copy('已关联', 'Connected') : copy('暂不可用', 'Unavailable') }}</span>
        </div>
      </section>

      <section class="portal-account-section" :aria-label="copy('账户安全', 'Account security')">
        <h2>{{ copy('账户安全', 'Account security') }}</h2>
        <div class="portal-account-row">
          <div class="portal-account-row__text">
            <strong>{{ copy('登录密码', 'Password') }}</strong>
            <small>{{ copy('定期更新密码，保护账户安全', 'Update your password to keep your account secure') }}</small>
          </div>
          <button type="button" class="portal-account-action" @click="openEditor('password')">{{ copy('更改', 'Change') }}</button>
        </div>
        <PortalAccountSecurity :passkey-enabled="settings?.passkey_enabled === true" />
      </section>

      <section v-if="settings?.balance_low_notify_enabled" class="portal-account-section" :aria-label="copy('通知', 'Notifications')">
        <PortalAccountNotifications :default-threshold="settings.balance_low_notify_threshold ?? 0" />
      </section>
    </div>
    <p v-if="settings?.contact_info" class="portal-account-support">{{ settings.contact_info }}</p>

    <PortalAccountDialog v-if="editor" :title="editorTitle" :busy="busy" :close-label="copy('关闭', 'Close')" @close="closeEditor">
      <p v-if="error" class="portal-account-error" role="alert">{{ error }}</p>
      <form class="portal-account-form" @submit.prevent="submit">
        <label v-if="editor === 'name'">
          {{ copy('账户昵称', 'Display name') }}
          <input v-model="name" class="portal-account-input" autocomplete="nickname" required autofocus />
        </label>

        <template v-else-if="editor === 'email'">
          <p>{{ emailBound ? copy('输入新邮箱，并使用当前密码确认更改。', 'Enter the new email address and confirm with your current password.') : copy('绑定邮箱并设置密码，之后可使用邮箱登录。', 'Connect your email and set a password to enable email sign-in.') }}</p>
          <label>
            {{ emailBound ? copy('新邮箱', 'New email address') : copy('邮箱地址', 'Email address') }}
            <input v-model="email" class="portal-account-input" type="email" autocomplete="email" required autofocus />
          </label>
          <label>
            {{ copy('邮箱验证码', 'Email verification code') }}
            <span class="portal-account-inline">
              <input v-model="emailCode" class="portal-account-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" required />
              <button type="button" class="portal-account-action" :disabled="busy || cooldown > 0 || !validEmail" @click="sendEmailCode">{{ cooldown > 0 ? `${cooldown}s` : copy('发送验证码', 'Send code') }}</button>
            </span>
          </label>
          <label>
            {{ emailBound ? copy('当前密码', 'Current password') : copy('设置密码', 'Set password') }}
            <input v-model="password" class="portal-account-input" type="password" :autocomplete="emailBound ? 'current-password' : 'new-password'" :minlength="emailBound ? undefined : 6" required />
          </label>
        </template>

        <template v-else-if="editor === 'password'">
          <label>{{ copy('当前密码', 'Current password') }}<input v-model="password" class="portal-account-input" type="password" autocomplete="current-password" required autofocus /></label>
          <label>{{ copy('新密码', 'New password') }}<input v-model="newPassword" class="portal-account-input" type="password" autocomplete="new-password" minlength="8" required /><span class="portal-account-note">{{ copy('至少 8 个字符', 'At least 8 characters') }}</span></label>
          <label>{{ copy('确认新密码', 'Confirm new password') }}<input v-model="confirmPassword" class="portal-account-input" type="password" autocomplete="new-password" minlength="8" required /></label>
        </template>

        <p v-else-if="editor === 'unbind'">{{ copy(`确认解除与 ${selectedProvider?.label} 的关联？解除后无法再使用这一身份登录。`, `Disconnect ${selectedProvider?.label}? You will no longer be able to use this identity to sign in.`) }}</p>

        <div class="portal-account-form__actions">
          <button type="button" class="portal-account-action" :disabled="busy" @click="closeEditor">{{ copy('取消', 'Cancel') }}</button>
          <button type="submit" class="portal-account-action portal-account-action--primary" :disabled="busy">{{ busy ? copy('处理中…', 'Processing…') : editor === 'unbind' ? copy('确认解绑', 'Disconnect') : editor === 'email' ? copy('确认', 'Confirm') : copy('保存', 'Save') }}</button>
        </div>
      </form>
    </PortalAccountDialog>
  </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { userAPI } from '@/api'
import { hasExplicitWeChatOAuthCapabilities, resolveWeChatOAuthStartStrict, type WeChatOAuthPublicSettings } from '@/api/auth'
import { startOAuthBinding, type BindableOAuthProvider } from '@/api/user'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { User, UserAuthBindingStatus, UserAuthProvider } from '@/types'
import PortalAccountDialog from '@/components/portal/PortalAccountDialog.vue'
import PortalAccountSecurity from '@/components/portal/PortalAccountSecurity.vue'
import PortalAccountNotifications from '@/components/portal/PortalAccountNotifications.vue'
import '@/styles/PortalAccount.css'

type SupportedProvider = 'linuxdo' | 'oidc' | 'wechat' | 'dingtalk'
const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const route = useRoute()
const app = useAppStore()
const auth = useAuthStore()
const user = computed(() => auth.user)
const settings = computed(() => app.cachedPublicSettings)
const initial = computed(() => (user.value?.username || user.value?.email || 'P').slice(0, 1).toUpperCase())
const loading = ref(false)
const loadError = ref('')
const editor = ref<'name' | 'email' | 'password' | 'unbind' | null>(null)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const name = ref('')
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
  // Preserve the legacy public-settings contract while using the existing strict resolver.
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
const editorTitle = computed(() => editor.value === 'name' ? copy('更改昵称', 'Edit display name') : editor.value === 'email' ? emailBound.value ? copy('更改邮箱', 'Change email') : copy('绑定邮箱', 'Connect email') : editor.value === 'password' ? copy('更改密码', 'Change password') : copy('解除绑定', 'Disconnect account'))

function resetSecrets() {
  emailCode.value = ''
  password.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
}
function openEditor(value: 'name' | 'email' | 'password') {
  name.value = user.value?.username || ''
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
async function loadAccount() {
  if (loading.value) return
  loading.value = true
  loadError.value = ''
  const results = await Promise.allSettled([auth.refreshUser(), app.fetchPublicSettings()])
  if (results.some(result => result.status === 'rejected') || !settings.value) loadError.value = copy('部分账户信息读取失败，请重试。', 'Some account information could not be loaded. Please retry.')
  loading.value = false
}
async function bindProvider(provider: BindableOAuthProvider) {
  const item = providers.value.find(value => value.provider === provider)
  if (!item?.canBind || busy.value) return
  busy.value = true
  loadError.value = ''
  try { await startOAuthBinding(provider, { redirectTo: route.fullPath || '/profile', wechatOAuthSettings: provider === 'wechat' ? wechatSettings.value : undefined }) }
  catch (cause) { loadError.value = extractApiErrorMessage(cause, copy('无法开始绑定，请重试。', 'Could not start the connection. Please retry.')) }
  finally { busy.value = false }
}
async function sendEmailCode() {
  if (!validEmail.value || busy.value || cooldown.value > 0) return
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
  } catch (cause) { error.value = extractApiErrorMessage(cause, copy('验证码发送失败，请重试。', 'Could not send the verification code.')) }
  finally { busy.value = false }
}
async function submit() {
  if (busy.value) return
  error.value = ''
  if (editor.value === 'name' && !name.value.trim()) { error.value = copy('请输入昵称。', 'Enter a display name.'); return }
  if (editor.value === 'email' && (!validEmail.value || !/^\d{6}$/.test(emailCode.value) || !password.value || (!emailBound.value && password.value.length < 6))) { error.value = copy('请填写有效邮箱、6 位验证码和密码。', 'Enter a valid email, 6-digit verification code, and password.'); return }
  if (editor.value === 'password' && (!password.value || newPassword.value.length < 8 || newPassword.value !== confirmPassword.value)) { error.value = copy('新密码至少 8 个字符，且两次输入必须一致。', 'The new password must be at least 8 characters and match the confirmation.'); return }
  if (editor.value === 'unbind' && !selectedProvider.value?.canUnbind) { error.value = copy('当前登录方式无法解除，请先绑定另一种登录方式。', 'Connect another sign-in method before disconnecting this one.'); return }
  busy.value = true
  try {
    if (editor.value === 'name') auth.user = await userAPI.updateProfile({ username: name.value.trim() })
    else if (editor.value === 'email') auth.user = await userAPI.bindEmailIdentity({ email: email.value.trim(), verify_code: emailCode.value, password: password.value })
    else if (editor.value === 'password') await userAPI.changePassword(password.value, newPassword.value)
    else if (editor.value === 'unbind' && targetProvider.value) auth.user = await userAPI.unbindAuthIdentity(targetProvider.value)
    notice.value = copy('账户设置已更新。', 'Account settings updated.')
    editor.value = null
    resetSecrets()
  } catch (cause) { error.value = extractApiErrorMessage(cause, copy('更新失败，请重试。', 'Could not update the account. Please retry.')) }
  finally { busy.value = false }
}

onMounted(() => { void loadAccount() })
onBeforeUnmount(() => {
  disposed = true
  if (timer) clearInterval(timer)
  resetSecrets()
})
</script>
