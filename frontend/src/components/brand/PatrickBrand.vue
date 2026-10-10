<template>
  <span class="patrick-brand" :class="{ 'patrick-brand--compact': compact }" aria-label="patrickapi">
    <span class="patrick-brand__mark" aria-hidden="true">
      <svg viewBox="0 0 32 32" fill="none">
        <path d="M10 25V9.5C10 7.6 11.6 6 13.5 6H18a7 7 0 0 1 0 14h-8" stroke="currentColor" stroke-width="2.7" stroke-linecap="round" />
        <path d="M15 11.5h3a1.5 1.5 0 0 1 0 3h-3" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" />
        <circle cx="23.5" cy="25" r="1.5" fill="currentColor" />
      </svg>
    </span>
    <span v-if="!compact" class="patrick-brand__name">patrick<span>api</span></span>
  </span>
</template>

<script setup lang="ts">
defineProps<{ compact?: boolean }>()
</script>

<style scoped>
.patrick-brand { display:inline-flex; align-items:center; gap:4px; flex-shrink:0; color:#000; line-height:1; }
.patrick-brand__mark { display:grid; place-items:center; width:28px; height:32px; }
.patrick-brand__mark svg { width:28px; height:28px; }
.patrick-brand__name { font-family:'Times New Roman',Times,serif; font-size:28px; font-weight:500; letter-spacing:-1px; }
.patrick-brand__name > span { color:inherit; font-weight:inherit; }
</style>
