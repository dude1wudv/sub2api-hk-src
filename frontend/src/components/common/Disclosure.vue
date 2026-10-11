<template>
  <details v-bind="$attrs" class="disclosure" :open="expanded" @toggle="expanded = ($event.target as HTMLDetailsElement).open">
    <summary>{{ label }}</summary>
    <div class="disclosure-content"><slot /></div>
  </details>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
defineOptions({ inheritAttrs: false })
const props = defineProps<{ label: string; active?: boolean }>()
const expanded = ref(!!props.active)
watch(() => props.active, active => { if (active) expanded.value = true })
</script>

<style scoped>
.disclosure { min-width: 0; border: 1px solid rgb(var(--line)); border-radius: 12px; background: rgb(var(--surface)); }
.disclosure > summary { min-height: 44px; padding: 12px 16px; color: rgb(var(--ink-2)); font-size: 14px; font-weight: 500; cursor: pointer; transition: color var(--motion-fast) var(--motion-ease), background-color var(--motion-fast) var(--motion-ease); }
.disclosure > summary:hover { color: rgb(var(--ink)); background: rgb(var(--surface-2)); }
.disclosure > summary:focus-visible { outline: 2px solid rgb(var(--focus-ring)); outline-offset: 2px; border-radius: 10px; }
.disclosure-content { padding: 0 16px 16px; }
@media (prefers-reduced-motion: reduce) { .disclosure > summary { transition: none; } }
</style>
