<template>
  <section class="atelier-surface space-y-5 p-6" :aria-label="t('adminAccessKeys.title')">
    <div class="flex flex-wrap items-start justify-between gap-4 border-b border-[var(--atelier-border)] pb-4">
      <div>
        <h2 class="text-lg font-semibold text-[var(--atelier-ink)]">{{ t('adminAccessKeys.title') }}</h2>
        <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">{{ t('adminAccessKeys.authorizationHint') }}</p>
      </div>
      <div class="flex gap-2">
        <UiButton variant="primary" data-test="create" :disabled="busy || loading || !loaded" @click="edit()">
          {{ t('adminAccessKeys.create') }}
        </UiButton>
        <UiButton variant="secondary" :disabled="busy || loading" @click="load">
          {{ t('adminAccessKeys.refresh') }}
        </UiButton>
      </div>
    </div>

    <div class="flex items-start gap-2.5 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-warn-bg)] px-4 py-3 text-sm text-[var(--atelier-warn)]">
      <UiStatusDot tone="warn" class="mt-1.5" />
      <p>{{ t('adminAccessKeys.sharedAccountWarning') }}</p>
    </div>

    <p v-if="error" role="alert" class="rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-sm text-[var(--atelier-err)]">{{ error }}</p>
    <p v-if="notice" role="status" class="rounded-lg bg-[var(--atelier-ok-bg)] px-3.5 py-2.5 text-sm text-[var(--atelier-ok)]">{{ notice }}</p>

    <div v-if="secret" class="space-y-2.5 rounded-xl border border-[var(--atelier-warn)]/40 bg-[var(--atelier-surface)] p-4">
      <p class="text-sm font-medium text-[var(--atelier-ink)]">{{ t('adminAccessKeys.secretWarning') }}</p>
      <code class="block select-all break-all rounded-lg bg-[var(--atelier-subtle)] px-3 py-2 font-mono text-xs text-[var(--atelier-ink)]">{{ secret }}</code>
      <div class="flex flex-wrap gap-2">
        <UiButton variant="secondary" size="sm" @click="copySecret">{{ t('adminAccessKeys.copySecret') }}</UiButton>
        <UiButton variant="secondary" size="sm" data-test="dismiss-secret" @click="secret = ''">{{ t('adminAccessKeys.dismissSecret') }}</UiButton>
      </div>
    </div>

    <p v-if="loading" class="text-sm text-[var(--atelier-ink-muted)]">{{ t('adminAccessKeys.loading') }}</p>

    <div class="space-y-3">
      <div v-for="item in items" :key="item.id" class="space-y-2.5 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-medium text-[var(--atelier-ink)]">{{ item.name }}</span>
            <code class="rounded bg-[var(--atelier-subtle)] px-1.5 py-0.5 font-mono text-xs text-[var(--atelier-ink-secondary)]">{{ item.key_prefix }}…</code>
            <span class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium" :class="statusBadgeClass(item)">
              <UiStatusDot :tone="statusTone(item)" />
              {{ status(item) }}
            </span>
          </div>
          <div v-if="!item.revoked_at" class="flex flex-wrap gap-2">
            <UiButton variant="secondary" size="sm" :disabled="busy" @click="edit(item)">{{ t('adminAccessKeys.editPermissions') }}</UiButton>
            <UiButton variant="secondary" size="sm" :disabled="busy" @click="toggle(item)">{{ item.enabled ? t('adminAccessKeys.disable') : t('adminAccessKeys.enable') }}</UiButton>
            <UiButton variant="secondary" size="sm" :disabled="busy" @click="rotateKey(item)">{{ t('adminAccessKeys.rotate') }}</UiButton>
            <UiButton variant="secondary" size="sm" class="!text-[var(--atelier-err)]" :disabled="busy" @click="revokeKey(item)">{{ t('adminAccessKeys.revoke') }}</UiButton>
          </div>
          <div v-else>
            <UiButton variant="secondary" size="sm" class="!text-[var(--atelier-err)]" :disabled="busy" @click="deleteRecord(item)">{{ t('adminAccessKeys.deleteRecord') }}</UiButton>
          </div>
        </div>
        <p class="text-xs text-[var(--atelier-ink-secondary)]">{{ item.description || t('adminAccessKeys.noDescription') }} · {{ t('adminAccessKeys.expires', { value: item.expires_at || t('adminAccessKeys.never') }) }}</p>
        <p class="text-xs tabular-nums text-[var(--atelier-ink-muted)]">{{ t('adminAccessKeys.lastUsed', { value: item.last_used_at || t('adminAccessKeys.neverUsed') }) }} {{ item.last_used_ip }}</p>
        <div class="space-y-1 rounded-lg bg-[var(--atelier-subtle)]/70 p-3 text-xs text-[var(--atelier-ink-secondary)]" :aria-label="t('adminAccessKeys.summaryLabel')">
          <p v-if="item.policy.full_access" class="font-semibold text-[var(--atelier-err)]">{{ t('adminAccessKeys.fullAccessLabel') }}</p>
          <p>{{ t('adminAccessKeys.summary', { globals: item.policy.global_permissions.length, rules: item.policy.rules.length }) }}</p>
          <p v-for="(rule, index) in item.policy.rules" :key="index">{{ t('adminAccessKeys.ruleLine', { index: index + 1, summary: ruleSummary(rule) }) }}</p>
        </div>
      </div>
    </div>

    <div v-if="draft" class="space-y-4 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-canvas)] p-5">
      <h3 class="text-base font-semibold text-[var(--atelier-ink)]">{{ editingID ? t('adminAccessKeys.editTitle') : t('adminAccessKeys.createTitle') }}</h3>
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
          <span>{{ t('adminAccessKeys.name') }}</span>
          <input v-model="draft.name" data-test="name" class="atelier-input mt-1.5" maxlength="100" />
        </label>
        <label class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
          <span>{{ t('adminAccessKeys.expiration') }}</span>
          <input v-model="expiration" type="datetime-local" class="atelier-input mt-1.5" />
        </label>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
          <span>{{ t('adminAccessKeys.description') }}</span>
          <textarea v-model="draft.description" class="atelier-input mt-1.5" rows="2" maxlength="1000" />
        </label>
        <label class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
          <span>{{ t('adminAccessKeys.allowedIps') }}</span>
          <textarea v-model="ips" class="atelier-input mt-1.5 font-mono text-xs" rows="2" placeholder="192.0.2.1&#10;2001:db8::/32" />
        </label>
      </div>
      <label class="inline-flex cursor-pointer items-center gap-2 text-sm text-[var(--atelier-ink)]">
        <input v-model="draft.enabled" type="checkbox" class="rounded border-[var(--atelier-border-strong)]" />
        {{ t('adminAccessKeys.enabled') }}
      </label>
      <div class="space-y-2 rounded-xl border border-[var(--atelier-err)]/30 bg-[var(--atelier-err-bg)]/50 p-3.5">
        <label class="inline-flex cursor-pointer items-center gap-2 text-sm font-medium text-[var(--atelier-ink)]">
          <input data-test="full-access" type="checkbox" :checked="draft.policy.full_access || fullAccessPrompt" :disabled="busy" @change="changeFullAccess" />
          {{ t('adminAccessKeys.fullAccessLabel') }}
        </label>
        <p class="text-xs text-[var(--atelier-err)]">{{ t('adminAccessKeys.fullAccessWarning') }}</p>
        <p v-if="draft.policy.full_access" class="text-xs text-[var(--atelier-ink-secondary)]">{{ t('adminAccessKeys.rulesSuspended') }}</p>
      </div>
      <fieldset :disabled="busy || !!draft.policy.full_access" class="space-y-4">
        <fieldset v-for="(rule, index) in draft.policy.rules" :key="index" class="space-y-3 rounded-xl border border-[var(--atelier-border)] bg-[var(--atelier-surface)] p-4">
          <legend class="px-1 text-xs font-semibold uppercase tracking-wider text-[var(--atelier-ink-muted)]">{{ t('adminAccessKeys.ruleLegend', { index: index + 1 }) }}</legend>
          <div class="flex flex-wrap gap-4">
            <label class="inline-flex cursor-pointer items-center gap-2 text-sm text-[var(--atelier-ink)]">
              <input v-model="rule.all_groups" type="checkbox" />
              {{ t('adminAccessKeys.allGroups') }}
            </label>
            <label class="inline-flex cursor-pointer items-center gap-2 text-sm text-[var(--atelier-ink)]">
              <input v-model="rule.include_ungrouped" type="checkbox" />
              {{ t('adminAccessKeys.includeUngrouped') }}
            </label>
          </div>
          <div>
            <p class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('adminAccessKeys.groups') }}</p>
            <div
              :data-test="`groups-${index}`"
              class="mt-1.5 max-h-40 overflow-y-auto rounded-lg border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-2.5"
              :class="{ 'pointer-events-none opacity-50': rule.all_groups }"
            >
              <div class="grid gap-1.5 sm:grid-cols-2 md:grid-cols-3">
                <label
                  v-for="group in groups"
                  :key="group.id"
                  class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-xs text-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]"
                >
                  <input
                    v-model="rule.group_ids"
                    type="checkbox"
                    :value="group.id"
                    :disabled="rule.all_groups"
                  />
                  <span class="truncate">{{ group.name }}</span>
                </label>
              </div>
              <p v-if="!groups.length" class="py-1 text-xs text-[var(--atelier-ink-muted)]">
                {{ t('adminAccessKeys.noGroups') }}
              </p>
            </div>
          </div>
          <div>
            <p class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('adminAccessKeys.accountTypes') }}</p>
            <div class="mt-1.5 flex flex-wrap gap-3">
              <label v-for="type in accountTypes" :key="type" class="inline-flex cursor-pointer items-center gap-1.5 text-xs text-[var(--atelier-ink)]">
                <input v-model="rule.account_types" type="checkbox" :value="type" />
                {{ type }}
              </label>
            </div>
          </div>
          <div>
            <p class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('adminAccessKeys.platforms') }}</p>
            <div class="mt-1.5 max-h-36 overflow-y-auto rounded-lg border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-2.5">
              <div class="grid gap-1.5 sm:grid-cols-2 md:grid-cols-3">
                <label
                  v-for="platform in CONCRETE_PLATFORM_OPTIONS"
                  :key="platform.value"
                  class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-xs text-[var(--atelier-ink)] hover:bg-[var(--atelier-hover)]"
                >
                  <input
                    v-model="rule.platforms"
                    type="checkbox"
                    :value="platform.value"
                  />
                  <span class="truncate">{{ platform.label }}</span>
                </label>
              </div>
            </div>
          </div>
          <div>
            <p class="text-xs font-medium text-[var(--atelier-ink-secondary)]">{{ t('adminAccessKeys.accountOperations') }}</p>
            <div class="mt-1.5 grid gap-2 sm:grid-cols-2">
              <label v-for="permission in accountPermissions" :key="permission.id" class="flex cursor-pointer items-center gap-2 text-xs text-[var(--atelier-ink)]">
                <input v-model="rule.permissions" type="checkbox" :value="permission.id" />
                <span>{{ permissionLabel(permission) }}</span>
                <span class="font-mono text-[11px] text-[var(--atelier-ink-muted)]">{{ permission.id }}</span>
              </label>
            </div>
          </div>
          <div v-if="groupPermissions.length" class="space-y-2 rounded-lg border border-[var(--atelier-warn)]/30 bg-[var(--atelier-warn-bg)]/40 p-3">
            <p class="text-xs font-medium text-[var(--atelier-ink)]">{{ t('adminAccessKeys.groupOperations') }}</p>
            <p class="text-xs text-[var(--atelier-warn)]">{{ t('adminAccessKeys.modelAllowlistHint') }}</p>
            <div class="grid gap-2 sm:grid-cols-2">
              <label v-for="permission in groupPermissions" :key="permission.id" class="flex cursor-pointer items-center gap-2 text-xs text-[var(--atelier-ink)]">
                <input v-model="rule.permissions" type="checkbox" :value="permission.id" />
                <span>{{ permissionLabel(permission) }}</span>
                <span class="font-mono text-[11px] text-[var(--atelier-ink-muted)]">{{ permission.id }}</span>
              </label>
            </div>
          </div>
          <UiButton variant="secondary" size="sm" @click="draft.policy.rules.splice(index, 1)">
            {{ t('adminAccessKeys.removeRule') }}
          </UiButton>
        </fieldset>
        <UiButton variant="secondary" size="sm" data-test="add-rule" @click="draft.policy.rules.push(emptyRule())">
          {{ t('adminAccessKeys.addRule') }}
        </UiButton>
        <fieldset class="space-y-3 rounded-xl border border-[var(--atelier-warn)]/35 bg-[var(--atelier-surface)] p-4">
          <legend class="px-1 text-xs font-semibold uppercase tracking-wider text-[var(--atelier-ink-muted)]">{{ t('adminAccessKeys.globalPermissions') }}</legend>
          <fieldset v-for="category in globalCategories" :key="category.name" class="space-y-2 rounded-lg border border-[var(--atelier-border)] p-3">
            <legend class="px-1 text-xs font-medium text-[var(--atelier-ink)]">{{ category.name }}</legend>
            <label v-for="permission in category.permissions" :key="permission.id" class="flex cursor-pointer items-center gap-2 text-xs text-[var(--atelier-ink)]">
              <input v-model="draft.policy.global_permissions" type="checkbox" :value="permission.id" />
              <span>{{ permissionLabel(permission) }}</span>
              <span class="font-mono text-[11px] text-[var(--atelier-ink-muted)]">{{ permission.id }}</span>
            </label>
          </fieldset>
        </fieldset>
      </fieldset>
      <div class="flex gap-2">
        <UiButton variant="primary" data-test="save" :disabled="busy || fullAccessPrompt" :loading="busy" @click="save">
          {{ busy ? t('adminAccessKeys.processing') : t('adminAccessKeys.save') }}
        </UiButton>
        <UiButton variant="secondary" :disabled="busy" @click="draft = null">
          {{ t('adminAccessKeys.cancel') }}
        </UiButton>
      </div>
    </div>

    <BaseDialog :show="fullAccessPrompt" :title="t('adminAccessKeys.fullAccessConfirmTitle')" width="normal" @close="cancelFullAccess">
      <p class="text-sm text-[var(--atelier-err)]">{{ t('adminAccessKeys.fullAccessWarning') }}</p>
      <label class="mt-4 block text-xs font-medium text-[var(--atelier-ink-secondary)]">
        <span>{{ t('adminAccessKeys.fullAccessPhraseHint', { phrase: t('adminAccessKeys.fullAccessPhrase') }) }}</span>
        <input v-model="fullAccessPhrase" data-test="full-access-phrase" class="atelier-input mt-1.5" autocomplete="off" />
      </label>
      <template #footer>
        <UiButton variant="secondary" data-test="full-access-cancel" @click="cancelFullAccess">{{ t('adminAccessKeys.cancel') }}</UiButton>
        <UiButton variant="danger" data-test="full-access-confirm" :disabled="fullAccessPhrase !== t('adminAccessKeys.fullAccessPhrase')" @click="confirmFullAccess">{{ t('adminAccessKeys.fullAccessConfirmTitle') }}</UiButton>
      </template>
    </BaseDialog>
    <TotpStepUpDialog :controller="stepUp" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as api from '@/api/admin/adminAccessKeys'
import type { AccessRule, AdminKey, AdminKeyInput, Permission } from '@/api/admin/adminAccessKeys'
import { getAllIncludingInactive } from '@/api/admin/groups'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { UiButton, UiStatusDot } from '@/ui'
import { useAdminAccessKeyPermissions } from '@/composables/useAdminAccessKeyPermissions'
import { isStepUpCancelled, useStepUp } from '@/composables/useStepUp'

const { t, locale } = useI18n()
const { permissionLabel, permissionCategory } = useAdminAccessKeyPermissions()
const stepUp = useStepUp()
const items = ref<AdminKey[]>([])
const permissions = ref<Permission[]>([])
const groups = ref<{ id: number; name: string }[]>([])
const loading = ref(false)
const loaded = ref(false)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const secret = ref('')
const draft = ref<AdminKeyInput | null>(null)
const editingID = ref('')
const expiration = ref('')
const initialExpiration = ref('')
const ips = ref('')
const fullAccessPrompt = ref(false)
const fullAccessPhrase = ref('')
const fullAccessConfirmed = ref(false)
// 切换语言后重新阅读当前语言的风险说明，不能复用尚未保存的扩权确认。
watch(locale, () => {
  if (fullAccessConfirmed.value && draft.value) draft.value.policy.full_access = false
  fullAccessConfirmed.value = false
  cancelFullAccess()
})
const accountTypes = ['oauth', 'setup-token', 'apikey', 'upstream', 'bedrock', 'service_account']
const accountPermissions = computed(() => permissions.value.filter(p => p.scope === 'account'))
const groupPermissions = computed(() => permissions.value.filter(p => p.scope === 'group'))
const globalCategories = computed(() => {
  const categories = new Map<string, Permission[]>()
  for (const permission of permissions.value.filter(p => p.scope === 'global')) {
    const name = permissionCategory(permission)
    categories.set(name, [...(categories.get(name) || []), permission])
  }
  return Array.from(categories, ([name, permissions]) => ({ name, permissions }))
})
// Go 的空切片可能序列化为 null；统一为空数组且不增加任何授权。
function normalizeItem(item: AdminKey): AdminKey {
  return { ...item, allowed_ips: item.allowed_ips ?? [], policy: {
    full_access: item.policy?.full_access ?? false,
    global_permissions: item.policy?.global_permissions ?? [],
    rules: (item.policy?.rules ?? []).map(rule => ({ ...rule,
      permissions: rule.permissions ?? [], group_ids: rule.group_ids ?? [],
      account_types: rule.account_types ?? [], platforms: rule.platforms ?? []
    }))
  } }
}
function ruleSummary(rule: AccessRule): string {
  const selectedGroups = rule.all_groups ? t('adminAccessKeys.allGroupsSummary') : rule.group_ids.map(id => groups.value.find(group => group.id === id)?.name || t('adminAccessKeys.groupFallback', { id })).join(t('adminAccessKeys.listSeparator')) || t('adminAccessKeys.noGroups')
  const groupLabel = selectedGroups + (rule.include_ungrouped ? t('adminAccessKeys.ungroupedSummary') : '')
  const types = rule.account_types.join(t('adminAccessKeys.listSeparator')) || t('adminAccessKeys.noAccountTypes')
  const platforms = rule.platforms.join(t('adminAccessKeys.listSeparator')) || t('adminAccessKeys.anyPlatform')
  return t('adminAccessKeys.ruleSummary', { groups: groupLabel, types, platforms, count: rule.permissions.length })
}
function emptyRule(): AccessRule { return { permissions: [], group_ids: [], all_groups: false, include_ungrouped: false, account_types: [], platforms: [] } }
function report(e: unknown) { if (!isStepUpCancelled(e)) error.value = (e as { message?: string })?.message || t('adminAccessKeys.operationFailed') }
async function load() {
  loading.value = true
  error.value = ''
  try {
    const [result, allGroups] = await Promise.all([api.list(), getAllIncludingInactive()])
    items.value = (result.items ?? []).map(normalizeItem)
    permissions.value = result.permissions ?? []
    groups.value = allGroups ?? []
    loaded.value = true
  } catch (e) { report(e) } finally { loading.value = false }
}
function edit(item?: AdminKey) {
  error.value = ''; notice.value = ''; secret.value = ''
  cancelFullAccess()
  fullAccessConfirmed.value = false
  editingID.value = item?.id || ''
  draft.value = item ? JSON.parse(JSON.stringify({ name: item.name, description: item.description, enabled: item.enabled, expires_at: item.expires_at, allowed_ips: item.allowed_ips, policy: item.policy })) : { name: '', description: '', enabled: true, expires_at: null, allowed_ips: [], policy: { rules: [emptyRule()], global_permissions: [] } }
  const date = item?.expires_at ? new Date(item.expires_at) : null
  expiration.value = date ? new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16) : ''
  initialExpiration.value = expiration.value
  ips.value = item?.allowed_ips.join('\n') || ''
}
function changeFullAccess(event: Event) {
  if (!draft.value || busy.value) return
  if ((event.target as HTMLInputElement).checked) {
    fullAccessPhrase.value = ''
    fullAccessPrompt.value = true
  } else {
    draft.value.policy.full_access = false
    fullAccessConfirmed.value = false
  }
}
function cancelFullAccess() {
  fullAccessPrompt.value = false
  fullAccessPhrase.value = ''
}
function confirmFullAccess() {
  if (!draft.value || fullAccessPhrase.value !== t('adminAccessKeys.fullAccessPhrase')) return
  draft.value.policy.full_access = true
  fullAccessConfirmed.value = true
  cancelFullAccess()
}
async function mutate(action: () => Promise<void>) {
  busy.value = true; error.value = ''; notice.value = ''; secret.value = ''
  try { await stepUp.run(action) } catch (e) { report(e) } finally { busy.value = false }
}
function replace(item: AdminKey) { items.value = [normalizeItem(item), ...items.value.filter(key => key.id !== item.id)] }
async function save() {
  if (!draft.value || busy.value || fullAccessPrompt.value) return
  if (!draft.value.name.trim()) { error.value = t('adminAccessKeys.nameRequired'); return }
  const date = expiration.value ? new Date(expiration.value) : null
  if (date && !Number.isFinite(date.getTime())) { error.value = t('adminAccessKeys.expirationInvalid'); return }
  const input: AdminKeyInput = JSON.parse(JSON.stringify(draft.value))
  input.confirm_full_access = !!input.policy.full_access && fullAccessConfirmed.value
  input.name = input.name.trim()
  // 未修改日期时保留服务端原值，避免丢失秒数、微秒或时区精度。
  if (expiration.value !== initialExpiration.value) input.expires_at = date?.toISOString() || null
  input.allowed_ips = ips.value.split(/[\n,]/).map(ip => ip.trim()).filter(Boolean)
  await mutate(async () => {
    if (editingID.value) replace(await api.update(editingID.value, input))
    else { const result = await api.create(input); replace(result.item); secret.value = result.key }
    draft.value = null
    notice.value = t('adminAccessKeys.saved')
  })
}
async function toggle(item: AdminKey) { await mutate(async () => { replace(await api.update(item.id, { name: item.name, description: item.description, enabled: !item.enabled, expires_at: item.expires_at, allowed_ips: item.allowed_ips, policy: item.policy })); notice.value = t('adminAccessKeys.statusUpdated') }) }
async function rotateKey(item: AdminKey) {
  if (!confirm(t('adminAccessKeys.rotateConfirm', { name: item.name }))) return
  await mutate(async () => { const result = await api.rotate(item.id); replace(result.item); secret.value = result.key; notice.value = t('adminAccessKeys.rotated') })
}
async function revokeKey(item: AdminKey) {
  if (!confirm(t('adminAccessKeys.revokeConfirm', { name: item.name }))) return
  await mutate(async () => { await api.revoke(item.id); replace({ ...item, enabled: false, revoked_at: new Date().toISOString() }); if (editingID.value === item.id) draft.value = null; notice.value = t('adminAccessKeys.revoked') })
}
async function deleteRecord(item: AdminKey) {
  if (busy.value || !item.revoked_at) return
  if (!confirm(t('adminAccessKeys.deleteConfirm', { name: item.name }))) return
  await mutate(async () => {
    await api.deleteRecord(item.id)
    items.value = items.value.filter(key => key.id !== item.id)
    if (editingID.value === item.id) draft.value = null
    notice.value = t('adminAccessKeys.removed')
  })
}
async function copySecret() { try { await navigator.clipboard.writeText(secret.value); notice.value = t('adminAccessKeys.copied') } catch { error.value = t('adminAccessKeys.copyFailed') } }
function status(item: AdminKey) { return item.revoked_at ? t('adminAccessKeys.statusRevoked') : !item.enabled ? t('adminAccessKeys.statusDisabled') : item.expires_at && new Date(item.expires_at).getTime() <= Date.now() ? t('adminAccessKeys.statusExpired') : t('adminAccessKeys.statusEnabled') }
function statusTone(item: AdminKey): 'ok' | 'warn' | 'err' | 'neutral' {
  if (item.revoked_at) return 'err'
  if (!item.enabled) return 'neutral'
  if (item.expires_at && new Date(item.expires_at).getTime() <= Date.now()) return 'warn'
  return 'ok'
}
function statusBadgeClass(item: AdminKey): string {
  const tone = statusTone(item)
  if (tone === 'ok') return 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]'
  if (tone === 'err') return 'bg-[var(--atelier-err-bg)] text-[var(--atelier-err)]'
  if (tone === 'warn') return 'bg-[var(--atelier-warn-bg)] text-[var(--atelier-warn)]'
  return 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-secondary)]'
}
onMounted(load)
</script>
