<template>
  <label class="appearance-select">
    <Icon :name="isDark ? 'moon' : 'sun'" size="sm" aria-hidden="true" />
    <select :value="mode" :aria-label="t('console.appearance')" @change="setMode(($event.target as HTMLSelectElement).value as ThemeMode)">
      <option value="dark">{{ t('nav.darkMode') }}</option>
      <option value="light">{{ t('nav.lightMode') }}</option>
      <option value="system">{{ t('console.systemTheme') }}</option>
    </select>
  </label>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppearance, type ThemeMode } from '@/composables/useAppearance'
const { t } = useI18n()
const { mode, isDark, setMode } = useAppearance()
</script>
