<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { clearSlowTTFT } from '@/api/admin/scheduling'
const props = defineProps<{ accountId: number; until?: string | null; reason?: string }>()
const emit = defineEmits<{ cleared: [] }>()
const { t } = useI18n()
const busy = ref(false)
const error = ref('')
async function clear() {
  busy.value = true; error.value = ''
  try { await clearSlowTTFT(props.accountId); emit('cleared') }
  catch { error.value = t('admin.accounts.slowTTFT.failed') }
  finally { busy.value = false }
}
</script>
<template>
  <div v-if="until && Date.parse(until) > Date.now()" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
    <p>{{ t('admin.accounts.slowTTFT.paused') }} · {{ t(`admin.accounts.slowTTFT.${reason === 'consecutive' ? 'consecutive' : 'window'}`) }}</p>
    <p>{{ t('admin.accounts.slowTTFT.until', { time: new Date(until).toLocaleString() }) }}</p>
    <button type="button" class="underline" :disabled="busy" @click="clear">{{ t('admin.accounts.slowTTFT.clear') }}</button>
    <p v-if="error" role="alert">{{ error }}</p>
  </div>
</template>
