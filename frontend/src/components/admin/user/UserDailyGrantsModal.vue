<template>
  <BaseDialog :show="show" :title="t('admin.users.dailyGrants')" width="wide" @close="$emit('close')">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100"><span class="text-lg font-medium text-primary-700">{{ user.email.charAt(0).toUpperCase() }}</span></div>
        <div class="flex-1"><p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p></div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
      <div v-else-if="grants.length === 0" class="py-12 text-center text-gray-500">{{ t('admin.users.noDailyGrants') }}</div>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="border-b border-gray-200 dark:border-dark-600">
            <tr class="text-left">
              <th class="pb-2 font-medium text-gray-700 dark:text-gray-300">{{ t('admin.users.dailyGrantAmount') }}</th>
              <th class="pb-2 font-medium text-gray-700 dark:text-gray-300">{{ t('admin.users.dailyGrantRemaining') }}</th>
              <th class="pb-2 font-medium text-gray-700 dark:text-gray-300">{{ t('admin.users.dailyGrantStatus') }}</th>
              <th class="pb-2 font-medium text-gray-700 dark:text-gray-300">{{ t('admin.users.grantedAt') }}</th>
              <th class="pb-2 font-medium text-gray-700 dark:text-gray-300">{{ t('admin.users.expiresAt') }}</th>
              <th class="pb-2 font-medium text-gray-700 dark:text-gray-300">{{ t('admin.users.dailyGrantSource') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="g in grants" :key="g.id" class="border-b border-gray-100 dark:border-dark-700">
              <td class="py-2 font-mono text-gray-900 dark:text-white">${{ g.amount.toFixed(2) }}</td>
              <td class="py-2 font-mono text-gray-700 dark:text-gray-300">${{ g.remaining.toFixed(2) }}</td>
              <td class="py-2"><span :class="statusClass(g.status)" class="inline-block rounded px-2 py-0.5 text-xs font-medium">{{ statusLabel(g.status) }}</span></td>
              <td class="py-2 text-gray-600 dark:text-gray-400">{{ formatDate(g.granted_at) }}</td>
              <td class="py-2 text-gray-600 dark:text-gray-400">{{ formatDate(g.expires_at) }}</td>
              <td class="py-2 text-gray-600 dark:text-gray-400">{{ g.source }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-secondary">{{ t('common.close') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminUser, DailyBalanceGrant } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const props = defineProps<{ show: boolean, user: AdminUser | null }>()
defineEmits(['close'])
const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const grants = ref<DailyBalanceGrant[]>([])

watch(() => props.show, async (v) => {
  if (v && props.user) {
    loading.value = true
    try {
      const data = await adminAPI.users.getUserDailyGrants(props.user.id)
      grants.value = data.grants
    } catch (e: any) {
      console.error('Failed to load daily grants:', e)
      appStore.showError(e.response?.data?.detail || t('common.error'))
      grants.value = []
    } finally { loading.value = false }
  }
})

const statusClass = (status: string) => {
  if (status === 'active') return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
  if (status === 'exhausted') return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
  return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
}

const statusLabel = (status: string) => {
  const labels: Record<string, string> = { active: '有效', exhausted: '已用尽', expired: '已过期' }
  return labels[status] || status
}

const formatDate = (iso: string) => new Date(iso).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
</script>
