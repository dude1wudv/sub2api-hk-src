<template>
  <div v-if="loadingStatus || statusError || status?.feature_enabled || status?.enabled" class="portal-account-row">
    <div class="portal-account-row__text">
      <strong>{{ copy('双重验证', 'Two-factor authentication') }}</strong>
      <small>{{ loadingStatus ? copy('正在读取安全设置…', 'Loading security settings…') : status?.enabled ? copy('已启用验证器保护', 'Authenticator protection is enabled') : copy('使用验证器为登录增加一道保护', 'Add an authenticator to protect sign-in') }}</small>
      <small v-if="statusError" role="alert">{{ statusError }}</small>
    </div>
    <button v-if="statusError" type="button" class="portal-account-action" @click="loadStatus">{{ copy('重试', 'Retry') }}</button>
    <button v-else type="button" class="portal-account-action" :disabled="loadingStatus" @click="openTotp">
      {{ status?.enabled ? copy('关闭', 'Disable') : copy('开启', 'Enable') }}
    </button>
  </div>

  <div v-if="passkeyEnabled" class="portal-account-row">
    <div class="portal-account-row__text">
      <strong>Passkey</strong>
      <small>{{ credentials.length ? copy(`已添加 ${credentials.length} 个通行密钥`, `${credentials.length} passkeys added`) : copy('使用设备指纹、面容或安全密钥登录', 'Sign in with your device or security key') }}</small>
    </div>
    <button type="button" class="portal-account-action" @click="openPasskeys">{{ copy('管理', 'Manage') }}</button>
  </div>

  <PortalAccountDialog v-if="view" :title="dialogTitle" :busy="busy" :close-label="copy('关闭', 'Close')" @close="close">
    <p v-if="error" class="portal-account-error" role="alert">{{ error }}</p>

    <form v-if="view === 'totp'" class="portal-account-form" @submit.prevent="submitTotp">
      <p v-if="methodLoading" class="portal-account-note" role="status">{{ copy('正在获取验证方式…', 'Loading verification method…') }}</p>
      <template v-else-if="verificationMethod">
        <template v-if="setup">
          <p>{{ copy('使用验证器扫描二维码，或手动输入以下密钥。', 'Scan the QR code with your authenticator or enter the secret below.') }}</p>
          <img v-if="qrCode" class="portal-account-qr" :src="qrCode" :alt="copy('验证器设置二维码', 'Authenticator setup QR code')" />
          <code class="portal-account-secret">{{ setup.secret }}</code>
          <p class="portal-account-note">{{ copy('请妥善保管密钥，不要与他人分享。', 'Keep this secret private and store it safely.') }}</p>
          <label>
            {{ copy('验证器验证码', 'Authenticator code') }}
            <input v-model="totpCode" class="portal-account-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" required />
          </label>
        </template>
        <template v-else>
          <p>{{ status?.enabled ? copy('关闭后，登录将不再要求验证器验证码。请验证身份以继续。', 'Disabling this removes the authenticator check at sign-in. Verify your identity to continue.') : copy('请先验证身份，然后连接你的验证器。', 'Verify your identity before connecting an authenticator.') }}</p>
          <label v-if="verificationMethod === 'password'">
            {{ copy('当前密码', 'Current password') }}
            <input v-model="password" class="portal-account-input" type="password" autocomplete="current-password" required />
          </label>
          <label v-else>
            {{ copy('邮箱验证码', 'Email verification code') }}
            <span class="portal-account-inline">
              <input v-model="emailCode" class="portal-account-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" required />
              <button type="button" class="portal-account-action" :disabled="busy || cooldown > 0" @click="sendTotpCode">{{ cooldown > 0 ? `${cooldown}s` : copy('发送验证码', 'Send code') }}</button>
            </span>
          </label>
        </template>
        <div class="portal-account-form__actions">
          <button type="button" class="portal-account-action" :disabled="busy" @click="close">{{ copy('取消', 'Cancel') }}</button>
          <button type="submit" class="portal-account-action portal-account-action--primary" :disabled="busy || !canSubmitTotp">{{ busy ? copy('处理中…', 'Processing…') : status?.enabled ? copy('确认关闭', 'Disable protection') : setup ? copy('确认开启', 'Enable protection') : copy('下一步', 'Continue') }}</button>
        </div>
      </template>
      <button v-else type="button" class="portal-account-action" @click="loadVerificationMethod">{{ copy('重新获取验证方式', 'Retry verification method') }}</button>
    </form>

    <template v-else-if="view === 'passkeys'">
      <p v-if="!supported" class="portal-account-note">{{ copy('此浏览器不支持创建 Passkey。你仍可管理已有密钥。', 'This browser cannot create passkeys. Existing keys can still be managed.') }}</p>
      <p v-if="loadingKeys" class="portal-account-note" role="status">{{ copy('正在读取…', 'Loading…') }}</p>
      <p v-else-if="credentials.length === 0" class="portal-account-note">{{ copy('尚未添加通行密钥。', 'No passkeys have been added.') }}</p>
      <ul v-else class="portal-account-list">
        <li v-for="credential in credentials" :key="credential.id">
          <div class="portal-account-list__text">
            {{ credential.name }}
            <small>{{ date(credential.created_at) }}{{ credential.backup ? copy(' · 已同步', ' · Synced') : '' }}</small>
            <small v-if="credential.last_used_at">{{ copy('最近使用 ', 'Last used ') }}{{ date(credential.last_used_at) }}</small>
          </div>
          <div class="portal-account-row__actions">
            <button type="button" class="portal-account-action" @click="editKey(credential, 'rename')">{{ copy('更名', 'Rename') }}</button>
            <button type="button" class="portal-account-action portal-account-action--danger" @click="editKey(credential, 'remove')">{{ copy('移除', 'Remove') }}</button>
          </div>
        </li>
      </ul>
      <div class="portal-account-form__actions" style="margin-top: 24px">
        <button type="button" class="portal-account-action" @click="loadKeys">{{ copy('刷新', 'Refresh') }}</button>
        <button v-if="supported" type="button" class="portal-account-action portal-account-action--primary" @click="view = 'add'; error = ''; name = ''; password = ''">{{ copy('添加 Passkey', 'Add passkey') }}</button>
      </div>
    </template>

    <form v-else class="portal-account-form" @submit.prevent="submitPasskey">
      <p v-if="view === 'remove'">{{ copy(`移除“${selectedKey?.name}”后，该密钥将不能用于登录。`, `Removing “${selectedKey?.name}” prevents using it to sign in.`) }}</p>
      <label v-else>
        {{ copy('密钥名称', 'Passkey name') }}
        <input v-model="name" class="portal-account-input" maxlength="100" :required="view === 'rename'" autofocus :placeholder="copy('例如：我的笔记本电脑', 'For example: My laptop')" />
      </label>
      <label v-if="view !== 'rename'">
        {{ copy('当前密码', 'Current password') }}
        <input v-model="password" class="portal-account-input" type="password" autocomplete="current-password" required />
      </label>
      <div class="portal-account-form__actions">
        <button type="button" class="portal-account-action" :disabled="busy" @click="backToKeys">{{ copy('返回', 'Back') }}</button>
        <button type="submit" class="portal-account-action portal-account-action--primary" :disabled="busy || (view === 'rename' ? !name.trim() : !password)">{{ busy ? copy('处理中…', 'Processing…') : view === 'remove' ? copy('确认移除', 'Remove passkey') : view === 'rename' ? copy('保存', 'Save') : copy('继续', 'Continue') }}</button>
      </div>
    </form>
  </PortalAccountDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import { passkeyAPI, totpAPI, type PasskeyCredentialSummary } from '@/api'
import type { TotpSetupResponse, TotpStatus } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import PortalAccountDialog from './PortalAccountDialog.vue'

const props = defineProps<{ passkeyEnabled: boolean }>()
const { locale } = useI18n()
const copy = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const status = ref<TotpStatus | null>(null)
const loadingStatus = ref(true)
const statusError = ref('')
const credentials = ref<PasskeyCredentialSummary[]>([])
const loadingKeys = ref(false)
const supported = passkeyAPI.isSupported()
const view = ref<'totp' | 'passkeys' | 'add' | 'rename' | 'remove' | null>(null)
const busy = ref(false)
const error = ref('')
const name = ref('')
const password = ref('')
const selectedKey = ref<PasskeyCredentialSummary | null>(null)
const methodLoading = ref(false)
const verificationMethod = ref<'email' | 'password' | null>(null)
const emailCode = ref('')
const totpCode = ref('')
const setup = ref<TotpSetupResponse | null>(null)
const qrCode = ref('')
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined
let disposed = false

const dialogTitle = computed(() => {
  if (view.value === 'totp') return status.value?.enabled ? copy('关闭双重验证', 'Disable two-factor authentication') : copy('开启双重验证', 'Enable two-factor authentication')
  if (view.value === 'add') return copy('添加 Passkey', 'Add passkey')
  if (view.value === 'rename') return copy('更改名称', 'Rename passkey')
  if (view.value === 'remove') return copy('移除 Passkey', 'Remove passkey')
  return copy('通行密钥', 'Passkeys')
})
const canSubmitTotp = computed(() => setup.value ? /^\d{6}$/.test(totpCode.value) : verificationMethod.value === 'email' ? /^\d{6}$/.test(emailCode.value) : verificationMethod.value === 'password' && password.value.length > 0)
const date = (value: string) => new Date(value).toLocaleDateString(locale.value)
const errorMessage = (cause: unknown) => extractApiErrorMessage(cause, copy('操作失败，请重试。', 'The operation failed. Please try again.'))

function clearSensitiveData() {
  password.value = ''
  emailCode.value = ''
  totpCode.value = ''
  setup.value = null
  qrCode.value = ''
}
function close() {
  if (busy.value) return
  view.value = null
  error.value = ''
  clearSensitiveData()
}
async function loadStatus() {
  loadingStatus.value = true
  statusError.value = ''
  try { status.value = await totpAPI.getStatus() }
  catch (cause) { statusError.value = errorMessage(cause) }
  finally { loadingStatus.value = false }
}
async function openTotp() {
  clearSensitiveData()
  error.value = ''
  view.value = 'totp'
  await loadVerificationMethod()
}
async function loadVerificationMethod() {
  methodLoading.value = true
  busy.value = true
  verificationMethod.value = null
  error.value = ''
  try { verificationMethod.value = (await totpAPI.getVerificationMethod()).method }
  catch (cause) { error.value = errorMessage(cause) }
  finally { methodLoading.value = false; busy.value = false }
}
async function sendTotpCode() {
  if (busy.value || cooldown.value > 0) return
  busy.value = true
  error.value = ''
  try {
    await totpAPI.sendVerifyCode()
    if (disposed) return
    cooldown.value = 60
    if (timer) clearInterval(timer)
    timer = setInterval(() => {
      cooldown.value = Math.max(0, cooldown.value - 1)
      if (cooldown.value === 0) { clearInterval(timer); timer = undefined }
    }, 1000)
  } catch (cause) { error.value = errorMessage(cause) }
  finally { busy.value = false }
}
async function submitTotp() {
  if (!canSubmitTotp.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    const proof = verificationMethod.value === 'email' ? { email_code: emailCode.value } : { password: password.value }
    if (status.value?.enabled) {
      await totpAPI.disable(proof)
    } else if (setup.value) {
      await totpAPI.enable({ totp_code: totpCode.value, setup_token: setup.value.setup_token })
    } else {
      const result = await totpAPI.initiateSetup(proof)
      if (disposed) return
      setup.value = result
      password.value = ''
      emailCode.value = ''
      try {
        const generated = await QRCode.toDataURL(result.qr_code_url, { width: 180, margin: 2, color: { dark: '#000000', light: '#fffdf7' } })
        if (!disposed) qrCode.value = generated
      }
      catch { qrCode.value = '' }
      return
    }
    clearSensitiveData()
    view.value = null
    await loadStatus()
  } catch (cause) { error.value = errorMessage(cause) }
  finally { busy.value = false }
}
async function loadKeys() {
  if (!props.passkeyEnabled) return
  loadingKeys.value = true
  try { credentials.value = await passkeyAPI.list() }
  catch (cause) { error.value = errorMessage(cause) }
  finally { loadingKeys.value = false }
}
function openPasskeys() {
  view.value = 'passkeys'
  error.value = ''
  void loadKeys()
}
function editKey(credential: PasskeyCredentialSummary, action: 'rename' | 'remove') {
  selectedKey.value = credential
  name.value = credential.name
  password.value = ''
  error.value = ''
  view.value = action
}
function backToKeys() {
  clearSensitiveData()
  error.value = ''
  view.value = 'passkeys'
}
async function submitPasskey() {
  if (busy.value) return
  if (view.value === 'rename' ? !name.value.trim() : !password.value) return
  busy.value = true
  error.value = ''
  try {
    if (view.value === 'add') await passkeyAPI.register(name.value.trim(), password.value)
    else if (view.value === 'rename' && selectedKey.value) await passkeyAPI.rename(selectedKey.value.id, name.value.trim())
    else if (view.value === 'remove' && selectedKey.value) await passkeyAPI.remove(selectedKey.value.id, password.value)
    backToKeys()
    await loadKeys()
  } catch (cause) {
    if (!(cause instanceof DOMException && cause.name === 'NotAllowedError')) error.value = errorMessage(cause)
  } finally { busy.value = false }
}

watch(() => props.passkeyEnabled, enabled => { if (enabled) void loadKeys() }, { immediate: true })
onMounted(() => { void loadStatus() })
onBeforeUnmount(() => {
  disposed = true
  if (timer) clearInterval(timer)
  clearSensitiveData()
})
</script>
