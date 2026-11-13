<script setup lang="ts">
defineProps<{
  base: string
  adjusted?: string
  adjustedLabel?: string
  caption?: string
  unit?: string
}>()
</script>

<template>
  <span class="portal-model-price">
    <span class="portal-model-price-base"><span v-if="caption" class="portal-model-price-caption">{{ caption + ' ' }}</span>{{ base }}<span v-if="unit">{{ ' ' + unit }}</span></span>
    <span v-if="adjusted !== undefined" class="portal-model-price-adjusted">{{ adjustedLabel }} <strong>{{ adjusted }}</strong><span v-if="unit">{{ ' ' + unit }}</span></span>
  </span>
</template>

<style scoped>
.portal-model-price { display:flex; flex-direction:column; align-items:flex-end; gap:5px; font-variant-numeric:tabular-nums; }
.portal-model-price + .portal-model-price { margin-top:12px; }
.portal-model-price-base { color:rgb(var(--ink)); white-space:nowrap; font-size:13px; line-height:1.5; }
.portal-model-price-caption { color:rgb(var(--ink-3)); font-size:11px; }
.portal-model-price-adjusted { display:inline-flex; align-items:baseline; gap:4px; flex-wrap:wrap; justify-content:flex-end; padding:3px 6px; border:1px solid rgb(var(--brand-moss) / .2); border-radius:5px; background:rgb(var(--brand-moss) / .08); color:rgb(var(--brand-moss)); font-size:11px; line-height:1.5; }
.portal-model-price-adjusted strong { font-size:12px; font-weight:600; white-space:nowrap; }
</style>
