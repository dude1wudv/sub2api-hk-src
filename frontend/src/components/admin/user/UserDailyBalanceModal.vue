<template>
  <BaseDialog :show="show" :title="t('admin.users.grantDailyBalance')" width="narrow" @close="$emit('close')">
    <form v-if="user" id="daily-balance-form" @submit.prevent="handleSubmit" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100"><span class="text-lg font-medium text-primary-700">{{ user.email.charAt(0).toUpperCase() }}</span></div>
        <div class="flex-1"><p class="font-medium text-gray-900">{{ user.email }}</p></div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.dailyBalanceGroup') }}</label>
        <select v-model="form.groupId" required class="input">
          <option value="" disabled>{{ dailyGroups.length ? t('admin.users.selectDailyGroup') : t('admin.users.noDailyBalanceGroups') }}</option>
          <option v-for="g in dailyGroups" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.dailyBalanceAmount') }}</label>
        <div class="relative flex-1"><div class="absolute left-3 top-1/2 -translate-y-1/2 font-medium text-gray-500">$</div><input v-model.number="form.amount" type="number" step="any" min="0" required class="input pl-8" :placeholder="t('admin.users.dailyBalanceAmountPlaceholder')" /></div>
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.users.dailyBalanceAmountHint') }}</p>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="daily-balance-form" :disabled="submitting || !form.groupId || !form.amount" class="btn bg-orange-600 text-white">{{ submitting ? t('common.saving') : t('common.confirm') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminUser, AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{ show: boolean, user: AdminUser | null, allGroups: AdminGroup[] }>()
const emit = defineEmits(['close', 'success'])
const { t } = useI18n()
const appStore = useAppStore()

const submitting = ref(false)
const form = reactive({ groupId: null as number | null, amount: 0 })
watch(() => props.show, (v) => { if(v) { form.groupId = null; form.amount = 0 } })

const dailyGroups = computed(() => props.allGroups.filter(g => g.daily_balance_enabled && g.is_exclusive))

const handleSubmit = async () => {
  if (!props.user || !form.groupId || !form.amount) return
  submitting.value = true
  try {
    await adminAPI.users.grantUserDailyBalance(props.user.id, form.groupId, form.amount)
    appStore.showSuccess(t('admin.users.dailyBalanceGranted'))
    emit('success')
    emit('close')
  } catch (e: any) {
    console.error('Failed to grant daily balance:', e)
    appStore.showError(e.response?.data?.detail || t('admin.users.failedToGrantDailyBalance'))
  } finally { submitting.value = false }
}
</script>
