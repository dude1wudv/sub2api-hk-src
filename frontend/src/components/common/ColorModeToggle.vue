<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useColorMode } from '@/composables/useColorMode'

const { locale } = useI18n()
const { isDark, toggleMode } = useColorMode()
const label = computed(() =>
  locale.value.startsWith('zh')
    ? isDark.value
      ? '切换至浅色'
      : '切换至深色'
    : isDark.value
      ? 'Switch to light mode'
      : 'Switch to dark mode'
)
</script>

<template>
  <button
    type="button"
    class="color-mode-toggle inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
    :aria-label="label"
    :title="label"
    :aria-pressed="isDark"
    @click="toggleMode"
  >
    <Icon :name="isDark ? 'sun' : 'moon'" size="sm" aria-hidden="true" />
  </button>
</template>
