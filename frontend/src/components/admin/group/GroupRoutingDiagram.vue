<template>
  <div class="routing-diagram" role="img" :aria-label="t('admin.groups.modelRouting.title')">
    <div class="route-node route-request">
      <Icon name="send" size="sm" />
      <span>{{ t('console.routing.request', 'Request') }}</span>
    </div>
    <span class="route-line" aria-hidden="true"></span>
    <div class="route-node route-group" :title="group.description || undefined">
      <span class="font-medium">{{ group.name }}</span>
      <span class="font-mono text-[10px] opacity-70">{{ group.platform }} · {{ group.rate_multiplier }}×</span>
    </div>
    <span class="route-line" aria-hidden="true"></span>
    <div class="route-node route-policy">
      <span>{{ t('admin.groups.modelRouting.title') }}</span>
      <span class="font-mono text-[10px] opacity-70">{{ group.model_routing_enabled ? t('admin.groups.modelRouting.enabled') : t('admin.groups.modelRouting.disabled') }}</span>
    </div>
    <span class="route-line" aria-hidden="true"></span>
    <div class="route-node route-accounts" :title="accountTitle">
      <span>{{ t('admin.groups.modelRouting.accounts') }}</span>
      <span class="font-mono text-[10px] opacity-70">{{ group.active_account_count ?? 0 }} / {{ group.account_count ?? 0 }}</span>
    </div>
  </div>

  <div v-if="rules.length" class="mt-3 space-y-1.5">
    <div v-for="rule in rules" :key="rule.model" class="route-rule">
      <code>{{ rule.model }}</code>
      <Icon name="arrowRight" size="xs" class="text-gray-400" />
      <span class="font-mono text-xs">{{ rule.accounts.join(', ') }}</span>
    </div>
  </div>
  <p v-else class="mt-3 text-xs text-gray-500 dark:text-dark-400">{{ group.model_routing_enabled ? t('admin.groups.modelRouting.noRulesHint') : t('admin.groups.modelRouting.disabledHint') }}</p>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup } from '@/types'

const props = defineProps<{ group: AdminGroup }>()
const { t } = useI18n()
const rules = computed(() => Object.entries(props.group.model_routing ?? {}).map(([model, accounts]) => ({ model, accounts })))
const accountTitle = computed(() => `${props.group.active_account_count ?? 0} / ${props.group.account_count ?? 0}`)
</script>

<style scoped>
.routing-diagram { @apply flex min-w-max items-center gap-2 overflow-x-auto py-1 text-xs text-gray-700 dark:text-dark-200; }
.route-node { @apply flex min-w-[6.75rem] flex-col items-center gap-1 rounded-md border border-gray-200 bg-white px-3 py-2 text-center shadow-sm dark:border-dark-700 dark:bg-dark-800; }
.route-request { @apply min-w-[5rem]; }
.route-group { @apply border-primary-300 dark:border-primary-700; }
.route-line { @apply h-px w-7 shrink-0 bg-gray-300 dark:bg-dark-600; }
.route-rule { @apply flex items-center gap-2 rounded-md border border-gray-100 bg-gray-50/60 px-2.5 py-2 text-gray-700 dark:border-dark-800 dark:bg-dark-800/40 dark:text-dark-200; }
.route-rule code { @apply max-w-[13rem] truncate font-mono text-xs text-primary-700 dark:text-primary-300; }
</style>
