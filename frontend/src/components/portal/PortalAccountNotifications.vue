<template>
  <div class="portal-account-row">
    <div class="portal-account-row__text">
      <strong>{{ copy('余额提醒', 'Balance alerts') }}</strong>
      <small>{{ user?.balance_notify_enabled !== false ? copy('余额不足时通过邮箱通知', 'Email notifications when your balance is low') : copy('已关闭', 'Disabled') }}</small>
    </div>
    <button type="button" class="portal-account-action" @click="open">{{ copy('管理', 'Manage') }}</button>
  </div>

  <PortalAccountDialog v-if="show" :title="copy('余额提醒', 'Balance alerts')" :busy="busy" :close-label="copy('关闭', 'Close')" @close="close">
    <p v-if="error" class="portal-account-error" role="alert">{{ error }}</p>
    <p v-if="success" class="portal-account-success" role="status">{{ success }}</p>
    <form v-if="removingEmail !== null" class="portal-account-form" @submit.prevent="removeEmail">
      <p>{{ copy(`移除 ${displayEmail(removingEmail)} 的余额提醒？`, `Remove balance alerts for ${displayEmail(removingEmail)}?`) }}</p>
      <div class="portal-account-form__actions">
        <button type="button" class="portal-account-action" :disabled="busy" @click="removingEmail = null">{{ copy('取消', 'Cancel') }}</button>
        <button type="submit" class="portal-account-action portal-account-action--primary" :disabled="busy">{{ copy('确认移除', 'Remove email') }}</button>
      </div>
    </form>
    <div v-else class="portal-account-form">
      <form class="portal-account-form" @submit.prevent="save">
        <label class="portal-account-check"><input v-model="enabled" type="checkbox" />{{ copy('启用余额提醒', 'Enable balance alerts') }}</label>
        <label v-if="enabled">
          {{ copy('提醒阈值（美元）', 'Alert threshold (USD)') }}
          <input v-model="threshold" class="portal-account-input" type="number" min="0" step="0.01" :placeholder="String(defaultThreshold)" />
          <span class="portal-account-note">{{ copy(`留空或填写 0，使用系统默认值 $${defaultThreshold}。`, `Leave empty or enter 0 to use the default $${defaultThreshold}.`) }}</span>
        </label>
        <div class="portal-account-form__actions">
          <button type="submit" class="portal-account-action" :disabled="busy">{{ copy('保存设置', 'Save settings') }}</button>
        </div>
      </form>
      <template v-if="enabled">
        <hr class="portal-account-divider" />
        <p>{{ copy('接收邮箱', 'Email recipients') }}</p>
        <p class="portal-account-note">{{ copy('最多 3 个接收地址；新增地址需要通过邮箱验证。', 'Up to 3 recipients. New addresses require email verification.') }}</p>
        <ul v-if="entries.length" class="portal-account-list">
          <li v-for="entry in entries" :key="entry.email">
            <label class="portal-account-check portal-account-list__text">
              <input type="checkbox" :checked="!entry.disabled" :disabled="busy" :aria-label="copy(`向 ${displayEmail(entry.email)} 发送提醒`, `Send alerts to ${displayEmail(entry.email)}`)" @change="toggle(entry)" />
              <span>{{ displayEmail(entry.email) }}<small>{{ entry.verified ? copy('已验证', 'Verified') : copy('尚未验证', 'Unverified') }}</small></span>
            </label>
            <div class="portal-account-row__actions">
              <button v-if="!entry.verified" type="button" class="portal-account-action" :disabled="busy" @click="verifyExisting(entry.email)">{{ copy('验证', 'Verify') }}</button>
              <button type="button" class="portal-account-action portal-account-action--danger" :disabled="busy" @click="removingEmail = entry.email; error = ''; success = ''">{{ copy('移除', 'Remove') }}</button>
            </div>
          </li>
        </ul>
        <form v-if="entries.length < 3 || verifyingExisting" class="portal-account-form" @submit.prevent="codeSent ? verifyEmail() : sendCode()">
          <label>
            {{ verifyingExisting ? copy('验证邮箱', 'Verify email') : copy('添加接收邮箱', 'Add recipient') }}
            <input v-model="email" class="portal-account-input" type="email" autocomplete="email" :readonly="codeSent || verifyingExisting" required :placeholder="copy('邮箱地址', 'Email address')" />
          </label>
          <label v-if="codeSent">
            {{ copy('邮箱验证码', 'Email verification code') }}
            <span class="portal-account-inline">
              <input v-model="code" class="portal-account-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" required />
              <button type="button" class="portal-account-action" :disabled="busy || cooldown > 0" @click="sendCode">{{ cooldown > 0 ? `${cooldown}s` : copy('重新发送', 'Resend') }}</button>
            </span>
          </label>
          <div class="portal-account-form__actions">
            <button v-if="codeSent || verifyingExisting" type="button" class="portal-account-action" :disabled="busy" @click="resetVerification">{{ copy('取消', 'Cancel') }}</button>
            <button type="submit" class="portal-account-action portal-account-action--primary" :disabled="busy || !email.trim() || (codeSent && !/^\d{6}$/.test(code))">{{ busy ? copy('处理中…', 'Processing…') : codeSent ? copy('验证并添加', 'Verify email') : copy('发送验证码', 'Send verification code') }}</button>
          </div>
        </form>
      </template>
    </div>
  </PortalAccountDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { NotifyEmailEntry } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import PortalAccountDialog from './PortalAccountDialog.vue'

defineProps<{ defaultThreshold: number }>()
const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const auth = useAuthStore()
const user = computed(() => auth.user)
const entries = computed(() => user.value?.balance_notify_extra_emails ?? [])
const show = ref(false)
const enabled = ref(true)
const threshold = ref<number | string>('')
const busy = ref(false)
const error = ref('')
const success = ref('')
const email = ref('')
const code = ref('')
const codeSent = ref(false)
const verifyingExisting = ref(false)
const removingEmail = ref<string | null>(null)
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined
let disposed = false
const displayEmail = (value: string) => value || user.value?.email || copy('主邮箱', 'Primary email')

function resetVerification() {
  email.value = ''
  code.value = ''
  codeSent.value = false
  verifyingExisting.value = false
}
function open() {
  enabled.value = user.value?.balance_notify_enabled ?? true
  threshold.value = user.value?.balance_notify_threshold ?? ''
  error.value = ''
  success.value = ''
  removingEmail.value = null
  resetVerification()
  show.value = true
}
function close() {
  if (busy.value) return
  show.value = false
  resetVerification()
}
async function perform(operation: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  success.value = ''
  try { await operation() }
  catch (cause) { error.value = extractApiErrorMessage(cause, copy('操作失败，请重试。', 'The operation failed. Please try again.')) }
  finally { busy.value = false }
}
async function save() {
  const value = threshold.value === '' ? 0 : Number(threshold.value)
  if (!Number.isFinite(value) || value < 0) {
    error.value = copy('请输入不小于 0 的提醒阈值。', 'Enter a threshold of 0 or greater.')
    return
  }
  await perform(async () => {
    auth.user = await userAPI.updateProfile({ balance_notify_enabled: enabled.value, balance_notify_threshold: value })
    success.value = copy('提醒设置已保存。', 'Alert settings saved.')
  })
}
async function toggle(entry: NotifyEmailEntry) {
  await perform(async () => { auth.user = await userAPI.toggleNotifyEmail(entry.email, !entry.disabled) })
}
async function removeEmail() {
  const target = removingEmail.value
  if (target === null) return
  await perform(async () => {
    await userAPI.removeNotifyEmail(target)
    auth.user = await userAPI.getProfile()
    removingEmail.value = null
    success.value = copy('接收邮箱已移除。', 'Recipient removed.')
  })
}
function verifyExisting(value: string) {
  resetVerification()
  email.value = displayEmail(value)
  verifyingExisting.value = true
}
async function sendCode() {
  const address = email.value.trim()
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(address)) {
    error.value = copy('请输入有效的邮箱地址。', 'Enter a valid email address.')
    return
  }
  if (cooldown.value > 0) {
    error.value = copy(`请 ${cooldown.value} 秒后再发送验证码。`, `Please wait ${cooldown.value} seconds before requesting another code.`)
    return
  }
  if (!verifyingExisting.value && (entries.value.length >= 3 || entries.value.some(entry => displayEmail(entry.email).toLowerCase() === address.toLowerCase()))) {
    error.value = copy('邮箱已存在，或已达到 3 个邮箱上限。', 'The address already exists or the 3-recipient limit has been reached.')
    return
  }
  await perform(async () => {
    await userAPI.sendNotifyEmailCode(address)
    if (disposed) return
    email.value = address
    codeSent.value = true
    cooldown.value = 60
    if (timer) clearInterval(timer)
    timer = setInterval(() => {
      cooldown.value = Math.max(0, cooldown.value - 1)
      if (cooldown.value === 0) { clearInterval(timer); timer = undefined }
    }, 1000)
    success.value = copy('验证码已发送，请查收邮件。', 'Verification code sent. Check your inbox.')
  })
}
async function verifyEmail() {
  if (!/^\d{6}$/.test(code.value)) return
  await perform(async () => {
    await userAPI.verifyNotifyEmail(email.value, code.value)
    auth.user = await userAPI.getProfile()
    resetVerification()
    success.value = copy('邮箱验证成功。', 'Email verified.')
  })
}
onBeforeUnmount(() => {
  disposed = true
  if (timer) clearInterval(timer)
  code.value = ''
})
</script>
