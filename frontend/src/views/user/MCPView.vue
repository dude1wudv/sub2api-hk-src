<template>
  <div class="mx-auto max-w-[1040px] space-y-6 pb-10">
    <header>
      <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
        {{ t('mcp.title') }}
      </h1>
      <p class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
        {{ t('mcp.connectionDescription') }}
      </p>
    </header>

    <!-- MCP Endpoint Connection Card -->
    <section class="atelier-surface p-6" aria-labelledby="mcp-connection-title" :aria-busy="connectionLoading">
      <div class="flex items-start gap-3">
        <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] text-[var(--atelier-ink)]">
          <Icon name="mcp" size="md" />
        </div>
        <div>
          <h2 id="mcp-connection-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ t('mcp.connectionTitle') }}
          </h2>
          <p class="mt-0.5 text-sm text-[var(--atelier-ink-secondary)]">
            {{ t('mcp.connectionDescription') }}
          </p>
        </div>
      </div>

      <div class="mt-5">
        <label for="mcp-endpoint" class="mb-1.5 block text-xs font-medium text-[var(--atelier-ink-secondary)]">
          {{ t('mcp.endpoint') }}
        </label>
        <div class="flex flex-col gap-2 sm:flex-row">
          <input
            id="mcp-endpoint"
            :value="endpoint"
            :placeholder="connectionLoading ? t('mcp.connectionLoading') : ''"
            class="atelier-input min-w-0 flex-1 font-mono text-xs"
            readonly
          />
          <UiButton
            data-testid="copy-endpoint"
            variant="secondary"
            size="md"
            type="button"
            class="shrink-0"
            :disabled="!endpoint || connectionLoading"
            @click="copyToClipboard(endpoint)"
          >
            {{ t('mcp.copyEndpoint') }}
          </UiButton>
        </div>

        <div v-if="connectionError" class="mt-3 flex flex-wrap items-center gap-3">
          <p role="alert" class="text-xs text-[var(--atelier-err)]">{{ connectionError }}</p>
          <UiButton
            data-testid="retry-connection"
            variant="secondary"
            size="sm"
            type="button"
            :disabled="connectionLoading"
            @click="loadConnection"
          >
            {{ t('common.refresh') }}
          </UiButton>
        </div>

        <ol class="mt-5 list-decimal space-y-1.5 pl-5 text-sm text-[var(--atelier-ink-secondary)]">
          <li>{{ t('mcp.connectStep1') }}</li>
          <li>{{ t('mcp.connectStep2') }}</li>
          <li>{{ t('mcp.connectStep3') }}</li>
        </ol>
      </div>

      <p class="mt-5 border-t border-[var(--atelier-border)] pt-4 text-xs text-[var(--atelier-ink-muted)]">
        {{ t('mcp.capabilities') }}
      </p>
    </section>

    <!-- Authorized MCP Grants -->
    <section class="atelier-surface overflow-hidden" aria-labelledby="mcp-grants-title">
      <header class="flex flex-wrap items-start justify-between gap-4 border-b border-[var(--atelier-border)] px-6 py-4">
        <div class="max-w-xl">
          <h2 id="mcp-grants-title" class="text-[15px] font-semibold tracking-[-0.015em] text-[var(--atelier-ink)]">
            {{ t('mcp.grantsTitle') }}
          </h2>
          <p class="mt-0.5 text-xs text-[var(--atelier-ink-secondary)]">
            {{ t('mcp.grantsDescription') }}
          </p>
        </div>
        <UiButton
          data-testid="refresh-grants"
          variant="secondary"
          size="sm"
          type="button"
          :disabled="loading || revoking"
          @click="loadGrants"
        >
          {{ t('common.refresh') }}
        </UiButton>
      </header>

      <div class="p-6" :aria-busy="loading">
        <p v-if="loadError" role="alert" class="rounded-lg bg-[var(--atelier-err-bg)] px-3.5 py-2.5 text-xs text-[var(--atelier-err)]">
          {{ loadError }}
        </p>
        <p v-else-if="loading" role="status" class="py-10 text-center text-sm text-[var(--atelier-ink-muted)]">
          {{ t('mcp.loading') }}
        </p>
        <p
          v-else-if="grants.length === 0"
          class="rounded-xl border border-dashed border-[var(--atelier-border-strong)] px-4 py-10 text-center text-sm text-[var(--atelier-ink-muted)]"
        >
          {{ t('mcp.empty') }}
        </p>
        <ul v-else class="divide-y divide-[var(--atelier-border)]">
          <li
            v-for="grant in grants"
            :key="grant.id"
            class="flex items-start justify-between gap-4 py-4 first:pt-0 last:pb-0"
            :data-grant-id="grant.id"
          >
            <div class="min-w-0 space-y-1.5">
              <div class="flex flex-wrap items-center gap-2">
                <span class="break-all text-sm font-semibold text-[var(--atelier-ink)]">{{ grant.client_name }}</span>
                <span class="rounded-md border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2 py-0.5 text-[11px] font-medium text-[var(--atelier-ink-secondary)]">
                  {{ grant.scope === 'admin' ? t('mcp.adminScope') : t('mcp.userScope') }}
                </span>
                <span class="rounded-md border border-[var(--atelier-border)] bg-[var(--atelier-subtle)] px-2 py-0.5 text-[11px] font-medium text-[var(--atelier-ink-secondary)]">
                  {{ grant.read_only ? t('mcp.readOnly') : t('mcp.readWrite') }}
                </span>
                <span
                  class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium"
                  :class="isActive(grant) ? 'bg-[var(--atelier-ok-bg)] text-[var(--atelier-ok)]' : 'bg-[var(--atelier-subtle)] text-[var(--atelier-ink-muted)]'"
                >
                  <UiStatusDot :tone="isActive(grant) ? 'ok' : 'neutral'" />
                  {{ grantStatus(grant) }}
                </span>
              </div>
              <p class="truncate font-mono text-xs text-[var(--atelier-ink-muted)]" :title="grant.client_id">
                {{ grant.client_id }}
              </p>
              <div class="flex flex-wrap gap-x-4 gap-y-1 font-mono text-[11px] tabular-nums text-[var(--atelier-ink-muted)]">
                <span>{{ t('mcp.createdAt', { date: formatDate(grant.created_at) }) }}</span>
                <span>{{ t('mcp.expiresAt', { date: formatDate(grant.expires_at) }) }}</span>
                <span>{{ grant.last_used_at ? t('mcp.lastUsedAt', { date: formatDate(grant.last_used_at) }) : t('mcp.neverUsed') }}</span>
              </div>
            </div>
            <UiButton
              v-if="!grant.revoked_at"
              variant="ghost"
              size="sm"
              type="button"
              class="shrink-0 !text-[var(--atelier-err)] hover:!bg-[var(--atelier-err-bg)]"
              :aria-label="`${t('mcp.revoke')} ${grant.client_name}`"
              @click="revokeTarget = grant"
            >
              {{ t('mcp.revoke') }}
            </UiButton>
          </li>
        </ul>
      </div>
    </section>

    <BaseDialog
      :show="!!revokeTarget"
      :title="t('mcp.revokeTitle')"
      width="narrow"
      :close-on-escape="!revoking"
      :show-close-button="!revoking"
      @close="closeRevoke"
    >
      <p class="text-sm text-[var(--atelier-ink-secondary)]">
        {{ t('mcp.revokeConfirm', { name: revokeTarget?.client_name || '' }) }}
      </p>
      <p v-if="revokeError" role="alert" class="mt-3 text-xs text-[var(--atelier-err)]">{{ revokeError }}</p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UiButton variant="secondary" size="md" type="button" :disabled="revoking" @click="closeRevoke">
            {{ t('common.cancel') }}
          </UiButton>
          <UiButton
            data-testid="confirm-revoke"
            variant="danger"
            size="md"
            type="button"
            :disabled="revoking || loading"
            :loading="revoking"
            @click="revokeGrant"
          >
            {{ revoking ? t('common.processing') : t('mcp.revoke') }}
          </UiButton>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { mcpAPI, type MCPGrant } from '@/api/mcp'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { userFacingError } from '@/utils/userFacingError'
import { formatDate } from '@/utils/format'
import { UiButton, UiStatusDot } from '@/ui'

const { t, locale } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const endpoint = ref('')
const connectionLoading = ref(false)
const connectionError = ref('')
const grants = ref<MCPGrant[]>([])
const loading = ref(false)
const loadError = ref('')
const revokeTarget = ref<MCPGrant | null>(null)
const revoking = ref(false)
const revokeError = ref('')
let unmounted = false

function errorMessage(error: unknown, fallback: string): string {
  return userFacingError(error, locale.value, fallback)
}

async function loadConnection(): Promise<void> {
  if (connectionLoading.value) return
  connectionLoading.value = true
  connectionError.value = ''
  endpoint.value = ''
  try {
    const result = await mcpAPI.getConnectionConfig()
    const resource = new URL(result.resource)
    if (resource.protocol !== 'https:' || resource.username || resource.password || resource.search || resource.hash || resource.pathname !== '/api/v1/mcp') throw new Error('Invalid MCP resource')
    if (!unmounted) endpoint.value = resource.href
  } catch (error) {
    if (!unmounted) connectionError.value = errorMessage(error, t('mcp.connectionLoadFailed'))
  } finally {
    connectionLoading.value = false
  }
}

function isActive(grant: MCPGrant): boolean {
  return !grant.revoked_at && !grant.invalidated && new Date(grant.expires_at).getTime() > Date.now()
}

function grantStatus(grant: MCPGrant): string {
  if (grant.revoked_at) return t('mcp.revoked')
  if (grant.invalidated) return t('mcp.invalidated')
  return isActive(grant) ? t('mcp.active') : t('mcp.expired')
}

async function loadGrants(): Promise<void> {
  if (loading.value || revoking.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await mcpAPI.listGrants()
    if (!unmounted) grants.value = result
  } catch (error) {
    if (!unmounted) loadError.value = errorMessage(error, t('mcp.loadFailed'))
  } finally {
    loading.value = false
  }
}

function closeRevoke(): void {
  if (revoking.value) return
  revokeTarget.value = null
  revokeError.value = ''
}

async function revokeGrant(): Promise<void> {
  if (!revokeTarget.value || revoking.value || loading.value) return
  revoking.value = true
  revokeError.value = ''
  const id = revokeTarget.value.id
  try {
    await mcpAPI.revokeGrant(id)
    if (unmounted) return
    grants.value = grants.value.map(grant => grant.id === id ? { ...grant, revoked_at: new Date().toISOString() } : grant)
    revokeTarget.value = null
    appStore.showSuccess(t('mcp.revokeSuccess'))
  } catch (error) {
    if (!unmounted) revokeError.value = errorMessage(error, t('mcp.revokeFailed'))
  } finally {
    revoking.value = false
  }
}

onMounted(() => {
  void loadConnection()
  void loadGrants()
})
onBeforeUnmount(() => { unmounted = true })
</script>
