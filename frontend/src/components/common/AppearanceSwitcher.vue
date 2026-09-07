<template>
  <div class="appearance-controls">
    <label class="appearance-select">
      <Icon name="sparkles" size="sm" aria-hidden="true" />
      <select :value="style" :aria-label="zh ? '主题风格' : 'Theme style'" @change="setStyle(($event.target as HTMLSelectElement).value as ThemeStyle)">
        <option value="aurora">{{ zh ? '星澜' : 'Astra' }}</option>
        <option value="lagoon">{{ zh ? '苔砂' : 'Grove' }}</option>
        <option value="graphite">{{ zh ? '墨铜' : 'Ember' }}</option>
      </select>
    </label>
    <button class="appearance-mode" :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')" :title="isDark ? t('nav.lightMode') : t('nav.darkMode')" @click="toggleTheme">
      <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppearance, type ThemeStyle } from '@/composables/useAppearance'
const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const { style, isDark, setStyle, toggleTheme } = useAppearance()
</script>
