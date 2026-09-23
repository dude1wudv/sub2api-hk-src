<template>
  <details v-if="isGlassLayout" v-bind="$attrs" class="glacier-disclosure" :open="expanded" @toggle="expanded = ($event.target as HTMLDetailsElement).open">
    <summary>{{ label }}</summary>
    <div class="glacier-disclosure-content"><slot /></div>
  </details>
  <slot v-else />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useGlacierPreferences } from '@/composables/useGlacierPreferences'
defineOptions({ inheritAttrs: false })
const props = defineProps<{ label: string; active?: boolean }>()
const { isGlassLayout } = useGlacierPreferences()
const expanded = ref(!!props.active)
watch(() => props.active, active => { if (active) expanded.value = true })
</script>
