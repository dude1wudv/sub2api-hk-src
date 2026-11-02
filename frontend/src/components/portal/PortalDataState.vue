<script setup lang="ts">
defineProps<{ loading?: boolean; error?: string; message?: string; retryLabel?: string; retryDisabled?: boolean }>()
defineEmits<{ retry: [] }>()
</script>

<template>
  <div :class="error ? 'portal-error' : 'portal-empty'" :role="error ? 'alert' : 'status'" :aria-busy="loading || undefined">
    <p>{{ error || message }}</p>
    <button v-if="error && retryLabel" type="button" class="portal-inline-button" :disabled="loading || retryDisabled" @click="$emit('retry')">{{ retryLabel }}</button>
    <slot />
  </div>
</template>
