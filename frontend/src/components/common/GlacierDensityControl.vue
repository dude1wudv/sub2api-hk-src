<template>
  <div v-if="isGlassLayout" class="glacier-density" role="group" :aria-label="zh ? '表格密度' : 'Table density'">
    <button type="button" :aria-pressed="density === 'comfortable'" @click="setDensity('comfortable')">{{ zh ? '舒适' : 'Comfortable' }}</button>
    <button type="button" :aria-pressed="density === 'compact'" @click="setDensity('compact')">{{ zh ? '紧凑' : 'Compact' }}</button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGlacierPreferences } from '@/composables/useGlacierPreferences'
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const { density, isGlassLayout, setDensity } = useGlacierPreferences()
</script>
