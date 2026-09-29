<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SlowTTFTConfig } from './slowTTFT'
const model = defineModel<SlowTTFTConfig>({ required: true })
const { t } = useI18n()
const fields = [
  { key: 'threshold_seconds', max: 3600 }, { key: 'consecutive_count', max: 1000 },
  { key: 'window_seconds', max: 86400 }, { key: 'window_count', max: 1000 },
  { key: 'pause_seconds', max: 604800 }
] as const
</script>
<template>
  <fieldset class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600" data-testid="slow-ttft-settings">
    <label class="flex items-center gap-2 font-medium">
      <input v-model="model.enabled" type="checkbox" class="checkbox" />
      {{ t('admin.accounts.slowTTFT.title') }}
    </label>
    <p class="input-hint">{{ t('admin.accounts.slowTTFT.hint') }}</p>
    <div v-if="model.enabled" class="grid grid-cols-2 gap-3">
      <label v-for="field in fields" :key="field.key" class="input-label">
        {{ t(`admin.accounts.slowTTFT.${field.key}`) }}
        <input v-model.number="model[field.key]" type="number" required min="1" :max="field.max" step="1" class="input mt-1" :data-testid="field.key" />
      </label>
    </div>
  </fieldset>
</template>
