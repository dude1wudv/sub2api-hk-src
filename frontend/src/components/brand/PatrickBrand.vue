<template>
  <span
    class="inline-flex shrink-0 items-center gap-2.5 leading-none text-[var(--atelier-ink)]"
    :aria-label="brandLabel"
  >
    <span
      class="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] text-[var(--atelier-ink)] shadow-[var(--atelier-shadow-xs)]"
      aria-hidden="true"
    >
      <BrandMark :size="16" />
    </span>
    <span
      v-if="!compact"
      class="font-sans text-[16px] font-semibold tracking-[-0.025em] text-[var(--atelier-ink)]"
    >
      {{ primaryWord }}<span v-if="secondaryWord" class="font-normal text-[var(--atelier-ink-muted)]">{{ secondaryWord }}</span>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getActivePinia } from 'pinia'
import { useAppStore } from '@/stores/app'
import BrandMark from '@/ui/icons/BrandMark.vue'

defineProps<{ compact?: boolean }>()

const appStore = getActivePinia() ? useAppStore() : null
const brandLabel = computed(() => appStore?.siteName || 'patrickapi')
const primaryWord = computed(() => {
  const raw = (appStore?.siteName || '').trim()
  if (!raw || raw.toLowerCase() === 'patrickapi') return 'patrick'
  return raw
})
const secondaryWord = computed(() => {
  const raw = (appStore?.siteName || '').trim()
  if (!raw || raw.toLowerCase() === 'patrickapi') return 'api'
  return ''
})
</script>
