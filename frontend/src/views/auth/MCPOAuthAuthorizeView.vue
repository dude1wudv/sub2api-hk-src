<template>
  <main class="min-h-screen bg-[var(--atelier-canvas)] px-4 py-8 sm:py-12">
    <section
      class="atelier-surface mx-auto max-w-2xl p-6 sm:p-8"
      aria-labelledby="mcp-consent-title"
      :aria-busy="loading || submitting"
    >
      <div class="mb-6 flex items-start gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] text-[var(--atelier-ink)]">
          <Icon name="mcp" size="md" />
        </div>
        <div>
          <h1 id="mcp-consent-title" class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
            {{ t('mcp.consentTitle') }}
          </h1>
          <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
            {{ t('mcp.consentDescription') }}
          </p>
        </div>
      </div>

      <p v-if="loading" role="status" class="py-10 text-center text-sm text-[var(--atelier-ink-muted)]">
        {{ t('mcp.consentLoading') }}
      </p>

      <div v-else-if="previewError" class="space-y-4">
        <p role="alert" class="rounded-lg border border-[var(--atelier-err)]/25 bg-[var(--atelier-err-bg)] px-4 py-3 text-sm text-[var(--atelier-err)]">
          {{ previewError }}
        </p>
        <p class="text-sm text-[var(--atelier-ink-secondary)]">{{ t('mcp.consentRetryHint') }}</p>
        <RouterLink to="/settings/mcp" class="atelier-btn atelier-btn-secondary h-9 px-4 text-sm">
          {{ t('mcp.backToConnections') }}
        </RouterLink>
      </div>

      <form v-else-if="consent" class="space-y-6" @submit.prevent="decide(true)">
        <div class="atelier-surface-subtle p-4 sm:p-5">
          <dl class="space-y-4 text-sm">
            <div>
              <dt class="text-xs text-[var(--atelier-ink-muted)]">{{ t('mcp.applicationName') }}</dt>
              <dd data-testid="client-name" class="mt-1 break-all text-base font-semibold text-[var(--atelier-ink)]">
                {{ consent.client_name }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-[var(--atelier-ink-muted)]">{{ t('mcp.clientId') }}</dt>
              <dd class="mt-1 truncate font-mono text-xs text-[var(--atelier-ink-secondary)]" :title="consent.client_id">
                {{ consent.client_id }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-[var(--atelier-ink-muted)]">{{ t('mcp.returnAddress') }}</dt>
              <dd data-testid="redirect-origin" class="mt-1 break-all font-semibold text-[var(--atelier-ink)]">
                {{ redirectOrigin }}
              </dd>
              <dd data-testid="redirect-uri" class="mt-1 break-all font-mono text-xs text-[var(--atelier-ink-secondary)]">
                {{ consent.redirect_uri }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-[var(--atelier-ink-muted)]">{{ t('mcp.requestedAccess') }}</dt>
              <dd class="mt-1 text-sm text-[var(--atelier-ink)]">{{ t('mcp.requestedAccessDescription') }}</dd>
              <dd class="mt-1 break-all font-mono text-xs text-[var(--atelier-ink-secondary)]">{{ consent.resource }}</dd>
            </div>
          </dl>
          <p class="mt-4 border-t border-[var(--atelier-border)] pt-3.5 text-xs text-[var(--atelier-warn)]">
            {{ t('mcp.untrustedApplicationHint') }}
          </p>
        </div>

        <p v-if="authStore.user?.email" class="break-all text-xs text-[var(--atelier-ink-secondary)]">
          {{ t('mcp.signedInAs', { email: authStore.user.email }) }}
        </p>

        <div class="space-y-1.5">
          <label for="mcp-consent-scope" class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ t('mcp.scope') }}
          </label>
          <UiSelect
            id="mcp-consent-scope"
            :model-value="scope"
            :options="scopeOptions"
            :disabled="submitting"
            :aria-label="t('mcp.scope')"
            @update:model-value="handleScopeChange"
          />
          <p id="mcp-scope-hint" class="text-xs text-[var(--atelier-ink-muted)]">
            {{ scope === 'admin' ? t('mcp.adminScopeHint') : t('mcp.userScopeHint') }}
          </p>
        </div>

        <div class="space-y-1.5">
          <label for="mcp-consent-permissions" class="block text-xs font-medium text-[var(--atelier-ink-secondary)]">
            {{ t('mcp.permissions') }}
          </label>
          <UiSelect
            id="mcp-consent-permissions"
            :model-value="readOnly"
            :options="permissionOptions"
            :disabled="submitting"
            :aria-label="t('mcp.permissions')"
            @update:model-value="handleReadOnlyChange"
          />
          <p
            id="mcp-permissions-hint"
            class="text-xs"
            :class="readOnly ? 'text-[var(--atelier-ink-muted)]' : 'text-[var(--atelier-warn)]'"
          >
            {{ readOnly ? t('mcp.readOnlyHint') : t('mcp.readWriteHint') }}
          </p>
          <p v-if="scope === 'admin' && !readOnly" data-testid="admin-write-hint" class="text-xs text-[var(--atelier-warn)]">
            {{ t('mcp.adminWriteHint') }}
          </p>
        </div>

        <p v-if="submitError" role="alert" class="rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]">
          {{ submitError }}
        </p>

        <div class="flex justify-end gap-2.5 border-t border-[var(--atelier-border)] pt-5">
          <UiButton data-testid="deny" variant="secondary" size="md" type="button" :disabled="submitting" @click="decide(false)">
            {{ t('mcp.deny') }}
          </UiButton>
          <UiButton data-testid="approve" variant="primary" size="md" type="submit" :disabled="submitting" :loading="submitting">
            {{ submitting ? t('common.processing') : t('mcp.approve') }}
          </UiButton>
        </div>
      </form>
    </section>

    <TotpStepUpDialog :controller="stepUp" />
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { mcpAPI, type MCPAccessScope, type MCPAuthorization, type MCPConsent } from '@/api/mcp'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { Icon } from '@/components/icons'
import { isStepUpBlocked, isStepUpCancelled, stepUpBlockReason, useStepUp } from '@/composables/useStepUp'
import { useAuthStore } from '@/stores/auth'
import { UiButton, UiSelect, type UiSelectOption } from '@/ui'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const stepUp = useStepUp()
const loading = ref(true)
const submitting = ref(false)
const previewError = ref('')
const submitError = ref('')
const consent = ref<MCPConsent | null>(null)
const scope = ref<MCPAccessScope>('user')
const readOnly = ref(true)
let authorization: MCPAuthorization | null = null
let generation = 0
const redirectOrigin = computed(() => consent.value ? new URL(consent.value.redirect_uri).origin : '')

const scopeOptions = computed<UiSelectOption[]>(() => {
  const opts: UiSelectOption[] = [{ value: 'user', label: t('mcp.userScope') }]
  if (authStore.isAdmin) opts.push({ value: 'admin', label: t('mcp.adminScope') })
  return opts
})

const permissionOptions = computed<UiSelectOption[]>(() => [
  { value: true, label: t('mcp.readOnly') },
  { value: false, label: t('mcp.readWrite') },
])

function handleScopeChange(val: string | number | boolean | null) {
  scope.value = val === 'admin' && authStore.isAdmin ? 'admin' : 'user'
}

function handleReadOnlyChange(val: string | number | boolean | null) {
  readOnly.value = Boolean(val)
}

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: unknown } | null)?.message
  return typeof message === 'string' && message ? message : fallback
}

function parseAuthorization(): MCPAuthorization {
  const required = ['response_type', 'client_id', 'redirect_uri', 'code_challenge', 'code_challenge_method', 'resource'] as const
  const allowed = [...required, 'scope', 'state']
  const values: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (!allowed.includes(key) || typeof value !== 'string' || value.length > 4096) {
      throw new Error(t('mcp.invalidRequest'))
    }
    values[key] = value
  }
  if (required.some(key => !values[key]) || values.response_type !== 'code' ||
    values.code_challenge_method !== 'S256' || !/^[A-Za-z0-9_-]{43}$/.test(values.code_challenge) ||
    (values.scope !== undefined && values.scope !== '' && values.scope !== 'mcp')) {
    throw new Error(t('mcp.invalidRequest'))
  }
  return values as unknown as MCPAuthorization
}

function validatePreview(result: MCPConsent, request: MCPAuthorization): void {
  if (result.client_id !== request.client_id || result.redirect_uri !== request.redirect_uri ||
    result.resource !== request.resource || result.scope !== 'mcp' || typeof result.client_name !== 'string') {
    throw new Error(t('mcp.invalidRequest'))
  }
  const redirect = new URL(result.redirect_uri)
  if (!['https:', 'http:'].includes(redirect.protocol) || redirect.username || redirect.password || redirect.hash) {
    throw new Error(t('mcp.invalidRequest'))
  }
}

async function loadConsent(): Promise<void> {
  const current = ++generation
  stepUp.onCancel()
  loading.value = true
  submitting.value = false
  previewError.value = ''
  submitError.value = ''
  consent.value = null
  authorization = null
  scope.value = 'user'
  readOnly.value = true
  try {
    const request = parseAuthorization()
    const result = await mcpAPI.previewAuthorization({ ...request })
    if (current !== generation) return
    validatePreview(result, request)
    authorization = Object.freeze({ ...request })
    consent.value = result
  } catch (error) {
    if (current === generation) previewError.value = errorMessage(error, t('mcp.consentLoadFailed'))
  } finally {
    if (current === generation) loading.value = false
  }
}

async function decide(approved: boolean): Promise<void> {
  if (loading.value || submitting.value || !consent.value || !authorization) return
  const current = generation
  const expectedRedirect = new URL(consent.value.redirect_uri)
  const input = {
    authorization: { ...authorization },
    approved,
    scope: approved && authStore.isAdmin ? scope.value : 'user' as const,
    read_only: approved ? readOnly.value : true
  }
  submitting.value = true
  submitError.value = ''
  try {
    const action = async () => {
      if (current !== generation) throw new Error(t('mcp.invalidRequest'))
      try {
        return await mcpAPI.authorize(input)
      } catch (error) {
        if (current !== generation) throw new Error(t('mcp.invalidRequest'))
        throw error
      }
    }
    const result = approved && input.scope === 'admin' ? await stepUp.run(action) : await action()
    if (current !== generation) return
    const destination = new URL(result.redirect_url)
    if (destination.origin !== expectedRedirect.origin || destination.pathname !== expectedRedirect.pathname ||
      destination.username || destination.password || destination.hash) {
      throw new Error(t('mcp.invalidRedirect'))
    }
    window.location.assign(result.redirect_url)
  } catch (error) {
    if (current !== generation) return
    if (!isStepUpCancelled(error)) {
      submitError.value = isStepUpBlocked(error)
        ? stepUpBlockReason(error) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN' ? t('stepUp.adminApiKeyForbidden') : t('stepUp.notEnabled')
        : errorMessage(error, t('mcp.consentSubmitFailed'))
    }
    submitting.value = false
  }
}

watch(() => route.fullPath, loadConsent, { immediate: true })
onBeforeUnmount(() => {
  generation++
  authorization = null
  stepUp.onCancel()
})
</script>
