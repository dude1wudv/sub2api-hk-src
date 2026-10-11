<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useColorMode } from '@/composables/useColorMode'

const { locale } = useI18n()
const { isDark, toggleMode } = useColorMode()
const label = computed(() => locale.value.startsWith('zh')
  ? (isDark.value ? '切换至浅色' : '切换至深色')
  : (isDark.value ? 'Switch to light mode' : 'Switch to dark mode'))
</script>

<template>
  <button type="button" class="color-mode-toggle" :aria-label="label" :title="label" :aria-pressed="isDark" @click="toggleMode">
    <Icon :name="isDark ? 'sun' : 'moon'" size="md" aria-hidden="true" />
  </button>
</template>
