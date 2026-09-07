<template>
  <Teleport to="body">
    <div v-if="show" class="fixed inset-0 z-[80] bg-black/45" @click.self="emit('close')">
      <aside
        ref="panel"
        class="absolute right-0 top-0 flex h-full w-full max-w-xl flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900"
        role="dialog"
        aria-modal="true"
        :aria-label="account?.name"
        tabindex="-1"
        @keydown.esc="emit('close')"
      >
        <header class="flex min-h-16 items-center justify-between border-b border-gray-200 px-5 dark:border-dark-700">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="h-2 w-2 rounded-full" :class="account?.status === 'active' ? 'bg-emerald-500' : account?.status === 'error' ? 'bg-red-500' : 'bg-gray-400'" />
              <h2 class="truncate text-base font-semibold text-gray-950 dark:text-white">{{ account?.name }}</h2>
            </div>
            <p class="mt-0.5 font-mono text-xs text-gray-500 dark:text-dark-400">#{{ account?.id }} · {{ account?.platform }} / {{ account?.type }}</p>
          </div>
          <button type="button" class="btn btn-secondary p-2" :aria-label="t('common.close')" @click="emit('close')">
            <Icon name="x" size="sm" />
          </button>
        </header>

        <div v-if="account" class="min-h-0 flex-1 overflow-y-auto p-5">
          <section class="detail-section">
            <div class="detail-grid">
              <div><dt>{{ t('admin.accounts.columns.status') }}</dt><dd class="capitalize">{{ account.status }}</dd></div>
              <div><dt>{{ t('admin.accounts.columns.priority') }}</dt><dd class="font-mono">{{ account.priority }}</dd></div>
              <div><dt>{{ t('admin.accounts.columns.schedulable') }}</dt><dd>{{ account.schedulable ? t('common.enabled') : t('common.disabled') }}</dd></div>
              <div><dt>{{ t('admin.accounts.columns.lastUsed') }}</dt><dd>{{ account.last_used_at ? formatDateTime(account.last_used_at) : '—' }}</dd></div>
            </div>
          </section>

          <section class="detail-section">
            <h3>{{ t('admin.accounts.columns.capacity') }}</h3>
            <div class="detail-grid">
              <div><dt>{{ t('admin.accounts.columns.capacity') }}</dt><dd class="font-mono">{{ account.current_concurrency ?? 0 }} / {{ account.concurrency }}</dd></div>
              <div v-if="account.base_rpm != null"><dt>{{ t('admin.accounts.quotaControl.rpmLimit.label') }}</dt><dd class="font-mono">{{ account.current_rpm ?? 0 }} / {{ account.base_rpm }}</dd></div>
              <div v-if="account.active_sessions != null"><dt>{{ t('admin.accounts.quotaControl.sessionLimit.label') }}</dt><dd class="font-mono">{{ account.active_sessions }} / {{ account.max_sessions ?? '—' }}</dd></div>
              <div v-if="account.current_window_cost != null"><dt>{{ t('admin.accounts.quotaControl.windowCost.label') }}</dt><dd class="font-mono">{{ account.current_window_cost }} / {{ account.window_cost_limit ?? '—' }}</dd></div>
            </div>
          </section>

          <section v-if="todayStats" class="detail-section">
            <h3>{{ t('admin.accounts.stats.todayOverview') }}</h3>
            <div class="detail-grid">
              <div><dt>{{ t('admin.accounts.stats.requests') }}</dt><dd class="font-mono">{{ todayStats.requests }}</dd></div>
              <div><dt>{{ t('admin.accounts.stats.tokens') }}</dt><dd class="font-mono">{{ todayStats.tokens }}</dd></div>
              <div><dt>{{ t('admin.accounts.stats.cost') }}</dt><dd class="font-mono">{{ todayStats.cost }}</dd></div>
            </div>
          </section>

          <section v-if="account.groups?.length || account.scheduler_scores?.length" class="detail-section">
            <h3>{{ t('admin.accounts.columns.groups') }}</h3>
            <div class="space-y-2">
              <div v-for="group in account.groups" :key="group.id" class="detail-route-row">
                <span class="truncate">{{ group.name }}</span>
                <span class="font-mono text-xs">{{ group.rate_multiplier }}×</span>
              </div>
              <div v-for="score in account.scheduler_scores" :key="String(score.group_id)" class="detail-route-row">
                <span class="truncate">{{ score.group_name || `#${score.group_id}` }}</span>
                <span class="font-mono text-xs">{{ score.base_score }}</span>
              </div>
            </div>
          </section>

          <section v-if="account.error_message || account.rate_limit_reset_at || account.overload_until || account.temp_unschedulable_until" class="detail-section">
            <h3>{{ t('common.error') }}</h3>
            <p v-if="account.error_message" class="rounded-md border border-red-200 bg-red-50 p-3 text-xs leading-5 text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">{{ account.error_message }}</p>
            <dl class="mt-2 space-y-1 text-xs text-gray-600 dark:text-dark-300">
              <div v-if="account.rate_limit_reset_at">429 · {{ formatDateTime(account.rate_limit_reset_at) }}</div>
              <div v-if="account.overload_until">529 · {{ formatDateTime(account.overload_until) }}</div>
              <div v-if="account.temp_unschedulable_until">{{ account.temp_unschedulable_reason || t('admin.accounts.status.tempUnschedulable') }} · {{ formatDateTime(account.temp_unschedulable_until) }}</div>
            </dl>
          </section>

          <section v-if="account.notes || account.proxy || account.expires_at || account.custom_base_url_enabled || account.credentials_status" class="detail-section">
            <h3>{{ t('common.settings') }}</h3>
            <div class="detail-grid">
              <div v-if="account.proxy"><dt>{{ t('admin.accounts.columns.proxy') }}</dt><dd>{{ account.proxy.name }}</dd></div>
              <div v-if="account.expires_at"><dt>{{ t('admin.accounts.columns.expiresAt') }}</dt><dd>{{ formatDateTime(account.expires_at) }}</dd></div>
              <div v-if="account.custom_base_url_enabled"><dt>{{ t('admin.accounts.quotaControl.customBaseUrl.label') }}</dt><dd class="break-all font-mono text-xs">{{ account.custom_base_url }}</dd></div>
              <div v-if="account.notes" class="col-span-full"><dt>{{ t('admin.accounts.columns.notes') }}</dt><dd>{{ account.notes }}</dd></div>
            </div>
          </section>
        </div>

        <footer class="flex justify-end gap-2 border-t border-gray-200 px-5 py-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
          <button type="button" class="btn btn-primary" @click="emit('edit', account)">{{ t('common.edit') }}</button>
        </footer>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { Account, WindowStats } from '@/types'

const props = defineProps<{ show: boolean; account: Account | null; todayStats?: WindowStats | null }>()
const emit = defineEmits<{ close: []; edit: [account: Account] }>()
const { t } = useI18n()
const panel = ref<HTMLElement | null>(null)

watch(() => props.show, async (show) => {
  if (show) {
    await nextTick()
    panel.value?.focus()
  }
})
</script>

<style scoped>
.detail-section { @apply border-b border-gray-100 py-5 first:pt-0 last:border-b-0 dark:border-dark-800; }
.detail-section h3 { @apply mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400; }
.detail-grid { @apply grid grid-cols-2 gap-x-5 gap-y-3; }
.detail-grid dt { @apply text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500; }
.detail-grid dd { @apply mt-1 text-sm text-gray-800 dark:text-dark-100; }
.detail-route-row { @apply flex items-center justify-between gap-3 rounded-md border border-gray-100 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:text-dark-200; }
</style>
