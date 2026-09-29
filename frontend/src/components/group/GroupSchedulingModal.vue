<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getGroupScheduling, saveGroupScheduling, type GroupScheduling } from '@/api/admin/scheduling'
const props = defineProps<{ groupId: number | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const settings = ref<GroupScheduling | null>(null)
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const levels = computed(() => [...new Set(settings.value?.accounts.map(a => a.priority))].sort((a, b) => a - b))
const rows = computed(() => [...(settings.value?.accounts ?? [])].sort((a, b) => a.priority - b.priority || a.account_id - b.account_id))
watch(() => props.groupId, async id => {
  settings.value = null; error.value = ''
  if (!id) return
  loading.value = true
  try { const data = await getGroupScheduling(id); if (props.groupId === id) settings.value = data }
  catch { if (props.groupId === id) error.value = t('admin.accounts.groupScheduling.failed') }
  finally { if (props.groupId === id) loading.value = false }
}, { immediate: true })
function moveLevel(priority: number, delta: number) {
  const other = levels.value[levels.value.indexOf(priority) + delta]
  if (other === undefined || !settings.value) return
  for (const row of settings.value.accounts) {
    if (row.priority === priority) row.priority = other
    else if (row.priority === other) row.priority = priority
  }
}
async function save() {
  if (!props.groupId || !settings.value) return
  if (settings.value.accounts.some(a => !Number.isInteger(a.priority) || a.priority < 0 || a.priority > 1000000)) {
    error.value = t('admin.accounts.groupScheduling.invalid'); return
  }
  saving.value = true; error.value = ''
  try { settings.value = await saveGroupScheduling(props.groupId, settings.value); emit('saved'); emit('close') }
  catch (e: unknown) {
    const status = (e as { status?: number }).status
    error.value = t(status === 409 ? 'admin.accounts.groupScheduling.conflict' : 'admin.accounts.groupScheduling.failed')
  } finally { saving.value = false }
}
</script>
<template>
  <BaseDialog :show="groupId !== null" :title="t('admin.accounts.groupScheduling.title')" width="wide" @close="!saving && emit('close')">
    <p v-if="loading">{{ t('common.loading') }}</p>
    <div v-if="settings" class="space-y-4">
      <p class="text-sm text-gray-500">{{ t('admin.accounts.groupScheduling.hint') }}</p>
      <div class="max-h-[55vh] space-y-3 overflow-auto">
        <section v-for="(level, index) in levels" :key="level" class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
          <div class="mb-2 flex items-center gap-2">
            <strong>{{ t('admin.accounts.priority') }} {{ level }}</strong>
            <button type="button" :disabled="index === 0" class="btn btn-secondary" :aria-label="t('admin.accounts.groupScheduling.up')" @click="moveLevel(level, -1)">↑</button>
            <button type="button" :disabled="index === levels.length - 1" class="btn btn-secondary" :aria-label="t('admin.accounts.groupScheduling.down')" @click="moveLevel(level, 1)">↓</button>
          </div>
          <div v-for="row in rows.filter(a => a.priority === level)" :key="row.account_id" class="flex items-center gap-3 py-2">
            <div class="min-w-0 flex-1"><p class="truncate">{{ row.name }} <span class="text-xs text-gray-500">#{{ row.account_id }}</span></p><p class="text-xs text-gray-500">{{ t('admin.accounts.loadFactor') }} {{ row.load_factor }} · {{ t('admin.accounts.concurrency') }} {{ row.concurrency }}</p></div>
            <input v-model.number.lazy="row.priority" class="input w-24" type="number" min="0" max="1000000" step="1" :aria-label="`${row.name} ${t('admin.accounts.priority')}`" />
          </div>
        </section>
      </div>
    </div>
    <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    <template #footer><div class="flex justify-end gap-2"><button class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button><button class="btn btn-primary" :disabled="loading || saving || !settings" @click="save">{{ t('common.save') }}</button></div></template>
  </BaseDialog>
</template>
