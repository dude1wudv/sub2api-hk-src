<script setup lang="ts">
defineProps<{ eyebrow: string; title: string; description?: string }>()
</script>

<template>
  <header class="portal-page-header">
    <div class="portal-page-header-copy">
      <p class="portal-eyebrow"><span aria-hidden="true"></span>{{ eyebrow }}</p>
      <h1>{{ title }}</h1>
      <p v-if="description" class="portal-page-description">{{ description }}</p>
    </div>
    <div v-if="$slots.default" class="portal-page-header-actions"><slot /></div>
  </header>
</template>
