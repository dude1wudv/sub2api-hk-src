<template>
  <span class="patrick-brand" :class="{ 'patrick-brand--compact': compact }" aria-label="patrickapi">
    <span class="patrick-brand__mark" aria-hidden="true">
      <svg viewBox="0 0 32 32" fill="none">
        <rect x="2.5" y="2.5" width="27" height="27" rx="8" fill="rgb(var(--action-bg))" />
        <path d="M11 22.5V10.5C11 9.4 11.9 8.5 13 8.5H17.2a4.8 4.8 0 0 1 0 9.6H11" stroke="rgb(var(--action-text))" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" />
        <circle cx="21.5" cy="21.5" r="2.2" fill="rgb(var(--brand-moss))" />
      </svg>
    </span>
    <span v-if="!compact" class="patrick-brand__name">patrick<span>api</span></span>
  </span>
</template>

<script setup lang="ts">
defineProps<{ compact?: boolean }>()
</script>

<style scoped>
.patrick-brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  flex-shrink: 0;
  color: rgb(var(--ink));
  line-height: 1;
}
.patrick-brand__mark {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
}
.patrick-brand__mark svg {
  width: 28px;
  height: 28px;
}
.patrick-brand__name {
  font-family: var(--font-sans);
  font-size: 19px;
  font-weight: 700;
  letter-spacing: -0.035em;
}
.patrick-brand__name > span {
  color: rgb(var(--brand-moss));
  font-weight: 550;
  margin-left: 1px;
}
</style>
