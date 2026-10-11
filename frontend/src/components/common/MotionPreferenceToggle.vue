<template>
  <button type="button" class="motion-preference-toggle" :aria-label="label" :title="label" :aria-pressed="preference === 'reduced'" @click="setPreference(preference === 'reduced' ? 'auto' : 'reduced')">
    <svg class="motion-preference-toggle__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" aria-hidden="true">
      <path v-if="reducedMotion" d="M9 5v14M15 5v14" />
      <path v-else d="M3 12c3-8 6-8 9 0s6 8 9 0M3 18c3-8 6-8 9 0" />
    </svg>
    <span v-if="showLabel">{{ zh ? (preference === 'reduced' ? '减少动效' : '动效跟随系统') : (preference === 'reduced' ? 'Reduced motion' : 'System motion') }}</span>
  </button>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMotionPreferences } from '@/composables/useMotionPreferences'
withDefaults(defineProps<{ showLabel?: boolean }>(), { showLabel: false })
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const { preference, reducedMotion, setPreference } = useMotionPreferences()
const label = computed(() => zh.value
  ? preference.value === 'reduced' ? '动效已减少；切换为跟随系统' : reducedMotion.value ? '已跟随系统减少动效；始终减少动效' : '减少动态效果'
  : preference.value === 'reduced' ? 'Reduced motion; follow system instead' : reducedMotion.value ? 'System motion is reduced; always reduce motion' : 'Reduce motion')
</script>
<style scoped>
.motion-preference-toggle { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-width: 44px; min-height: 44px; padding: 8px; border: 1px solid transparent; border-radius: 999px; background: transparent; color: rgb(var(--ink-2)); font: inherit; cursor: pointer; transition: color var(--motion-fast,160ms), background-color var(--motion-fast,160ms); }
.motion-preference-toggle:hover, .motion-preference-toggle[aria-pressed='true'] { color: rgb(var(--ink)); background: rgb(var(--surface-2)); }
.motion-preference-toggle:focus-visible { outline: 2px solid rgb(var(--focus-ring)); outline-offset: 3px; }
.motion-preference-toggle__icon { width: 19px; height: 19px; }
</style>
